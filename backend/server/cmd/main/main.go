package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/szplanner/snakerule/internal/bootstrap"
)

func main() {
	app, err := bootstrap.New()
	if err != nil {
		slog.Error("initialize application", "error", err)
		os.Exit(1)
	}
	defer app.StopBackground()

	serverErr := make(chan error, 1)
	go func() { serverErr <- app.Server.ListenAndServe() }()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-sigCtx.Done():
	case err := <-serverErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("HTTP server stopped", "error", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.Server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
