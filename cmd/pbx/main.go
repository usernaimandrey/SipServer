package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "SipServer/internal/http_server"
	"SipServer/internal/metrics"
	"SipServer/internal/registrar"
	"SipServer/internal/router"
	"SipServer/internal/sipserver"
	"SipServer/pkg/dbconnecter"

	"github.com/emiago/sipgo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const (
	retry           int = 3
	defaultHttpPort     = "8080"
	defaultSipPort      = "5060"
)

func main() {
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

	defer dbCloser()
	// ---------------- METRICS ----------------

	regM := prometheus.NewRegistry()
	regM.MustRegister(
		collectors.NewGoCollector(), // goroutines, GC, memory
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	metrics.MustRegister(regM)
	// ---------------- HTTP -------------------

	sh := httpserver.NewHttpServer(db)
	r := router.NewRouter(sh, regM)
	handler := httpserver.MetricsMiddleware(r)
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = defaultHttpPort
	}
	go func() {
		log.Printf("HTTP server listening on http://0.0.0.0:%s", port)
		if err := http.ListenAndServe(":"+port, handler); err != nil && err != http.ErrServerClosed {
			log.Println("http server stopped:", err)
		}
	}()

	// ------------------- SIP -------------------
	sip, err := sipserver.New(ua, reg, db)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer func() {
		stop()
	}()

	go func() {
		log.Println("SIP server listening on udp://0.0.0.0:5060")
		if err := sip.ListenAndServe(ctx, "udp", "0.0.0.0:5060"); err != nil {
			log.Println("server stopped:", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown")
}
