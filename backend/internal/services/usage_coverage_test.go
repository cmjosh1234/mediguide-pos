package services

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"mediguide/internal/models"
	"testing"
	"time"
)

func TestUsageCoverageCountersAndRetryKeys(t *testing.T) {
	s := progressUsageTestService(t)
	if err := s.DB.AutoMigrate(&models.Drug{}, &models.HealthFacility{}); err != nil {
		t.Fatal(err)
	}
	owner := uuid.New()
	abbreviation := models.Abbreviation{Abbreviation: "ABC", Meaning: "Test abbreviation"}
	drug := models.Drug{Name: "Test drug"}
	facility := models.HealthFacility{Name: "Test facility"}
	for _, value := range []any{&abbreviation, &drug, &facility} {
		if err := s.DB.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	document := createProgressTestGuideline(t, s)
	for _, item := range []struct {
		kind, resourceType string
		id                 uuid.UUID
		model              any
	}{
		{"guideline", "guideline_document", document, nil},
		{"abbreviation", "", abbreviation.ID, &models.Abbreviation{}},
	} {
		resource := item.id.String()
		in := UsageEventInput{ResourceID: &resource, ResourceType: item.resourceType, IdempotencyKey: item.resourceType + item.kind}
		for range 2 {
			if _, err := s.RecordUsage(owner, item.kind, in); err != nil {
				t.Fatal(err)
			}
		}
		if item.model != nil {
			var count int64
			if err := s.DB.Model(item.model).Where("id = ?", item.id).Select("usage_count").Scan(&count).Error; err != nil || count != 1 {
				t.Fatalf("%s counter = %d, %v", item.kind, count, err)
			}
		}
		missing := uuid.New().String()
		in.ResourceID = &missing
		if _, err := s.RecordUsage(owner, item.kind, in); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("missing %s accepted: %v", item.kind, err)
		}
	}
	for range 2 {
		if _, err := (DrugService{DB: s.DB}).RecordUsage(owner, drug.ID, "drug-retry"); err != nil {
			t.Fatal(err)
		}
		if _, err := (FacilityService{DB: s.DB}).RecordUsage(owner, facility.ID, "facility-retry"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RecordUsage(owner, "ai", UsageEventInput{IdempotencyKey: "ai-retry"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RecordUsage(owner, "feature", UsageEventInput{IdempotencyKey: "feature-retry", Feature: "outbreaks"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct {
		model any
		id    uuid.UUID
	}{{&models.Drug{}, drug.ID}, {&models.HealthFacility{}, facility.ID}} {
		var count int64
		s.DB.Model(item.model).Where("id = ?", item.id).Select("usage_count").Scan(&count)
		if count != 1 {
			t.Fatalf("counter duplicated: %d", count)
		}
	}
	// A retry key cannot be reassigned to a different resource or feature.
	other := models.Drug{Name: "Other drug"}
	s.DB.Create(&other)
	if _, err := (DrugService{DB: s.DB}).RecordUsage(owner, other.ID, "drug-retry"); !errors.Is(err, ErrProgressUsageInvalid) {
		t.Fatalf("key reused for another drug: %v", err)
	}
	if _, err := s.RecordUsage(owner, "feature", UsageEventInput{IdempotencyKey: "feature-retry", Feature: "search"}); !errors.Is(err, ErrProgressUsageInvalid) {
		t.Fatalf("feature key changed: %v", err)
	}
	if _, err := s.RecordUsage(owner, "feature", UsageEventInput{IdempotencyKey: "invalid-feature", Feature: "raw search text"}); !errors.Is(err, ErrProgressUsageInvalid) {
		t.Fatalf("invalid feature accepted: %v", err)
	}
	// Historical opens remain part of engagement after their content table is retired.
	if err := s.DB.Create(&models.HistoricalGuidelineUsageLog{UserID: owner, LegacyGuidelineID: uuid.New()}).Error; err != nil {
		t.Fatal(err)
	}
	retiredID := uuid.New().String()
	if _, err := s.RecordUsage(owner, "guideline", UsageEventInput{ResourceID: &retiredID, ResourceType: "medical_guideline", IdempotencyKey: "retired"}); !errors.Is(err, ErrProgressUsageInvalid) {
		t.Fatalf("retired resource type accepted: %v", err)
	}
	rows, err := s.UsageAggregates(nil)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int64{}
	for _, row := range rows {
		counts[row.EventType] = row.Count
	}
	for key, want := range map[string]int64{"guideline": 2, "abbreviation": 1, "drug": 1, "facility": 1, "ai": 1, "feature_outbreaks": 1} {
		if counts[key] != want {
			t.Fatalf("%s aggregate = %d want %d", key, counts[key], want)
		}
	}
	tomorrow := time.Now().Add(24 * time.Hour)
	filtered, err := s.UsageAggregates(&tomorrow)
	if err != nil || len(filtered) != 0 {
		t.Fatalf("since filter: %#v %v", filtered, err)
	}
}
