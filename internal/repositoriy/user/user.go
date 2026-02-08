package user

import (
	"database/sql"
	"errors"
)

type UserNotFoundError struct {
	err error
}

func (e UserNotFoundError) Error() string {
	return e.err.Error()
}

func NewUserError(msg string) error {
	return &UserNotFoundError{err: errors.New(msg)}
}

type UserRepositoriy struct {
	Db *sql.DB
}

type User struct {
	Id     int
	Login  string
	Role   string
	Config *UserConfig
}

type UserConfig struct {
	CallSchema string
}

func NewUser() *User {
	return &User{Config: &UserConfig{}}
}

func NewUserRepo(db *sql.DB) *UserRepositoriy {
	return &UserRepositoriy{
		Db: db,
	}
}

func (u *UserRepositoriy) FindByLogin(login string) (*User, error) {
	user := NewUser()
	row := u.Db.QueryRow("SELECT id, login, role FROM users where login = $1", login)
	err := row.Scan(&user.Id, &user.Login, &user.Role)

	if err != nil {
		if err == sql.ErrNoRows {
			return user, NewUserError("user not found")
		} else {
			return user, err
		}
	}
	return user, nil
}

func (u *UserRepositoriy) FindByLoginWithConfig(login string) (*User, error) {
	user := NewUser()
	row := u.Db.QueryRow("SELECT u.id, u.login, u.role, uc.call_schema FROM users u LEFT JOIN user_configs uc ON uc.user_id = u.id where login = $1", login)
	err := row.Scan(&user.Id, &user.Login, &user.Role, &user.Config.CallSchema)

	if err != nil {
		if err == sql.ErrNoRows {
			return user, NewUserError("user not found")
		} else {
			return user, err
		}
	}
	return user, nil
}
