package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mediguide/internal/models"
	"mediguide/internal/services"
)

type readerSeedStore struct{ objects map[string][]byte }

func (s *readerSeedStore) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(r)
	s.objects[key] = data
	return err
}
func (s *readerSeedStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.objects[key])), nil
}
func (s *readerSeedStore) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}
func (s *readerSeedStore) PresignGet(_ context.Context, key string, _ time.Duration) (*url.URL, error) {
	return url.Parse("https://example.org/" + key)
}

func TestReaderFixturesHaveValidBlocksAndThreePageOriginals(t *testing.T) {
	fixtures := demoReaderDocuments()
	seen := map[string]bool{}
	for _, fixture := range fixtures {
		seen[fixture.DocumentKindSlug] = true
		for _, section := range fixture.Sections {
			for _, block := range section.Blocks {
				payload, err := json.Marshal(block.Payload)
				if err != nil {
					t.Fatal(err)
				}
				if err := services.ValidateGuidelineBlockContent(models.GuidelineBlockType(block.Type), payload); err != nil {
					t.Fatal(err)
				}
			}
		}
		data := readerFixturePDF(fixture)
		document, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatal(err)
		}
		if document.NumPage() != 3 {
			t.Fatalf("%s: expected three original pages", fixture.Key)
		}
		text, err := document.GetPlainText()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(text)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(contents), "not clinical guidance") {
			t.Fatalf("%s: missing sample label", fixture.Key)
		}
	}
	if len(seen) != 19 || len(fixtures) != 20 {
		t.Fatalf("expected 19 kinds and one partial sample, got %d kinds, %d fixtures", len(seen), len(fixtures))
	}
}

func TestReaderSeedIsIdempotentAndRespectsPublicationModes(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.DocumentKind{}, &models.GuidelineDocument{}, &models.GuidelineVersion{}, &models.GuidelineVersionManifest{}, &models.GuidelineSection{}, &models.GuidelineContentBlock{}, &models.GuidelineChunk{}, &models.GuidelineAsset{}); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range demoReaderDocuments() {
		kind := models.DocumentKind{Slug: fixture.DocumentKindSlug, Name: fixture.DocumentKindSlug, Status: "active", PublishAsUploaded: fixture.DocumentKindSlug == "form" || fixture.DocumentKindSlug == "situation_report_attachment", PublishAsLink: fixture.DocumentKindSlug == "link"}
		kind.ID = demoID("reader-kind", fixture.DocumentKindSlug)
		if err := database.Where("slug = ?", kind.Slug).FirstOrCreate(&kind).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := &readerSeedStore{objects: map[string][]byte{}}
	for range 2 {
		if err := seedDemoReaderDocuments(context.Background(), database, store, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	database.Model(&models.GuidelineDocument{}).Count(&count)
	if count != 20 {
		t.Fatalf("seed duplicated documents: %d", count)
	}
	for _, slug := range []string{"form", "situation_report_attachment", "link"} {
		versionID := demoID("guideline-version", "reader-test-"+strings.ReplaceAll(slug, "_", "-")+"-1.0")
		database.Model(&models.GuidelineContentBlock{}).Where("version_id = ?", versionID).Count(&count)
		if count != 0 {
			t.Fatalf("%s must have no structured blocks", slug)
		}
		var version models.GuidelineVersion
		if err := database.First(&version, "id = ?", versionID).Error; err != nil {
			t.Fatal(err)
		}
		if slug == "link" && (version.ExternalURL != "https://example.org" || version.OriginalFileKey != "") {
			t.Fatal("link must publish only its URL")
		}
		if slug != "link" && version.OriginalFileKey == "" {
			t.Fatal("form must have an original PDF")
		}
	}
	versionID := demoID("guideline-version", "reader-test-partial-sop-1.0")
	var manifest models.GuidelineVersionManifest
	if err := database.First(&manifest, "version_id = ?", versionID).Error; err != nil {
		t.Fatal(err)
	}
	if manifest.ExtractionQuality != models.GuidelineExtractionPartiallyReviewed || !manifest.HasOriginalPDF || manifest.HasOfflinePackage || manifest.ReviewedSectionCount != 2 {
		t.Fatalf("invalid partial reader fixture: %+v", manifest)
	}
	database.Model(&models.GuidelineAsset{}).Where("version_id = ? AND type = ?", versionID, "offline_package").Count(&count)
	if count != 0 {
		t.Fatal("partial sample must not expose unreviewed text in an offline ZIP")
	}
	// Reseeding must not silently publish a document an administrator withdrew.
	for _, key := range []string{"reader-test-sop", "reader-test-form", "reader-test-link"} {
		if err := database.Model(&models.GuidelineDocument{}).Where("id = ?", demoID("guideline", key)).Updates(map[string]any{
			"current_version_id": nil, "withdrawn_at": time.Now().UTC(), "withdrawal_reason": "Under review",
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := seedDemoReaderDocuments(context.Background(), database, store, uuid.New()); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"reader-test-sop", "reader-test-form", "reader-test-link"} {
		var document models.GuidelineDocument
		if err := database.First(&document, "id = ?", demoID("guideline", key)).Error; err != nil {
			t.Fatal(err)
		}
		if document.CurrentVersionID != nil || document.WithdrawnAt == nil || document.WithdrawalReason != "Under review" {
			t.Fatalf("reseed restored withdrawn %s", key)
		}
	}
}
