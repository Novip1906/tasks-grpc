package main

import (
	"log/slog"
	"time"

	"github.com/Novip1906/tasks-grpc/auth/internal/app"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/Novip1906/tasks-grpc/auth/pkg/logging"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logging.SetupLogger(slog.LevelDebug)

	time.Sleep(time.Second)

	srv := app.NewServer(cfg, log)

	log.Info("starting server", "address", cfg.Address)
	if err := srv.Run(); err != nil {
		log.Error("server run error", logging.Err(err))
		return
	}
}
