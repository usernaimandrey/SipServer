package usecase

import (
	"SipServer/internal/repository/user"
	"database/sql"
)

type UserUsecase struct {
	userRepo *user.UserRepositoriy
}

func NewUserUseCase(db *sql.DB) *UserUsecase {
	return &UserUsecase{
		userRepo: user.NewUserRepo(db),
	}
}

func (u *UserUsecase) ListUsers() ([]*user.User, error) {
	return u.userRepo.List()
}

func (u *UserUsecase) GetUser(id string) (*user.User, error) {
	return u.userRepo.FindByIDWithConfig(id)
}

func (u *UserUsecase) CreateUser(user *user.User) (*user.User, error) {
	return u.userRepo.CreateUserWithConfig(user)
}

func (u *UserUsecase) UpdateUser(user_id string, arg *user.UpdateUserRequest) error {
	return u.userRepo.UpdateUser(user_id, arg)
}
