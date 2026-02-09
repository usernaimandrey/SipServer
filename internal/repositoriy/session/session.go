package session

import (
	"SipServer/internal/repositoriy"
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
	Id            int
	JoornalId     int
	CallID        string
	FromTag       string
	ToTag         string
	SessionState  SessionState
	RemoteTarget  string
	RouteSet      interface{}
	CreatedAt     time.Time
	EstablishedAt time.Time
	TerminatedAt  time.Time
	EndedBy       repositoriy.CallEndedBy
	TermCode      int
	TermReason    string
	UpdatedAt     time.Time
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
