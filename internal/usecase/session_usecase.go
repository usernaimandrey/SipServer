package usecase

import (
	"SipServer/internal/repository/session"
	"database/sql"
)

type SessionUsecase struct {
	repo *session.SessionRepo
}

func NewSessionUsecase(db *sql.DB) *SessionUsecase {
	return &SessionUsecase{
		repo: session.NewSessionRepo(db),
	}
}

func (s *SessionUsecase) List() ([]session.Session, error) {
	return s.repo.List()
}
