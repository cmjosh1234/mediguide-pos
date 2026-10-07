package mailer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"mediguide/internal/config"
)

func TestNewSelectsExplicitMailDriver(t *testing.T) {
	disabled, err := New(config.Config{MailDriver: "disabled"})
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(disabled.Send(context.Background(), Message{}), ErrDisabled) {
		t.Fatal("disabled mailer must reject delivery honestly")
	}

	development, err := New(config.Config{MailDriver: "development"})
	if err != nil {
		t.Fatal(err)
	}
	if err := development.Send(context.Background(), Message{To: "user@example.test", Subject: "subject", Text: "body"}); err != nil {
		t.Fatal(err)
	}
}

func TestSMTPConfigurationIsValidated(t *testing.T) {
	if _, err := New(config.Config{MailDriver: "smtp"}); err == nil {
		t.Fatal("expected incomplete SMTP configuration to fail")
	}
	if _, err := New(config.Config{MailDriver: "unknown"}); err == nil {
		t.Fatal("expected unknown mail driver to fail")
	}
}

func TestDevelopmentDriverCannotLeakTokensInProduction(t *testing.T) {
	if _, err := New(config.Config{MailDriver: "development", AppEnv: "production"}); err == nil {
		t.Fatal("development mail enabled in production")
	}
}

func TestSMTPAcceptedAndRejectedDelivery(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprint("reject=", reject), func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			body := make(chan string, 1)
			done := make(chan struct{})
			go func() {
				defer close(done)
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				protocol := textproto.NewConn(conn)
				defer protocol.Close()
				_ = protocol.PrintfLine("220 local.test ESMTP")
				for {
					line, err := protocol.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"), strings.HasPrefix(line, "MAIL"):
						_ = protocol.PrintfLine("250 local.test")
					case strings.HasPrefix(line, "RCPT"):
						if reject {
							_ = protocol.PrintfLine("550 recipient rejected")
						} else {
							_ = protocol.PrintfLine("250 recipient accepted")
						}
					case line == "DATA":
						_ = protocol.PrintfLine("354 send message")
						payload, err := protocol.ReadDotBytes()
						if err != nil {
							return
						}
						body <- string(payload)
						_ = protocol.PrintfLine("250 queued")
					case line == "QUIT":
						_ = protocol.PrintfLine("221 goodbye")
						return
					default:
						_ = protocol.PrintfLine("500 unsupported")
					}
				}
			}()
			sender := SMTPSender{Address: listener.Addr().String(), Host: "127.0.0.1", From: "local@example.test"}
			err = sender.Send(context.Background(), Message{To: "test@example.test", Subject: "Verify", Text: "local test link"})
			if reject && err == nil {
				t.Fatal("SMTP rejection reported as success")
			}
			if !reject {
				if err != nil {
					t.Fatal(err)
				}
				if message := <-body; !strings.Contains(message, "local test link") {
					t.Fatal("message body missing")
				}
			}
			<-done
		})
	}
}

func TestSMTPStopsWhenContextIsCancelled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(time.Second))
		_, _ = io.Copy(io.Discard, conn)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	sender := SMTPSender{Address: listener.Addr().String(), Host: "127.0.0.1", From: "local@example.test"}
	if err := sender.Send(ctx, Message{To: "test@example.test"}); err == nil {
		t.Fatal("stalled SMTP succeeded")
	}
	if time.Since(started) > time.Second {
		t.Fatal("SMTP ignored context cancellation")
	}
}
