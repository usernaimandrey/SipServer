package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"SipServer/internal/entity/user"
	"SipServer/internal/registrar"
	"SipServer/internal/sipserver"
	"SipServer/pkg/dbconnecter"

	"github.com/emiago/sipgo"
	"github.com/joho/godotenv"
)

const (
	retry int = 3
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	sipHost := os.Getenv("PUBLIC_HOST")
	sipPort, err := strconv.Atoi(os.Getenv("PUBLIC_PORT"))

	if err != nil {
		log.Fatal(err)
	}

	ua, err := sipgo.NewUA() // создание UserAgent :contentReference[oaicite:5]{index=5}
	if err != nil {
		log.Fatal(err)
	}
	defer ua.Close()

	reg := registrar.New(60 * time.Second)

	db, _, dbCloser, err := dbconnecter.DbConnecter(false, retry)

	if err != nil {
		log.Fatal(err)
	}

	userRepo := user.NewUserRepo(db)

	defer dbCloser()

	s, err := sipserver.New(ua, reg, db, sipHost, sipPort, userRepo)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// UDP достаточно для прототипа (можно добавить TCP второй строкой)
	go func() {
		log.Println("SIP server listening on udp://0.0.0.0:5060")
		if err := s.ListenAndServe(ctx, "udp", "0.0.0.0:5060"); err != nil {
			log.Println("server stopped:", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown")
}
