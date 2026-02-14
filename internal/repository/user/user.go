package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var ErrNoFieldsToUpdate = errors.New("no fields to update")

const (
	queryUserWithConfig string = "SELECT u.id, u.login, u.role, uc.call_schema FROM users u LEFT JOIN user_configs uc ON uc.user_id = u.id"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepositoriy struct {
	Db *sql.DB
}

type User struct {
	Id           int         `json:"id"`
	Login        string      `json:"login" validate:"required,min=4,max=64"`
	Role         string      `json:"role" validate:"required,oneof=admin user"`
	PasswordHash string      `json:"-"`
	Config       *UserConfig `json:"config" validate:"required"`
}

type UpdateUserRequest struct {
	Id           int                      `json:"id"`
	Login        string                   `json:"login" validate:"min=4,max=64"`
	Role         string                   `json:"role" validate:"oneof=admin user"`
	PasswordHash string                   `json:"-"`
	Config       *UpdateUserConfigRequest `json:"config"`
}

type UpdateUserConfigRequest struct {
	CallSchema string `json:"call_schema" vlidate:"oneof=redirect proxy"`
}

type UserConfig struct {
	CallSchema string `json:"call_schema" validate:"required,oneof=redirect proxy"`
}

func NewUser() *User {
	return &User{Config: &UserConfig{}}
}

func NewUserUpdateReq() *UpdateUserRequest {
	return &UpdateUserRequest{
		Config: &UpdateUserConfigRequest{},
	}
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		} else {
			return nil, err
		}
	}
	return user, nil
}

func (u *UserRepositoriy) FindByLoginWithConfig(login string) (*User, error) {
	user := NewUser()
	row := u.Db.QueryRow(queryUserWithConfig+" where login = $1", login)
	err := row.Scan(&user.Id, &user.Login, &user.Role, &user.Config.CallSchema)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		} else {
			return nil, err
		}
	}
	return user, nil
}

func (u *UserRepositoriy) FindByIDWithConfig(id string) (*User, error) {
	user := NewUser()
	row := u.Db.QueryRow(queryUserWithConfig+" where u.id = $1", id)
	err := row.Scan(&user.Id, &user.Login, &user.Role, &user.Config.CallSchema)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		} else {
			return nil, err
		}
	}
	return user, nil
}

func (u *UserRepositoriy) List() ([]*User, error) {
	users := make([]*User, 0)

	rows, err := u.Db.Query("SELECT u.id, u.login, u.role, uc.call_schema FROM users u LEFT JOIN user_configs uc ON uc.user_id = u.id")

	if err != nil {
		return nil, err
	}
	for rows.Next() {
		u := NewUser()
		err := rows.Scan(&u.Id, &u.Login, &u.Role, &u.Config.CallSchema)

		if err != nil {
			return nil, err
		}

		users = append(users, u)
	}
	return users, nil
}

func (u *UserRepositoriy) CreateUserWithConfig(user *User) (*User, error) {
	ctx := context.Background()
	tx, err := u.Db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})

	if err != nil {
		return nil, err
	}

	defer func() {
		tx.Rollback()
	}()

	hash, _ := u.GenerateRandomHash(1)

	var userID int64

	err = tx.QueryRowContext(ctx,
		`INSERT INTO users(login, role, password_hash) VALUES($1,$2,$3) RETURNING id`,
		user.Login, user.Role, hash,
	).Scan(&userID)

	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO user_configs(user_id, call_schema) VALUES($1,$2)`,
		userID,
		user.Config.CallSchema,
	)

	if err != nil {
		return nil, err
	}
	tx.Commit()
	user.Id = int(userID)
	return user, nil
}

func (u *UserRepositoriy) UpdateUser(userID string, arg *UpdateUserRequest) error {
	ctx := context.Background()

	tx, err := u.Db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1) users
	userSets := map[string]any{}
	if arg.Login != "" {
		userSets["login"] = arg.Login
	}
	if arg.Role != "" {
		userSets["role"] = arg.Role
	}

	if len(userSets) > 0 {
		qUsers, argsUsers, err := func() (string, []any, error) {
			where := fmt.Sprintf("id = $%d", len(userSets)+1)
			return buildUpdate("users", userSets, where, userID)
		}()
		if err != nil && !errors.Is(err, ErrNoFieldsToUpdate) {
			return err
		}
		if err == nil {
			if _, err := tx.ExecContext(ctx, qUsers, argsUsers...); err != nil {
				return err
			}
		}
	}

	// 2) user_configs (опционально)
	if arg.Config != nil && arg.Config.CallSchema != "" {
		configSets := map[string]any{
			"call_schema": arg.Config.CallSchema,
		}

		qCfg, argsCfg, err := func() (string, []any, error) {
			where := fmt.Sprintf("user_id = $%d", len(configSets)+1)
			return buildUpdate("user_configs", configSets, where, userID)
		}()
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, qCfg, argsCfg...); err != nil {
			return err
		}
	}
	if len(userSets) == 0 && (arg.Config == nil || arg.Config.CallSchema == "") {
		return ErrNoFieldsToUpdate // или return nil
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (u *UserRepositoriy) GenerateRandomHash(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func buildUpdate(table string, sets map[string]any, where string, whereArgs ...any) (string, []any, error) {
	if len(sets) == 0 {
		return "", nil, ErrNoFieldsToUpdate
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(table)
	sb.WriteString(" SET ")

	args := make([]any, 0, len(sets)+len(whereArgs))
	i := 1

	first := true
	for col, val := range sets {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		sb.WriteString(col)
		sb.WriteString(" = ")
		sb.WriteString(fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}

	sb.WriteString(" WHERE ")
	sb.WriteString(where)

	args = append(args, whereArgs...)
	return sb.String(), args, nil
}
