package kafka

import (
	"context"
	"time"

	"github.com/Novip1906/tasks-grpc/tasks/internal/config"
	"github.com/segmentio/kafka-go"
)

type producer struct {
	writer *kafka.Writer
}

func newProducer(kafkaCfg *config.Kafka) *producer {
	return &producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(kafkaCfg.Brokers...),
			BatchSize:    1,
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
	}
}

func (p *producer) SendMessage(ctx context.Context, topic string, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

func (p *producer) Close() error {
	return p.writer.Close()
}
