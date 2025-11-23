package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Novip1906/tasks-grpc/notifications/internal/email"
	"github.com/Novip1906/tasks-grpc/notifications/internal/models"
)

type eventsHandler struct {
	emailService *email.EmailSenderService
	log          *slog.Logger
}

func (h *eventsHandler) HandleMessage(ctx context.Context, message []byte) error {
	var eventMsg models.EventMessage
	if err := json.Unmarshal(message, &eventMsg); err != nil {
		return fmt.Errorf("unmarshal event message: %w", err)
	}

	h.log.Info("Received event request", "email", eventMsg.Email)
	return h.emailService.SendEventEmail(eventMsg)
}
