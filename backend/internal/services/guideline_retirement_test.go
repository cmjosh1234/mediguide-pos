package services

import (
	"mediguide/internal/models"
	"testing"
)

func TestGuidelineStatsCountCurrentPublishedDocuments(t *testing.T) {
	s := progressUsageTestService(t)
	if err := s.DB.AutoMigrate(&models.GuidelineVersion{}); err != nil {
		t.Fatal(err)
	}
	published := models.GuidelineDocument{Title: "Published"}
	draft := models.GuidelineDocument{Title: "Draft"}
	old := models.GuidelineDocument{Title: "Has historical published version only"}
	for _, d := range []*models.GuidelineDocument{&published, &draft, &old} {
		if err := s.DB.Create(d).Error; err != nil {
			t.Fatal(err)
		}
	}
	current := models.GuidelineVersion{DocumentID: published.ID, Version: "1", Status: "published"}
	historical := models.GuidelineVersion{DocumentID: old.ID, Version: "1", Status: "published"}
	for _, v := range []*models.GuidelineVersion{&current, &historical} {
		if err := s.DB.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DB.Model(&published).Update("current_version_id", current.ID).Error; err != nil {
		t.Fatal(err)
	}
	count, err := (LegacyAPIService{DB: s.DB}).countPublishedGuidelineDocuments()
	if err != nil || count != 1 {
		t.Fatalf("published documents = %d, err=%v", count, err)
	}
	if err := s.DB.Model(&current).Update("deleted_at", "2026-01-01").Error; err != nil {
		t.Fatal(err)
	}
	count, err = (LegacyAPIService{DB: s.DB}).countPublishedGuidelineDocuments()
	if err != nil || count != 0 {
		t.Fatalf("deleted version counted: %d, err=%v", count, err)
	}
	if s.DB.Migrator().HasTable("medical_guidelines") {
		t.Fatal("retired table recreated")
	}
}
