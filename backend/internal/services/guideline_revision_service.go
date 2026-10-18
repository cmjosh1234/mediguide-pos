package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mediguide/internal/models"
)

var ErrGuidelineRevisionInput = errors.New("a new version number of 1 to 100 characters is required")

type ReviseGuidelineInput struct {
	SourceVersionID uuid.UUID `json:"source_version_id" binding:"required"`
	Version         string    `json:"version" binding:"required"`
}

type GuidelineRevisionResult struct {
	Version  models.GuidelineVersion `json:"version"`
	Existing bool                    `json:"existing"`
}

// ReviseWithdrawnDocument branches the retained publication into an unapproved
// draft. The document lock serializes retries and concurrent publication changes.
func (s GuidelineService) ReviseWithdrawnDocument(ctx context.Context, id, actor uuid.UUID, in ReviseGuidelineInput, ip string) (*GuidelineRevisionResult, error) {
	in.Version = strings.TrimSpace(in.Version)
	if in.Version == "" || len([]rune(in.Version)) > 100 {
		return nil, ErrGuidelineRevisionInput
	}
	var result GuidelineRevisionResult
	objectKey := ""
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var doc models.GuidelineDocument
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&doc, "id = ?", id).Error; err != nil {
			return err
		}
		if doc.CurrentVersionID != nil || doc.WithdrawnAt == nil || doc.WithdrawnVersionID == nil || *doc.WithdrawnVersionID != in.SourceVersionID {
			return ErrGuidelinePublicationConflict
		}
		var source models.GuidelineVersion
		if err := tx.First(&source, "id = ? AND document_id = ? AND status = ?", in.SourceVersionID, id, "published").Error; err != nil {
			return err
		}
		if err := tx.Where("document_id = ? AND source_version_id = ? AND status <> ?", id, source.ID, "published").Order("created_at DESC").First(&result.Version).Error; err == nil {
			result.Existing = true
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var count int64
		if err := tx.Model(&models.GuidelineVersion{}).Where("document_id = ? AND lower(version) = lower(?)", id, in.Version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrGuidelineVersionExists
		}
		version := models.GuidelineVersion{DocumentID: id, SourceVersionID: &source.ID, Version: in.Version, Status: "draft", RevisionWarnings: []byte(`[]`), StructuredContentStatus: "not_generated"}
		content := ""
		var parent *uuid.UUID
		anchors := []byte(`{}`)
		warnings := []string{}
		asLink, err := versionPublishesAsLink(tx, source.ID)
		if err != nil {
			return err
		}
		asFile, err := versionPublishesAsUploaded(tx, source.ID)
		if err != nil {
			return err
		}
		switch {
		case asLink:
			version.RevisionSource = "link"
			version.ExternalURL = source.ExternalURL
		case asFile:
			version.RevisionSource = "replacement_upload"
			warnings = append(warnings, "Upload a replacement file. The withdrawn file has not been copied or approved for this draft.")
		default:
			version.OriginalFileKey = source.OriginalFileKey
			if source.PublishedMarkdownRevisionID != nil {
				var revision models.GuidelineMarkdownRevision
				if err := tx.First(&revision, "id = ? AND version_id = ?", *source.PublishedMarkdownRevisionID, source.ID).Error; err != nil {
					return err
				}
				content, err = s.readMarkdownObject(ctx, revision.StorageKey)
				if err != nil {
					return err
				}
				parent = &revision.ID
				anchors = revision.AnchorMetadataJSON
				version.RevisionSource = "published_markdown"
			} else {
				content, warnings, err = reviewedRevisionMarkdown(tx, &source)
				if err != nil {
					return err
				}
				version.RevisionSource = "reviewed_content"
			}
		}
		version.RevisionWarnings, err = json.Marshal(warnings)
		if err != nil {
			return err
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if !asLink && !asFile {
			revision := models.GuidelineMarkdownRevision{DocumentID: id, VersionID: version.ID, RevisionNumber: 1, SourceType: "duplicated", ParentRevisionID: parent, CreatedBy: &actor, IsCurrent: true, StructuredContentStatus: "outdated", ReviewState: "draft", PublicationState: "draft", AnchorMetadataJSON: anchors, ChangeSummary: fmt.Sprintf("Revision of withdrawn version %s; all content requires review", source.Version)}
			revision.ID = uuid.New()
			data := []byte(content)
			objectKey = fmt.Sprintf("guidelines/%s/revisions/%s.md", version.ID, revision.ID)
			if err := s.Store.Put(ctx, objectKey, bytes.NewReader(data), int64(len(data)), "text/markdown; charset=utf-8"); err != nil {
				return err
			}
			revision.StorageKey = objectKey
			revision.SizeBytes = int64(len(data))
			revision.Checksum = markdownContentChecksum(data)
			if err := tx.Create(&revision).Error; err != nil {
				return err
			}
			if err := tx.Model(&version).Updates(map[string]any{"current_markdown_revision_id": revision.ID, "markdown_file_key": objectKey, "checksum": revision.Checksum, "structured_content_status": "outdated"}).Error; err != nil {
				return err
			}
		}
		if err := tx.First(&result.Version, "id = ?", version.ID).Error; err != nil {
			return err
		}
		return writeGuidelineAudit(tx, actor, "guideline.document.revision_started", "guideline_document", id, ip, map[string]any{"version_id": version.ID, "source_version_id": source.ID, "revision_source": version.RevisionSource, "warnings": warnings})
	})
	if err != nil {
		if objectKey != "" {
			_ = s.Store.Delete(ctx, objectKey)
		}
		return nil, err
	}
	return &result, nil
}

