package email

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/Novip1906/tasks-grpc/notifications/internal/config"
	"github.com/Novip1906/tasks-grpc/notifications/internal/models"
	"github.com/Novip1906/tasks-grpc/notifications/pkg/logging"
)

type EmailSenderService struct {
	smtpConfig *config.SMTP
	log        *slog.Logger
}

func NewEmailSender(smtpCfg *config.SMTP, log *slog.Logger) *EmailSenderService {
	return &EmailSenderService{smtpConfig: smtpCfg, log: log}
}

func (s *EmailSenderService) SendVerificationEmail(msg models.EmailVerificationMessage) error {
	subject := "Подтверждение эл. почты"

	body, err := s.renderVerificationTemplate(msg.Username, msg.Code)
	if err != nil {
		s.log.Error("vericfation template render error", logging.Err(err))
		return ErrRenderTemplate
	}

	return s.sendEmail(msg.Email, subject, body)
}

func (s *EmailSenderService) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.smtpConfig.Email, s.smtpConfig.Password, s.smtpConfig.Host)

	msg := []byte(
		"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-version: 1.0;\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
			"\r\n" + body,
	)

	addr := fmt.Sprintf("%s:%d", s.smtpConfig.Host, s.smtpConfig.Port)
	return smtp.SendMail(addr, auth, s.smtpConfig.Email, []string{to}, msg)
}
