package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emiago/sipgo"

	"SipServer/internal/registrar"
	"SipServer/internal/sipserver"
)

func main() {
	ua, err := sipgo.NewUA() // создание UserAgent :contentReference[oaicite:5]{index=5}
	if err != nil {
		log.Fatal(err)
	}
	defer ua.Close()

	reg := registrar.New(60 * time.Second)

	s, err := sipserver.New(ua, reg)
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
