package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"mediguide/internal/models"
	"mediguide/internal/storage"
)

// These deterministic fixtures own only reader-test-* records. They are neutral
// UI examples, not clinical guidance, and can be refreshed without reseeding users.
func demoReaderDocuments() []demoGuideline {
	kinds := []struct{ slug, name string }{
		{"guideline", "Guideline"}, {"manual", "Manual"}, {"framework", "Framework"},
		{"sop", "SOP"}, {"case_definition", "Case Definition"}, {"ipc_protocol", "IPC Protocol"},
		{"laboratory_protocol", "Laboratory Protocol"}, {"surveillance_protocol", "Surveillance Protocol"},
		{"contact_tracing_guide", "Contact Tracing Guide"}, {"treatment_protocol", "Treatment Protocol"},
		{"referral_protocol", "Referral Protocol"}, {"training_material", "Training Material"},
		{"checklist", "Checklist"}, {"communication_material", "Communication Material"},
		{"form", "Form"}, {"policy", "Policy"}, {"situation_report_attachment", "Situation Report Attachment"},
		{"other", "Other"}, {"link", "Link"},
	}
	var fixtures []demoGuideline
	for _, kind := range kinds {
		titles := []string{"Purpose and scope", "Sample workflow", "Review and references"}
		if kind.slug == "case_definition" {
			titles = []string{"Sample category A", "Sample category B", "Sample category C"}
		}
		if kind.slug == "training_material" {
			titles = []string{"Learning objectives", "Practice activity", "Review"}
		}
		if kind.slug == "manual" {
			titles = []string{"Chapter 1: Introduction", "Chapter 2: Using the reader", "Chapter 3: Reference material"}
		}
		if kind.slug == "framework" {
			titles = []string{"Principles", "Implementation areas", "Monitoring and review"}
		}
		fixture := demoGuideline{
			Key: "reader-test-" + strings.ReplaceAll(kind.slug, "_", "-"), Title: "Local test - " + kind.name,
			DocumentKindSlug: kind.slug, SourceOrganization: "Local reader fixtures",
			Description: "Local development sample for document-reader testing. Not clinical guidance.",
			ProgramArea: "Reader testing", Population: "Development testing only", Healthcare: "Development testing only",
			Version: "1.0", Publication: "2026-10-13", Review: "2027-10-13",
			Sections: []demoSection{
				{Title: titles[0], Slug: "overview", Level: 1, Blocks: []demoBlock{
					{Type: "paragraph", Payload: map[string]any{"type": "paragraph", "text": "This sample checks reader layout and navigation. It must not be used for clinical decisions."}},
				}},
				{Title: titles[1], Slug: "workflow", Level: 1, Blocks: []demoBlock{
					{Type: "ordered_list", Payload: map[string]any{"type": "ordered_list", "items": []string{"Open a section in the reader.", "Read the sample content.", "Continue to the next section."}}},
					{Type: "table", Payload: map[string]any{"type": "table", "title": "Sample review matrix", "columns": []string{"Item", "Purpose"}, "rows": [][]string{{"Navigation", "Find a section"}, {"Source page", "Open the original PDF"}}}},
				}},
				{Title: titles[2], Slug: "references", Level: 1, Blocks: []demoBlock{
					{Type: "paragraph", Payload: map[string]any{"type": "paragraph", "text": "Check text scaling, dark mode and source-page navigation with this test publication."}},
				}},
			},
		}
		fixtures = append(fixtures, fixture)
	}
	partial := fixtures[3]
	partial.Key = "reader-test-partial-sop"
	partial.Title = "Local test - Partially reviewed SOP"
	fixtures = append(fixtures, partial)
	return fixtures
}

