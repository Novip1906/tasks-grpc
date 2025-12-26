package main

import (
	"time"

	"github.com/Novip1906/tasks-grpc/tasks/internal/app"
	"github.com/Novip1906/tasks-grpc/tasks/internal/config"
	"github.com/Novip1906/tasks-grpc/tasks/pkg/logging"
)

func main() {
	cfg := config.MustLoadConfig()
	log := logging.SetupLogger(cfg.Env)

	time.Sleep(time.Second)

	srv := app.NewServer(cfg, log)
	defer srv.Close()

	log.Info("starting server", "address", cfg.TasksAddress)
	if err := srv.Run(); err != nil {
		log.Error("server run error", logging.Err(err))
		return
	}
}
