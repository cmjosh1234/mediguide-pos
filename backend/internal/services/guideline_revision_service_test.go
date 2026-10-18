package services

import (
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"mediguide/internal/models"
)

func TestWithdrawnRevisionSourcesAndRetry(t *testing.T) {
	for _, mode := range []string{"published_markdown", "reviewed_content", "replacement_upload", "link"} {
		t.Run(mode, func(t *testing.T) {
			s, source, actor := markdownServiceFixture(t)
			ctx := context.Background()
			if mode == "published_markdown" {
				draft, err := s.SaveMarkdownDraft(ctx, source.ID, actor, MarkdownDraftInput{Content: "# Exact published content", SourceType: "blank"})
				if err != nil {
					t.Fatal(err)
				}
				if err := s.DB.Model(&source).Update("published_markdown_revision_id", draft.Revision.ID).Error; err != nil {
					t.Fatal(err)
				}
			} else if mode == "reviewed_content" {
				section := models.GuidelineSection{VersionID: source.ID, Title: "Clinical care", Slug: "clinical-care", Level: 1}
				if err := s.DB.Create(&section).Error; err != nil {
					t.Fatal(err)
				}
				blocks := []models.GuidelineContentBlock{
					{VersionID: source.ID, SectionID: &section.ID, Type: models.GuidelineBlockParagraph, ContentJSON: []byte(`{"type":"paragraph","text":"Reviewed text"}`), SourceFingerprint: "reviewed", ReviewStatus: models.GuidelineBlockReviewed},
					{VersionID: source.ID, SectionID: &section.ID, Type: models.GuidelineBlockParagraph, ContentJSON: []byte(`{"type":"paragraph","text":"SECRET UNREVIEWED TEXT"}`), SourceFingerprint: "draft", ReviewStatus: models.GuidelineBlockDraft},
					{VersionID: source.ID, SectionID: &section.ID, Type: models.GuidelineBlockAlgorithm, ContentJSON: []byte(`{"type":"algorithm","nodes":[]}`), SourceFingerprint: "algorithm", ReviewStatus: models.GuidelineBlockReviewed},
				}
				if err := s.DB.Create(&blocks).Error; err != nil {
					t.Fatal(err)
				}
				if err := s.DB.Create(&models.GuidelineSection{VersionID: source.ID, Title: "References", Slug: "references", Level: 1, SortOrder: 1}).Error; err != nil {
					t.Fatal(err)
				}
			} else {
				kind := models.DocumentKind{Name: mode, Slug: mode, Status: "active", PublishAsLink: mode == "link", PublishAsUploaded: mode == "replacement_upload"}
				if err := s.DB.Create(&kind).Error; err != nil {
					t.Fatal(err)
				}
				if err := s.DB.Model(&models.GuidelineDocument{}).Where("id = ?", source.DocumentID).Update("document_kind_id", kind.ID).Error; err != nil {
					t.Fatal(err)
				}
				if err := s.DB.Model(&source).Updates(map[string]any{"external_url": "https://example.org/old", "original_file_key": "retained.pdf"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			approvedAt := time.Now().UTC().Format(time.RFC3339)
			if err := s.DB.Model(&source).Updates(map[string]any{"status": "published", "approved_by": actor, "approved_at": approvedAt}).Error; err != nil {
				t.Fatal(err)
			}
			if err := s.DB.Model(&models.GuidelineDocument{}).Where("id = ?", source.DocumentID).Update("current_version_id", source.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := s.WithdrawDocument(source.DocumentID, actor, WithdrawGuidelineInput{Reason: "Clinical revision", CurrentVersionID: source.ID}, ""); err != nil {
				t.Fatal(err)
			}
			input := ReviseGuidelineInput{SourceVersionID: source.ID, Version: "2"}
			result, err := s.ReviseWithdrawnDocument(ctx, source.DocumentID, actor, input, "")
			if err != nil {
				t.Fatal(err)
			}
			v := result.Version
			if result.Existing || v.Status != "draft" || v.SourceVersionID == nil || *v.SourceVersionID != source.ID || v.DocumentID != source.DocumentID || v.ApprovedBy != nil || v.ApprovedAt != nil || v.PublishedMarkdownRevisionID != nil || v.RevisionSource != mode {
				t.Fatalf("invalid draft: %#v", v)
			}
			if mode == "replacement_upload" && (v.OriginalFileKey != "" || v.MarkdownFileKey != "") {
				t.Fatal("file draft reused withdrawn source")
			}
			if mode == "link" && v.ExternalURL != "https://example.org/old" {
				t.Fatal("link not copied")
			}
			if mode == "published_markdown" || mode == "reviewed_content" {
				draft, err := s.GetMarkdownDraft(ctx, v.ID)
				if err != nil {
					t.Fatal(err)
				}
				if draft.Revision.ReviewState != "draft" || draft.Revision.PublicationState != "draft" {
					t.Fatal("approval copied")
				}
				if mode == "published_markdown" && draft.Content != "# Exact published content" {
					t.Fatal("published Markdown changed")
				}
				if mode == "reviewed_content" && (!strings.Contains(draft.Content, "Reviewed text") || strings.Contains(draft.Content, "SECRET") || !strings.Contains(string(v.RevisionWarnings), "Omitted algorithm")) {
					t.Fatal("unsafe or incomplete structured fallback")
				}
				if mode == "reviewed_content" {
					titles := 0
					for _, line := range strings.Split(draft.Content, "\n") {
						if strings.HasPrefix(line, "# ") {
							titles++
						}
					}
					if titles != 1 || !strings.Contains(draft.Content, "## References") || !strings.Contains(string(v.RevisionWarnings), "References") {
						t.Fatal("fallback did not normalize headings and disclose empty sections")
					}
				}
			}
			retry, err := s.ReviseWithdrawnDocument(ctx, source.DocumentID, actor, input, "")
			if err != nil || !retry.Existing || retry.Version.ID != v.ID {
				t.Fatalf("retry created duplicate: %#v %v", retry, err)
			}
			doc, err := s.GetDocument(source.DocumentID)
			if err != nil {
				t.Fatal(err)
			}
			if doc.CurrentVersionID != nil || doc.WithdrawnAt == nil || doc.WithdrawalReason != "Clinical revision" {
				t.Fatal("draft creation changed public visibility")
			}
			if _, err := (PublicGuidelineService{DB: s.DB}).Get(ctx, doc.ID); !errors.Is(err, ErrPublicGuidelineNotFound) {
				t.Fatalf("draft visible: %v", err)
			}
			if mode == "link" || mode == "replacement_upload" {
				if mode == "link" {
					if _, err := s.SetVersionLink(v.ID, actor, "https://example.org/replacement", ""); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := s.PublishVersion(v.ID, actor); err == nil {
						t.Fatal("file draft published without a replacement upload")
					}
					data := testDocx(t, "Replacement file requiring publisher approval")
					job, err := s.UploadAsUploadedSource(ctx, v.ID, asUploadedTestFile(data), &multipart.FileHeader{Filename: "replacement.docx", Size: int64(len(data))})
					if err != nil {
						t.Fatal(err)
					}
					if err := s.PublishVersion(v.ID, actor); err == nil {
						t.Fatal("file published before indexing completed")
					}
					if err := s.DB.Model(&models.IngestionJob{}).Where("id = ?", job.ID).Update("status", "completed").Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := s.PublishVersion(v.ID, actor); err != nil {
					t.Fatal(err)
				}
				visible, err := (PublicGuidelineService{DB: s.DB}).Get(ctx, doc.ID)
				if err != nil || visible.Version != "2" {
					t.Fatalf("replacement not visible: %#v %v", visible, err)
				}
				if mode == "link" && visible.ExternalURL != "https://example.org/replacement" {
					t.Fatal("replacement URL missing")
				}
			}
			if err := s.DB.Model(&models.GuidelineDocument{}).Where("id = ?", doc.ID).Update("current_version_id", uuid.New()).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := s.ReviseWithdrawnDocument(ctx, doc.ID, actor, input, ""); !errors.Is(err, ErrGuidelinePublicationConflict) {
				t.Fatalf("stale withdrawal accepted: %v", err)
			}
		})
	}
}

func TestWithdrawnRevisionRejectsInvalidRequestsAndRollsBackUnavailableSource(t *testing.T) {
	s, source, actor := markdownServiceFixture(t)
	ctx := context.Background()
	draft, err := s.SaveMarkdownDraft(ctx, source.ID, actor, MarkdownDraftInput{Content: "# Source", SourceType: "blank"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&source).Updates(map[string]any{"status": "published", "published_markdown_revision_id": draft.Revision.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&models.GuidelineDocument{}).Where("id = ?", source.DocumentID).Update("current_version_id", source.ID).Error; err != nil {
		t.Fatal(err)
	}
	input := ReviseGuidelineInput{SourceVersionID: source.ID, Version: "2"}
	if _, err := s.ReviseWithdrawnDocument(ctx, source.DocumentID, actor, input, ""); !errors.Is(err, ErrGuidelinePublicationConflict) {
		t.Fatalf("non-withdrawn document accepted: %v", err)
	}
	if err := s.WithdrawDocument(source.DocumentID, actor, WithdrawGuidelineInput{Reason: "Correct content", CurrentVersionID: source.ID}, ""); err != nil {
		t.Fatal(err)
	}
	for _, number := range []string{"", "   ", strings.Repeat("x", 101)} {
		invalid := input
		invalid.Version = number
		if _, err := s.ReviseWithdrawnDocument(ctx, source.DocumentID, actor, invalid, ""); !errors.Is(err, ErrGuidelineRevisionInput) {
			t.Fatalf("invalid version accepted: %v", err)
		}
	}
	duplicate := input
	duplicate.Version = "1"
	if _, err := s.ReviseWithdrawnDocument(ctx, source.DocumentID, actor, duplicate, ""); !errors.Is(err, ErrGuidelineVersionExists) {
		t.Fatalf("duplicate number accepted: %v", err)
	}
	s.Store.(*fakePublicStore).err = errors.New("source unavailable")
	if _, err := s.ReviseWithdrawnDocument(ctx, source.DocumentID, actor, input, ""); err == nil {
		t.Fatal("missing source accepted")
	}
	var count int64
	if err := s.DB.Model(&models.GuidelineVersion{}).Where("document_id = ?", source.DocumentID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed creation left %d versions", count)
	}
	document, err := s.GetDocument(source.DocumentID)
	if err != nil {
		t.Fatal(err)
	}
	if document.CurrentVersionID != nil || document.WithdrawnAt == nil {
		t.Fatal("failure changed withdrawal")
	}
}

func TestRevisionFallbackPreservesClinicalCalloutTypeAndMetadata(t *testing.T) {
	block := models.GuidelineContentBlock{Type: models.GuidelineBlockWarning, ContentJSON: []byte(`{"type":"warning","title":"Escalation","content":"Seek urgent review.","severity":"critical","evidence_grade":"A","source":"Reviewed source"}`)}
	markdown, err := revisionBlockMarkdown(block)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{":::warning", `title="Escalation"`, `severity="critical"`, `evidence_grade="A"`, `source="Reviewed source"`, "Seek urgent review.\n:::"} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("clinical metadata lost: %s", markdown)
		}
	}
}