func seedDemoReaderDocuments(ctx context.Context, database *gorm.DB, store storage.ObjectStore, reviewerID uuid.UUID) error {
	for _, fixture := range demoReaderDocuments() {
		var kind models.DocumentKind
		if err := database.Where("slug = ?", fixture.DocumentKindSlug).First(&kind).Error; err != nil {
			return err
		}
		documentID := demoID("guideline", fixture.Key)
		versionID := demoID("guideline-version", fixture.Key+"-"+fixture.Version)
		if !kind.PublishAsUploaded && !kind.PublishAsLink {
			if err := seedDemoGuideline(ctx, database, store, reviewerID, fixture); err != nil {
				return err
			}
		} else {
			if err := upsertByID(database, "guideline_documents", map[string]any{
				"id": documentID, "document_kind_id": kind.ID, "title": fixture.Title, "description": fixture.Description,
				"country": "Uganda", "source_org": fixture.SourceOrganization, "program_area": fixture.ProgramArea, "language": "en",
			}); err != nil {
				return err
			}
			externalURL := ""
			if kind.PublishAsLink {
				externalURL = "https://example.org"
			}
			sourceBytes := readerFixturePDF(fixture)
			if kind.PublishAsLink {
				sourceBytes = []byte(externalURL)
			}
			sourceSum := sha256.Sum256(sourceBytes)
			sourceChecksum := hex.EncodeToString(sourceSum[:])
			if err := upsertByID(database, "guideline_versions", map[string]any{
				"id": versionID, "document_id": documentID, "version": fixture.Version, "publication_date": fixture.Publication,
				"review_date": fixture.Review, "status": "published", "external_url": externalURL, "markdown_file_key": "",
				"approved_by": reviewerID, "approved_at": fixture.Publication + "T09:00:00Z", "extraction_schema_version": 1,
				"structured_content_status": "approved", "checksum": sourceChecksum,
			}); err != nil {
				return err
			}
			if err := database.Table("guideline_documents").Where("id = ?", documentID).Update("current_version_id", versionID).Error; err != nil {
				return err
			}
			if err := upsertByID(database, "guideline_version_manifests", map[string]any{
				"id": demoID("guideline-manifest", fixture.Key), "guideline_id": documentID, "version_id": versionID,
				"version": fixture.Version, "schema_version": 1, "package_version": 1, "extraction_quality": "unreviewed",
				"has_chapters": false, "has_key_points": false, "has_offline_package": false, "generated_at": time.Now().UTC(),
				"checksum": sourceChecksum, "etag": "sha256-" + sourceChecksum,
			}); err != nil {
				return err
			}
		}
		if !kind.PublishAsLink {
			if err := seedReaderOriginal(ctx, database, store, reviewerID, fixture); err != nil {
				return err
			}
		}
		if fixture.Key == "reader-test-partial-sop" {
			sectionID := demoID("guideline-section", fixture.Key+"-references")
			if err := database.Model(&models.GuidelineContentBlock{}).Where("version_id = ? AND section_id = ?", versionID, sectionID).
				Updates(map[string]any{"review_status": "draft", "reviewed_by": nil, "reviewed_at": nil}).Error; err != nil {
				return err
			}
			if err := database.Model(&models.GuidelineChunk{}).Where("version_id = ? AND section_id = ?", versionID, sectionID).
				Update("review_status", "draft").Error; err != nil {
				return err
			}
			if err := database.Table("guideline_version_manifests").Where("version_id = ?", versionID).
				Updates(map[string]any{"extraction_quality": "partially_reviewed", "block_count": 3}).Error; err != nil {
				return err
			}
			if err := database.Table("guideline_versions").Where("id = ?", versionID).Update("markdown_file_key", "").Error; err != nil {
				return err
			}
			// The generic offline ZIP includes all fixture text; expose only the source
			// PDF for this partial fixture, leaving unreviewed prose out of the package.
			if err := database.Table("guideline_version_manifests").Where("version_id = ?", versionID).Update("has_offline_package", false).Error; err != nil {
				return err
			}
			if err := database.Unscoped().Where("id = ?", demoID("guideline-asset", fixture.Key+"-offline")).Delete(&models.GuidelineAsset{}).Error; err != nil {
				return err
			}
		}
		if !kind.PublishAsUploaded && !kind.PublishAsLink {
			count := 3
			if fixture.Key == "reader-test-partial-sop" {
				count = 2
			}
			if err := database.Table("guideline_version_manifests").Where("version_id = ?", versionID).
				Updates(map[string]any{"reviewed_section_count": count, "leaf_section_count": 3, "reviewed_leaf_section_count": count, "empty_leaf_section_count": 3 - count, "reviewed_paragraph_count": count - 1, "has_key_points": false}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedReaderOriginal(ctx context.Context, database *gorm.DB, store storage.ObjectStore, reviewerID uuid.UUID, fixture demoGuideline) error {
	pdf := readerFixturePDF(fixture)
	sum := sha256.Sum256(pdf)
	checksum := hex.EncodeToString(sum[:])
	key := fmt.Sprintf("demo/guidelines/%s/original.pdf", fixture.Key)
	versionID := demoID("guideline-version", fixture.Key+"-"+fixture.Version)
	if err := store.Put(ctx, key, bytes.NewReader(pdf), int64(len(pdf)), "application/pdf"); err != nil {
		return err
	}
	if err := database.Table("guideline_versions").Where("id = ?", versionID).Update("original_file_key", key).Error; err != nil {
		return err
	}
	if err := database.Table("guideline_version_manifests").Where("version_id = ?", versionID).Update("has_original_pdf", true).Error; err != nil {
		return err
	}
	return upsertByID(database, "guideline_assets", map[string]any{
		"id": demoID("guideline-asset", fixture.Key+"-original"), "version_id": versionID, "type": "original_pdf",
		"mime_type": "application/pdf", "checksum": checksum, "storage_key": key, "size_bytes": len(pdf),
		"original_filename": fixture.Key + ".pdf", "source": "development_seed", "uploaded_by": reviewerID,
		"source_fingerprint": "reader-original:" + fixture.Key,
		"provenance_json":    mustJSON(`{"source":"development_seed","fixture":"reader_documents"}`),
		"review_status":      "reviewed", "reviewed_by": reviewerID, "reviewed_at": time.Now().UTC(),
	})
}

// A small, dependency-free three-page PDF fixture with a real cross-reference
// table. Pages match the seeded sections so source links can be exercised.
func readerFixturePDF(fixture demoGuideline) []byte {
	escape := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [4 0 R 6 0 R 8 0 R] /Count 3 >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	for i, section := range fixture.Sections {
		var stream strings.Builder
		fmt.Fprintf(&stream, "BT /F1 16 Tf 48 760 Td (%s) Tj /F1 12 Tf 0 -30 Td (%s) Tj 0 -26 Td (Local development sample - not clinical guidance.) Tj", escape.Replace(fixture.Title), escape.Replace(section.Title))
		for _, block := range section.Blocks {
			text := demoBlockText(block.Payload)
			// Fixed-size lines avoid clipping while preserving all sample text.
			words := strings.Fields(text)
			line := ""
			for _, word := range words {
				if len(line)+len(word)+1 > 78 {
					fmt.Fprintf(&stream, " 0 -22 Td (%s) Tj", escape.Replace(line))
					line = ""
				}
				if line != "" {
					line += " "
				}
				line += word
			}
			if line != "" {
				fmt.Fprintf(&stream, " 0 -22 Td (%s) Tj", escape.Replace(line))
			}
		}
		stream.WriteString(" ET")
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", 5+i*2), fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", stream.Len(), stream.String()))
	}
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, output.Len())
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&output, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return output.Bytes()
}
