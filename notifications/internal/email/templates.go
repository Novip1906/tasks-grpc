package email

import (
	"bytes"

	"github.com/Novip1906/tasks-grpc/notifications/internal/models"
)

func (s *EmailSenderService) renderVerificationTemplate(msg models.EmailVerificationMessage) (string, error) {
	var buf bytes.Buffer

	if err := s.verificationTmpl.Execute(&buf, msg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *EmailSenderService) renderEventTemplate(msg models.EventMessage) (string, error) {
	var buf bytes.Buffer

	if err := s.eventTmpl.Execute(&buf, msg); err != nil {
		return "", err
	}

	return buf.String(), nil
}
