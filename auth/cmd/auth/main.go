package main

import (
	"log/slog"

	"github.com/Novip1906/tasks-grpc/auth/internal/app"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/Novip1906/tasks-grpc/auth/pkg/logging"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logging.SetupLogger(slog.LevelDebug)

	srv := app.NewServer(cfg, log)

	log.Info("Starting server at " + cfg.Address)
	if err := srv.Run(); err != nil {
		log.Error(err.Error())
		return
	}
}
