package infrastructure

import (
	"fmt"
	"net/smtp"
)

// EmailService defins the contract for sending emails.
type EmailService interface {
	SendPasswordResetEmail(toEmail, username, resetToken string) error
	SendActivationEmail(toEmail, username, activationToken string) error
}

type smtpEmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTPEmailService(host string, port int, username, password, from string) EmailService {
	return &smtpEmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *smtpEmailService) SendPasswordResetEmail(toEmail, username, resetToken string) error {
	subject := "Reset Your Password"
	body := fmt.Sprintf(`
	Hi %s,

	You requested to reset your password.

	Use the following token to reset your password:
	%s

	Or click the link below:
	http://yourfrontend.com/reset-password?token=%s

	If you did not request this, please ignore this email.
	`, username, resetToken, resetToken)

	return s.send(toEmail, subject, body)
}

func (s *smtpEmailService) SendActivationEmail(toEmail, username, activationToken string) error {
	subject := "Activate Your Account"
	body := fmt.Sprintf(`
	Hi %s,

	Welcome to our app!

	Activate your account using the token below:
	%s

	Or click this link:
	http://yourfrontend.com/activate-account?token=%s

	If you did not create an account, ignore this email.
	`, username, activationToken, activationToken)

	return s.send(toEmail, subject, body)
}

func (s *smtpEmailService) send(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	message := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.from, to, subject, body))

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	return smtp.SendMail(addr, auth, s.from, []string{to}, message)
}