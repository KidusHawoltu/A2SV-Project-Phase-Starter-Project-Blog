package infrastructure

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/gomail.v2"
)

// --- Mock Dialer ---
type mockDialer struct {
	sentMessages []*gomail.Message
	shouldFail   bool
}

func (m *mockDialer) DialAndSend(msg ...*gomail.Message) error {
	if m.shouldFail {
		return fmt.Errorf("mock send failure")
	}
	m.sentMessages = append(m.sentMessages, msg...)
	return nil
}

// Helper to create a service with a mock dialer
func newTestEmailService(from string, fail bool) (*SmtpEmailService, *mockDialer) {
	mock := &mockDialer{shouldFail: fail}
	svc := &SmtpEmailService{
		from:   from,
		dialer: mock, // interface accepts mock
	}
	return svc, mock
}

func TestSendPasswordResetEmail_Success(t *testing.T) {
	svc, mock := newTestEmailService("test@example.com", false)

	err := svc.SendPasswordResetEmail("user@example.com", "John", "reset123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mock.sentMessages) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(mock.sentMessages))
	}

	msg := mock.sentMessages[0]
	if !strings.Contains(getBody(msg), "reset123") {
		t.Errorf("expected token in email body")
	}
}

func TestSendActivationEmail_Success(t *testing.T) {
	svc, mock := newTestEmailService("test@example.com", false)

	err := svc.SendActivationEmail("user@example.com", "Alice", "activate123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(mock.sentMessages) != 1 {
		t.Fatalf("expected 1 message sent, got %d", len(mock.sentMessages))
	}

	msg := mock.sentMessages[0]
	if !strings.Contains(getBody(msg), "activate123") {
		t.Errorf("expected token in email body")
	}
}

func TestSendPasswordResetEmail_Failure(t *testing.T) {
	svc, _ := newTestEmailService("test@example.com", true)

	err := svc.SendPasswordResetEmail("user@example.com", "John", "reset123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Utility to extract raw email content from gomail.Message ---
func getBody(msg *gomail.Message) string {
	var sb strings.Builder
	_, _ = msg.WriteTo(&sb)
	return sb.String()
}
