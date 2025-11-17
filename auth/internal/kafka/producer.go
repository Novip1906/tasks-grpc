package kafka

import (
	"context"

	"github.com/Novip1906/tasks-grpc/auth/internal/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg *config.Config) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr: kafka.TCP(cfg.Kafka.Brokers...),
		},
	}
}

func (p *Producer) SendMessage(ctx context.Context, topic string, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
