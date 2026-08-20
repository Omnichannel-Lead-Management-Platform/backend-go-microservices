package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aarondl/authboss/v3"
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
		fmt.Printf("Failed to write email to file: %v\n", err)
		return err
	}

	fmt.Printf("Email successfully saved to %s\n", filePath)
	return nil
}
