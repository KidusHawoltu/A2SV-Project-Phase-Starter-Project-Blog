package infrastructure

import (
	"fmt"

	"gopkg.in/gomail.v2"
)

// EmailService defines the contract for sending emails.
type EmailService interface {
	SendPasswordResetEmail(toEmail, username, resetToken string) error
	SendActivationEmail(toEmail, username, activationToken string) error
}

// dialer interface allows mocking the gomail.Dialer
type dialer interface {
	DialAndSend(...*gomail.Message) error
}

type SmtpEmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	dialer   dialer
}

func NewSMTPEmailService(host string, port int, username, password, from string) EmailService {
	d := gomail.NewDialer(host, port, username, password)

	return &SmtpEmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		dialer:   d,
	}
}

func (s *SmtpEmailService) SendPasswordResetEmail(toEmail, username, resetToken string) error {
	subject := "Reset Your Password"
	body := fmt.Sprintf(`
	Hi %s,

	You requested to reset your password.

	Use the following token to reset your password:
	%s

	Or click the link below:
	http://localhost:8080/api/v1/password/reset?token=%s

	If you did not request this, please ignore this email.
	`, username, resetToken, resetToken)

	return s.send(toEmail, subject, body)
}

func (s *SmtpEmailService) SendActivationEmail(toEmail, username, activationToken string) error {
	subject := "Activate Your Account"
	body := fmt.Sprintf(`
	Hi %s,

	Welcome to our app!

	Activate your account using the token below:
	%s

	Or click this link:
	http://localhost:8080/api/v1/auth/activate?token=%s
	If you did not create an account, ignore this email.
	`, username, activationToken, activationToken)

	return s.send(toEmail, subject, body)
}

func (s *SmtpEmailService) send(to, subject, body string) error {
	m := gomail.NewMessage()

	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	return s.dialer.DialAndSend(m)
}
