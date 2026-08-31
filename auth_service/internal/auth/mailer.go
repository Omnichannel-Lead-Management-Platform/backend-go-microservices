package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/omnichannel/common/logger"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

// FileMailer implements authboss.Mailer and saves emails to disk
type FileMailer struct {
	OutDir string
}

func NewFileMailer(outDir string) *FileMailer {
	// Ensure directory exists
	os.MkdirAll(outDir, os.ModePerm)
	return &FileMailer{OutDir: outDir}
}

func (f *FileMailer) Send(ctx context.Context, email authboss.Email) error {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.txt", timestamp, email.To[0])
	filePath := filepath.Join(f.OutDir, filename)

	content := fmt.Sprintf("To: %v\nFrom: %s\nSubject: %s\n\n=== TEXT BODY ===\n%s\n\n=== HTML BODY ===\n%s\n",
		email.To, email.From, email.Subject, email.TextBody, email.HTMLBody)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		logger.Log.Error("Failed to write email to file", zap.Error(err))
		return err
	}

	logger.Log.Info("Email successfully saved", zap.String("path", filePath))
	return nil
}

// SMTPMailer implements authboss.Mailer and sends real emails via SMTP
type SMTPMailer struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	FromName   string
}

func NewSMTPMailer() *SMTPMailer {
	port := 587 // default
	if p := os.Getenv("MAIL_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	return &SMTPMailer{
		Host:     os.Getenv("MAIL_HOST"),
		Port:     port,
		Username: os.Getenv("MAIL_USERNAME"),
		Password: os.Getenv("MAIL_PASSWORD"),
		From:     os.Getenv("MAIL_FROM_ADDRESS"),
		FromName: os.Getenv("MAIL_FROM_NAME"),
	}
}

func (m *SMTPMailer) Send(ctx context.Context, email authboss.Email) error {
	msg := gomail.NewMessage()
	
	from := m.From
	if m.FromName != "" {
		from = m.FromName + " <" + m.From + ">"
	}
	// Fallback to email.From if our env is empty
	if from == "" {
		from = email.From
	}

	msg.SetHeader("From", from)
	msg.SetHeader("To", email.To...)
	msg.SetHeader("Subject", email.Subject)

	if email.TextBody != "" {
		msg.SetBody("text/plain", email.TextBody)
	}
	if email.HTMLBody != "" {
		if email.TextBody != "" {
			msg.AddAlternative("text/html", email.HTMLBody)
		} else {
			msg.SetBody("text/html", email.HTMLBody)
		}
	}

	dialer := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)
	
	if err := dialer.DialAndSend(msg); err != nil {
		logger.Log.Error("Failed to send email via SMTP", zap.Error(err))
		return err
	}

	logger.Log.Info("Email successfully sent", zap.Any("to", email.To))
	return nil
}
