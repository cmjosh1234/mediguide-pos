package services

import (
	"context"
	"errors"
	"mediguide/internal/config"
	"mediguide/internal/models"
	"strings"
	"testing"
	"time"
)

func TestAccountRegistrationValidatesAndQueuesVerification(t *testing.T) {
	s, _ := testPasswordResetService(t)
	for _, in := range []RegisterInput{{Name: "", Email: "new@example.test", Password: "Password8"}, {Name: "User", Email: "bad", Password: "Password8"}, {Name: "User", Email: "new@example.test", Password: "weak"}, {Name: "User", Email: "new@example.test", Password: strings.Repeat("a", 72) + "8"}} {
		if _, err := s.Register(in); err == nil {
			t.Fatal("invalid registration accepted")
		}
	}
	s.Cfg.AppEnv = "production"
	s.Cfg.PublicAppURL = "https://example.test"
	s.Mailer = &recordingMailer{}
	u, err := s.Register(RegisterInput{Name: "New User", Email: " NEW@example.test ", Password: "Password8"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Verified || u.EmailVerified {
		t.Fatal("creation must not imply verification or approval")
	}
	var delivery models.AccountEmailDelivery
	if err := s.DB.Where("user_id = ?", u.ID).First(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	if delivery.Status != "pending" || delivery.EncryptedToken == "" {
		t.Fatal("verification was not durably queued")
	}
	var counts int64
	s.DB.Model(&models.AuditLog{}).Where("entity_id = ? AND action = ?", u.ID.String(), "user.account_created").Count(&counts)
	if counts != 1 {
		t.Fatalf("creation event count %d", counts)
	}
	if err := s.DrainAccountEmails(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.DB.First(&delivery, "id = ?", delivery.ID)
	if delivery.Status != "sent" || delivery.EncryptedToken != "" {
		t.Fatal("sent payload was not cleared")
	}
	if _, err := s.Register(RegisterInput{Name: "Again", Email: u.Email, Password: "Password8"}); err == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestAccountEmailRetriesAreDurableAndTokensNeverAppearInAudits(t *testing.T) {
	s, u := testPasswordResetService(t)
	sender := &recordingMailer{err: errors.New("provider temporarily unavailable")}
	s.Mailer = sender
	s.Cfg.PublicAppURL = "https://example.test"
	result, err := s.RequestPasswordReset(u.Email)
	if err != nil {
		t.Fatal(err)
	}
	var delivery models.AccountEmailDelivery
	s.DB.Where("user_id = ?", u.ID).First(&delivery)
	if delivery.Status != "pending" || delivery.Attempts != 1 || delivery.LastErrorCode != "provider_rejected" {
		t.Fatalf("missing failure/retry state %#v", delivery)
	}
	if strings.Contains(delivery.EncryptedToken, result.DevelopmentToken) {
		t.Fatal("raw token stored")
	}
	var audits []models.AuditLog
	s.DB.Find(&audits)
	for _, audit := range audits {
		if strings.Contains(audit.MetadataJSON, result.DevelopmentToken) {
			t.Fatal("raw token audited")
		}
	}
	s.DB.Model(&delivery).Update("next_attempt_at", time.Now().UTC().Add(-time.Minute))
	sender.err = nil
	if err := s.DrainAccountEmails(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.DB.First(&delivery, "id = ?", delivery.ID)
	if delivery.Status != "sent" || delivery.Attempts != 2 || delivery.SentAt == nil || delivery.EncryptedToken != "" {
		t.Fatalf("retry failed %#v", delivery)
	}
	if err := s.DrainAccountEmails(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 2 {
		t.Fatal("already sent message was delivered twice")
	}
	if sender.messages[0].IdempotencyKey != "account-email/"+delivery.ID.String() || sender.messages[0].IdempotencyKey != sender.messages[1].IdempotencyKey {
		t.Fatal("account email retries must reuse the same provider idempotency key")
	}
	if err := s.ConfirmPasswordReset(result.DevelopmentToken, "NewPassword9"); err != nil {
		t.Fatal(err)
	}
	var n int64
	s.DB.Model(&models.AuditLog{}).Where("action = ?", "user.password_reset_completed").Count(&n)
	if n != 1 {
		t.Fatal("missing reset completion audit")
	}
}
func TestAccountEmailCancelsSupersededAndConsumedTokens(t *testing.T) {
	s, u := testPasswordResetService(t)
	s.Cfg.AppEnv = "production"
	s.Cfg.PublicAppURL = "https://example.test"
	sender := &recordingMailer{}
	s.Mailer = sender
	if _, err := s.RequestPasswordReset(u.Email); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RequestPasswordReset(u.Email); err != nil {
		t.Fatal(err)
	}
	if err := s.DrainAccountEmails(context.Background()); err != nil {
		t.Fatal(err)
	}
	var deliveries []models.AccountEmailDelivery
	s.DB.Find(&deliveries)
	var sent, cancelled int
	for _, d := range deliveries {
		if d.Status == "sent" {
			sent++
		}
		if d.Status == "cancelled" {
			cancelled++
		}
	}
	if sent != 1 || cancelled != 1 || len(sender.messages) != 1 {
		t.Fatalf("wrong delivery eligibility: sent=%d cancelled=%d", sent, cancelled)
	}
}
func TestAccountEmailHonorsLeaseAndExhaustsRetries(t *testing.T) {
	s, u := testPasswordResetService(t)
	s.Cfg.AppEnv = "production"
	s.Cfg.PublicAppURL = "https://example.test"
	sender := &recordingMailer{err: errors.New("offline")}
	s.Mailer = sender
	s.RequestPasswordReset(u.Email)
	var d models.AccountEmailDelivery
	s.DB.First(&d)
	lease := time.Now().UTC().Add(time.Hour)
	s.DB.Model(&d).Updates(map[string]any{"status": "sending", "lease_until": lease})
	s.DrainAccountEmails(context.Background())
	if len(sender.messages) != 0 {
		t.Fatal("active lease stolen")
	}
	s.DB.Model(&d).Updates(map[string]any{"lease_until": time.Now().UTC().Add(-time.Minute), "attempts": 4})
	s.DrainAccountEmails(context.Background())
	s.DB.First(&d, "id = ?", d.ID)
	if d.Status != "failed" || d.Attempts != 5 || d.EncryptedToken != "" {
		t.Fatalf("retry limit ignored %#v", d)
	}
	var failures int64
	s.DB.Model(&models.AuditLog{}).Where("entity_id = ? AND action = ?", d.UserID.String(), "user.account_email_failed").Count(&failures)
	if failures != 1 {
		t.Fatal("terminal mail failure was not counted once")
	}

}
func TestAccountEmailLinkUsesDashboardBasePath(t *testing.T) {
	for _, base := range []string{"https://example.test", "https://example.test/admin", "https://example.test/admin/login"} {
		s := AuthService{Cfg: config.Config{AppEnv: "production", PublicAppURL: base}}
		link, err := s.accountActionLink("reset-password", "token+value")
		if err != nil || link != "https://example.test/admin/reset-password?token=token%2Bvalue" {
			t.Fatalf("invalid link %s: %v", link, err)
		}
	}
	s := AuthService{Cfg: config.Config{AppEnv: "production", AccountActionURL: "http://example.test"}}
	if _, err := s.accountActionLink("verify-email", "x"); err == nil {
		t.Fatal("unsafe production link accepted")
	}
}
func TestEmailOwnershipDoesNotGrantAdministrativeApproval(t *testing.T) {
	s, u := testPasswordResetService(t)
	s.Mailer = &recordingMailer{}
	s.Cfg.PublicAppURL = "https://example.test"
	result, err := s.RequestEmailVerification(u.Email)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmEmailVerification(result.DevelopmentToken); err != nil {
		t.Fatal(err)
	}
	s.DB.First(&u, "id = ?", u.ID)
	if !u.EmailVerified || u.Verified {
		t.Fatal("email ownership granted administrative approval")
	}
}

func TestEmailChangeInvalidatesOldLinksAndResetsOwnership(t *testing.T) {
	s, u := testPasswordResetService(t)
	if err := s.DB.AutoMigrate(&models.Role{}, &models.Permission{}); err != nil {
		t.Fatal(err)
	}
	s.Cfg.PublicAppURL = "https://example.test"
	old, err := s.RequestEmailVerification(u.Email)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmEmailVerification(old.DevelopmentToken); err != nil {
		t.Fatal(err)
	}
	reset, err := s.RequestPasswordReset(u.Email)
	if err != nil {
		t.Fatal(err)
	}
	email := "new-address@example.test"
	if _, err := (UserService{DB: s.DB, Auth: &s}).UpdateUser(u.ID, UserUpdateInput{Email: &email}); err != nil {
		t.Fatal(err)
	}
	var updated models.User
	if err := s.DB.First(&updated, "id = ?", u.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.EmailVerified || updated.Email != email {
		t.Fatal("new email inherited old email ownership")
	}
	if err := s.ConfirmPasswordReset(reset.DevelopmentToken, "NewPassword8"); err == nil {
		t.Fatal("old email reset link still valid")
	}
	var pending int64
	s.DB.Model(&models.AccountActionToken{}).Where("user_id = ? AND purpose = ? AND consumed_at IS NULL", u.ID, "email_verification").Count(&pending)
	if pending != 1 {
		t.Fatalf("expected new verification token, got %d", pending)
	}
}

func TestEmailVerificationDatabaseFailureIsNotSilentlyAccepted(t *testing.T) {
	s, _ := testPasswordResetService(t)
	if err := s.DB.Migrator().DropTable(&models.User{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RequestEmailVerification("known@example.test"); err == nil {
		t.Fatal("database failure was silently accepted")
	}
}
