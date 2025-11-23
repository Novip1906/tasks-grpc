package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Novip1906/tasks-grpc/tasks/internal/config"
	"github.com/Novip1906/tasks-grpc/tasks/internal/models"
)

type EmailProducer struct {
	producer    *producer
	eventsTopic string
}

func NewEmailProducer(kafkaCfg *config.Kafka) *EmailProducer {
	kafkaProducer := newProducer(kafkaCfg)
	return &EmailProducer{
		producer:    kafkaProducer,
		eventsTopic: kafkaCfg.EventsTopic,
	}
}

func (e *EmailProducer) SendEventEmail(ctx context.Context, message *models.EventMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal email events message: %w", err)
	}

	err = e.producer.SendMessage(
		ctx,
		e.eventsTopic,
		[]byte(message.Email),
		jsonData,
	)

	if err != nil {
		return fmt.Errorf("failed to send email events message: %w", err)
	}

	return nil
}

func (e *EmailProducer) Close() {
	e.producer.Close()
}
