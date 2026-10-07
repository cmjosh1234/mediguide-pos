package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"

	"mediguide/internal/config"
)

var ErrDisabled = errors.New("email delivery is disabled")

type Message struct {
	To      string
	Subject string
	Text    string
	// Stable per queued email, including across retries and lease recovery.
	IdempotencyKey string
}

type Sender interface {
	Send(context.Context, Message) error
}

func New(cfg config.Config) (Sender, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.MailDriver)) {
	case "", "disabled":
		return DisabledSender{}, nil
	case "development":
		if cfg.AppEnv == "production" {
			return nil, errors.New("development mail driver is not allowed in production")
		}
		return DevelopmentSender{}, nil
	case "resend":
		return newResendSender(cfg)
	case "smtp":
		if cfg.SMTPHost == "" || cfg.SMTPPort == 0 || cfg.MailFrom == "" {
			return nil, errors.New("MAIL_DRIVER=smtp requires SMTP_HOST, SMTP_PORT, and MAIL_FROM")
		}
		return SMTPSender{
			Address:  fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort),
			Host:     cfg.SMTPHost,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.MailFrom,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported MAIL_DRIVER %q", cfg.MailDriver)
	}
}

type DisabledSender struct{}

func (DisabledSender) Send(context.Context, Message) error { return ErrDisabled }

// DevelopmentSender accepts messages locally and writes their content to the
// development log. It must never be enabled in production.
type DevelopmentSender struct{}

func (DevelopmentSender) Send(_ context.Context, message Message) error {
	log.Printf("development email accepted to=%q subject=%q body=%q", message.To, message.Subject, message.Text)
	return nil
}

type SMTPSender struct {
	Address  string
	Host     string
	Username string
	Password string
	From     string
}

func (sender SMTPSender) Send(ctx context.Context, message Message) error {
	var auth smtp.Auth
	if sender.Username != "" {
		auth = smtp.PlainAuth("", sender.Username, sender.Password, sender.Host)
	}
	payload := []byte("From: " + sender.From + "\r\n" +
		"To: " + message.To + "\r\n" +
		"Subject: " + message.Subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + message.Text)
	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", sender.Address)
	if err != nil {
		return err
	}
	defer conn.Close()
	stopCancellation := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stopCancellation()
	deadline := time.Now().Add(15 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, sender.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: sender.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if sender.Host != "localhost" && sender.Host != "127.0.0.1" {
		return errors.New("SMTP provider must support STARTTLS")
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(sender.From); err != nil {
		return err
	}
	if err := client.Rcpt(message.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write(payload); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	// DATA acceptance is the delivery boundary. A failed QUIT must not resend
	// an already accepted message.
	_ = client.Quit()
	return nil
}
