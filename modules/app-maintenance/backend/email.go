package main

import (
	"fmt"
	"log"
	"net/smtp"
)

// EmailSender sends a plain-text email. Swappable so forgot-password (and
// anything else later) doesn't care whether it's really going out over
// SMTP or just being logged.
type EmailSender interface {
	Send(to, subject, body string) error
}

// newEmailSender picks a sender based on env: a real SMTP sender once
// SMTP_HOST is set, a console sender (logs the message instead of sending
// it) otherwise - so the whole forgot-password flow is testable today,
// and switches to actually sending mail later by setting env vars only,
// no code changes.
func newEmailSender() EmailSender {
	host := getenv("SMTP_HOST", "")
	if host == "" {
		log.Println("SMTP_HOST not set - emails will be logged, not sent (see .env.example)")
		return &consoleSender{}
	}
	return &smtpSender{
		host:     host,
		port:     getenv("SMTP_PORT", "587"),
		user:     getenv("SMTP_USER", ""),
		password: getenv("SMTP_PASSWORD", ""),
		from:     getenv("SMTP_FROM", "no-reply@bung.local"),
	}
}

// consoleSender logs the email instead of sending it - the dev/testing
// fallback (see newEmailSender).
type consoleSender struct{}

func (c *consoleSender) Send(to, subject, body string) error {
	log.Printf("[email:console] to=%s subject=%q\n%s", to, subject, body)
	return nil
}

// smtpSender sends via net/smtp (stdlib only, no new dependency). Works
// with STARTTLS-capable relays on the usual submission port (587),
// including most internal relays and providers like Gmail (with an app
// password).
type smtpSender struct {
	host, port, user, password, from string
}

func (s *smtpSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s", s.from, to, subject, body)

	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}
	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
