package service

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Mailer SMTP 邮件发送（465 直连 SSL / 587 STARTTLS）
type Mailer struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func (m *Mailer) Send(to []string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	from := m.From
	if from == "" {
		from = m.User
	}
	msg := buildMessage(from, to, subject, body)

	auth := smtp.PlainAuth("", m.User, m.Password, m.Host)
	if m.Port == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.Host})
		if err != nil {
			return fmt.Errorf("smtp tls dial: %w", err)
		}
		c, err := smtp.NewClient(conn, m.Host)
		if err != nil {
			return err
		}
		defer c.Close()
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		if err = c.Mail(from); err != nil {
			return err
		}
		for _, rcpt := range to {
			if err = c.Rcpt(rcpt); err != nil {
				return err
			}
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err = w.Write(msg); err != nil {
			return err
		}
		if err = w.Close(); err != nil {
			return err
		}
		return c.Quit()
	}
	return smtp.SendMail(addr, auth, from, to, msg)
}

func buildMessage(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ",") + "\r\n")
	b.WriteString("Subject: =?UTF-8?B?" + base64Encode(subject) + "?=\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
