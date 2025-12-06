package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Novip1906/tasks-grpc/notifications/internal/app"
	"github.com/Novip1906/tasks-grpc/notifications/internal/config"
	"github.com/Novip1906/tasks-grpc/notifications/pkg/logging"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logging.SetupLogger(slog.LevelDebug)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv, err := app.NewServer(cfg, log)

	if err != nil {
		log.Error("error creating server", logging.Err(err))
		return
	}

	log.Info("starting server")
	if err := srv.Run(ctx); err != nil {
		log.Error("server run error", logging.Err(err))
		return
	}
}
