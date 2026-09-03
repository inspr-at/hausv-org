// Command hausv-org is the composition root: it builds the app from the
// environment, starts the background workers, and serves.
//
// Everything else lives under internal/. Keeping main thin is what makes the
// rest of the code testable without a process environment.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/inspr-at/hausv-org/internal/homeconnector"
	"github.com/inspr-at/hausv-org/internal/server"
)

const (
	httpReadHeaderTimeout = 5 * time.Second
	httpReadTimeout       = 60 * time.Second
	httpWriteTimeout      = 120 * time.Second
	httpIdleTimeout       = 90 * time.Second
	httpMaxHeaderBytes    = 32 << 10
)

type httpServerLimits struct {
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	maxHeaderBytes    int
}

func productionHTTPServerLimits() httpServerLimits {
	return httpServerLimits{
		readHeaderTimeout: httpReadHeaderTimeout,
		readTimeout:       httpReadTimeout,
		writeTimeout:      httpWriteTimeout,
		idleTimeout:       httpIdleTimeout,
		maxHeaderBytes:    httpMaxHeaderBytes,
	}
}

func main() {
	// Structured JSON logs to stderr; the request/panic middleware uses the
	// default logger (HAUSV-141).
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	if len(os.Args) > 1 && os.Args[1] == "connector" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := homeconnector.Run(ctx, os.Args[2:], os.Stdout, os.Stderr); err != nil {
			slog.Error("connector stopped", "error", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "charging-mode" {
		if err := runChargingMode(os.Args[2:]); err != nil {
			slog.Error("charging mode update failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate-data" {
		if err := runMigrateData(os.Args[2:], os.Stdout, os.Stderr, os.Getenv); err != nil {
			slog.Error("data migration did not complete", "error", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "demo-seed" {
		if err := runDemoSeed(os.Args[2:], os.Stdout, os.Stderr, os.Getenv); err != nil {
			slog.Error("demo seed did not complete", "error", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "predeploy-snapshot" {
		if err := runPredeploySnapshot(os.Args[2:], os.Stdout, os.Stderr); err != nil {
			slog.Error("pre-deploy snapshot failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		target := "http://127.0.0.1:8080/healthz"
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		if err := server.RunHealthcheck(target); err != nil {
			slog.Error("healthcheck failed", "error", err)
			os.Exit(1)
		}
		return
	}

	app, err := server.New()
	if err != nil {
		slog.Error("application initialization failed", "error", err)
		os.Exit(1)
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
	stopEnergyRetention := app.StartEnergyRetentionWorker()
	defer stopEnergyRetention()
	stopHomeReservationRetention := app.StartHomeReservationRetentionWorker()
	defer stopHomeReservationRetention()
	stopEnergySampler := app.StartEnergyIntervalSampler()
	defer stopEnergySampler()

	srv := newHTTPServer(app.Addr(), app.Handler())

	// Graceful shutdown: SIGTERM arrives on every deploy. Without this, in-flight
	// requests are killed mid-response — including a store write between its two
	// commits. Give them 15s to finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", app.Addr())
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
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

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return newHTTPServerWithLimits(addr, handler, productionHTTPServerLimits())
}

func newHTTPServerWithLimits(addr string, handler http.Handler, limits httpServerLimits) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: limits.readHeaderTimeout,
		ReadTimeout:       limits.readTimeout,
		WriteTimeout:      limits.writeTimeout,
		IdleTimeout:       limits.idleTimeout,
		MaxHeaderBytes:    limits.maxHeaderBytes,
	}
}
