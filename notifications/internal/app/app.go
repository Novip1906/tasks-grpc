package app

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Novip1906/tasks-grpc/notifications/internal/config"
	"github.com/Novip1906/tasks-grpc/notifications/internal/email"
	"github.com/Novip1906/tasks-grpc/notifications/internal/kafka"
)

type Server struct {
	cfg      *config.Config
	log      *slog.Logger
	consumer *kafka.Consumer
	wg       sync.WaitGroup
}

func NewServer(cfg *config.Config, log *slog.Logger) *Server {
	emailService := email.NewEmailSender(&cfg.SMTP, log)

	consumer := kafka.NewConsumer(cfg.Kafka, emailService, log)
	return &Server{cfg: cfg, log: log, consumer: consumer}
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.consumer.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	s.log.Info("shutting down server, stopping consumer")
	return s.consumer.Stop()
}
