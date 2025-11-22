package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Novip1906/tasks-grpc/notifications/internal/app"
	"github.com/Novip1906/tasks-grpc/notifications/internal/config"
	"github.com/Novip1906/tasks-grpc/notifications/pkg/logging"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logging.SetupLogger(slog.LevelDebug)

	time.Sleep(20 * time.Second)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := app.NewServer(cfg, log)

	log.Info("starting server")
	if err := srv.Run(ctx); err != nil {
		log.Error("server run error", logging.Err(err))
		return
	}
}
