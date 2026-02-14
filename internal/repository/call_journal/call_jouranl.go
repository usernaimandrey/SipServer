package calljournal

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"SipServer/internal/repository"
)

type CallResult string

const (
	CallResultAnswered  CallResult = "answered"
	CallResultRejected  CallResult = "rejected"
	CallResultCancelled CallResult = "cancelled"
	CAllResultNoAnswer  CallResult = "no_answer"
	CallResultFailed    CallResult = "failed"
)

var ErrNotFound = errors.New("cdr: not found")

type CallJournal struct {
	Id          int
	CallId      string
	InitBranch  string
	FromTag     string
	ToTag       string
	CallerUser  string
	CaleeUser   string
	CalerUri    string
	CaleeURi    string
	InviteAt    time.Time
	First18xAt  time.Time
	AnswerAt    time.Time
	EndAt       time.Time
	Result      CallResult
	FinalCode   int
	FinalReason string
	RingMs      int
	TalkMs      int
	EndedBy     repository.CallEndedBy
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewCallJournal() *CallJournal {
	return &CallJournal{}
}

type CallJournalRepo struct {
	DB *sql.DB
}

func NewCallJournalRepo(db *sql.DB) *CallJournalRepo {
	return &CallJournalRepo{DB: db}
}

func (r *CallJournalRepo) StartCallAttempt(
	ctx context.Context,
	callID string,
	initBranch string,
	callerUser string,
	calleeUser string,
	inviteAt time.Time,
) (int64, error) {

	const q = `
		INSERT INTO call_journals (
			call_id, init_branch, caller_user, callee_user, invite_at
		)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id
	`

	var id int64
	err := r.DB.QueryRowContext(
		ctx, q,
		callID,
		repository.NullIfEmpty(initBranch),
		callerUser,
		calleeUser,
		inviteAt,
	).Scan(&id)

	return id, err
}

func (r *CallJournalRepo) MarkRejected(
	ctx context.Context,
	journalID int64,
	code int,
	reason string,
	endAt time.Time,
) error {

	const q = `
		UPDATE call_journals
		SET
			end_at       = COALESCE(end_at, $2),
			result       = COALESCE(result, 'rejected'),
			final_code   = COALESCE(final_code, $3),
			final_reason = COALESCE(final_reason, $4),
			ended_by     = COALESCE(ended_by, 'callee')
		WHERE id = $1
	`
	_, err := r.DB.ExecContext(ctx, q, journalID, endAt, code, repository.NullIfEmpty(reason))
	return err
}

func (r *CallJournalRepo) MarkCancelled(
	ctx context.Context,
	journalID int64,
	endAt time.Time,
) error {

	const q = `
		UPDATE call_journals
		SET
			end_at       = COALESCE(end_at, $2),
			result       = COALESCE(result, 'cancelled'),
			final_code   = COALESCE(final_code, 487),
			final_reason = COALESCE(final_reason, 'Request Terminated'),
			ended_by     = COALESCE(ended_by, 'caller')
		WHERE id = $1
	`
	_, err := r.DB.ExecContext(ctx, q, journalID, endAt)
	return err
}

func (r *CallJournalRepo) MarkAnswered(
	ctx context.Context,
	journalID int64,
	callID, fromTag, toTag string,
	remoteTarget string,
	routeSetJSON []byte,
	answerAt time.Time,
	ringMs int,
) error {

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1) Обновляем журнал (один раз)
	const qJournal = `
		UPDATE call_journals
		SET
			answer_at    = COALESCE(answer_at, $2),
			ring_ms   	 = COALESCE(ring_ms, $3),
			result       = COALESCE(result, 'answered'),
			final_code   = COALESCE(final_code, 200),
			final_reason = COALESCE(final_reason, 'OK')
		WHERE id = $1
	`
	if _, err = tx.ExecContext(ctx, qJournal, journalID, answerAt, ringMs); err != nil {
		return err
	}

	// 2) Создаём / обновляем сессию
	const qSession = `
		INSERT INTO call_sessions (
			journal_id, call_id, from_tag, to_tag,
			state, remote_target, route_set,
			created_at, established_at
		)
		VALUES (
			$1,$2,$3,$4,
			'active', $5, $6,
			$7, $7
		)
		ON CONFLICT (call_id, from_tag, to_tag)
		DO UPDATE SET
			state = CASE
				WHEN call_sessions.state = 'terminated' THEN call_sessions.state
				ELSE 'active'
			END,
			remote_target  = COALESCE(call_sessions.remote_target, EXCLUDED.remote_target),
			route_set      = COALESCE(call_sessions.route_set, EXCLUDED.route_set),
			established_at = COALESCE(call_sessions.established_at, EXCLUDED.established_at)
	`
	if _, err = tx.ExecContext(
		ctx,
		qSession,
		journalID,
		callID,
		fromTag,
		toTag,
		repository.NullIfEmpty(remoteTarget),
		routeSetJSON,
		answerAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *CallJournalRepo) EndByBye(
	ctx context.Context,
	callID, fromTag, toTag string,
	endedBy repository.CallEndedBy,
	endAt time.Time,
	talkMs int,
) error {

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1) Завершаем сессию (только если ещё активна)
	const qSession = `
		UPDATE call_sessions
		SET
			state         = 'terminated',
			terminated_at = COALESCE(terminated_at, $4),
			ended_by      = COALESCE(ended_by, $5),
			term_code     = COALESCE(term_code, 200),
			term_reason   = COALESCE(term_reason, 'BYE')
		WHERE call_id = $1
		  AND from_tag = $2
		  AND to_tag = $3
		  AND state <> 'terminated'
		RETURNING journal_id
	`

	var journalID int64
	err = tx.QueryRowContext(
		ctx, qSession,
		callID, fromTag, toTag, endAt, string(endedBy),
	).Scan(&journalID)

	if err == sql.ErrNoRows {
		// already terminated or session missing — идемпотентно
		return tx.Commit()
	}
	if err != nil {
		return err
	}

	// 2) Закрываем журнал
	const qJournal = `
		UPDATE call_journals
		SET
			end_at   = COALESCE(end_at, $2),
			ended_by = COALESCE(ended_by, $3),
			final_code = COALESCE(final_code, 200),
      talk_ms = COALESCE(talk_ms, $4)
		WHERE id = $1
	`
	if _, err = tx.ExecContext(ctx, qJournal, journalID, endAt, string(endedBy), talkMs); err != nil {
		return err
	}

	return tx.Commit()
}
