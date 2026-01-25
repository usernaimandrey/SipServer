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

type User struct {
	Id    int
	Login string
	Role  string
}

func NewUser() *User {
	return &User{}
}

func (u *User) FindByLogin(login string, db *sql.DB) (*User, error) {
	user := NewUser()
	row := db.QueryRow("SELECT id, login, role FROM users where login = $1", login)
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
