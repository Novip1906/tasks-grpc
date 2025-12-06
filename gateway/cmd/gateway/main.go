package main

import (
	"log/slog"
	"os"

	"github.com/Novip1906/tasks-grpc/gateway/internal/app"
	"github.com/Novip1906/tasks-grpc/gateway/internal/config"
	"github.com/Novip1906/tasks-grpc/gateway/pkg/logging"
)

func main() {
	cfg := config.MustLoadConfig()

	log := logging.SetupLogger(slog.LevelDebug)

	srv, err := app.NewServer(cfg, log)
	if err != nil {
		log.Error("failed to create server", logging.Err(err))
		os.Exit(1)
	}

	if err = srv.Run(); err != nil {
		log.Error("server run error", logging.Err(err))
		os.Exit(1)
	}

}
