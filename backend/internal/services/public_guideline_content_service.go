package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const publicAssetURLTTL = 10 * time.Minute

type PublicGuidelineContentQuery struct {
	Page     PageInput
	ParentID *uuid.UUID
	Sort     string
	Order    string
}

type PublicGuidelineManifest struct {
	GuidelineID              uuid.UUID                         `json:"guideline_id"`
	VersionID                uuid.UUID                         `json:"version_id"`
	Version                  string                            `json:"version"`
	SchemaVersion            int                               `json:"schema_version"`
	PackageVersion           int                               `json:"package_version"`
	ExtractionQuality        models.GuidelineExtractionQuality `json:"extraction_quality"`
	RecommendedMode          string                            `json:"recommended_mode" enums:"structured,partial,original_document"`
	HasChapters              bool                              `json:"has_chapters"`
	HasKeyPoints             bool                              `json:"has_key_points"`
	HasTables                bool                              `json:"has_tables"`
	HasFigures               bool                              `json:"has_figures"`
	HasAlgorithms            bool                              `json:"has_algorithms"`
	HasOriginalPDF           bool                              `json:"has_original_pdf"`
	HasOfflinePackage        bool                              `json:"has_offline_package"`
	SectionCount             int                               `json:"section_count"`
	ReviewedSectionCount     int                               `json:"reviewed_section_count"`
	LeafSectionCount         int                               `json:"leaf_section_count"`
	ReviewedLeafSectionCount int                               `json:"reviewed_leaf_section_count"`
	EmptyLeafSectionCount    int                               `json:"empty_leaf_section_count"`
	BlockCount               int                               `json:"block_count"`
	ReviewedParagraphCount   int                               `json:"reviewed_paragraph_count"`
	TableCount               int                               `json:"table_count"`
	FigureCount              int                               `json:"figure_count"`
	AlgorithmCount           int                               `json:"algorithm_count"`
	Checksum                 string                            `json:"checksum"`
	ETag                     string                            `json:"etag"`
	GeneratedAt              time.Time                         `json:"generated_at"`
}

