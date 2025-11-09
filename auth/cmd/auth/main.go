package main

import (
	"log"

	"github.com/Novip1906/tasks-grpc/auth/internal/app"
	"github.com/Novip1906/tasks-grpc/auth/internal/config"
)

func main() {
	cfg := config.MustLoadConfig()

	srv := app.NewServer(cfg)

	log.Println("Starting server at", cfg.Address)
	if err := srv.Run(); err != nil {
		log.Fatal("Error starting a server:", err)
		return
	}
}
