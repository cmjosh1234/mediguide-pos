package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCalculatorServiceCRUDAndFilters(t *testing.T) {
	service := testCalculatorService(t)
	userID := uuid.New()

	created, err := service.Create(userID, CreateCalculatorInput{
		Name:        "Emergency Triage",
		Description: "Urgency assessment",
		AppFileJSON: json.RawMessage(`{"path":"emergency-triage-assessment.html"}`),
		Version:     "1.0.0",
		Type:        "decision_tool",
		Status:      "active",
		Featured:    true,
		CategoryID:  testCategoryID(t, service),
	})
	if err != nil {
		t.Fatalf("create calculator: %v", err)
	}

	result, err := service.List(CalculatorListInput{
		Page:   PageInput{Page: 1, PerPage: 10},
		Search: "triage",
		Type:   "decision_tool",
		Status: "active",
		Sort:   "created_at",
		Order:  "desc",
	})
	if err != nil {
		t.Fatalf("list calculators: %v", err)
	}
	if result.TotalItems != 1 || len(result.Items) != 1 || result.Items[0].ID != created.ID {
		t.Fatalf("unexpected calculator list: %#v", result)
	}

	category := testCategoryID(t, service).String()
	byCategory, err := service.List(CalculatorListInput{Page: PageInput{Page: 1, PerPage: 10}, CategoryID: category})
	if err != nil || byCategory.TotalItems != 1 {
		t.Fatalf("category filter: %v %#v", err, byCategory)
	}
	uncategorised, err := service.List(CalculatorListInput{Page: PageInput{Page: 1, PerPage: 10}, CategoryID: "none"})
	if err != nil || uncategorised.TotalItems != 0 {
		t.Fatalf("uncategorised filter: %v %#v", err, uncategorised)
	}
	if _, err := service.Create(userID, CreateCalculatorInput{Name: "No category", Version: "1", Type: "calculator", AppFileJSON: json.RawMessage(`{"path":"bmi-calculator.html"}`)}); !errors.Is(err, ErrCalculatorCategoryRequired) {
		t.Fatalf("expected a category to be required, got %v", err)
	}

	name := "Emergency Triage Assessment"
	status := "archived"
	updated, err := service.Update(created.ID, CalculatorEditor{UserID: userID}, UpdateCalculatorInput{Name: &name, Status: &status})
	if err != nil {
		t.Fatalf("update calculator: %v", err)
	}
	if updated.Name != name || updated.Status != status {
		t.Fatalf("unexpected updated calculator: %#v", updated)
	}

	if err := service.DB.Model(&models.ToolCategory{}).Where("id = ?", *testCategoryID(t, service)).Update("status", "inactive").Error; err != nil {
		t.Fatal(err)
	}
	description := "Edited after its category was retired"
	if _, err := service.Update(created.ID, CalculatorEditor{UserID: userID}, UpdateCalculatorInput{Description: &description, CategoryID: testCategoryID(t, service)}); err != nil {
		t.Fatalf("editing a tool in a retired category: %v", err)
	}

	if err := service.Delete(created.ID); err != nil {
		t.Fatalf("delete calculator: %v", err)
	}
	if _, err := service.Get(created.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected deleted calculator to be unavailable, got %v", err)
	}
}

