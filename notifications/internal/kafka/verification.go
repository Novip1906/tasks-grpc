package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Novip1906/tasks-grpc/notifications/internal/email"
	"github.com/Novip1906/tasks-grpc/notifications/internal/models"
)

type emailVerificationHandler struct {
	emailService *email.EmailSenderService
	log          *slog.Logger
}

func (h *emailVerificationHandler) HandleMessage(ctx context.Context, message []byte) error {
	var verificationMsg models.EmailVerificationMessage
	if err := json.Unmarshal(message, &verificationMsg); err != nil {
		return fmt.Errorf("unmarshal verification message: %w", err)
	}

	h.log.Info("Received verification email request", "email", verificationMsg.Email)
	return h.emailService.SendVerificationEmail(verificationMsg)
}
