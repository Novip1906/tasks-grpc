package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Novip1906/tasks-grpc/auth/internal/models"
)

type EmailVerificationProducer struct {
	producer *Producer
	topic    string
}

func NewEmailVerificationProducer(producer *Producer, topic string) *EmailVerificationProducer {
	return &EmailVerificationProducer{
		producer: producer,
		topic:    topic,
	}
}

func (e *EmailVerificationProducer) SendVerificationEmail(ctx context.Context, message *models.EmailVerificationMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal email verification message: %w", err)
	}

	err = e.producer.SendMessage(
		ctx,
		e.topic,
		[]byte(message.Email),
		jsonData,
	)

	if err != nil {
		return fmt.Errorf("failed to send email verification message: %w", err)
	}

	return nil
}
