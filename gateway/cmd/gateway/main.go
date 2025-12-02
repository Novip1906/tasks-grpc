package main

import (
	"log/slog"

	"github.com/Novip1906/tasks-grpc/gateway/internal/app"
	"github.com/Novip1906/tasks-grpc/gateway/internal/config"
	"github.com/Novip1906/tasks-grpc/gateway/pkg/logging"
)

func main() {
	cfg := config.MustLoadConfig()

	log := logging.SetupLogger(slog.LevelDebug)

	srv := app.NewServer(cfg, log)

	if err := srv.Run(); err != nil {
		log.Error("server run error", logging.Err(err))
		return
	}

}
