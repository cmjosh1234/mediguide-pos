package services

import (
	"errors"
	"mediguide/internal/models"
	"testing"
)

func TestAccountDeletionRequiresCredentialsAndDeduplicates(t *testing.T) {
	service, user := testPasswordResetService(t)
	if err := service.DB.AutoMigrate(&models.SupportTicket{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestAccountDeletion(nil, user.Email, "wrong"); !errors.Is(err, ErrDeletionCredentials) {
		t.Fatalf("wrong password accepted: %v", err)
	}
	if _, err := service.RequestAccountDeletion(nil, "missing@example.test", "OriginalPassword8"); !errors.Is(err, ErrDeletionCredentials) {
		t.Fatalf("unknown account exposed: %v", err)
	}
	result, err := service.RequestAccountDeletion(&user.ID, "ignored@example.test", "OriginalPassword8")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "requested" {
		t.Fatalf("unexpected status %s", result.Status)
	}
	again, err := service.RequestAccountDeletion(nil, user.Email, "OriginalPassword8")
	if err != nil {
		t.Fatal(err)
	}
	if again.RequestID != result.RequestID {
		t.Fatal("duplicate active deletion request")
	}
	var count int64
	service.DB.Model(&models.SupportTicket{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected one ticket, got %d", count)
	}
	service.DB.Model(&models.AuditLog{}).Where("action = ?", "user.deletion_requested").Count(&count)
	if count != 1 {
		t.Fatalf("expected one audit record, got %d", count)
	}
	var ticket models.SupportTicket
	if err := service.DB.First(&ticket, "id = ?", result.RequestID).Error; err != nil {
		t.Fatal(err)
	}
	if ticket.UserID == nil || *ticket.UserID != user.ID || ticket.Status != "open" {
		t.Fatal("request owner/status incorrect")
	}
	// Creating a request must not silently erase or deactivate the account.
	var unchanged models.User
	if err := service.DB.First(&unchanged, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !unchanged.IsActive {
		t.Fatal("request unexpectedly deactivated account")
	}
}

func TestAccountDeletionRejectsInactiveAccount(t *testing.T) {
	service, user := testPasswordResetService(t)
	if err := service.DB.Model(&user).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestAccountDeletion(&user.ID, "", "OriginalPassword8"); !errors.Is(err, ErrDeletionCredentials) {
		t.Fatalf("inactive account accepted: %v", err)
	}
}

func TestAccountDeletionRollsBackWhenAuditFails(t *testing.T) {
	service, user := testPasswordResetService(t)
	if err := service.DB.AutoMigrate(&models.SupportTicket{}); err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Migrator().DropTable(&models.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RequestAccountDeletion(&user.ID, "", "OriginalPassword8"); err == nil {
		t.Fatal("expected audit failure")
	}
	var count int64
	service.DB.Model(&models.SupportTicket{}).Count(&count)
	if count != 0 {
		t.Fatal("request persisted without audit")
	}
}

func TestGenericSupportCannotForgeVerifiedDeletion(t *testing.T) {
	category := AccountDeletionCategory
	_, err := newSupportTicket(SupportTicketCreate{
		Subject: "Delete", Description: "Forged verification", Category: &category,
	})
	if !errors.Is(err, ErrSupportInvalid) {
		t.Fatalf("reserved category accepted: %v", err)
	}
}
