package main

import (
	connecter "SipServer/pkg/dbconnecter"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		fmt.Printf("Load .env with error: %v", err)
		os.Exit(1)
	}

	db, dbName, closer, err := connecter.DbConnecter(true)

	defer closer()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = db.Ping()

	if err != nil {
		fmt.Printf("DB connection err: %v", err)
		os.Exit(1)
	}

	defer db.Close()

	_, err = db.Exec("DROP DATABASE IF EXISTS " + dbName)
	if err != nil {
		fmt.Printf("DB does not deleted %v", err)
		os.Exit(1)
	}

	_, err = db.Exec("create database " + dbName)
	if err != nil {
		fmt.Printf("DB %v not create: %v", dbName, err)
		os.Exit(1)
	}
}