// Only reviewed blocks are exported. Unsupported blocks and empty/unreviewed
// sections are explicitly reported, never silently converted or approved.
func reviewedRevisionMarkdown(tx *gorm.DB, source *models.GuidelineVersion) (string, []string, error) {
	var document models.GuidelineDocument
	if err := tx.Select("title").First(&document, "id = ?", source.DocumentID).Error; err != nil {
		return "", nil, err
	}
	var sections []models.GuidelineSection
	if err := tx.Where("version_id = ?", source.ID).Order("sort_order, id").Find(&sections).Error; err != nil {
		return "", nil, err
	}
	var blocks []models.GuidelineContentBlock
	if err := tx.Where("version_id = ?", source.ID).Order("sort_order, id").Find(&blocks).Error; err != nil {
		return "", nil, err
	}
	warnings := []string{"Draft created from reviewed structured content only. Verify it against the retained original; previous approvals do not carry over."}
	var out strings.Builder
	out.WriteString("# " + document.Title + "\n\n")
	rootCount := 0
	var root *models.GuidelineSection
	sectionByID := map[uuid.UUID]models.GuidelineSection{}
	for i := range sections {
		sectionByID[sections[i].ID] = sections[i]
		if sections[i].ParentID == nil {
			rootCount++
			root = &sections[i]
		}
	}
	shift := 1
	if rootCount == 1 && root.Level == 1 && strings.EqualFold(strings.TrimSpace(root.Title), strings.TrimSpace(document.Title)) {
		shift = 0
	}
	seen := map[uuid.UUID]bool{}
	writeBlock := func(block models.GuidelineContentBlock) bool {
		if block.ReviewStatus != models.GuidelineBlockReviewed {
			warnings = append(warnings, fmt.Sprintf("Omitted %s block %s: not reviewed.", block.Type, block.ID))
			return false
		}
		rendered, err := revisionBlockMarkdown(block)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Omitted %s block %s: restore it from the original during editing.", block.Type, block.ID))
			return false
		}
		out.WriteString(rendered)
		out.WriteString("\n\n")
		return true
	}
	for _, section := range sections {
		level := 1 + shift
		parentID := section.ParentID
		ancestors := map[uuid.UUID]bool{section.ID: true}
		for parentID != nil {
			parent, exists := sectionByID[*parentID]
			if !exists || ancestors[*parentID] {
				warnings = append(warnings, fmt.Sprintf("Section %q has an invalid source hierarchy; verify its placement.", section.Title))
				break
			}
			ancestors[*parentID] = true
			level++
			parentID = parent.ParentID
		}
		if level > 6 {
			warnings = append(warnings, fmt.Sprintf("Section %q exceeded Markdown heading depth; verify its placement.", section.Title))
			level = 6
		}
		if !(shift == 0 && root != nil && section.ID == root.ID) {
			out.WriteString(strings.Repeat("#", level) + " " + section.Title + "\n\n")
		}
		copied := false
		for _, block := range blocks {
			if block.SectionID != nil && *block.SectionID == section.ID {
				seen[block.ID] = true
				if block.Type == models.GuidelineBlockHeading && block.ReviewStatus == models.GuidelineBlockReviewed {
					var heading models.GuidelineHeadingBlockPayload
					if json.Unmarshal(block.ContentJSON, &heading) == nil && heading.Text == section.Title {
						copied = true
						continue
					}
				}
				if writeBlock(block) {
					copied = true
				}
			}
		}
		if !copied {
			warnings = append(warnings, fmt.Sprintf("Section %q has no reviewed content in this draft.", section.Title))
		}
	}
	for _, block := range blocks {
		if !seen[block.ID] {
			writeBlock(block)
		}
	}
	return out.String(), warnings, nil
}

