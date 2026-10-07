package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"mediguide/internal/config"
)

type resendTransport func(*http.Request) (*http.Response, error)

func (transport resendTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func testResendSender(t *testing.T) ResendSender {
	t.Helper()
	sender, err := New(config.Config{MailDriver: "resend", ResendAPIKey: "private-fixture", MailFrom: "MediGuide <no-reply@example.test>"})
	if err != nil {
		t.Fatal(err)
	}
	return sender.(ResendSender)
}

func TestResendRequiresPrivateKeyAndSender(t *testing.T) {
	for _, cfg := range []config.Config{
		{MailDriver: "resend", MailFrom: "no-reply@example.test"},
		{MailDriver: "resend", ResendAPIKey: "private-fixture"},
		{MailDriver: "resend", ResendAPIKey: "private-fixture", MailFrom: "invalid-address"},
		{MailDriver: "resend", ResendAPIKey: "private-fixture\r\ninvalid", MailFrom: "no-reply@example.test"},
	} {
		_, err := New(cfg)
		if err == nil {
			t.Fatal("incomplete or malformed Resend configuration accepted")
		}
		if strings.Contains(err.Error(), "private-fixture") {
			t.Fatal("configuration error exposed credential")
		}
	}
}

func TestResendSendsAccountEmailWithStableIdempotencyKey(t *testing.T) {
	for _, action := range []string{"verify-email", "reset-password"} {
		t.Run(action, func(t *testing.T) {
			sender := testResendSender(t)
			message := Message{To: "user@example.test", Subject: action, Text: "https://example.test/admin/" + action + "?token=private-token", IdempotencyKey: "account-email/fixture-id"}
			calls := 0
			sender.client.Transport = resendTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost || req.URL.String() != resendEmailEndpoint {
					t.Fatal("incorrect Resend endpoint")
				}
				if req.Header.Get("Authorization") != "Bearer private-fixture" || req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Idempotency-Key") != message.IdempotencyKey {
					t.Fatal("missing authentication, content type, or stable idempotency key")
				}
				var payload struct {
					From, Subject, Text string
					To                  []string
				}
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if payload.From != sender.from || len(payload.To) != 1 || payload.To[0] != message.To || payload.Subject != message.Subject || payload.Text != message.Text {
					t.Fatal("account email content changed")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"provider-message-id"}`)), Header: make(http.Header)}, nil
			})
			for range 2 {
				if err := sender.Send(context.Background(), message); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 2 {
				t.Fatal("adapter performed extra sends")
			}
		})
	}
}

func TestResendRejectsFailuresAndMalformedReceiptsWithoutLeakingProviderBody(t *testing.T) {
	for _, status := range []int{200, 201, 401, 403, 422, 429, 500} {
		sender := testResendSender(t)
		sender.client.Transport = resendTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"message":"private-fixture private-token"}`)), Header: make(http.Header)}, nil
		})
		err := sender.Send(context.Background(), Message{})
		if err == nil {
			t.Fatalf("HTTP %d without acceptance receipt succeeded", status)
		}
		if strings.Contains(err.Error(), "private-fixture") || strings.Contains(err.Error(), "private-token") {
			t.Fatal("provider error exposed private data")
		}
	}
}

func TestResendDoesNotFollowRedirects(t *testing.T) {
	sender := testResendSender(t)
	calls := 0
	sender.client.Transport = resendTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": []string{"https://other.example.test/emails"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if err := sender.Send(context.Background(), Message{Text: "private-token"}); err == nil || calls != 1 {
		t.Fatal("authenticated account email followed a redirect")
	}
}

func TestResendHonorsCancellationAndSanitizesTransportErrors(t *testing.T) {
	sender := testResendSender(t)
	sender.client.Transport = resendTransport(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sender.Send(ctx, Message{}); !errors.Is(err, context.Canceled) {
		t.Fatal("Resend ignored cancellation")
	}
	sender.client.Transport = resendTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("private-fixture private-token")
	})
	err := sender.Send(context.Background(), Message{})
	if err == nil || strings.Contains(err.Error(), "private-fixture") || strings.Contains(err.Error(), "private-token") {
		t.Fatal("transport failure leaked private content or was reported as success")
	}
}
