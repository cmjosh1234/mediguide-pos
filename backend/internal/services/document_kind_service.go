package services

import (
	"errors"
	"regexp"
	"strings"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrDocumentKindInUse         = errors.New("document kind is assigned to documents")
	ErrDocumentKindRequired      = errors.New("at least one active document kind is required")
	ErrDocumentKindSlugImmutable = errors.New("document kind slug cannot be changed")
	ErrDocumentKindModeInUse     = errors.New("publish as uploaded cannot change while guideline documents use this kind")
)

// documentKindSlugPattern accepts the existing outbreak codes (contact_tracing_guide)
// as well as hyphenated slugs.
var documentKindSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:[_-][a-z0-9]+)*$`)

type DocumentKindInput struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	Status      *string `json:"status"`
	// PublishAsUploaded keeps uploaded files as the published document instead
	// of extracting them into editable content.
	PublishAsUploaded *bool `json:"publish_as_uploaded"`
}

func validDocumentKindSlug(value string) bool { return documentKindSlugPattern.MatchString(value) }

func documentKindSlugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "_")
	return strings.Trim(value, "_")
}

// documentKindQuery selects live kinds with the number of live guideline and
// outbreak documents using each, so editors can see what a delete would affect.
func (s GuidelineContentService) documentKindQuery() *gorm.DB {
	return s.DB.Table("document_kinds dk").
		Select(`dk.*,
			(SELECT COUNT(*) FROM guideline_documents gd WHERE gd.document_kind_id = dk.id AND gd.deleted_at IS NULL) AS guideline_document_count,
			(SELECT COUNT(*) FROM outbreak_resources r WHERE r.document_kind = dk.slug AND r.deleted_at IS NULL) AS outbreak_document_count`).
		Where("dk.deleted_at IS NULL")
}

func (s GuidelineContentService) ListDocumentKinds(editor bool, in GuidelineContentQuery) (*PageResult[models.DocumentKind], error) {
	p := in.Page.Normalize(50, 100)
	q := s.documentKindQuery()
	if !editor {
		q = q.Where("dk.status = ?", "active")
	} else if in.Status != "" {
		if !oneOf(in.Status, "active", "inactive") {
			return nil, ErrGuidelineContentInvalid
		}
		q = q.Where("dk.status = ?", in.Status)
	}
	if search := strings.TrimSpace(in.Search); search != "" {
		like := "%" + search + "%"
		q = q.Where("LOWER(dk.name) LIKE LOWER(?) OR LOWER(dk.slug) LIKE LOWER(?) OR LOWER(COALESCE(dk.description,'')) LIKE LOWER(?)", like, like, like)
	}
	return pageHelp[models.DocumentKind](q, p, map[string]string{"name": "dk.name", "sort_order": "dk.sort_order", "created_at": "dk.created_at", "updated_at": "dk.updated_at"}, in.Sort, in.Order, "dk.sort_order ASC, dk.name ASC")
}

func (s GuidelineContentService) GetDocumentKind(id uuid.UUID, editor bool) (*models.DocumentKind, error) {
	q := s.documentKindQuery().Where("dk.id = ?", id)
	if !editor {
		q = q.Where("dk.status = ?", "active")
	}
	var item models.DocumentKind
	err := q.First(&item).Error
	return &item, err
}

func (s GuidelineContentService) SaveDocumentKind(id *uuid.UUID, in DocumentKindInput) (*models.DocumentKind, error) {
	item := models.DocumentKind{Status: "active"}
	if id != nil {
		existing, err := s.GetDocumentKind(*id, true)
		if err != nil {
			return nil, err
		}
		item = *existing
	}
	wasActive := id != nil && item.Status == "active"
	if in.Name != nil {
		item.Name = strings.TrimSpace(*in.Name)
	}
	if in.Slug != nil {
		slug := strings.TrimSpace(*in.Slug)
		// Documents and clients refer to kinds by slug, so it is fixed once created.
		if id != nil && slug != "" && slug != item.Slug {
			return nil, ErrDocumentKindSlugImmutable
		}
		if id == nil {
			item.Slug = slug
		}
	}
	if item.Slug == "" {
		item.Slug = documentKindSlugify(item.Name)
	}
	item.Description = mergeOptionalString(item.Description, in.Description)
	if in.SortOrder != nil {
		item.SortOrder = *in.SortOrder
	}
	if in.Status != nil {
		item.Status = strings.TrimSpace(*in.Status)
	}
	if in.PublishAsUploaded != nil && *in.PublishAsUploaded != item.PublishAsUploaded {
		// Existing guideline documents were uploaded under the old mode.
		if item.GuidelineDocumentCount > 0 {
			return nil, ErrDocumentKindModeInUse
		}
		item.PublishAsUploaded = *in.PublishAsUploaded
	}
	if item.Name == "" || len(item.Name) > 120 || !validDocumentKindSlug(item.Slug) || !oneOf(item.Status, "active", "inactive") || item.SortOrder < 0 || item.SortOrder > 10_000 {
		return nil, ErrGuidelineContentInvalid
	}
	if id == nil {
		// Archived kinds keep their slug because older documents still reference it.
		var count int64
		if err := s.DB.Unscoped().Model(&models.DocumentKind{}).Where("slug = ?", item.Slug).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrGuidelineContentConflict
		}
	}
	if wasActive && item.Status != "active" {
		if err := s.ensureOtherActiveDocumentKind(*id); err != nil {
			return nil, err
		}
	}
	if err := s.DB.Save(&item).Error; err != nil {
		return nil, err
	}
	return s.GetDocumentKind(item.ID, true)
}

// DeleteDocumentKind archives a kind that no live document uses. The last
// active kind is kept so new documents always have a kind to default to.
func (s GuidelineContentService) DeleteDocumentKind(id uuid.UUID) error {
	kind, err := s.GetDocumentKind(id, true)
	if err != nil {
		return err
	}
	if kind.GuidelineDocumentCount+kind.OutbreakDocumentCount > 0 {
		return ErrDocumentKindInUse
	}
	if kind.Status == "active" {
		if err := s.ensureOtherActiveDocumentKind(id); err != nil {
			return err
		}
	}
	return deleteExisting(s.DB, &models.DocumentKind{}, id)
}

func (s GuidelineContentService) ensureOtherActiveDocumentKind(id uuid.UUID) error {
	var count int64
	if err := s.DB.Model(&models.DocumentKind{}).Where("status = ? AND id <> ?", "active", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrDocumentKindRequired
	}
	return nil
}

// activeDocumentKindSlug reports whether slug names an active, live kind that
// can be assigned to a document.
func activeDocumentKindSlug(tx *gorm.DB, slug string) (bool, error) {
	if !validDocumentKindSlug(slug) {
		return false, nil
	}
	var count int64
	err := tx.Model(&models.DocumentKind{}).Where("slug = ? AND status = ?", slug, "active").Count(&count).Error
	return count > 0, err
}
