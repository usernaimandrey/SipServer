package dbconnecter

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type DbCloser func()

func DbConnecter(defaultDB bool) (*sql.DB, string, DbCloser, error) {
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

	dbPort := os.Getenv("POSTGRES_PORT")

	connStr := fmt.Sprintf("postgres://postgres@localhost:%v/%v?sslmode=disable", dbPort, connectDb)

	db, err := sql.Open("postgres", connStr)

	closer = func() {
		db.Close()
	}

	if err != nil {
		return db, "", closer, err
	}

	err = db.Ping()

	if err != nil {
		return db, "", closer, err
	}
	return db, dbName, closer, nil
}
