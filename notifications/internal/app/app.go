package app

import (
	"context"
	"log/slog"

	"github.com/Novip1906/tasks-grpc/notifications/internal/config"
	"github.com/Novip1906/tasks-grpc/notifications/internal/email"
	"github.com/Novip1906/tasks-grpc/notifications/internal/kafka"
)

type Server struct {
	cfg      *config.Config
	log      *slog.Logger
	consumer *kafka.Consumer
}

func NewServer(cfg *config.Config, log *slog.Logger) (*Server, error) {
	emailService, err := email.NewEmailSender(&cfg.SMTP, log)
	if err != nil {
		return nil, err
	}

	consumer := kafka.NewConsumer(cfg.Kafka, emailService, log)
	return &Server{cfg: cfg, log: log, consumer: consumer}, err
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.consumer.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	s.log.Info("shutting down server, stopping consumer")
	return s.consumer.Stop()
}
