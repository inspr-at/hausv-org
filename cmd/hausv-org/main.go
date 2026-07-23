// Command hausv-org is the composition root: it builds the app from the
// environment, starts the background workers, and serves.
//
// Everything else lives under internal/. Keeping main thin is what makes the
// rest of the code testable without a process environment.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/markus-barta/hausv-org/internal/server"
)

func main() {
	// Structured JSON logs to stderr; the request/panic middleware uses the
	// default logger (HAUSV-141).
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		target := "http://127.0.0.1:8080/healthz"
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		if err := server.RunHealthcheck(target); err != nil {
			log.Printf("healthcheck failed: %v", err)
			os.Exit(1)
		}
		return
	}

	app, err := server.New()
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	stopSampler := app.StartParkingSampler()
	defer stopSampler()
	stopVoteReminders := app.StartVoteReminderWorker()
	defer stopVoteReminders()
	stopCharging := app.StartChargingController()
	defer stopCharging()
	stopTelegram := app.StartTelegramBot()
	defer stopTelegram()

	srv := &http.Server{
		Addr:              app.Addr(),
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown: SIGTERM arrives on every deploy. Without this, in-flight
	// requests are killed mid-response — including a store write between its two
	// commits. Give them 15s to finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", app.Addr())
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
