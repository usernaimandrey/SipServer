package dbconnecter

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type DbCloser func()

func DbConnecter(defaultDB bool, retry int) (*sql.DB, string, DbCloser, error) {
	err := godotenv.Load()

	closer := func() {}

	if err != nil {
		return &sql.DB{}, "", closer, fmt.Errorf("%s", "File .env does not exists")
	}

	env := os.Getenv("ENV")
	var dbName string
	var connectDb string

	if env == "test" {
		dbName = os.Getenv("POSTGRES_DB_TEST")
	} else {
		dbName = os.Getenv("POSTGRES_DB")
	}

	if defaultDB {
		connectDb = "postgres"
	} else {
		connectDb = dbName
	}

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("POSTGRES_PASSWORD")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		connectDb,
	)

	db, err := sql.Open("postgres", connStr)

	closer = func() {
		db.Close()
	}

	if err != nil {
		return db, "", closer, err
	}

	err = db.Ping()

	if err != nil && retry == 0 {
		return db, "", closer, err
	} else if err != nil && retry != 0 {
		return DbConnecter(defaultDB, retry-1)
	}
	return db, dbName, closer, nil
}