func TestCalculatorArtifactContainsReviewedStaticHTML(t *testing.T) {
	root := t.TempDir()
	content := reviewedLegacyFixture(t, "bmi-calculator.html")
	if err := os.WriteFile(filepath.Join(root, "bmi-calculator.html"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	artifact, err := resolveCalculatorArtifact([]byte(`{"path":"bmi-calculator.html"}`), root)
	if err != nil {
		t.Fatalf("resolve static artifact: %v", err)
	}
	if !strings.Contains(string(artifact.Content), "Content-Security-Policy") || artifact.ContentType != "text/html; charset=utf-8" || artifact.Checksum == "" {
		t.Fatalf("unexpected static artifact: %#v", artifact)
	}

	if _, err := resolveCalculatorArtifact([]byte(`{"name":"custom.html","html":"<html><head></head><body>Custom</body></html>"}`), root); !errors.Is(err, ErrCalculatorArtifactUnsafe) {
		t.Fatalf("expected embedded HTML rejection, got %v", err)
	}

	if _, err := resolveCalculatorArtifact([]byte(`{"path":"../secret.html"}`), root); !errors.Is(err, ErrCalculatorArtifactUnsafe) {
		t.Fatalf("expected path traversal rejection, got %v", err)
	}
}

func TestLegacyCalculatorArtifactRejectsDriftRemoteDependenciesAndOversize(t *testing.T) {
	if _, err := verifyLegacyCalculatorArtifact("bmi-calculator.html", []byte("changed")); !errors.Is(err, ErrCalculatorArtifactChecksum) {
		t.Fatalf("expected checksum rejection, got %v", err)
	}
	original := reviewedLegacyCalculatorChecksums["bmi-calculator.html"]
	content := []byte(`<html><head><script src="https://example.org/x.js"></script></head></html>`)
	digest := sha256.Sum256(content)
	reviewedLegacyCalculatorChecksums["bmi-calculator.html"] = hex.EncodeToString(digest[:])
	t.Cleanup(func() { reviewedLegacyCalculatorChecksums["bmi-calculator.html"] = original })
	if _, err := verifyLegacyCalculatorArtifact("bmi-calculator.html", content); !errors.Is(err, ErrCalculatorArtifactDependency) {
		t.Fatalf("expected remote dependency rejection, got %v", err)
	}
	if _, err := verifyLegacyCalculatorArtifact("bmi-calculator.html", make([]byte, maxLegacyCalculatorArtifactBytes+1)); !errors.Is(err, ErrCalculatorArtifactUnsafe) {
		t.Fatalf("expected size rejection, got %v", err)
	}
}

func TestEveryReviewedLegacyCalculatorArtifactPassesContainment(t *testing.T) {
	if len(reviewedLegacyCalculatorChecksums) != 14 {
		t.Fatalf("reviewed artifact allowlist has %d entries, want 14", len(reviewedLegacyCalculatorChecksums))
	}
	for filename := range reviewedLegacyCalculatorChecksums {
		content := reviewedLegacyFixture(t, filename)
		if _, err := verifyLegacyCalculatorArtifact(filename, content); err != nil {
			t.Fatalf("%s failed containment: %v", filename, err)
		}
		if _, err := containLegacyCalculatorHTML(content); err != nil {
			t.Fatalf("%s cannot receive CSP containment: %v", filename, err)
		}
	}
}

func TestCalculatorUsageIsOwnedByAuthenticatedUser(t *testing.T) {
	service := testCalculatorService(t)
	userID := uuid.New()
	calculator, err := service.Create(userID, CreateCalculatorInput{
		Name:        "BMI",
		AppFileJSON: json.RawMessage(`{"path":"bmi-calculator.html"}`),
		Version:     "1",
		Type:        "calculator",
		Status:      "active",
		CategoryID:  testCategoryID(t, service),
	})
	if err != nil {
		t.Fatal(err)
	}
	versionID := uuid.New()
	if err := service.DB.Model(&models.Calculator{}).Where("id = ?", calculator.ID).Update("current_version_id", versionID).Error; err != nil {
		t.Fatal(err)
	}

	usage, err := service.StartUsage(userID, calculator.ID, StartCalculatorUsageInput{
		SessionStart:   "2026-07-30T12:00:00Z",
		CalculatorType: "calculator",
	})
	if err != nil {
		t.Fatalf("start usage: %v", err)
	}
	if usage.CalculatorVersionID == nil || *usage.CalculatorVersionID != versionID {
		t.Fatalf("usage did not retain the active immutable version: %#v", usage)
	}

	if _, err := service.FinishUsage(uuid.New(), usage.ID, FinishCalculatorUsageInput{
		SessionEnd: "2026-07-30T12:05:00Z",
	}); !errors.Is(err, ErrCalculatorUsageForbidden) {
		t.Fatalf("expected usage ownership failure, got %v", err)
	}
	if _, err := service.FinishUsage(userID, usage.ID, FinishCalculatorUsageInput{
		SessionEnd: "2026-07-30T12:05:00Z",
	}); err != nil {
		t.Fatalf("finish owned usage: %v", err)
	}
}

func reviewedLegacyFixture(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "dashboard", "samples", name))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestCalculatorValidationRejectsUnknownEnumsAndEmptyArtifacts(t *testing.T) {
	if err := validateCalculatorInput("BMI", "1", "unknown", "active", json.RawMessage(`{}`)); !errors.Is(err, ErrCalculatorInvalidPayload) {
		t.Fatalf("expected invalid type, got %v", err)
	}
	if err := validateCalculatorInput("BMI", "1", "calculator", "active", nil); !errors.Is(err, ErrCalculatorInvalidPayload) {
		t.Fatalf("expected missing artifact, got %v", err)
	}
}

func testCalculatorService(t *testing.T) CalculatorService {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.Calculator{}, &models.CalculatorUsageLog{}, &models.CalculatorVersion{}, &models.ToolCategory{}, &models.CalculatorVersionAudit{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&models.ToolCategory{Name: "General Assessment", Slug: "general-assessment", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	return CalculatorService{DB: database, LegacyClinicalToolsDir: t.TempDir()}
}

// testCategoryID returns the active category every test service starts with.
func testCategoryID(t *testing.T, s CalculatorService) *uuid.UUID {
	t.Helper()
	var category models.ToolCategory
	if err := s.DB.First(&category, "slug = ?", "general-assessment").Error; err != nil {
		t.Fatal(err)
	}
	return &category.ID
}

func TestCalculatorUsageRetriesValidationAndExactVersion(t *testing.T) {
	s := testCalculatorService(t)
	owner := uuid.New()
	tool, err := s.Create(owner, CreateCalculatorInput{Name: "BMI", Version: "1", Type: "calculator", Status: "active", CategoryID: testCategoryID(t, s), AppFileJSON: json.RawMessage(`{"path":"bmi-calculator.html"}`)})
	if err != nil {
		t.Fatal(err)
	}
	published := time.Now().UTC()
	oldVersion := models.CalculatorVersion{CalculatorID: tool.ID, SemanticVersion: "1", SchemaVersion: "1", DefinitionJSON: datatypes.JSON(`{}`), DefinitionChecksum: "old", PublishedAt: &published, Status: "superseded"}
	if err := s.DB.Create(&oldVersion).Error; err != nil {
		t.Fatal(err)
	}
	in := StartCalculatorUsageInput{SessionStart: "2026-07-30T15:00:00+03:00", CalculatorType: "calculator", CalculatorVersionID: &oldVersion.ID, IdempotencyKey: "offline-retry"}
	first, err := s.StartUsage(owner, tool.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.StartUsage(owner, tool.ID, in)
	if err != nil || replay.ID != first.ID || *first.CalculatorVersionID != oldVersion.ID {
		t.Fatalf("retry/version: %#v %v", replay, err)
	}
	updated, _ := s.Get(tool.ID)
	if updated.UsageCount != 1 {
		t.Fatalf("retry counted twice: %d", updated.UsageCount)
	}
	if _, err := s.StartUsage(uuid.New(), tool.ID, in); err != nil {
		t.Fatal(err)
	}
	updated, _ = s.Get(tool.ID)
	if updated.UsageCount != 2 {
		t.Fatal("key should be scoped to user")
	}
	for _, mutation := range []func(*StartCalculatorUsageInput){
		func(i *StartCalculatorUsageInput) { i.SessionStart = "invalid" },
		func(i *StartCalculatorUsageInput) { i.CalculatorType = "checklist" },
		func(i *StartCalculatorUsageInput) { i.SessionStart = "2026-07-30T12:01:00Z" },
		func(i *StartCalculatorUsageInput) { id := uuid.New(); i.CalculatorVersionID = &id },
	} {
		invalid := in
		mutation(&invalid)
		if _, err := s.StartUsage(owner, tool.ID, invalid); !errors.Is(err, ErrCalculatorInvalidPayload) {
			t.Fatalf("invalid start accepted: %v", err)
		}
	}
	draft := oldVersion
	draft.ID = uuid.New()
	draft.PublishedAt = nil
	draft.Status = "draft"
	if err := s.DB.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}
	invalid := in
	invalid.CalculatorVersionID = &draft.ID
	if _, err := s.StartUsage(owner, tool.ID, invalid); !errors.Is(err, ErrCalculatorInvalidPayload) {
		t.Fatalf("draft accepted: %v", err)
	}
	for _, end := range []string{"invalid", "2026-07-30T11:59:59Z"} {
		if _, err := s.FinishUsage(owner, first.ID, FinishCalculatorUsageInput{SessionEnd: end}); !errors.Is(err, ErrCalculatorInvalidPayload) {
			t.Fatalf("invalid end accepted: %v", err)
		}
	}
	finish := FinishCalculatorUsageInput{SessionEnd: "2026-07-30T12:00:01Z"}
	for range 2 {
		if _, err := s.FinishUsage(owner, first.ID, finish); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.FinishUsage(owner, first.ID, FinishCalculatorUsageInput{SessionEnd: "2026-07-30T12:01:00Z"}); !errors.Is(err, ErrCalculatorInvalidPayload) {
		t.Fatalf("closed session altered: %v", err)
	}
}