func revisionBlockMarkdown(block models.GuidelineContentBlock) (string, error) {
	var p struct {
		Text          string     `json:"text"`
		Content       string     `json:"content"`
		Title         string     `json:"title"`
		Severity      string     `json:"severity"`
		EvidenceGrade string     `json:"evidence_grade"`
		Source        string     `json:"source"`
		Level         int        `json:"level"`
		Items         []string   `json:"items"`
		Columns       []string   `json:"columns"`
		Rows          [][]string `json:"rows"`
		Footnotes     []string   `json:"footnotes"`
		Citation      string     `json:"citation"`
		URL           string     `json:"url"`
		Page          int        `json:"page"`
	}
	if err := json.Unmarshal(block.ContentJSON, &p); err != nil {
		return "", err
	}
	switch block.Type {
	case models.GuidelineBlockParagraph:
		return p.Text, nil
	case models.GuidelineBlockHeading:
		if p.Level < 1 || p.Level > 6 {
			return "", errors.New("invalid heading")
		}
		if p.Level == 1 {
			p.Level = 2
		}
		return strings.Repeat("#", p.Level) + " " + p.Text, nil
	case models.GuidelineBlockOrderedList, models.GuidelineBlockUnorderedList:
		lines := []string{}
		for i, item := range p.Items {
			prefix := "- "
			if block.Type == models.GuidelineBlockOrderedList {
				prefix = fmt.Sprintf("%d. ", i+1)
			}
			lines = append(lines, prefix+strings.ReplaceAll(item, "\n", "\n  "))
		}
		return strings.Join(lines, "\n"), nil
	case models.GuidelineBlockTable:
		cell := func(value string) string {
			return strings.ReplaceAll(strings.ReplaceAll(value, "|", `\|`), "\n", "<br>")
		}
		row := func(values []string) string {
			cells := []string{}
			for _, value := range values {
				cells = append(cells, cell(value))
			}
			return "| " + strings.Join(cells, " | ") + " |\n"
		}
		var out strings.Builder
		if p.Title != "" {
			out.WriteString("**" + p.Title + "**\n\n")
		}
		out.WriteString(row(p.Columns))
		separators := make([]string, len(p.Columns))
		for i := range separators {
			separators[i] = "---"
		}
		out.WriteString(row(separators))
		for _, values := range p.Rows {
			out.WriteString(row(values))
		}
		for _, note := range p.Footnotes {
			out.WriteString("\n" + note + "\n")
		}
		return out.String(), nil
	case models.GuidelineBlockReference:
		return strings.TrimSpace(p.Citation + " " + p.URL), nil
	case models.GuidelineBlockPageBreak:
		return fmt.Sprintf("<!-- page: %d -->", p.Page), nil
	case models.GuidelineBlockRecommendation, models.GuidelineBlockWarning, models.GuidelineBlockCaution, models.GuidelineBlockKeyPoint, models.GuidelineBlockContraindication, models.GuidelineBlockDosage, models.GuidelineBlockEvidence, models.GuidelineBlockDefinition, models.GuidelineBlockProcedure, models.GuidelineBlockClinicalNote, models.GuidelineBlockReferralCriteria, models.GuidelineBlockAlgorithmReference:
		for _, line := range strings.Split(p.Content, "\n") {
			if strings.TrimSpace(line) == ":::" {
				return "", errors.New("nested callout delimiter")
			}
		}
		// The worker's native fenced syntax retains both the clinical risk type
		// and its metadata. A generic blockquote could silently become prose.
		header := ":::" + strings.ReplaceAll(string(block.Type), "_", "-")
		quote := func(value string) string {
			return "\"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", " ", "\r", " ").Replace(value) + "\""
		}
		for _, field := range []struct{ key, value string }{{"title", p.Title}, {"severity", p.Severity}, {"evidence_grade", p.EvidenceGrade}, {"source", p.Source}} {
			if field.value != "" {
				header += " " + field.key + "=" + quote(field.value)
			}
		}
		return header + "\n" + p.Content + "\n:::", nil
	default:
		return "", errors.New("unsupported block")
	}
}
