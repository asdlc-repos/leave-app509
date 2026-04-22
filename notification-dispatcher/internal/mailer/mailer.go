package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Mailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func New(host string, port int, username, password, from string) *Mailer {
	if from == "" {
		from = "noreply@leave-app.local"
	}
	return &Mailer{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

// Send delivers a single email. When Host is empty the mailer runs in
// log-simulation mode so the dispatcher still functions in dev environments
// that lack an SMTP relay.
func (m *Mailer) Send(to, subject, body string) error {
	if m.Host == "" {
		log.Printf("[email-simulation] from=%s to=%s subject=%q body=%s",
			m.From, to, subject, summarize(body, 200))
		return nil
	}

	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}

	msg := buildMessage(m.From, to, subject, body)
	if err := smtp.SendMail(addr, auth, m.From, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send to %s: %w", to, err)
	}
	log.Printf("[email-sent] to=%s subject=%q", to, subject)
	return nil
}

func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

func summarize(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
