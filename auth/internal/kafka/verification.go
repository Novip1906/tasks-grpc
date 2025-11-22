package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Novip1906/tasks-grpc/auth/internal/models"
)

type EmailProducer struct {
	producer          *Producer
	verificationTopic string
}

func NewEmailProducer(producer *Producer, verificationTopic string) *EmailProducer {
	return &EmailProducer{
		producer:          producer,
		verificationTopic: verificationTopic,
	}
}

func (e *EmailProducer) SendVerificationEmail(ctx context.Context, message *models.EmailVerificationMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal email verification message: %w", err)
	}

	err = e.producer.SendMessage(
		ctx,
		e.verificationTopic,
		[]byte(message.Email),
		jsonData,
	)

	if err != nil {
		return fmt.Errorf("failed to send email verification message: %w", err)
	}

	return nil
}
