package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aarondl/authboss/v3"
)

func TestFileMailer_Send(t *testing.T) {
	// Create a temp directory for emails
	tempDir := t.TempDir()

	mailer := &FileMailer{
		OutDir: tempDir,
	}

	email := authboss.Email{
		To:       []string{"test@example.com"},
		From:     "noreply@example.com",
		FromName: "No Reply",
		ReplyTo:  "support@example.com",
		Subject:  "Test Email",
		TextBody: "This is a test email text body.",
		HTMLBody: "<p>This is a test email HTML body.</p>",
	}

	err := mailer.Send(context.Background(), email)
	if err != nil {
		t.Fatalf("unexpected error sending email: %v", err)
	}

	// Wait a tiny bit just to ensure file is written, though it's synchronous
	time.Sleep(10 * time.Millisecond)

	// Verify a file was created in the temp directory
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 file in temp dir, got %d", len(entries))
	}

	fileName := entries[0].Name()
	filePath := filepath.Join(tempDir, fileName)

	// Check file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read email file: %v", err)
	}
	
	contentStr := string(content)

	if !contains(contentStr, "To: [test@example.com]") {
		t.Errorf("missing To field in email file: %s", contentStr)
	}
	if !contains(contentStr, "Subject: Test Email") {
		t.Errorf("missing Subject field in email file: %s", contentStr)
	}
	if !contains(contentStr, "This is a test email text body.") {
		t.Errorf("missing TextBody in email file: %s", contentStr)
	}
	if !contains(contentStr, "<p>This is a test email HTML body.</p>") {
		t.Errorf("missing HTMLBody in email file: %s", contentStr)
	}
}

// Simple helper to check if a string contains another
func contains(s, substr string) bool {
	// Not using strings.Contains directly to avoid another import if possible, 
	// but we can just import "strings". Let's use strings.Contains
	return stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewSMTPMailer(t *testing.T) {
	os.Setenv("MAIL_PORT", "1025")
	os.Setenv("MAIL_HOST", "localhost")
	defer os.Unsetenv("MAIL_PORT")
	defer os.Unsetenv("MAIL_HOST")

	mailer := NewSMTPMailer()
	if mailer.Port != 1025 {
		t.Errorf("expected port 1025, got %d", mailer.Port)
	}
	if mailer.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", mailer.Host)
	}
}

