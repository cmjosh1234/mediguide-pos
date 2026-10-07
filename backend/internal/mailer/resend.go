package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"mediguide/internal/config"
)

const resendEmailEndpoint = "https://api.resend.com/emails"

type ResendSender struct {
	apiKey string
	from   string
	client *http.Client
}

func newResendSender(cfg config.Config) (Sender, error) {
	key, from := strings.TrimSpace(cfg.ResendAPIKey), strings.TrimSpace(cfg.MailFrom)
	if key == "" || from == "" {
		return nil, errors.New("MAIL_DRIVER=resend requires RESEND_API_KEY and MAIL_FROM")
	}
	if strings.ContainsAny(key, "\r\n") {
		return nil, errors.New("invalid RESEND_API_KEY configuration")
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, errors.New("MAIL_FROM must be a valid sender address")
	}
	return ResendSender{
		apiKey: key,
		from:   from,
		client: &http.Client{
			Timeout: 15 * time.Second,
			// Never redirect an authenticated request containing account tokens.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func (sender ResendSender) Send(ctx context.Context, message Message) error {
	payload, err := json.Marshal(struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}{sender.from, []string{message.To}, message.Subject, message.Text})
	if err != nil {
		return errors.New("could not encode Resend email")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEmailEndpoint, bytes.NewReader(payload))
	if err != nil {
		return errors.New("could not create Resend request")
	}
	req.Header.Set("Authorization", "Bearer "+sender.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if message.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", message.IdempotencyKey)
	}
	response, err := sender.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Provider/transport errors may contain credentials or message contents.
		return errors.New("Resend request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Resend rejected email (HTTP %d)", response.StatusCode)
	}
	var receipt struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&receipt); err != nil || strings.TrimSpace(receipt.ID) == "" {
		return errors.New("Resend returned an invalid acceptance receipt")
	}
	return nil
}