type PublicGuidelineSection struct {
	ID        uuid.UUID  `json:"id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Title     string     `json:"title"`
	Slug      string     `json:"slug"`
	Level     int        `json:"level"`
	PageStart *int       `json:"page_start,omitempty"`
	PageEnd   *int       `json:"page_end,omitempty"`
	SortOrder int        `json:"sort_order"`
}

type PublicGuidelineBlock struct {
	ID        uuid.UUID                 `json:"id"`
	SectionID *uuid.UUID                `json:"section_id,omitempty"`
	Type      models.GuidelineBlockType `json:"type"`
	SortOrder int                       `json:"sort_order"`
	Content   json.RawMessage           `json:"content" swaggertype:"object"`
	PageStart *int                      `json:"page_start,omitempty"`
	PageEnd   *int                      `json:"page_end,omitempty"`
}

type PublicGuidelineSectionDetail struct {
	GuidelineID    uuid.UUID              `json:"guideline_id"`
	VersionID      uuid.UUID              `json:"version_id"`
	PackageVersion int                    `json:"package_version"`
	Checksum       string                 `json:"checksum"`
	Section        PublicGuidelineSection `json:"section"`
	Blocks         []PublicGuidelineBlock `json:"blocks"`
}

// PublicGuidelineContent is the complete reviewed structured projection used by
// readers. Keeping sections and blocks in one response avoids one request per
// section for large publications while the section endpoints remain available
// for deep links and backwards-compatible clients.
type PublicGuidelineContent struct {
	GuidelineID    uuid.UUID                `json:"guideline_id"`
	VersionID      uuid.UUID                `json:"version_id"`
	PackageVersion int                      `json:"package_version"`
	Checksum       string                   `json:"checksum"`
	Sections       []PublicGuidelineSection `json:"sections"`
	Blocks         []PublicGuidelineBlock   `json:"blocks"`
}

type PublicGuidelineTable struct {
	ID        uuid.UUID                         `json:"id"`
	SectionID *uuid.UUID                        `json:"section_id,omitempty"`
	SortOrder int                               `json:"sort_order"`
	PageStart *int                              `json:"page_start,omitempty"`
	PageEnd   *int                              `json:"page_end,omitempty"`
	Content   models.GuidelineTableBlockPayload `json:"content"`
}

type PublicGuidelineFigure struct {
	ID        uuid.UUID                          `json:"id"`
	SectionID *uuid.UUID                         `json:"section_id,omitempty"`
	SortOrder int                                `json:"sort_order"`
	PageStart *int                               `json:"page_start,omitempty"`
	PageEnd   *int                               `json:"page_end,omitempty"`
	Content   models.GuidelineFigureBlockPayload `json:"content"`
	Asset     PublicGuidelineAssetLink           `json:"asset"`
}

type PublicGuidelineAlgorithm struct {
	ID        uuid.UUID                             `json:"id"`
	SectionID *uuid.UUID                            `json:"section_id,omitempty"`
	SortOrder int                                   `json:"sort_order"`
	PageStart *int                                  `json:"page_start,omitempty"`
	PageEnd   *int                                  `json:"page_end,omitempty"`
	Content   models.GuidelineAlgorithmBlockPayload `json:"content"`
}

type PublicGuidelineAssetLink struct {
	AssetID          *uuid.UUID `json:"asset_id,omitempty"`
	Type             string     `json:"type"`
	MIMEType         string     `json:"mime_type"`
	Checksum         string     `json:"checksum,omitempty"`
	SizeBytes        int64      `json:"size_bytes,omitempty"`
	OriginalFilename string     `json:"original_filename,omitempty"`
	URL              string     `json:"url"`
	ExpiresAt        time.Time  `json:"expires_at"`
}

// PublicGuidelineAssetDownload keeps managed object-storage reads behind the
// public API boundary. Clients must never need to resolve an internal MinIO or
// S3 hostname to download a published guideline asset.
type PublicGuidelineAssetDownload struct {
	Body      io.ReadCloser
	Filename  string
	MIMEType  string
	SizeBytes int64
	Checksum  string
}

type publicGuidelineAssetSource struct {
	asset      *models.GuidelineAsset
	storageKey string
	filename   string
	mimeType   string
	sizeBytes  int64
	checksum   string
}

func (s PublicGuidelineService) Manifest(ctx context.Context, guidelineID uuid.UUID) (*PublicGuidelineManifest, error) {
	var manifest models.GuidelineVersionManifest
	err := s.visibleQuery(ctx).
		Joins("JOIN guideline_version_manifests AS gvm ON gvm.version_id = gv.id AND gvm.guideline_id = gd.id AND gvm.deleted_at IS NULL").
		Select("gvm.*").Where("gd.id = ?", guidelineID).Take(&manifest).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPublicGuidelineNotFound
	}
	if err != nil {
		return nil, err
	}
	return publicManifest(&manifest), nil
}

func (s PublicGuidelineService) Sections(ctx context.Context, guidelineID uuid.UUID, input PublicGuidelineContentQuery) (*PageResult[PublicGuidelineSection], error) {
	page := input.Page.Normalize(100, 500)
	query := s.publicSectionsQuery(ctx, guidelineID)
	if input.ParentID != nil {
		query = query.Where("gs.parent_id = ?", *input.ParentID)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("gs.id").Count(&total).Error; err != nil {
		return nil, err
	}
	order, err := publicOrder(input.Sort, input.Order, map[string]string{
		"sort_order": "gs.sort_order", "title": "gs.title", "page_start": "gs.page_start",
	}, "gs.sort_order ASC")
	if err != nil {
		return nil, err
	}
	var rows []models.GuidelineSection
	if err := query.Session(&gorm.Session{}).Select("gs.*").Order(order).Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PublicGuidelineSection, 0, len(rows))
	for _, row := range rows {
		items = append(items, publicSection(row))
	}
	return NewPageResult(items, page, total), nil
}

func (s PublicGuidelineService) Content(ctx context.Context, guidelineID uuid.UUID) (*PublicGuidelineContent, error) {
	var manifest models.GuidelineVersionManifest
	var sectionRows []models.GuidelineSection
	var blockRows []models.GuidelineContentBlock
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := visibleManifestQuery(tx, guidelineID).Take(&manifest).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPublicGuidelineNotFound
			}
			return err
		}
		if err := tx.Where("version_id = ? AND deleted_at IS NULL", manifest.VersionID).
			Order("sort_order ASC, id ASC").Find(&sectionRows).Error; err != nil {
			return err
		}
		return tx.Where("version_id = ? AND deleted_at IS NULL AND review_status = ?", manifest.VersionID, models.GuidelineBlockReviewed).
			Where("section_id IS NULL OR EXISTS (SELECT 1 FROM guideline_sections active_section WHERE active_section.id = guideline_content_blocks.section_id AND active_section.version_id = ? AND active_section.deleted_at IS NULL)", manifest.VersionID).
			Order("sort_order ASC, id ASC").Find(&blockRows).Error
	})
	if err != nil {
		return nil, err
	}

	sections := make([]PublicGuidelineSection, 0, len(sectionRows))
	for _, row := range sectionRows {
		sections = append(sections, publicSection(row))
	}
	blocks := make([]PublicGuidelineBlock, 0, len(blockRows))
	for _, row := range blockRows {
		blocks = append(blocks, publicBlock(row))
	}
	return &PublicGuidelineContent{
		GuidelineID: guidelineID, VersionID: manifest.VersionID,
		PackageVersion: manifest.PackageVersion, Checksum: manifest.Checksum,
		Sections: sections, Blocks: blocks,
	}, nil
}

func (s PublicGuidelineService) Section(ctx context.Context, guidelineID, sectionID uuid.UUID) (*PublicGuidelineSectionDetail, error) {
	var manifest models.GuidelineVersionManifest
	var section models.GuidelineSection
	var rows []models.GuidelineContentBlock
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := visibleManifestQuery(tx, guidelineID).Take(&manifest).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPublicGuidelineNotFound
			}
			return err
		}
		if err := tx.Where("id = ? AND version_id = ? AND deleted_at IS NULL", sectionID, manifest.VersionID).Take(&section).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPublicGuidelineNotFound
			}
			return err
		}
		return tx.Where("version_id = ? AND section_id = ? AND deleted_at IS NULL AND review_status = ?", manifest.VersionID, sectionID, models.GuidelineBlockReviewed).
			Order("sort_order ASC, id ASC").Find(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	blocks := make([]PublicGuidelineBlock, 0, len(rows))
	for _, row := range rows {
		blocks = append(blocks, publicBlock(row))
	}
	return &PublicGuidelineSectionDetail{
		GuidelineID: guidelineID, VersionID: manifest.VersionID,
		PackageVersion: manifest.PackageVersion, Checksum: manifest.Checksum,
		Section: publicSection(section), Blocks: blocks,
	}, nil
}

func visibleManifestQuery(db *gorm.DB, guidelineID uuid.UUID) *gorm.DB {
	return db.Table("guideline_version_manifests AS gvm").
		Joins("JOIN guideline_versions AS gv ON gv.id = gvm.version_id AND gv.deleted_at IS NULL AND lower(gv.status) = 'published'").
		Joins("JOIN guideline_documents AS gd ON gd.id = gvm.guideline_id AND gd.current_version_id = gv.id AND gd.deleted_at IS NULL").
		Where("gvm.deleted_at IS NULL AND gd.id = ?", guidelineID).
		Select("gvm.*")
}

func (s PublicGuidelineService) Tables(ctx context.Context, guidelineID uuid.UUID, input PublicGuidelineContentQuery) (*PageResult[PublicGuidelineTable], error) {
	page := input.Page.Normalize(50, 200)
	query := s.publicBlocksQuery(ctx, guidelineID).Where("gcb.type = ?", models.GuidelineBlockTable)
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("gcb.id").Count(&total).Error; err != nil {
		return nil, err
	}
	order, err := publicOrder(input.Sort, input.Order, publicBlockSortColumns(), "gcb.sort_order ASC")
	if err != nil {
		return nil, err
	}
	var rows []models.GuidelineContentBlock
	if err := query.Session(&gorm.Session{}).Select("gcb.*").Order(order).Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PublicGuidelineTable, 0, len(rows))
	for _, row := range rows {
		var payload models.GuidelineTableBlockPayload
		if err := json.Unmarshal(row.ContentJSON, &payload); err != nil {
			return nil, fmt.Errorf("published table payload is invalid: %w", err)
		}
		items = append(items, PublicGuidelineTable{ID: row.ID, SectionID: row.SectionID, SortOrder: row.SortOrder, PageStart: row.PageStart, PageEnd: row.PageEnd, Content: payload})
	}
	return NewPageResult(items, page, total), nil
}

func (s PublicGuidelineService) Figures(ctx context.Context, guidelineID uuid.UUID, input PublicGuidelineContentQuery) (*PageResult[PublicGuidelineFigure], error) {
	page := input.Page.Normalize(50, 200)
	query := s.publicBlocksQuery(ctx, guidelineID).Where("gcb.type = ?", models.GuidelineBlockFigure)
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("gcb.id").Count(&total).Error; err != nil {
		return nil, err
	}
	order, err := publicOrder(input.Sort, input.Order, publicBlockSortColumns(), "gcb.sort_order ASC")
	if err != nil {
		return nil, err
	}
	var rows []models.GuidelineContentBlock
	if err := query.Session(&gorm.Session{}).Select("gcb.*").Order(order).Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PublicGuidelineFigure, 0, len(rows))
	for _, row := range rows {
		var payload models.GuidelineFigureBlockPayload
		if err := json.Unmarshal(row.ContentJSON, &payload); err != nil {
			return nil, fmt.Errorf("published figure payload is invalid: %w", err)
		}
		asset, err := s.assetLink(ctx, guidelineID, payload.AssetID, "")
		if err != nil {
			return nil, err
		}
		items = append(items, PublicGuidelineFigure{ID: row.ID, SectionID: row.SectionID, SortOrder: row.SortOrder, PageStart: row.PageStart, PageEnd: row.PageEnd, Content: payload, Asset: *asset})
	}
	return NewPageResult(items, page, total), nil
}

func (s PublicGuidelineService) Algorithms(ctx context.Context, guidelineID uuid.UUID, input PublicGuidelineContentQuery) (*PageResult[PublicGuidelineAlgorithm], error) {
	page := input.Page.Normalize(50, 200)
	query := s.publicBlocksQuery(ctx, guidelineID).Where("gcb.type = ?", models.GuidelineBlockAlgorithm)
	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("gcb.id").Count(&total).Error; err != nil {
		return nil, err
	}
	order, err := publicOrder(input.Sort, input.Order, publicBlockSortColumns(), "gcb.sort_order ASC")
	if err != nil {
		return nil, err
	}
	var rows []models.GuidelineContentBlock
	if err := query.Session(&gorm.Session{}).Select("gcb.*").Order(order).Limit(page.PerPage).Offset(page.Offset()).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PublicGuidelineAlgorithm, 0, len(rows))
	for _, row := range rows {
		var payload models.GuidelineAlgorithmBlockPayload
		if err := json.Unmarshal(row.ContentJSON, &payload); err != nil {
			return nil, fmt.Errorf("published algorithm payload is invalid: %w", err)
		}
		items = append(items, PublicGuidelineAlgorithm{ID: row.ID, SectionID: row.SectionID, SortOrder: row.SortOrder, PageStart: row.PageStart, PageEnd: row.PageEnd, Content: payload})
	}
	return NewPageResult(items, page, total), nil
}

func (s PublicGuidelineService) Original(ctx context.Context, guidelineID uuid.UUID) (*PublicGuidelineAssetLink, error) {
	return s.assetLink(ctx, guidelineID, uuid.Nil, string(models.GuidelineAssetOriginalPDF))
}

func (s PublicGuidelineService) OfflinePackage(ctx context.Context, guidelineID uuid.UUID) (*PublicGuidelineAssetLink, error) {
	return s.assetLink(ctx, guidelineID, uuid.Nil, string(models.GuidelineAssetOfflinePackage))
}

func (s PublicGuidelineService) AssetDownload(ctx context.Context, guidelineID, assetID uuid.UUID, assetType string) (*PublicGuidelineAssetDownload, error) {
	if s.Store == nil {
		return nil, ErrPublicGuidelineNotFound
	}
	source, err := s.publicAssetSource(ctx, guidelineID, assetID, assetType)
	if err != nil {
		return nil, err
	}
	body, err := s.Store.Get(ctx, source.storageKey)
	if err != nil {
		return nil, err
	}
	return &PublicGuidelineAssetDownload{
		Body: body, Filename: source.filename, MIMEType: source.mimeType,
		SizeBytes: source.sizeBytes, Checksum: source.checksum,
	}, nil
}

func (s PublicGuidelineService) publicSectionsQuery(ctx context.Context, guidelineID uuid.UUID) *gorm.DB {
	return s.DB.WithContext(ctx).Table("guideline_sections AS gs").
		Joins("JOIN guideline_versions AS gv ON gv.id = gs.version_id AND gv.deleted_at IS NULL AND lower(gv.status) = 'published'").
		Joins("JOIN guideline_documents AS gd ON gd.id = gv.document_id AND gd.current_version_id = gv.id AND gd.deleted_at IS NULL").
		Where("gs.deleted_at IS NULL AND gd.id = ?", guidelineID)
}

func (s PublicGuidelineService) publicBlocksQuery(ctx context.Context, guidelineID uuid.UUID) *gorm.DB {
	return s.DB.WithContext(ctx).Table("guideline_content_blocks AS gcb").
		Joins("JOIN guideline_versions AS gv ON gv.id = gcb.version_id AND gv.deleted_at IS NULL AND lower(gv.status) = 'published'").
		Joins("JOIN guideline_documents AS gd ON gd.id = gv.document_id AND gd.current_version_id = gv.id AND gd.deleted_at IS NULL").
		Where("gcb.deleted_at IS NULL AND gcb.review_status = ? AND gd.id = ?", models.GuidelineBlockReviewed, guidelineID).
		Where("gcb.section_id IS NULL OR EXISTS (SELECT 1 FROM guideline_sections active_section WHERE active_section.id = gcb.section_id AND active_section.version_id = gv.id AND active_section.deleted_at IS NULL)")
}

func (s PublicGuidelineService) assetLink(ctx context.Context, guidelineID, assetID uuid.UUID, assetType string) (*PublicGuidelineAssetLink, error) {
	if s.Store == nil {
		return nil, ErrPublicGuidelineNotFound
	}
	source, err := s.publicAssetSource(ctx, guidelineID, assetID, assetType)
	if err != nil {
		return nil, err
	}
	downloadURL := fmt.Sprintf("/api/public/guidelines/%s/assets/%s/download", guidelineID, assetID)
	if assetID == uuid.Nil {
		switch assetType {
		case string(models.GuidelineAssetOriginalPDF):
			downloadURL = fmt.Sprintf("/api/public/guidelines/%s/original/download", guidelineID)
		case string(models.GuidelineAssetOfflinePackage):
			downloadURL = fmt.Sprintf("/api/public/guidelines/%s/offline-package/download", guidelineID)
		default:
			return nil, ErrPublicGuidelineNotFound
		}
	}
	result := &PublicGuidelineAssetLink{
		Type: assetType, MIMEType: source.mimeType, Checksum: source.checksum,
		SizeBytes: source.sizeBytes, OriginalFilename: source.filename,
		URL: downloadURL, ExpiresAt: time.Now().UTC().Add(publicAssetURLTTL),
	}
	if source.asset != nil {
		result.AssetID = &source.asset.ID
		result.Type = string(source.asset.Type)
	}
	return result, nil
}

func (s PublicGuidelineService) publicAssetSource(ctx context.Context, guidelineID, assetID uuid.UUID, assetType string) (*publicGuidelineAssetSource, error) {
	row, err := s.getVisibleRow(ctx, guidelineID)
	if err != nil {
		return nil, err
	}
	var asset models.GuidelineAsset
	query := s.DB.WithContext(ctx).Where("version_id = ?", row.VersionID)
	if assetID != uuid.Nil {
		query = query.Where("id = ? AND review_status = ?", assetID, models.GuidelineBlockReviewed)
	} else {
		query = query.Where("type = ?", assetType)
		if assetType == string(models.GuidelineAssetOfflinePackage) {
			query = query.Where("review_status = ?", models.GuidelineBlockReviewed)
		}
	}
	err = query.Order("created_at DESC").First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) && assetType == string(models.GuidelineAssetOriginalPDF) && strings.TrimSpace(row.OriginalFileKey) != "" {
		extension, mimeType := originalFileType(row.OriginalFileKey)
		return &publicGuidelineAssetSource{
			storageKey: row.OriginalFileKey,
			filename:   slugify(row.Title) + "." + extension,
			mimeType:   mimeType,
		}, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPublicGuidelineNotFound
	}
	if err != nil {
		return nil, err
	}
	filename := ""
	if asset.OriginalFilename != nil {
		filename = *asset.OriginalFilename
	}
	return &publicGuidelineAssetSource{
		asset: &asset, storageKey: asset.StorageKey, filename: filename,
		mimeType: asset.MIMEType, sizeBytes: asset.SizeBytes, checksum: asset.Checksum,
	}, nil
}

func publicManifest(row *models.GuidelineVersionManifest) *PublicGuidelineManifest {
	return &PublicGuidelineManifest{
		GuidelineID: row.GuidelineID, VersionID: row.VersionID, Version: row.Version,
		SchemaVersion: row.SchemaVersion, PackageVersion: row.PackageVersion,
		ExtractionQuality: row.ExtractionQuality, HasChapters: row.HasChapters,
		RecommendedMode: recommendedGuidelineReaderMode(row),
		HasKeyPoints:    row.HasKeyPoints, HasTables: row.HasTables, HasFigures: row.HasFigures,
		HasAlgorithms: row.HasAlgorithms, HasOriginalPDF: row.HasOriginalPDF,
		HasOfflinePackage: row.HasOfflinePackage, SectionCount: row.SectionCount,
		ReviewedSectionCount: row.ReviewedSectionCount, LeafSectionCount: row.LeafSectionCount,
		ReviewedLeafSectionCount: row.ReviewedLeafSectionCount, EmptyLeafSectionCount: row.EmptyLeafSectionCount,
		BlockCount: row.BlockCount, TableCount: row.TableCount, FigureCount: row.FigureCount,
		ReviewedParagraphCount: row.ReviewedParagraphCount,
		AlgorithmCount:         row.AlgorithmCount, Checksum: row.Checksum, ETag: row.ETag,
		GeneratedAt: row.GeneratedAt,
	}
}

func recommendedGuidelineReaderMode(row *models.GuidelineVersionManifest) string {
	switch row.ExtractionQuality {
	case models.GuidelineExtractionReviewed:
		return "structured"
	case models.GuidelineExtractionPartiallyReviewed:
		return "partial"
	case models.GuidelineExtractionMarkdownFallback:
		if row.SectionCount > 0 {
			return "partial"
		}
	}
	return "original_document"
}

func publicSection(row models.GuidelineSection) PublicGuidelineSection {
	return PublicGuidelineSection{ID: row.ID, ParentID: row.ParentID, Title: row.Title, Slug: row.Slug, Level: row.Level, PageStart: row.PageStart, PageEnd: row.PageEnd, SortOrder: row.SortOrder}
}

func publicBlock(row models.GuidelineContentBlock) PublicGuidelineBlock {
	content := append(json.RawMessage(nil), row.ContentJSON...)
	return PublicGuidelineBlock{ID: row.ID, SectionID: row.SectionID, Type: row.Type, SortOrder: row.SortOrder, Content: content, PageStart: row.PageStart, PageEnd: row.PageEnd}
}

func publicBlockSortColumns() map[string]string {
	return map[string]string{"sort_order": "gcb.sort_order", "page_start": "gcb.page_start", "type": "gcb.type"}
}

func publicOrder(sortValue, orderValue string, allowed map[string]string, fallback string) (string, error) {
	field := strings.TrimSpace(sortValue)
	if field == "" {
		return fallback, nil
	}
	column, ok := allowed[field]
	if !ok {
		return "", ErrPublicGuidelineQuery
	}
	direction := strings.ToUpper(strings.TrimSpace(orderValue))
	if direction == "" {
		direction = "ASC"
	}
	if direction != "ASC" && direction != "DESC" {
		return "", ErrPublicGuidelineQuery
	}
	return column + " " + direction + ", " + strings.Split(fallback, " ")[0] + " ASC", nil
}
