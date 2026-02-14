package session

import (
	"SipServer/internal/repository"
	"database/sql"
	"time"
)

type SessionState string

const (
	SessionStateActive     SessionState = "active"
	SessionStateEarly      SessionState = "early"
	SessionStateTerminated SessionState = "terminated"
)

type Session struct {
	Id            int                     `json:"id"`
	JournalId     int                     `json:"journal_id"`
	CallID        string                  `json:"call_id"`
	FromTag       string                  `json:"from_tag,omitempty"`
	ToTag         string                  `json:"to_tag,omitempty"`
	SessionState  SessionState            `json:"session_state"`
	RemoteTarget  string                  `json:"remote_target,omitempty"`
	RouteSet      interface{}             `json:"route_set,omitempty"`
	CreatedAt     *time.Time              `json:"created_at"`
	EstablishedAt *time.Time              `json:"established_at,omitempty"`
	TerminatedAt  *time.Time              `json:"terminated_at,omitempty"`
	EndedBy       *repository.CallEndedBy `json:"ended_by,omitempty"`
	TermCode      sql.NullInt64           `json:"term_code,omitempty"`
	TermReason    sql.NullString          `json:"term_reason,omitempty"`
	UpdatedAt     time.Time               `json:"updated_at"`
}

func NewSession() *Session {
	return &Session{}
}

type SessionRepo struct {
	DB *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{DB: db}
}

func (r *SessionRepo) List() ([]Session, error) {
	query := `
SELECT
	id,
	journal_id,
	call_id,
	from_tag,
	to_tag,
	state,
	remote_target,
	route_set,
	created_at,
	established_at,
	terminated_at,
	ended_by,
	term_code,
	term_reason,
	updated_at
FROM call_sessions
`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	var (
		est  sql.NullTime
		term sql.NullTime
	)
	for rows.Next() {

		var s Session
		if err := rows.Scan(
			&s.Id,
			&s.JournalId,
			&s.CallID,
			&s.FromTag,
			&s.ToTag,
			&s.SessionState,
			&s.RemoteTarget,
			&s.RouteSet,
			&s.CreatedAt,
			&est,
			&term,
			&s.EndedBy,
			&s.TermCode,
			&s.TermReason,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if est.Valid {
			t := est.Time
			s.EstablishedAt = &t
		}
		if term.Valid {
			t := term.Time
			s.TerminatedAt = &t
		}

		sessions = append(sessions, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}
