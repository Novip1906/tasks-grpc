package email

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"

	"github.com/Novip1906/tasks-grpc/notifications/internal/config"
	"github.com/Novip1906/tasks-grpc/notifications/internal/models"
)

//go:embed templates
var templateFS embed.FS

type EmailSenderService struct {
	smtpConfig       *config.SMTP
	log              *slog.Logger
	verificationTmpl *template.Template
	eventTmpl        *template.Template
}

func NewEmailSender(smtpCfg *config.SMTP, log *slog.Logger) (*EmailSenderService, error) {
	verificationTmpl, err := template.ParseFS(templateFS, "templates/email_verification.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse verification template: %w", err)
	}

	eventTmpl, err := template.ParseFS(templateFS, "templates/event.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse event template: %w", err)
	}
	return &EmailSenderService{
		smtpConfig:       smtpCfg,
		log:              log,
		verificationTmpl: verificationTmpl,
		eventTmpl:        eventTmpl,
	}, nil
}

func (s *EmailSenderService) SendVerificationEmail(msg models.EmailVerificationMessage) error {
	subject := "Подтверждение эл. почты"

	body, err := s.renderVerificationTemplate(msg)
	if err != nil {
		return err
	}

	return s.sendEmail(msg.Email, subject, body)
}

func (s *EmailSenderService) SendEventEmail(msg models.EventMessage) error {
	subject := "Уведомление"

	body, err := s.renderEventTemplate(msg)
	if err != nil {
		return err
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
