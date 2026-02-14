package usecase

import (
	"database/sql"

	calljournal "SipServer/internal/repository/call_journal"
)

type CallJournalUsecase struct {
	repo *calljournal.CallJournalRepo
}

func NewCallJournalUsecase(db *sql.DB) *CallJournalUsecase {
	return &CallJournalUsecase{
		repo: calljournal.NewCallJournalRepo(db),
	}
}

func (c *CallJournalUsecase) List() ([]calljournal.CallJournal, error) {
	return c.repo.List()
}
