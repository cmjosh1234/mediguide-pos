package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"mediguide/internal/mailer"
	"mediguide/internal/models"
)

func accountLifecycleAudit(tx *gorm.DB, userID uuid.UUID, event string) error {
	return tx.Create(&models.AuditLog{ActorID: userID.String(), EntityType: "user", EntityID: userID.String(), Action: "user." + event, MetadataJSON: "{}"}).Error
}

func (s AuthService) tokenCipher() (cipher.AEAD, error) {
	key := sha256.Sum256([]byte("mediguide-account-email:" + s.Cfg.JWTSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
func (s AuthService) queueAccountEmail(tx *gorm.DB, token models.AccountActionToken, raw string) error {
	aead, err := s.tokenCipher()
	if err != nil {
		return err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	sealed := aead.Seal(nonce, nonce, []byte(raw), []byte(token.ID.String()))
	return tx.Create(&models.AccountEmailDelivery{UserID: token.UserID, TokenID: token.ID, Purpose: token.Purpose, EncryptedToken: base64.RawStdEncoding.EncodeToString(sealed), Status: "pending", NextAttemptAt: time.Now().UTC()}).Error
}

func (s AuthService) accountActionLink(path, token string) (string, error) {
	base := strings.TrimSpace(s.Cfg.AccountActionURL)
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(s.Cfg.PublicAppURL), "/")
		parsed, err := url.Parse(base)
		if err != nil {
			return "", err
		}
		// Legacy deployments used either the public site root or a dashboard login URL.
		if strings.HasSuffix(parsed.Path, "/login") {
			parsed.Path = strings.TrimSuffix(parsed.Path, "/login")
		}
		if parsed.Path == "" && s.Cfg.AppEnv == "production" {
			parsed.Path = "/admin"
		}
		base = parsed.String()
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(s.Cfg.AppEnv == "development" && parsed.Scheme == "http")) || parsed.User != nil {
		return "", errors.New("invalid account action URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + path
	parsed.RawQuery = url.Values{"token": {token}}.Encode()
	parsed.Fragment = ""
	return parsed.String(), nil
}

// Development previews deliver immediately; production requests only queue work,
// avoiding provider latency and keeping account-existence responses neutral.
func (s AuthService) processAccountEmail(tokenID uuid.UUID) {
	if s.Cfg.AppEnv != "development" {
		return
	}
	var delivery models.AccountEmailDelivery
	if err := s.DB.Where("token_id = ?", tokenID).First(&delivery).Error; err == nil {
		s.deliverAccountEmail(context.Background(), delivery)
	}
}

// DrainAccountEmails is safe across replicas: a compare-and-set lease claims work.
func (s AuthService) DrainAccountEmails(ctx context.Context) error {
	now := time.Now().UTC()
	var pending []models.AccountEmailDelivery
	if err := s.DB.WithContext(ctx).Where("(status = ? AND next_attempt_at <= ?) OR (status = ? AND lease_until < ?)", "pending", now, "sending", now).Order("next_attempt_at ASC").Limit(20).Find(&pending).Error; err != nil {
		return err
	}
	for _, delivery := range pending {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.deliverAccountEmail(ctx, delivery)
	}
	return nil
}

func (s AuthService) deliverAccountEmail(ctx context.Context, d models.AccountEmailDelivery) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	lease := now.Add(2 * time.Minute)
	claim := s.DB.WithContext(ctx).Model(&models.AccountEmailDelivery{}).Where("id = ? AND ((status = ? AND next_attempt_at <= ?) OR (status = ? AND lease_until < ?))", d.ID, "pending", now, "sending", now).Updates(map[string]any{"status": "sending", "lease_until": lease, "attempts": gorm.Expr("attempts + 1")})
	if claim.Error != nil || claim.RowsAffected != 1 {
		return
	}
	d.Attempts++
	var token models.AccountActionToken
	if err := s.DB.Where("id = ? AND consumed_at IS NULL AND expires_at > ?", d.TokenID, now).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.finishAccountEmail(d, lease, "cancelled", "token_inactive", nil)
		} else {
			s.retryAccountEmail(d, lease, "database_unavailable")
		}
		return
	}
	var user models.User
	if err := s.DB.Where("id = ? AND deleted_at IS NULL", d.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.finishAccountEmail(d, lease, "cancelled", "account_unavailable", nil)
		} else {
			s.retryAccountEmail(d, lease, "database_unavailable")
		}
		return
	}
	aead, err := s.tokenCipher()
	if err != nil {
		s.retryAccountEmail(d, lease, "encryption_unavailable")
		return
	}
	sealed, err := base64.RawStdEncoding.DecodeString(d.EncryptedToken)
	if err != nil || len(sealed) < aead.NonceSize() {
		s.finishAccountEmail(d, lease, "failed", "invalid_payload", nil)
		return
	}
	raw, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], []byte(d.TokenID.String()))
	if err != nil {
		s.finishAccountEmail(d, lease, "failed", "invalid_payload", nil)
		return
	}
	path, subject := "verify-email", "Verify your MediGuide email"
	if d.Purpose == "password_reset" {
		path, subject = "reset-password", "Reset your MediGuide password"
	}
	link, err := s.accountActionLink(path, string(raw))
	if err != nil {
		s.retryAccountEmail(d, lease, "invalid_link_configuration")
		return
	}
	if s.Mailer == nil {
		s.retryAccountEmail(d, lease, "mail_disabled")
		return
	}
	sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := s.Mailer.Send(sendCtx, mailer.Message{To: user.Email, Subject: subject, Text: fmt.Sprintf("Open this link to continue: %s\n\nIf you did not request this action, ignore this message.", link), IdempotencyKey: "account-email/" + d.ID.String()}); err != nil {
		code := "provider_rejected"
		if errors.Is(err, mailer.ErrDisabled) {
			code = "mail_disabled"
		}
		s.retryAccountEmail(d, lease, code)
		return
	}
	sentAt := time.Now().UTC()
	s.finishAccountEmail(d, lease, "sent", "", &sentAt)
}
func (s AuthService) retryAccountEmail(d models.AccountEmailDelivery, lease time.Time, code string) {
	status := "pending"
	if d.Attempts >= 5 {
		status = "failed"
	}
	next := time.Now().UTC().Add(time.Duration(1<<d.Attempts) * time.Minute)
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": status, "lease_until": nil, "next_attempt_at": next, "last_error_code": code}
		if status == "failed" {
			updates["encrypted_token"] = ""
		}
		result := tx.Model(&models.AccountEmailDelivery{}).Where("id = ? AND status = ? AND lease_until = ?", d.ID, "sending", lease).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		if err := accountLifecycleAudit(tx, d.UserID, "account_email_send_failed"); err != nil {
			return err
		}
		if status == "failed" {
			return accountLifecycleAudit(tx, d.UserID, "account_email_failed")
		}
		return nil
	})
	log.Error().Str("event", "account_email_send_failed").Str("delivery_id", d.ID.String()).Str("reason", code).Int("attempt", d.Attempts).Bool("recorded", err == nil).Msg("account email needs retry or operator attention")
}
func (s AuthService) finishAccountEmail(d models.AccountEmailDelivery, lease time.Time, status, code string, sentAt *time.Time) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.AccountEmailDelivery{}).Where("id = ? AND status = ? AND lease_until = ?", d.ID, "sending", lease).Updates(map[string]any{"status": status, "lease_until": nil, "sent_at": sentAt, "last_error_code": code, "encrypted_token": ""})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		return accountLifecycleAudit(tx, d.UserID, "account_email_"+status)
	})
	if err != nil {
		log.Error().Str("event", "account_email_result_persistence_failed").Str("delivery_id", d.ID.String()).Msg("account email result could not be persisted")
	}
}

func (s AuthService) queueVerificationForUser(tx *gorm.DB, user models.User) error {
	raw, err := generateRefreshToken()
	if err != nil {
		return err
	}
	token := models.AccountActionToken{UserID: user.ID, Purpose: "email_verification", TokenHash: hashRefreshToken(raw), ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}
	if err := tx.Create(&token).Error; err != nil {
		return err
	}
	if err := s.queueAccountEmail(tx, token, raw); err != nil {
		return err
	}
	return accountLifecycleAudit(tx, user.ID, "email_verification_requested")
}
