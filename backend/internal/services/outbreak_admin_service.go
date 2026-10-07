package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"mediguide/internal/models"
	"mediguide/internal/storage"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OutbreakActor struct {
	ID uuid.UUID
	IP string
}

type OutbreakAdminService struct {
	DB                   *gorm.DB
	Store                storage.ObjectStore
	AllowedExternalHosts []string
}

type OutbreakAdminQuery struct {
	Page                                                                                                    PageInput
	Search, Status, Disease, Area, RegionID, VisualTone, EffectiveFrom, EffectiveTo, UpdatedFrom, UpdatedTo string
	Sort, Order                                                                                             string
}
type OutbreakMetricsInput struct {
	Metrics     []OutbreakMetric `json:"metrics" binding:"required"`
	LockVersion *int             `json:"lock_version"`
}
type OutbreakInput struct {
	Title              *string           `json:"title"`
	DiseaseID          *uuid.UUID        `json:"disease_id"`
	GeographicArea     *string           `json:"geographic_area"`
	RegionID           *uuid.UUID        `json:"region_id"`
	DistrictID         *uuid.UUID        `json:"district_id"`
	Summary            *string           `json:"summary"`
	StartDate          *time.Time        `json:"start_date"`
	LastUpdate         *time.Time        `json:"last_update"`
	VisualTone         *string           `json:"visual_tone"`
	SourceOrganization *string           `json:"source_organization"`
	SourceURL          *string           `json:"source_url"`
	SourceReference    *string           `json:"source_reference"`
	EffectiveAt        *time.Time        `json:"effective_at"`
	DataAsOf           *time.Time        `json:"data_as_of"`
	LastVerifiedAt     *time.Time        `json:"last_verified_at"`
	Metrics            *[]OutbreakMetric `json:"metrics"`
	LockVersion        *int              `json:"lock_version"`
}
type ChildContentInput struct {
	Title               *string `json:"title"`
	Summary             *string `json:"summary"`
	Description         *string `json:"description"`
	IssuingOrganization *string `json:"issuing_organization"`
	ResourceType        *string `json:"resource_type"`
	DocumentKind        *string `json:"document_kind"`
	URL                 *string `json:"url"`
	AssetURL            *string `json:"asset_url"`
	SortOrder           *int    `json:"sort_order"`
	LockVersion         *int    `json:"lock_version"`
}
type SituationReportInput struct {
	OutbreakID         *uuid.UUID        `json:"outbreak_id"`
	RegionID           *uuid.UUID        `json:"region_id"`
	DistrictID         *uuid.UUID        `json:"district_id"`
	Title              *string           `json:"title"`
	GeographicArea     *string           `json:"geographic_area"`
	Summary            *string           `json:"summary"`
	SourceOrganization *string           `json:"source_organization"`
	PublicationDate    *time.Time        `json:"publication_date"`
	StandaloneAllowed  *bool             `json:"standalone_allowed"`
	SourceURL          *string           `json:"source_url"`
	SourceReference    *string           `json:"source_reference"`
	EffectiveAt        *time.Time        `json:"effective_at"`
	DataAsOf           *time.Time        `json:"data_as_of"`
	LastVerifiedAt     *time.Time        `json:"last_verified_at"`
	KeyHighlights      *[]string         `json:"key_highlights"`
	Metrics            *[]OutbreakMetric `json:"metrics"`
	LockVersion        *int              `json:"lock_version"`
}

// ResourceCorrectionInput replaces a published resource with an edited version
// that goes through review before it supersedes the original.
type ResourceCorrectionInput struct {
	LockVersion int                `json:"lock_version"`
	Reason      string             `json:"reason"`
	Changes     *ChildContentInput `json:"changes"`
}

type TransitionInput struct {
	LockVersion       int    `json:"lock_version"`
	Reason            string `json:"reason,omitempty"`
	OperationalStatus string `json:"operational_status,omitempty"`
}

type OutbreakReviewCommentInput struct {
	Comment string `json:"comment" binding:"required,max=4000"`
}

type OutbreakAdminDTO struct {
	ID                 uuid.UUID        `json:"id"`
	Title              string           `json:"title"`
	DiseaseID          *uuid.UUID       `json:"disease_id,omitempty"`
	DiseaseName        string           `json:"disease_name,omitempty"`
	Status             string           `json:"status"`
	GeographicArea     string           `json:"geographic_area"`
	RegionID           *uuid.UUID       `json:"region_id,omitempty"`
	DistrictID         *uuid.UUID       `json:"district_id,omitempty"`
	Summary            string           `json:"summary"`
	StartDate          *time.Time       `json:"start_date,omitempty"`
	LastUpdate         time.Time        `json:"last_update"`
	VisualTone         string           `json:"visual_tone"`
	SourceOrganization string           `json:"source_organization"`
	PublishedAt        *time.Time       `json:"published_at,omitempty"`
	AuthorID           *uuid.UUID       `json:"author_id,omitempty"`
	ReviewedBy         *uuid.UUID       `json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time       `json:"reviewed_at,omitempty"`
	ApprovedBy         *uuid.UUID       `json:"approved_by,omitempty"`
	ApprovedAt         *time.Time       `json:"approved_at,omitempty"`
	WithdrawnAt        *time.Time       `json:"withdrawn_at,omitempty"`
	WithdrawalReason   string           `json:"withdrawal_reason,omitempty"`
	SupersedesID       *uuid.UUID       `json:"supersedes_id,omitempty"`
	SourceURL          string           `json:"source_url,omitempty"`
	SourceReference    string           `json:"source_reference,omitempty"`
	EffectiveAt        *time.Time       `json:"effective_at,omitempty"`
	DataAsOf           *time.Time       `json:"data_as_of,omitempty"`
	LastVerifiedAt     *time.Time       `json:"last_verified_at,omitempty"`
	LockVersion        int              `json:"lock_version"`
	Metrics            []OutbreakMetric `json:"metrics"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	// OpenCorrectionID is the correction of this outbreak that is still in
	// draft or review, if any. Only set when a single outbreak is fetched.
	OpenCorrectionID *uuid.UUID `json:"open_correction_id,omitempty"`
}
type OutbreakUpdateAdminDTO struct {
	ID               uuid.UUID  `json:"id"`
	OutbreakID       uuid.UUID  `json:"outbreak_id"`
	Title            string     `json:"title"`
	Summary          string     `json:"summary"`
	Status           string     `json:"status"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	AuthorID         *uuid.UUID `json:"author_id,omitempty"`
	ReviewedBy       *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ApprovedBy       *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	WithdrawnAt      *time.Time `json:"withdrawn_at,omitempty"`
	WithdrawalReason string     `json:"withdrawal_reason,omitempty"`
	SupersedesID     *uuid.UUID `json:"supersedes_id,omitempty"`
	LockVersion      int        `json:"lock_version"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
type OutbreakResourceAdminDTO struct {
	ID                  uuid.UUID  `json:"id"`
	OutbreakID          uuid.UUID  `json:"outbreak_id"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	IssuingOrganization string     `json:"issuing_organization"`
	ResourceType        string     `json:"resource_type"`
	DocumentKind        string     `json:"document_kind"`
	URL                 string     `json:"url"`
	AssetURL            string     `json:"asset_url"`
	SortOrder           int        `json:"sort_order"`
	Status              string     `json:"status"`
	PublishedAt         *time.Time `json:"published_at,omitempty"`
	AuthorID            *uuid.UUID `json:"author_id,omitempty"`
	ReviewedBy          *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	ApprovedBy          *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
	WithdrawnAt         *time.Time `json:"withdrawn_at,omitempty"`
	WithdrawalReason    string     `json:"withdrawal_reason,omitempty"`
	SupersedesID        *uuid.UUID `json:"supersedes_id,omitempty"`
	LockVersion         int        `json:"lock_version"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
type SituationReportAdminDTO struct {
	ID                 uuid.UUID        `json:"id"`
	OutbreakID         *uuid.UUID       `json:"outbreak_id,omitempty"`
	RegionID           *uuid.UUID       `json:"region_id,omitempty"`
	DistrictID         *uuid.UUID       `json:"district_id,omitempty"`
	Title              string           `json:"title"`
	GeographicArea     string           `json:"geographic_area"`
	Summary            string           `json:"summary"`
	SourceOrganization string           `json:"source_organization"`
	PublicationDate    time.Time        `json:"publication_date"`
	Status             string           `json:"status"`
	ReportAssetURL     string           `json:"report_asset_url,omitempty"`
	ReportAssetID      *uuid.UUID       `json:"report_asset_id,omitempty"`
	StandaloneAllowed  bool             `json:"standalone_allowed"`
	AuthorID           *uuid.UUID       `json:"author_id,omitempty"`
	PublishedAt        *time.Time       `json:"published_at,omitempty"`
	SubmittedBy        *uuid.UUID       `json:"submitted_by,omitempty"`
	SubmittedAt        *time.Time       `json:"submitted_at,omitempty"`
	ReviewedBy         *uuid.UUID       `json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time       `json:"reviewed_at,omitempty"`
	ApprovedBy         *uuid.UUID       `json:"approved_by,omitempty"`
	ApprovedAt         *time.Time       `json:"approved_at,omitempty"`
	WithdrawnAt        *time.Time       `json:"withdrawn_at,omitempty"`
	WithdrawalReason   string           `json:"withdrawal_reason,omitempty"`
	CorrectionReason   string           `json:"correction_reason,omitempty"`
	SupersedesID       *uuid.UUID       `json:"supersedes_id,omitempty"`
	SourceURL          string           `json:"source_url,omitempty"`
	SourceReference    string           `json:"source_reference,omitempty"`
	EffectiveAt        *time.Time       `json:"effective_at,omitempty"`
	DataAsOf           *time.Time       `json:"data_as_of,omitempty"`
	LastVerifiedAt     *time.Time       `json:"last_verified_at,omitempty"`
	LockVersion        int              `json:"lock_version"`
	KeyHighlights      []string         `json:"key_highlights"`
	Metrics            []OutbreakMetric `json:"metrics"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	// OpenCorrectionID is the correction of this report that is still in
	// draft or review, if any. Only set when a single report is fetched.
	OpenCorrectionID *uuid.UUID `json:"open_correction_id,omitempty"`
}
type OutbreakAuditDTO struct {
	ID         uuid.UUID      `json:"id"`
	ActorID    string         `json:"actor_id"`
	ActorName  string         `json:"actor_name,omitempty"`
	ActorEmail string         `json:"actor_email,omitempty"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Metadata   map[string]any `json:"metadata"`
	// Labels names the people and records the metadata refers to by ID, such
	// as who wrote an applied correction or the districts it switched between.
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// auditReferences are the metadata fields that hold record IDs, and where
// their display names live.
var auditReferences = map[string]struct{ table, column string }{
	"region_id":   {"regions", "name"},
	"district_id": {"districts", "name"},
	"disease_id":  {"diseases", "name"},
	"outbreak_id": {"outbreaks", "title"},
}

func (s OutbreakAdminService) ListAudit(entityType string, id uuid.UUID, page PageInput) (*PageResult[OutbreakAuditDTO], error) {
	if !validOutbreakValue(entityType, "outbreak", "situation_report") {
		return nil, ErrOutbreakInvalid
	}
	page = page.Normalize(20, 100)
	query := s.DB.Model(&models.AuditLog{}).Where("entity_type = ? AND entity_id = ?", entityType, id.String())
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []models.AuditLog
	if err := query.Order("created_at DESC, id DESC").Offset(page.Offset()).Limit(page.PerPage).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]OutbreakAuditDTO, len(rows))
	people := map[uuid.UUID]bool{}
	records := map[string]map[uuid.UUID]bool{}
	mentioned := make([][]string, len(rows))
	note := func(index int, set map[uuid.UUID]bool, value any) {
		text, _ := value.(string)
		if ref, err := uuid.Parse(text); err == nil && ref != uuid.Nil {
			set[ref] = true
			mentioned[index] = append(mentioned[index], text)
		}
	}
	for index, row := range rows {
		metadata := map[string]any{}
		_ = json.Unmarshal([]byte(row.MetadataJSON), &metadata)
		items[index] = OutbreakAuditDTO{ID: row.ID, ActorID: row.ActorID, Action: row.Action, EntityType: row.EntityType, EntityID: row.EntityID, Metadata: metadata, CreatedAt: row.CreatedAt}
		note(index, people, row.ActorID)
		note(index, people, metadata["corrected_by"])
		changes, _ := metadata["changes"].(map[string]any)
		for field, change := range changes {
			value, _ := change.(map[string]any)
			if _, ok := auditReferences[field]; !ok || value == nil {
				continue
			}
			if records[field] == nil {
				records[field] = map[uuid.UUID]bool{}
			}
			note(index, records[field], value["from"])
			note(index, records[field], value["to"])
		}
	}

	// Deleted users and records keep their names in the history.
	names, emails := map[string]string{}, map[string]string{}
	if len(people) > 0 {
		var users []models.User
		if err := s.DB.Unscoped().Select("id", "name", "email").Where("id IN ?", setKeys(people)).Find(&users).Error; err != nil {
			return nil, err
		}
		for _, user := range users {
			names[user.ID.String()], emails[user.ID.String()] = user.Name, user.Email
		}
	}
	for field, ids := range records {
		ref := auditReferences[field]
		var found []struct {
			ID    uuid.UUID
			Label string
		}
		if err := s.DB.Table(ref.table).Select("id, "+ref.column+" AS label").Where("id IN ?", setKeys(ids)).Scan(&found).Error; err != nil {
			return nil, err
		}
		for _, row := range found {
			names[row.ID.String()] = row.Label
		}
	}
	for index := range items {
		items[index].ActorName, items[index].ActorEmail = names[items[index].ActorID], emails[items[index].ActorID]
		for _, ref := range mentioned[index] {
			if name, ok := names[ref]; ok && ref != items[index].ActorID {
				if items[index].Labels == nil {
					items[index].Labels = map[string]string{}
				}
				items[index].Labels[ref] = name
			}
		}
	}
	return NewPageResult(items, page, total), nil
}

func setKeys(set map[uuid.UUID]bool) []uuid.UUID {
	keys := make([]uuid.UUID, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	return keys
}

// transitionAudit records why a workflow step was taken and, when the step
// changes the status, which status it moved from and to.
func transitionAudit(reason, from string, updates map[string]any) map[string]any {
	meta := map[string]any{"reason": reason}
	if to, ok := updates["status"].(string); ok && to != from {
		meta["from_status"], meta["to_status"] = from, to
	}
	return meta
}

func (s OutbreakAdminService) AddReviewComment(actor OutbreakActor, entityType string, id uuid.UUID, comment string) error {
	comment = strings.TrimSpace(comment)
	if !validOutbreakValue(entityType, "outbreak", "situation_report") || comment == "" || len(comment) > 4000 {
		return ErrOutbreakInvalid
	}
	model := any(&models.Outbreak{})
	if entityType == "situation_report" {
		model = &models.SituationReport{}
	}
	var count int64
	if err := s.DB.Model(model).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return auditOutbreak(s.DB, actor, entityType+".review_comment", entityType, id, map[string]any{"comment": comment})
}

func (s OutbreakAdminService) ListOutbreaks(q OutbreakAdminQuery) (*PageResult[OutbreakAdminDTO], error) {
	p := q.Page.Normalize(20, 100)
	db := s.DB.Model(&models.Outbreak{})
	if v := strings.TrimSpace(q.Search); v != "" {
		like := "%" + strings.ToLower(v) + "%"
		db = db.Where("lower(title) LIKE ? OR lower(summary) LIKE ? OR EXISTS (SELECT 1 FROM diseases d WHERE d.id = outbreaks.disease_id AND lower(d.name) LIKE ?)", like, like, like)
	}
	if v := strings.TrimSpace(q.Status); v != "" {
		if !validOutbreakValue(v, "draft", "pending_review", "published", "active", "monitoring", "contained", "closed", "withdrawn") {
			return nil, ErrOutbreakInvalid
		}
		db = db.Where("status = ?", v)
	}
	if v := strings.TrimSpace(q.Area); v != "" {
		db = db.Where("lower(geographic_area) LIKE ?", "%"+strings.ToLower(v)+"%")
	}
	if v := strings.TrimSpace(q.Disease); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, ErrOutbreakInvalid
		}
		db = db.Where("disease_id = ?", id)
	}
	if v := strings.TrimSpace(q.RegionID); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, ErrOutbreakInvalid
		}
		db = db.Where("region_id = ?", id)
	}
	if v := strings.TrimSpace(q.VisualTone); v != "" {
		if !validOutbreakValue(v, "neutral", "info", "warning", "critical", "success") {
			return nil, ErrOutbreakInvalid
		}
		db = db.Where("visual_tone = ?", v)
	}
	for _, filter := range []struct {
		value, predicate string
	}{{q.EffectiveFrom, "effective_at >= ?"}, {q.EffectiveTo, "effective_at <= ?"}, {q.UpdatedFrom, "updated_at >= ?"}, {q.UpdatedTo, "updated_at <= ?"}} {
		if strings.TrimSpace(filter.value) == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, filter.value)
		if err != nil {
			return nil, ErrOutbreakInvalid
		}
		db = db.Where(filter.predicate, parsed)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	order, err := outbreakAdminOrder(q.Sort, q.Order)
	if err != nil {
		return nil, err
	}
	var rows []models.Outbreak
	if err = db.Select("outbreaks.*, (SELECT name FROM diseases WHERE diseases.id = outbreaks.disease_id) AS disease_name").Order(order).Offset(p.Offset()).Limit(p.PerPage).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]OutbreakAdminDTO, len(rows))
	for i := range rows {
		items[i] = outbreakAdminDTO(rows[i])
	}
	return NewPageResult(items, p, total), nil
}
func (s OutbreakAdminService) GetOutbreak(id uuid.UUID) (*OutbreakAdminDTO, error) {
	var row models.Outbreak
	if err := s.DB.Model(&models.Outbreak{}).Select("outbreaks.*, (SELECT name FROM diseases WHERE diseases.id = outbreaks.disease_id) AS disease_name").First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	v := outbreakAdminDTO(row)
	var open models.Outbreak
	if err := s.DB.Select("id").Where("supersedes_id = ? AND status IN ?", id, []string{"draft", "pending_review"}).Order("created_at DESC").Limit(1).Find(&open).Error; err != nil {
		return nil, err
	}
	if open.ID != uuid.Nil {
		v.OpenCorrectionID = &open.ID
	}
	return &v, nil
}
func (s OutbreakAdminService) CreateOutbreak(actor OutbreakActor, in OutbreakInput) (*OutbreakAdminDTO, error) {
	now := time.Now()
	row := models.Outbreak{Title: "", Status: "draft", LastUpdate: now, VisualTone: "warning", AuthorID: &actor.ID, LockVersion: 1, Metrics: datatypes.JSON("[]")}
	if err := applyOutbreak(&row, in); err != nil {
		return nil, err
	}
	if err := s.validateOutbreakFields(row, false); err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "outbreak.created", "outbreak", row.ID, map[string]any{"status": row.Status})
	}); err != nil {
		return nil, err
	}
	return s.GetOutbreak(row.ID)
}
func (s OutbreakAdminService) UpdateOutbreak(actor OutbreakActor, id uuid.UUID, in OutbreakInput) (*OutbreakAdminDTO, error) {
	if in.LockVersion == nil {
		return nil, ErrOutbreakInvalid
	}
	var row models.Outbreak
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if publicOutbreakStatus(row.Status) || row.Status == "withdrawn" {
		return nil, ErrOutbreakImmutable
	}
	if row.SupersedesID != nil {
		// A correction only carries core metadata and source; metrics stay on
		// the live outbreak, so any sent here are ignored.
		in.Metrics = nil
	}
	if err := applyOutbreak(&row, in); err != nil {
		return nil, err
	}
	if err := s.validateOutbreakFields(row, false); err != nil {
		return nil, err
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		updates := outbreakCoreFields(row)
		updates["metrics"] = row.Metrics
		updates["lock_version"] = gorm.Expr("lock_version + 1")
		r := tx.Model(&models.Outbreak{}).Where("id = ? AND lock_version = ?", id, *in.LockVersion).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "outbreak.updated", "outbreak", id, nil)
	})
	if err != nil {
		return nil, err
	}
	return s.GetOutbreak(id)
}

// UpdateMetrics lets metrics be kept current on an outbreak regardless of its
// lifecycle status. Unlike UpdateOutbreak, it is not blocked once an outbreak
// is published, because case counts and similar figures need to keep moving
// after publication without going through the correction/re-approval flow
// that guards the outbreak's other, editorially-reviewed fields.
func (s OutbreakAdminService) UpdateMetrics(actor OutbreakActor, id uuid.UUID, in OutbreakMetricsInput) (*OutbreakAdminDTO, error) {
	if in.LockVersion == nil {
		return nil, ErrOutbreakInvalid
	}
	if err := s.ensureOutbreak(id); err != nil {
		return nil, err
	}
	if err := validateOutbreakMetrics(in.Metrics); err != nil {
		return nil, err
	}
	encoded, err := encodeMetrics(in.Metrics)
	if err != nil {
		return nil, err
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var row models.Outbreak
		if err := tx.Select("id", "title", "status", "published_at", "metrics", "supersedes_id").First(&row, "id = ?", id).Error; err != nil {
			return err
		}
		if row.SupersedesID != nil {
			return invalid("Metrics are kept on the live outbreak, not on its correction. Change them there.")
		}
		r := tx.Model(&models.Outbreak{}).Where("id = ? AND lock_version = ?", id, *in.LockVersion).Updates(map[string]any{"metrics": datatypes.JSON(encoded), "lock_version": gorm.Expr("lock_version + 1")})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		if err := auditOutbreak(tx, actor, "outbreak.metrics_updated", "outbreak", id, nil); err != nil {
			return err
		}
		// New figures on a live outbreak are announced; removals and draft
		// outbreaks are not.
		added := addedMetrics(decodeMetrics(row.Metrics), in.Metrics)
		if len(added) == 0 || row.PublishedAt == nil || !publicOutbreakStatus(row.Status) {
			return nil
		}
		return enqueueOutbreakTopicAlertTx(tx, outbreakTopicAlert{SourceType: "outbreak_metric", SourceID: uuid.New(), OutbreakID: id, Title: row.Title, Body: metricAlertBody(added)}, time.Now())
	})
	if err != nil {
		return nil, err
	}
	return s.GetOutbreak(id)
}

// OutbreakDeleteInput records why an outbreak is being deleted.
type OutbreakDeleteInput struct {
	Reason string `json:"reason"`
}

// OutbreakDeleteResult says what deleting an outbreak removed and what it only
// unlinked.
type OutbreakDeleteResult struct {
	DeletedUpdates     int64 `json:"deleted_updates"`
	DeletedResources   int64 `json:"deleted_resources"`
	DeletedCorrections int64 `json:"deleted_corrections"`
	CancelledAlerts    int64 `json:"cancelled_alerts"`
	UnlinkedReports    int64 `json:"unlinked_reports"`
	UnlinkedHubs       int64 `json:"unlinked_hubs"`
}

// DeleteOutbreak soft-deletes an outbreak in any state, together with what
// only exists inside it: its updates, resources, open corrections, disease
// tags and unsent alerts. Content managed elsewhere and later linked to it is
// kept: situation reports lose their outbreak and content hubs lose the
// mapping. Removing anything beyond a draft or a submission in review takes
// it off the public app, so it needs the withdraw permission too.
func (s OutbreakAdminService) DeleteOutbreak(actor OutbreakActor, id uuid.UUID, lock int, in OutbreakDeleteInput, canWithdraw bool) (*OutbreakDeleteResult, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, invalidField("Give a reason for deleting this outbreak.", "reason")
	}
	var row models.Outbreak
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if row.Status != "draft" && row.Status != "pending_review" && !canWithdraw {
		return nil, ErrOutbreakForbidden
	}
	wasLive := row.PublishedAt != nil && row.WithdrawnAt == nil && publicOutbreakStatus(row.Status)
	var result OutbreakDeleteResult
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Where("id = ? AND lock_version = ?", id, lock).Delete(&models.Outbreak{})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}

		// Owned by the outbreak: deleted with it.
		if r = tx.Where("outbreak_id = ?", id).Delete(&models.OutbreakUpdate{}); r.Error != nil {
			return r.Error
		}
		result.DeletedUpdates = r.RowsAffected
		if r = tx.Where("outbreak_id = ?", id).Delete(&models.OutbreakResource{}); r.Error != nil {
			return r.Error
		}
		result.DeletedResources = r.RowsAffected
		if r = tx.Where("supersedes_id = ? AND status IN ?", id, []string{"draft", "pending_review"}).Delete(&models.Outbreak{}); r.Error != nil {
			return r.Error
		}
		result.DeletedCorrections = r.RowsAffected
		// A correction published on its own before corrections were applied in
		// place becomes an ordinary outbreak, as the foreign key intends.
		if err := tx.Model(&models.Outbreak{}).Where("supersedes_id = ?", id).Update("supersedes_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("content_type = ? AND content_id = ?", models.ContentDiseaseOutbreak, id).Delete(&models.ContentDiseaseAssignment{}).Error; err != nil {
			return err
		}
		// Alerts already being sent can't be recalled; queued ones are dropped.
		if r = tx.Where("topic = ? AND status IN ? AND payload_json->'action'->>'resource_id' = ?", PublicOutbreakTopic, []string{"pending", "retry"}, id.String()).Delete(&models.NotificationTopicJob{}); r.Error != nil {
			return r.Error
		}
		result.CancelledAlerts = r.RowsAffected

		// Linked from elsewhere: kept, only unlinked.
		var reportIDs []uuid.UUID
		if err := tx.Model(&models.SituationReport{}).Where("outbreak_id = ?", id).Pluck("id", &reportIDs).Error; err != nil {
			return err
		}
		if len(reportIDs) > 0 {
			// A report without an outbreak is public only when it may stand
			// alone, so each report stays exactly as visible as it was: shown
			// if the outbreak was live, hidden if the outbreak was hiding it.
			if err := tx.Model(&models.SituationReport{}).Where("id IN ?", reportIDs).Updates(map[string]any{"outbreak_id": nil, "standalone_allowed": wasLive, "lock_version": gorm.Expr("lock_version + 1")}).Error; err != nil {
				return err
			}
		}
		result.UnlinkedReports = int64(len(reportIDs))
		var hubIDs []uuid.UUID
		if err := tx.Model(&models.ContentHubOutbreak{}).Where("outbreak_id = ?", id).Pluck("content_hub_id", &hubIDs).Error; err != nil {
			return err
		}
		if err := tx.Where("outbreak_id = ?", id).Delete(&models.ContentHubOutbreak{}).Error; err != nil {
			return err
		}
		result.UnlinkedHubs = int64(len(hubIDs))

		return auditOutbreak(tx, actor, "outbreak.deleted", "outbreak", id, map[string]any{
			"reason": reason, "status": row.Status, "title": row.Title,
			"deleted_updates": result.DeletedUpdates, "deleted_resources": result.DeletedResources,
			"deleted_corrections": result.DeletedCorrections, "cancelled_alerts": result.CancelledAlerts,
			"unlinked_report_ids": reportIDs, "unlinked_hub_ids": hubIDs,
		})
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (s OutbreakAdminService) TransitionOutbreak(actor OutbreakActor, id uuid.UUID, action string, in TransitionInput) (*OutbreakAdminDTO, error) {
	var row models.Outbreak
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if row.LockVersion != in.LockVersion {
		return nil, ErrOutbreakConflict
	}
	now := time.Now()
	updates := map[string]any{"lock_version": gorm.Expr("lock_version + 1")}
	var alert *outbreakTopicAlert
	switch action {
	case "submit":
		if row.Status != "draft" {
			return nil, invalidf("Only a draft can be submitted for review. This outbreak is %s.", outbreakStatusLabel(row.Status))
		}
		if err := s.validateOutbreakFields(row, false); err != nil {
			return nil, err
		}
		updates["status"] = "pending_review"
		updates["reviewed_by"] = nil
		updates["reviewed_at"] = nil
		updates["approved_by"] = nil
		updates["approved_at"] = nil
	case "approve":
		if row.Status != "pending_review" {
			return nil, invalidf("Only an outbreak in review can be approved. This outbreak is %s.", outbreakStatusLabel(row.Status))
		}
		if row.ApprovedAt != nil {
			return nil, invalid("This outbreak is already approved and is waiting to be published.")
		}
		if row.AuthorID != nil && *row.AuthorID == actor.ID {
			if row.SupersedesID != nil {
				return nil, invalid("You created this correction, so a different reviewer has to approve it.")
			}
			return nil, invalid("You created this outbreak, so a different reviewer has to approve it.")
		}
		if row.SupersedesID != nil {
			return s.applyCorrection(actor, row, in)
		}
		updates["reviewed_by"] = actor.ID
		updates["reviewed_at"] = now
		updates["approved_by"] = actor.ID
		updates["approved_at"] = now
	case "publish":
		if row.SupersedesID != nil {
			return nil, invalid("A correction isn't published on its own. Approving it applies the changes to the live outbreak.")
		}
		if row.Status != "pending_review" {
			return nil, invalidf("Only an outbreak in review can be published. This outbreak is %s.", outbreakStatusLabel(row.Status))
		}
		if row.ApprovedAt == nil {
			return nil, invalid("This outbreak has to be approved before it can be published.")
		}
		if row.VisualTone == "critical" && row.ApprovedBy != nil && *row.ApprovedBy == actor.ID {
			return nil, invalid("Critical outbreaks have to be published by someone other than the person who approved them.")
		}
		if err := s.validateOutbreakFields(row, true); err != nil {
			return nil, err
		}
		status := strings.TrimSpace(in.OperationalStatus)
		if status == "" {
			status = "active"
		}
		if !validOutbreakValue(status, "published", "active", "monitoring", "contained", "closed") {
			return nil, invalid("Operational status must be published, active, monitoring, contained or closed.")
		}
		updates["status"] = status
		updates["published_at"] = now
		updates["withdrawn_at"] = nil
		updates["withdrawal_reason"] = ""
		if outbreakPublishAnnounced(row, status) {
			alert = &outbreakTopicAlert{SourceType: "outbreak", SourceID: row.ID, OutbreakID: row.ID, Title: "Outbreak alert: " + row.Title, Body: row.Summary, Urgent: row.VisualTone == "critical"}
		}
	case "withdraw":
		if !publicOutbreakStatus(row.Status) {
			return nil, invalidf("Only a published outbreak can be withdrawn. This outbreak is %s.", outbreakStatusLabel(row.Status))
		}
		if strings.TrimSpace(in.Reason) == "" {
			return nil, invalid("A reason is required to withdraw an outbreak.")
		}
		updates["status"] = "withdrawn"
		updates["withdrawn_at"] = now
		updates["withdrawal_reason"] = strings.TrimSpace(in.Reason)
	case "update_status":
		if !publicOutbreakStatus(row.Status) {
			return nil, invalidf("Only a published outbreak's status can be changed. This outbreak is %s.", outbreakStatusLabel(row.Status))
		}
		if row.Status == "closed" {
			return nil, invalid("Closed outbreaks can't be moved to another status. Create a correction instead.")
		}
		target := strings.TrimSpace(in.OperationalStatus)
		if !validOutbreakValue(target, "published", "active", "monitoring", "contained", "closed") {
			return nil, invalid("Status must be published, active, monitoring, contained or closed.")
		}
		if target == row.Status {
			return nil, invalidf("This outbreak is already %s.", outbreakStatusLabel(target))
		}
		if strings.TrimSpace(in.Reason) == "" {
			return nil, invalid("A reason is required to change an outbreak's status.")
		}
		updates["status"] = target
	default:
		return nil, invalid("Unknown workflow action.")
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&models.Outbreak{}).Where("id = ? AND lock_version = ?", id, in.LockVersion).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		if err := auditOutbreak(tx, actor, "outbreak."+action, "outbreak", id, transitionAudit(in.Reason, row.Status, updates)); err != nil {
			return err
		}
		if alert != nil {
			return enqueueOutbreakTopicAlertTx(tx, *alert, now)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetOutbreak(id)
}
func (s OutbreakAdminService) CorrectOutbreak(actor OutbreakActor, id uuid.UUID, in TransitionInput) (*OutbreakAdminDTO, error) {
	var old models.Outbreak
	if err := s.DB.First(&old, "id = ? AND lock_version = ?", id, in.LockVersion).Error; err != nil {
		return nil, err
	}
	if !publicOutbreakStatus(old.Status) {
		return nil, invalidf("Only a published outbreak can be corrected. This outbreak is %s.", outbreakStatusLabel(old.Status))
	}
	if strings.TrimSpace(in.Reason) == "" {
		return nil, invalid("Give a reason for this correction.")
	}
	var open int64
	if err := s.DB.Model(&models.Outbreak{}).Where("supersedes_id = ? AND status IN ?", old.ID, []string{"draft", "pending_review"}).Count(&open).Error; err != nil {
		return nil, err
	}
	if open > 0 {
		return nil, invalid("A correction of this outbreak is already in progress. Open that correction to make further changes.")
	}
	copy := old
	copy.Base = models.Base{}
	copy.Status = "draft"
	copy.AuthorID = &actor.ID
	copy.PublishedAt = nil
	copy.ReviewedBy = nil
	copy.ReviewedAt = nil
	copy.ApprovedBy = nil
	copy.ApprovedAt = nil
	copy.WithdrawnAt = nil
	copy.WithdrawalReason = ""
	copy.SupersedesID = &old.ID
	copy.LockVersion = 1
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&copy).Error; err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "outbreak.correction_created", "outbreak", copy.ID, map[string]any{"supersedes_id": old.ID, "reason": in.Reason})
	})
	if err != nil {
		return nil, err
	}
	return s.GetOutbreak(copy.ID)
}

// applyCorrection writes an approved correction's core metadata and source
// onto the live outbreak and removes the correction. The live outbreak keeps
// its ID, status, metrics, children and publication history, so public links,
// alerts and hub mappings keep pointing at it. No alert is sent.
func (s OutbreakAdminService) applyCorrection(actor OutbreakActor, correction models.Outbreak, in TransitionInput) (*OutbreakAdminDTO, error) {
	var original models.Outbreak
	if err := s.DB.First(&original, "id = ?", *correction.SupersedesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalid("The outbreak this corrects no longer exists, so the correction can't be applied.")
		}
		return nil, err
	}
	if !publicOutbreakStatus(original.Status) {
		return nil, invalidf("This correction can't be applied because the outbreak it corrects is %s.", outbreakStatusLabel(original.Status))
	}
	corrected := original
	copyOutbreakCoreFields(&corrected, correction)
	if err := s.validateOutbreakFields(corrected, true); err != nil {
		return nil, err
	}
	changes := changedOutbreakCoreFields(original, corrected)
	reason := correctionReason(s.DB, "outbreak", correction.ID)
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Where("id = ? AND lock_version = ? AND status = ?", correction.ID, in.LockVersion, "pending_review").Delete(&models.Outbreak{})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		updates := outbreakCoreFields(corrected)
		updates["lock_version"] = gorm.Expr("lock_version + 1")
		// Only core fields are written, so metric or status changes made
		// while the correction was in review are kept.
		r = tx.Model(&models.Outbreak{}).Where("id = ? AND status IN ?", original.ID, []string{"published", "active", "monitoring", "contained", "closed"}).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		if err := auditOutbreak(tx, actor, "outbreak.approve", "outbreak", correction.ID, map[string]any{"applied_to": original.ID}); err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "outbreak.correction_applied", "outbreak", original.ID, map[string]any{"correction_id": correction.ID, "corrected_by": correction.AuthorID, "reason": reason, "changes": changes})
	}); err != nil {
		return nil, err
	}
	return s.GetOutbreak(original.ID)
}

func validateUpdateFields(row models.OutbreakUpdate) error {
	switch {
	case strings.TrimSpace(row.Title) == "":
		return invalid("Title is required.")
	case len(row.Title) > 240:
		return invalid("Title must be 240 characters or fewer.")
	case len(row.Summary) > 10_000:
		return invalid("Summary must be 10,000 characters or fewer.")
	}
	return nil
}

// outbreakStatusLabel renders a status the way the dashboard shows it.
func outbreakStatusLabel(status string) string {
	switch status {
	case "pending_review":
		return "in review"
	case "":
		return "in an unknown state"
	}
	return strings.ReplaceAll(status, "_", " ")
}

// childLabel renders a transitionChildWithHook "kind" the way an administrator
// would refer to it, e.g. "outbreak_update" -> "update".
func childLabel(kind string) string {
	return strings.ReplaceAll(strings.TrimPrefix(kind, "outbreak_"), "_", " ")
}
func (s OutbreakAdminService) ListUpdates(id uuid.UUID, p PageInput) (*PageResult[OutbreakUpdateAdminDTO], error) {
	return listAdminChildren(s.DB, id, p, func(r models.OutbreakUpdate) OutbreakUpdateAdminDTO { return updateAdminDTO(r) })
}
func (s OutbreakAdminService) CreateUpdate(actor OutbreakActor, id uuid.UUID, in ChildContentInput) (*OutbreakUpdateAdminDTO, error) {
	var parent models.Outbreak
	if err := s.DB.Select("id", "supersedes_id").First(&parent, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if parent.SupersedesID != nil {
		return nil, invalid("Updates are added to the live outbreak, not to its correction.")
	}
	row := models.OutbreakUpdate{OutbreakID: id, Status: "draft", AuthorID: &actor.ID, LockVersion: 1}
	applyUpdate(&row, in)
	if err := validateUpdateFields(row); err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "outbreak_update.created", "outbreak_update", row.ID, map[string]any{"outbreak_id": id})
	}); err != nil {
		return nil, err
	}
	v := updateAdminDTO(row)
	return &v, nil
}
func (s OutbreakAdminService) GetUpdate(id, child uuid.UUID) (*OutbreakUpdateAdminDTO, error) {
	var row models.OutbreakUpdate
	if err := s.DB.First(&row, "id = ? AND outbreak_id = ?", child, id).Error; err != nil {
		return nil, err
	}
	v := updateAdminDTO(row)
	return &v, nil
}
func (s OutbreakAdminService) UpdateUpdate(actor OutbreakActor, id, child uuid.UUID, in ChildContentInput) (*OutbreakUpdateAdminDTO, error) {
	if in.LockVersion == nil {
		return nil, ErrOutbreakInvalid
	}
	var row models.OutbreakUpdate
	if err := s.DB.First(&row, "id = ? AND outbreak_id = ?", child, id).Error; err != nil {
		return nil, err
	}
	if row.Status == "published" || row.Status == "withdrawn" {
		return nil, ErrOutbreakImmutable
	}
	applyUpdate(&row, in)
	if err := validateUpdateFields(row); err != nil {
		return nil, err
	}
	updates := map[string]any{"title": row.Title, "summary": row.Summary, "lock_version": gorm.Expr("lock_version + 1")}
	if row.Status == "pending_review" {
		// Changed content needs a fresh approval. Edits to a published update
		// stay in review; any other update goes back to draft.
		updates["reviewed_by"], updates["reviewed_at"], updates["approved_by"], updates["approved_at"] = nil, nil, nil, nil
		if row.SupersedesID == nil {
			updates["status"] = "draft"
		}
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&models.OutbreakUpdate{}).Where("id = ? AND outbreak_id = ? AND lock_version = ?", child, id, *in.LockVersion).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "outbreak_update.updated", "outbreak_update", child, nil)
	}); err != nil {
		return nil, err
	}
	return s.GetUpdate(id, child)
}
func (s OutbreakAdminService) DeleteUpdate(actor OutbreakActor, id, child uuid.UUID, lock int) error {
	return s.deleteChild(actor, id, child, lock, "outbreak_update", &models.OutbreakUpdate{})
}
func (s OutbreakAdminService) TransitionUpdate(actor OutbreakActor, id, child uuid.UUID, action string, in TransitionInput) (*OutbreakUpdateAdminDTO, error) {
	var row models.OutbreakUpdate
	if err := s.DB.First(&row, "id = ? AND outbreak_id = ?", child, id).Error; err != nil {
		return nil, err
	}
	if err := validateUpdateFields(row); err != nil {
		return nil, err
	}
	var hook func(*gorm.DB) error
	if action == "publish" {
		// A published edit replaces the original and is announced as a change.
		body := row.Title
		if row.SupersedesID != nil {
			body = "Updated: " + row.Title
		}
		hook = func(tx *gorm.DB) error {
			if row.SupersedesID != nil {
				if err := retireSuperseded(tx, actor, "outbreak_update", &models.OutbreakUpdate{}, id, child, *row.SupersedesID); err != nil {
					return err
				}
			}
			return enqueueOutbreakChildAlertTx(tx, "outbreak_update", row.ID, id, body)
		}
	}
	if err := s.transitionChildWithHook(actor, id, child, action, in, "outbreak_update", &models.OutbreakUpdate{}, hook); err != nil {
		return nil, err
	}
	return s.GetUpdate(id, child)
}
func (s OutbreakAdminService) CorrectUpdate(actor OutbreakActor, id, child uuid.UUID, in TransitionInput) (*OutbreakUpdateAdminDTO, error) {
	return s.correctUpdate(actor, id, child, in, nil, false)
}

// EditPublishedUpdate stores the edits as a correction that is submitted for
// review straight away; the published original stays live until it is published.
func (s OutbreakAdminService) EditPublishedUpdate(actor OutbreakActor, id, child uuid.UUID, in ResourceCorrectionInput) (*OutbreakUpdateAdminDTO, error) {
	if in.Changes == nil {
		return nil, invalid("Include the changes to make to this update.")
	}
	changes := *in.Changes
	return s.correctUpdate(actor, id, child, TransitionInput{LockVersion: in.LockVersion, Reason: in.Reason}, func(copy *models.OutbreakUpdate) error {
		applyUpdate(copy, changes)
		return validateUpdateFields(*copy)
	}, true)
}

// correctUpdate copies a published update into a correction. When prepare is
// given it applies the editor's changes (only one such edit may be open at a
// time); submit sends the correction straight to review.
func (s OutbreakAdminService) correctUpdate(actor OutbreakActor, id, child uuid.UUID, in TransitionInput, prepare func(*models.OutbreakUpdate) error, submit bool) (*OutbreakUpdateAdminDTO, error) {
	var old models.OutbreakUpdate
	if err := s.DB.First(&old, "id = ? AND outbreak_id = ? AND lock_version = ?", child, id, in.LockVersion).Error; err != nil {
		return nil, err
	}
	if old.Status != "published" || strings.TrimSpace(in.Reason) == "" {
		if prepare != nil && old.Status == "published" {
			return nil, invalid("Give a reason for this change.")
		}
		return nil, ErrOutbreakInvalid
	}
	copy := old
	copy.Base = models.Base{}
	copy.Status = "draft"
	copy.PublishedAt = nil
	copy.AuthorID = &actor.ID
	copy.ReviewedBy, copy.ReviewedAt, copy.ApprovedBy, copy.ApprovedAt, copy.WithdrawnAt = nil, nil, nil, nil, nil
	copy.WithdrawalReason = ""
	copy.SupersedesID = &old.ID
	copy.LockVersion = 1
	if prepare != nil {
		var open int64
		if err := s.DB.Model(&models.OutbreakUpdate{}).Where("outbreak_id = ? AND supersedes_id = ? AND status IN ?", id, old.ID, []string{"draft", "pending_review"}).Count(&open).Error; err != nil {
			return nil, err
		}
		if open > 0 {
			return nil, invalid("An edit of this is already in progress. Open that edit to make further changes.")
		}
		if err := prepare(&copy); err != nil {
			return nil, err
		}
	}
	if submit {
		copy.Status = "pending_review"
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&copy).Error; err != nil {
			return err
		}
		if err := auditOutbreak(tx, actor, "outbreak_update.correction_created", "outbreak_update", copy.ID, map[string]any{"supersedes_id": old.ID, "reason": in.Reason}); err != nil {
			return err
		}
		if submit {
			return auditOutbreak(tx, actor, "outbreak_update.submit", "outbreak_update", copy.ID, map[string]any{"reason": in.Reason})
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetUpdate(id, copy.ID)
}

func (s OutbreakAdminService) ListResources(id uuid.UUID, p PageInput) (*PageResult[OutbreakResourceAdminDTO], error) {
	p = p.Normalize(20, 100)
	var total int64
	q := s.DB.Model(&models.OutbreakResource{}).Where("outbreak_id = ?", id)
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []models.OutbreakResource
	if err := q.Order("sort_order,id").Offset(p.Offset()).Limit(p.PerPage).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]OutbreakResourceAdminDTO, len(rows))
	for i := range rows {
		items[i] = resourceAdminDTO(rows[i])
	}
	return NewPageResult(items, p, total), nil
}
func (s OutbreakAdminService) CreateResource(actor OutbreakActor, id uuid.UUID, in ChildContentInput) (*OutbreakResourceAdminDTO, error) {
	var parent models.Outbreak
	if err := s.DB.Select("id", "status", "supersedes_id").First(&parent, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if parent.SupersedesID != nil {
		return nil, invalid("Resources are added to the live outbreak, not to its correction.")
	}
	if parent.Status == "closed" || parent.Status == "withdrawn" {
		return nil, invalidf("Resources can't be added to an outbreak that is %s.", outbreakStatusLabel(parent.Status))
	}
	row := models.OutbreakResource{OutbreakID: id, Status: "draft", DocumentKind: "other", AuthorID: &actor.ID, LockVersion: 1}
	applyResource(&row, in)
	if err := s.validateResource(row); err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "outbreak_resource.created", "outbreak_resource", row.ID, map[string]any{"outbreak_id": id})
	}); err != nil {
		return nil, err
	}
	v := resourceAdminDTO(row)
	return &v, nil
}
func (s OutbreakAdminService) GetResource(id, child uuid.UUID) (*OutbreakResourceAdminDTO, error) {
	var row models.OutbreakResource
	if err := s.DB.First(&row, "id = ? AND outbreak_id = ?", child, id).Error; err != nil {
		return nil, err
	}
	v := resourceAdminDTO(row)
	return &v, nil
}
func (s OutbreakAdminService) UpdateResource(actor OutbreakActor, id, child uuid.UUID, in ChildContentInput) (*OutbreakResourceAdminDTO, error) {
	if in.LockVersion == nil {
		return nil, ErrOutbreakInvalid
	}
	var row models.OutbreakResource
	if err := s.DB.First(&row, "id = ? AND outbreak_id = ?", child, id).Error; err != nil {
		return nil, err
	}
	if row.Status == "published" || row.Status == "withdrawn" {
		return nil, ErrOutbreakImmutable
	}
	applyResource(&row, in)
	if err := s.validateResource(row); err != nil {
		return nil, err
	}
	updates := map[string]any{"title": row.Title, "description": row.Description, "issuing_authority": row.IssuingAuthority, "resource_type": row.ResourceType, "document_kind": row.DocumentKind, "url": row.URL, "asset_url": row.AssetURL, "sort_order": row.SortOrder, "lock_version": gorm.Expr("lock_version + 1")}
	if row.Status == "pending_review" {
		// Changed content needs a fresh approval. Edits to a published resource
		// stay in review; any other resource goes back to draft.
		updates["reviewed_by"], updates["reviewed_at"], updates["approved_by"], updates["approved_at"] = nil, nil, nil, nil
		if row.SupersedesID == nil {
			updates["status"] = "draft"
		}
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&models.OutbreakResource{}).Where("id = ? AND outbreak_id = ? AND lock_version = ?", child, id, *in.LockVersion).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "outbreak_resource.updated", "outbreak_resource", child, nil)
	}); err != nil {
		return nil, err
	}
	return s.GetResource(id, child)
}
func (s OutbreakAdminService) DeleteResource(actor OutbreakActor, id, child uuid.UUID, lock int) error {
	return s.deleteChild(actor, id, child, lock, "outbreak_resource", &models.OutbreakResource{})
}
func (s OutbreakAdminService) TransitionResource(actor OutbreakActor, id, child uuid.UUID, action string, in TransitionInput) (*OutbreakResourceAdminDTO, error) {
	var row models.OutbreakResource
	if err := s.DB.First(&row, "id = ? AND outbreak_id = ?", child, id).Error; err != nil {
		return nil, err
	}
	if err := s.validateResource(row); err != nil {
		return nil, err
	}
	var hook func(*gorm.DB) error
	if action == "publish" {
		body := "New resource: " + row.Title
		if row.SupersedesID != nil {
			body = "Updated resource: " + row.Title
		}
		hook = func(tx *gorm.DB) error {
			if row.SupersedesID != nil {
				if err := retireSuperseded(tx, actor, "outbreak_resource", &models.OutbreakResource{}, id, child, *row.SupersedesID); err != nil {
					return err
				}
			}
			return enqueueOutbreakChildAlertTx(tx, "outbreak_resource", row.ID, id, body)
		}
	}
	if err := s.transitionChildWithHook(actor, id, child, action, in, "outbreak_resource", &models.OutbreakResource{}, hook); err != nil {
		return nil, err
	}
	return s.GetResource(id, child)
}
func (s OutbreakAdminService) CorrectResource(actor OutbreakActor, id, child uuid.UUID, in TransitionInput) (*OutbreakResourceAdminDTO, error) {
	return s.correctResource(actor, id, child, in, nil, false)
}

// EditPublishedResource stores the edits as a correction that is submitted for
// review straight away; the published original stays live until it is published.
func (s OutbreakAdminService) EditPublishedResource(actor OutbreakActor, id, child uuid.UUID, in ResourceCorrectionInput) (*OutbreakResourceAdminDTO, error) {
	if in.Changes == nil {
		return nil, invalid("Include the changes to make to this resource.")
	}
	changes := *in.Changes
	return s.correctResource(actor, id, child, TransitionInput{LockVersion: in.LockVersion, Reason: in.Reason}, func(copy *models.OutbreakResource) error {
		applyResource(copy, changes)
		return s.validateResource(*copy)
	}, true)
}

// correctResource copies a published resource into a correction. When prepare is
// given it applies the editor's changes (only one such edit may be open at a
// time); submit sends the correction straight to review.
func (s OutbreakAdminService) correctResource(actor OutbreakActor, id, child uuid.UUID, in TransitionInput, prepare func(*models.OutbreakResource) error, submit bool) (*OutbreakResourceAdminDTO, error) {
	var old models.OutbreakResource
	if err := s.DB.First(&old, "id = ? AND outbreak_id = ? AND lock_version = ?", child, id, in.LockVersion).Error; err != nil {
		return nil, err
	}
	if old.Status != "published" || strings.TrimSpace(in.Reason) == "" {
		if prepare != nil && old.Status == "published" {
			return nil, invalid("Give a reason for this change.")
		}
		return nil, ErrOutbreakInvalid
	}
	copy := old
	copy.Base = models.Base{}
	copy.Status = "draft"
	copy.PublishedAt = nil
	copy.AuthorID = &actor.ID
	copy.ReviewedBy, copy.ReviewedAt, copy.ApprovedBy, copy.ApprovedAt, copy.WithdrawnAt = nil, nil, nil, nil, nil
	copy.WithdrawalReason = ""
	copy.SupersedesID = &old.ID
	copy.LockVersion = 1
	if prepare != nil {
		var open int64
		if err := s.DB.Model(&models.OutbreakResource{}).Where("outbreak_id = ? AND supersedes_id = ? AND status IN ?", id, old.ID, []string{"draft", "pending_review"}).Count(&open).Error; err != nil {
			return nil, err
		}
		if open > 0 {
			return nil, invalid("An edit of this is already in progress. Open that edit to make further changes.")
		}
		if err := prepare(&copy); err != nil {
			return nil, err
		}
	}
	if submit {
		copy.Status = "pending_review"
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&copy).Error; err != nil {
			return err
		}
		if err := auditOutbreak(tx, actor, "outbreak_resource.correction_created", "outbreak_resource", copy.ID, map[string]any{"supersedes_id": old.ID, "reason": in.Reason}); err != nil {
			return err
		}
		if submit {
			return auditOutbreak(tx, actor, "outbreak_resource.submit", "outbreak_resource", copy.ID, map[string]any{"reason": in.Reason})
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetResource(id, copy.ID)
}

func (s OutbreakAdminService) ListReports(q OutbreakAdminQuery, outbreakID string) (*PageResult[SituationReportAdminDTO], error) {
	p := q.Page.Normalize(20, 100)
	db := s.DB.Model(&models.SituationReport{})
	if v := strings.TrimSpace(outbreakID); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, ErrOutbreakInvalid
		}
		db = db.Where("outbreak_id = ?", id)
	}
	if v := strings.TrimSpace(q.Search); v != "" {
		like := "%" + strings.ToLower(v) + "%"
		db = db.Where("lower(title) LIKE ? OR lower(summary) LIKE ? OR lower(geographic_area) LIKE ?", like, like, like)
	}
	if q.Status != "" {
		if !validOutbreakValue(q.Status, "draft", "pending_review", "published", "archived", "withdrawn") {
			return nil, ErrOutbreakInvalid
		}
		db = db.Where("status = ?", q.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	order, err := reportAdminOrder(q.Sort, q.Order)
	if err != nil {
		return nil, err
	}
	var rows []models.SituationReport
	if err := db.Order(order).Offset(p.Offset()).Limit(p.PerPage).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]SituationReportAdminDTO, len(rows))
	for i := range rows {
		items[i] = reportAdminDTO(rows[i])
	}
	return NewPageResult(items, p, total), nil
}
func (s OutbreakAdminService) GetReport(id uuid.UUID) (*SituationReportAdminDTO, error) {
	var row models.SituationReport
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	v := reportAdminDTO(row)
	var open models.SituationReport
	if err := s.DB.Select("id").Where("supersedes_id = ? AND status IN ?", id, []string{"draft", "pending_review"}).Order("created_at DESC").Limit(1).Find(&open).Error; err != nil {
		return nil, err
	}
	if open.ID != uuid.Nil {
		v.OpenCorrectionID = &open.ID
	}
	return &v, nil
}
func (s OutbreakAdminService) CreateReport(actor OutbreakActor, in SituationReportInput) (*SituationReportAdminDTO, error) {
	row := models.SituationReport{Status: "draft", AuthorID: &actor.ID, LockVersion: 1, Metrics: datatypes.JSON("[]"), KeyHighlights: datatypes.JSON("[]")}
	if err := applyReport(&row, in); err != nil {
		return nil, err
	}
	if err := s.validateDraftReport(row); err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "situation_report.created", "situation_report", row.ID, nil)
	}); err != nil {
		return nil, err
	}
	return s.GetReport(row.ID)
}
func (s OutbreakAdminService) UpdateReport(actor OutbreakActor, id uuid.UUID, in SituationReportInput) (*SituationReportAdminDTO, error) {
	if in.LockVersion == nil {
		return nil, ErrOutbreakInvalid
	}
	var row models.SituationReport
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if row.Status == "published" || row.Status == "withdrawn" {
		return nil, ErrOutbreakImmutable
	}
	before := row
	if err := applyReport(&row, in); err != nil {
		return nil, err
	}
	// The published report's place in outbreak hubs follows its outbreak, so a
	// correction can't move it.
	if row.SupersedesID != nil && (!sameUUID(row.OutbreakID, before.OutbreakID) || row.StandaloneAllowed != before.StandaloneAllowed) {
		return nil, invalidField("A correction keeps the report's related outbreak. To move the report, withdraw it and publish a new one.", "outbreak_id")
	}
	if err := s.validateDraftReport(row); err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		updates := reportContentFields(row)
		updates["lock_version"] = gorm.Expr("lock_version + 1")
		r := tx.Model(&models.SituationReport{}).Where("id = ? AND lock_version = ?", id, *in.LockVersion).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "situation_report.updated", "situation_report", id, nil)
	}); err != nil {
		return nil, err
	}
	return s.GetReport(id)
}
// UpdateReportMetrics saves a report's metrics on their own, as soon as one is
// added or removed. Unlike an outbreak's, they are part of what is reviewed and
// published, so a published report's metrics change only through a correction.
func (s OutbreakAdminService) UpdateReportMetrics(actor OutbreakActor, id uuid.UUID, in OutbreakMetricsInput) (*SituationReportAdminDTO, error) {
	if in.LockVersion == nil {
		return nil, ErrOutbreakInvalid
	}
	var row models.SituationReport
	if err := s.DB.Select("id", "status").First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if row.Status == "published" || row.Status == "withdrawn" {
		return nil, invalid("Metrics are locked once the report is published. Create a correction to change them.")
	}
	if err := validateOutbreakMetrics(in.Metrics); err != nil {
		return nil, err
	}
	encoded, err := encodeMetrics(in.Metrics)
	if err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&models.SituationReport{}).Where("id = ? AND lock_version = ? AND status IN ?", id, *in.LockVersion, []string{"draft", "pending_review"}).Updates(map[string]any{"metrics": datatypes.JSON(encoded), "lock_version": gorm.Expr("lock_version + 1")})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "situation_report.metrics_updated", "situation_report", id, nil)
	}); err != nil {
		return nil, err
	}
	return s.GetReport(id)
}
func (s OutbreakAdminService) DeleteReport(actor OutbreakActor, id uuid.UUID, lock int) error {
	var row models.SituationReport
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return err
	}
	if row.Status == "published" || row.Status == "withdrawn" {
		return ErrOutbreakImmutable
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Where("id = ? AND lock_version = ?", id, lock).Delete(&models.SituationReport{})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "situation_report.deleted", "situation_report", id, nil)
	})
}
func (s OutbreakAdminService) TransitionReport(actor OutbreakActor, id uuid.UUID, action string, in TransitionInput) (*SituationReportAdminDTO, error) {
	var row models.SituationReport
	if err := s.DB.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if row.LockVersion != in.LockVersion {
		return nil, ErrOutbreakConflict
	}
	now := time.Now()
	updates := map[string]any{"lock_version": gorm.Expr("lock_version + 1")}
	switch action {
	case "submit":
		if row.Status != "draft" {
			return nil, invalidf("Only a draft report can be submitted for review. This report is %s.", outbreakStatusLabel(row.Status))
		}
		if err := s.validateDraftReport(row); err != nil {
			return nil, err
		}
		updates["status"] = "pending_review"
		updates["submitted_by"] = actor.ID
		updates["submitted_at"] = now
	case "approve":
		if row.Status != "pending_review" {
			return nil, invalidf("Only a submitted report can be approved. This report is %s.", outbreakStatusLabel(row.Status))
		}
		if row.ApprovedAt != nil {
			return nil, invalid("This report is already approved and is waiting to be published.")
		}
		// Approval is independent of both who wrote the report and who put it
		// forward for review. Publishing is open to anyone allowed to publish.
		if row.AuthorID != nil && *row.AuthorID == actor.ID {
			return nil, invalid("You created this report, so a different reviewer has to approve it.")
		}
		if row.SubmittedBy != nil && *row.SubmittedBy == actor.ID {
			return nil, invalid("You submitted this report for review, so a different reviewer has to approve it.")
		}
		if row.SupersedesID != nil {
			return s.applyReportCorrection(actor, row, in)
		}
		updates["reviewed_by"] = actor.ID
		updates["reviewed_at"] = now
		updates["approved_by"] = actor.ID
		updates["approved_at"] = now
	case "publish":
		if row.SupersedesID != nil {
			return nil, invalid("A correction isn't published on its own. Approving it applies the changes to the published report.")
		}
		if row.Status != "pending_review" {
			return nil, invalidf("Only a submitted report can be published. This report is %s.", outbreakStatusLabel(row.Status))
		}
		if row.ApprovedAt == nil {
			return nil, invalid("This report has to be approved before it can be published.")
		}
		if row.ApprovedBy != nil && *row.ApprovedBy == actor.ID {
			var parent models.Outbreak
			if row.OutbreakID != nil && s.DB.Select("visual_tone").First(&parent, "id = ?", *row.OutbreakID).Error == nil && parent.VisualTone == "critical" {
				return nil, invalid("Critical outbreak reports have to be published by someone other than the person who approved them.")
			}
		}
		if err := s.validatePublishReport(row); err != nil {
			return nil, err
		}
		updates["status"] = "published"
		updates["published_at"] = now
	case "withdraw":
		if row.Status != "published" {
			return nil, invalidf("Only a published report can be withdrawn. This report is %s.", outbreakStatusLabel(row.Status))
		}
		if strings.TrimSpace(in.Reason) == "" {
			return nil, invalid("A reason is required to withdraw this report.")
		}
		updates["status"] = "withdrawn"
		updates["withdrawn_at"] = now
		updates["withdrawal_reason"] = strings.TrimSpace(in.Reason)
	default:
		return nil, invalidf("%q is not a supported action for a report.", action)
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&models.SituationReport{}).Where("id = ? AND lock_version = ?", id, in.LockVersion).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		if err := auditOutbreak(tx, actor, "situation_report."+action, "situation_report", id, transitionAudit(in.Reason, row.Status, updates)); err != nil {
			return err
		}
		if action == "publish" {
			return syncReportIntoOutbreakHubs(tx, actor, row)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetReport(id)
}
func (s OutbreakAdminService) CorrectReport(actor OutbreakActor, id uuid.UUID, in TransitionInput) (*SituationReportAdminDTO, error) {
	var old models.SituationReport
	if err := s.DB.First(&old, "id = ? AND lock_version = ?", id, in.LockVersion).Error; err != nil {
		return nil, err
	}
	if old.Status != "published" {
		return nil, invalidf("Only a published report can be corrected. This report is %s.", outbreakStatusLabel(old.Status))
	}
	if strings.TrimSpace(in.Reason) == "" {
		return nil, invalid("Give a reason for this correction.")
	}
	var open int64
	if err := s.DB.Model(&models.SituationReport{}).Where("supersedes_id = ? AND status IN ?", old.ID, []string{"draft", "pending_review"}).Count(&open).Error; err != nil {
		return nil, err
	}
	if open > 0 {
		return nil, invalid("A correction of this report is already in progress. Open that correction to make further changes.")
	}
	copy := old
	copy.Base = models.Base{}
	copy.Status = "draft"
	copy.AuthorID = &actor.ID
	copy.PublishedAt = nil
	copy.ReviewedBy = nil
	copy.ReviewedAt = nil
	copy.ApprovedBy = nil
	copy.ApprovedAt = nil
	copy.WithdrawnAt = nil
	copy.WithdrawalReason = ""
	copy.SubmittedBy, copy.SubmittedAt = nil, nil
	copy.CorrectionReason = strings.TrimSpace(in.Reason)
	copy.SupersedesID = &old.ID
	copy.LockVersion = 1
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&copy).Error; err != nil {
			return err
		}
		// The correction starts with the same attachments, to change or keep.
		var attachments []models.SituationReportAttachment
		if err := tx.Where("situation_report_id = ?", old.ID).Find(&attachments).Error; err != nil {
			return err
		}
		for _, attachment := range attachments {
			attachment.Base = models.Base{}
			attachment.SituationReportID = copy.ID
			attachment.LockVersion = 1
			if err := tx.Create(&attachment).Error; err != nil {
				return err
			}
		}
		return auditOutbreak(tx, actor, "situation_report.correction_created", "situation_report", copy.ID, map[string]any{"supersedes_id": old.ID, "reason": in.Reason})
	}); err != nil {
		return nil, err
	}
	return s.GetReport(copy.ID)
}

// applyReportCorrection replaces the published report's content and
// attachments with an approved correction's, then removes the correction. The
// report keeps its ID, status, publication date and hub items, so links to it
// keep working and nothing is announced again.
func (s OutbreakAdminService) applyReportCorrection(actor OutbreakActor, correction models.SituationReport, in TransitionInput) (*SituationReportAdminDTO, error) {
	var original models.SituationReport
	if err := s.DB.First(&original, "id = ?", *correction.SupersedesID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invalid("The report this corrects no longer exists, so the correction can't be applied.")
		}
		return nil, err
	}
	if original.Status != "published" {
		return nil, invalidf("This correction can't be applied because the report it corrects is %s.", outbreakStatusLabel(original.Status))
	}
	corrected := original
	copyReportContent(&corrected, correction)
	if err := s.validateReportForPublishing(corrected, correction.ID); err != nil {
		return nil, err
	}
	changes := changedFields(reportContentFields(original), reportContentFields(corrected))
	var was, now []models.SituationReportAttachment
	if err := s.DB.Where("situation_report_id = ?", original.ID).Order("sort_order, id").Find(&was).Error; err != nil {
		return nil, err
	}
	if err := s.DB.Where("situation_report_id = ?", correction.ID).Order("sort_order, id").Find(&now).Error; err != nil {
		return nil, err
	}
	if attached := func(rows []models.SituationReportAttachment) []string {
		titles := make([]string, len(rows))
		for i, row := range rows {
			titles[i] = row.Title + " (" + row.URL + ")"
		}
		return titles
	}; !slicesEqual(attached(was), attached(now)) {
		changes["attachments"] = map[string]any{"from": attached(was), "to": attached(now)}
	}
	reason := correctionReason(s.DB, "situation_report", correction.ID)
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Where("id = ? AND lock_version = ? AND status = ?", correction.ID, in.LockVersion, "pending_review").Delete(&models.SituationReport{})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		// The correction's attachments become the report's.
		if err := tx.Where("situation_report_id = ?", original.ID).Delete(&models.SituationReportAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SituationReportAttachment{}).Where("situation_report_id = ?", correction.ID).Update("situation_report_id", original.ID).Error; err != nil {
			return err
		}
		updates := reportContentFields(corrected)
		updates["lock_version"] = gorm.Expr("lock_version + 1")
		r = tx.Model(&models.SituationReport{}).Where("id = ? AND status = ?", original.ID, "published").Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		if err := auditOutbreak(tx, actor, "situation_report.approve", "situation_report", correction.ID, map[string]any{"applied_to": original.ID}); err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "situation_report.correction_applied", "situation_report", original.ID, map[string]any{"correction_id": correction.ID, "corrected_by": correction.AuthorID, "reason": reason, "changes": changes})
	}); err != nil {
		return nil, err
	}
	return s.GetReport(original.ID)
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SituationReportAttachmentInput picks a published guideline-library document
// for a report. URL is /public/guidelines/{id}; DocumentKind is its kind slug.
type SituationReportAttachmentInput struct {
	Title               *string `json:"title"`
	Description         *string `json:"description"`
	IssuingOrganization *string `json:"issuing_organization"`
	DocumentKind        *string `json:"document_kind"`
	URL                 *string `json:"url"`
	SortOrder           *int    `json:"sort_order"`
	LockVersion         *int    `json:"lock_version"`
}

type SituationReportAttachmentDTO struct {
	ID                  uuid.UUID `json:"id"`
	SituationReportID   uuid.UUID `json:"situation_report_id"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	IssuingOrganization string    `json:"issuing_organization"`
	DocumentKind        string    `json:"document_kind"`
	URL                 string    `json:"url"`
	SortOrder           int       `json:"sort_order"`
	LockVersion         int       `json:"lock_version"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func reportAttachmentDTO(r models.SituationReportAttachment) SituationReportAttachmentDTO {
	return SituationReportAttachmentDTO{ID: r.ID, SituationReportID: r.SituationReportID, Title: r.Title, Description: r.Description, IssuingOrganization: r.IssuingOrganization, DocumentKind: r.DocumentKind, URL: r.URL, SortOrder: r.SortOrder, LockVersion: r.LockVersion, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func applyReportAttachment(r *models.SituationReportAttachment, in SituationReportAttachmentInput) {
	if in.Title != nil {
		r.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		r.Description = strings.TrimSpace(*in.Description)
	}
	if in.IssuingOrganization != nil {
		r.IssuingOrganization = strings.TrimSpace(*in.IssuingOrganization)
	}
	if in.DocumentKind != nil {
		r.DocumentKind = strings.ToLower(strings.TrimSpace(*in.DocumentKind))
	}
	if in.URL != nil {
		r.URL = strings.TrimSpace(*in.URL)
	}
	if in.SortOrder != nil {
		r.SortOrder = *in.SortOrder
	}
}

// editableReport loads a report whose attachments can still change: they are
// reviewed and published with the report, so they lock when it is published.
func (s OutbreakAdminService) editableReport(id uuid.UUID) (*models.SituationReport, error) {
	var report models.SituationReport
	if err := s.DB.Select("id", "status").First(&report, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if report.Status == "published" || report.Status == "withdrawn" {
		return nil, invalid("Attachments are locked once the report is published. Create a correction to change them.")
	}
	return &report, nil
}

func (s OutbreakAdminService) validateReportAttachment(row models.SituationReportAttachment) error {
	if err := validateReportAttachmentFields(row); err != nil {
		return err
	}
	_, err := s.validatePublishedDocument(row.URL, row.DocumentKind)
	return err
}

func (s OutbreakAdminService) ListReportAttachments(reportID uuid.UUID) ([]SituationReportAttachmentDTO, error) {
	var count int64
	if err := s.DB.Model(&models.SituationReport{}).Where("id = ?", reportID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var rows []models.SituationReportAttachment
	if err := s.DB.Where("situation_report_id = ?", reportID).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]SituationReportAttachmentDTO, len(rows))
	for i := range rows {
		items[i] = reportAttachmentDTO(rows[i])
	}
	return items, nil
}

func (s OutbreakAdminService) CreateReportAttachment(actor OutbreakActor, reportID uuid.UUID, in SituationReportAttachmentInput) (*SituationReportAttachmentDTO, error) {
	if _, err := s.editableReport(reportID); err != nil {
		return nil, err
	}
	row := models.SituationReportAttachment{SituationReportID: reportID, LockVersion: 1}
	applyReportAttachment(&row, in)
	if in.SortOrder == nil {
		var last struct{ SortOrder int }
		s.DB.Model(&models.SituationReportAttachment{}).Select("COALESCE(MAX(sort_order), 0) AS sort_order").Where("situation_report_id = ?", reportID).Scan(&last)
		row.SortOrder = last.SortOrder + 1
	}
	if err := s.validateReportAttachment(row); err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return auditOutbreak(tx, actor, "situation_report.attachment_added", "situation_report", reportID, map[string]any{"attachment_id": row.ID, "title": row.Title, "url": row.URL})
	}); err != nil {
		return nil, err
	}
	v := reportAttachmentDTO(row)
	return &v, nil
}

func (s OutbreakAdminService) UpdateReportAttachment(actor OutbreakActor, reportID, attachmentID uuid.UUID, in SituationReportAttachmentInput) (*SituationReportAttachmentDTO, error) {
	if in.LockVersion == nil {
		return nil, ErrOutbreakInvalid
	}
	if _, err := s.editableReport(reportID); err != nil {
		return nil, err
	}
	var row models.SituationReportAttachment
	if err := s.DB.First(&row, "id = ? AND situation_report_id = ?", attachmentID, reportID).Error; err != nil {
		return nil, err
	}
	applyReportAttachment(&row, in)
	if err := s.validateReportAttachment(row); err != nil {
		return nil, err
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&models.SituationReportAttachment{}).Where("id = ? AND situation_report_id = ? AND lock_version = ?", attachmentID, reportID, *in.LockVersion).Updates(map[string]any{"title": row.Title, "description": row.Description, "issuing_organization": row.IssuingOrganization, "document_kind": row.DocumentKind, "url": row.URL, "sort_order": row.SortOrder, "lock_version": gorm.Expr("lock_version + 1")})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "situation_report.attachment_updated", "situation_report", reportID, map[string]any{"attachment_id": attachmentID, "title": row.Title, "url": row.URL})
	}); err != nil {
		return nil, err
	}
	if err := s.DB.First(&row, "id = ?", attachmentID).Error; err != nil {
		return nil, err
	}
	v := reportAttachmentDTO(row)
	return &v, nil
}

func (s OutbreakAdminService) DeleteReportAttachment(actor OutbreakActor, reportID, attachmentID uuid.UUID, lock int) error {
	if _, err := s.editableReport(reportID); err != nil {
		return err
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Where("id = ? AND situation_report_id = ? AND lock_version = ?", attachmentID, reportID, lock).Delete(&models.SituationReportAttachment{})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, actor, "situation_report.attachment_removed", "situation_report", reportID, map[string]any{"attachment_id": attachmentID})
	})
}

// Helpers intentionally keep persistence models internal to this package.
func applyOutbreak(r *models.Outbreak, in OutbreakInput) error {
	if in.Title != nil {
		r.Title = strings.TrimSpace(*in.Title)
	}
	if in.DiseaseID != nil {
		r.DiseaseID = in.DiseaseID
	}
	if in.GeographicArea != nil {
		r.GeographicArea = strings.TrimSpace(*in.GeographicArea)
	}
	if in.RegionID != nil {
		r.RegionID = in.RegionID
	}
	if in.DistrictID != nil {
		r.DistrictID = in.DistrictID
	}
	if in.Summary != nil {
		r.Summary = strings.TrimSpace(*in.Summary)
	}
	if in.StartDate != nil {
		r.StartDate = in.StartDate
	}
	if in.LastUpdate != nil {
		r.LastUpdate = *in.LastUpdate
	}
	if in.VisualTone != nil {
		r.VisualTone = strings.TrimSpace(*in.VisualTone)
	}
	if in.SourceOrganization != nil {
		r.SourceOrganization = strings.TrimSpace(*in.SourceOrganization)
	}
	if in.SourceURL != nil {
		r.SourceURL = strings.TrimSpace(*in.SourceURL)
	}
	if in.SourceReference != nil {
		r.SourceReference = strings.TrimSpace(*in.SourceReference)
	}
	if in.EffectiveAt != nil {
		r.EffectiveAt = in.EffectiveAt
	}
	if in.DataAsOf != nil {
		r.DataAsOf = in.DataAsOf
	}
	if in.LastVerifiedAt != nil {
		r.LastVerifiedAt = in.LastVerifiedAt
	}
	if in.Metrics != nil {
		value, err := encodeMetrics(*in.Metrics)
		if err != nil {
			return err
		}
		r.Metrics = datatypes.JSON(value)
	}
	return nil
}
func publicOutbreakStatus(v string) bool {
	return validOutbreakValue(v, "published", "active", "monitoring", "contained", "closed")
}
func outbreakAdminOrder(sort, order string) (string, error) {
	cols := map[string]string{"": "updated_at", "title": "title", "status": "status", "visual_tone": "visual_tone", "effective_at": "effective_at", "data_as_of": "data_as_of", "last_verified_at": "last_verified_at", "last_update": "last_update", "created_at": "created_at", "updated_at": "updated_at"}
	c, ok := cols[strings.TrimSpace(sort)]
	if !ok {
		return "", ErrOutbreakInvalid
	}
	d := strings.ToUpper(strings.TrimSpace(order))
	if d == "" {
		d = "DESC"
	}
	if d != "ASC" && d != "DESC" {
		return "", ErrOutbreakInvalid
	}
	return c + " " + d + ", id " + d, nil
}
func reportAdminOrder(sort, order string) (string, error) {
	cols := map[string]string{"": "updated_at", "title": "title", "status": "status", "publication_date": "publication_date", "created_at": "created_at", "updated_at": "updated_at"}
	c, ok := cols[strings.TrimSpace(sort)]
	if !ok {
		return "", ErrOutbreakInvalid
	}
	d := strings.ToUpper(strings.TrimSpace(order))
	if d == "" {
		d = "DESC"
	}
	if d != "ASC" && d != "DESC" {
		return "", ErrOutbreakInvalid
	}
	return c + " " + d + ", id " + d, nil
}
func outbreakAdminDTO(r models.Outbreak) OutbreakAdminDTO {
	return OutbreakAdminDTO{ID: r.ID, Title: r.Title, DiseaseID: r.DiseaseID, DiseaseName: r.DiseaseName, Status: r.Status, GeographicArea: r.GeographicArea, RegionID: r.RegionID, DistrictID: r.DistrictID, Summary: r.Summary, StartDate: r.StartDate, LastUpdate: r.LastUpdate, VisualTone: r.VisualTone, SourceOrganization: r.SourceOrganization, PublishedAt: r.PublishedAt, AuthorID: r.AuthorID, ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt, ApprovedBy: r.ApprovedBy, ApprovedAt: r.ApprovedAt, WithdrawnAt: r.WithdrawnAt, WithdrawalReason: r.WithdrawalReason, SupersedesID: r.SupersedesID, SourceURL: r.SourceURL, SourceReference: r.SourceReference, EffectiveAt: r.EffectiveAt, DataAsOf: r.DataAsOf, LastVerifiedAt: r.LastVerifiedAt, LockVersion: r.LockVersion, Metrics: decodeMetrics(r.Metrics), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

// outbreakCoreFields are the columns on the "Core metadata and source" form:
// what a draft save writes and what an approved correction replaces. Metrics,
// status and child content each have their own path and are never included.
func outbreakCoreFields(r models.Outbreak) map[string]any {
	return map[string]any{"title": r.Title, "disease_id": r.DiseaseID, "geographic_area": r.GeographicArea, "region_id": r.RegionID, "district_id": r.DistrictID, "summary": r.Summary, "start_date": r.StartDate, "last_update": r.LastUpdate, "visual_tone": r.VisualTone, "source_organization": r.SourceOrganization, "source_url": r.SourceURL, "source_reference": r.SourceReference, "effective_at": r.EffectiveAt, "data_as_of": r.DataAsOf, "last_verified_at": r.LastVerifiedAt}
}

func copyOutbreakCoreFields(dst *models.Outbreak, src models.Outbreak) {
	dst.Title, dst.DiseaseID, dst.GeographicArea, dst.RegionID, dst.DistrictID = src.Title, src.DiseaseID, src.GeographicArea, src.RegionID, src.DistrictID
	dst.Summary, dst.StartDate, dst.LastUpdate, dst.VisualTone = src.Summary, src.StartDate, src.LastUpdate, src.VisualTone
	dst.SourceOrganization, dst.SourceURL, dst.SourceReference = src.SourceOrganization, src.SourceURL, src.SourceReference
	dst.EffectiveAt, dst.DataAsOf, dst.LastVerifiedAt = src.EffectiveAt, src.DataAsOf, src.LastVerifiedAt
}

// changedOutbreakCoreFields lists each core field that differs between two
// versions as {"from": old, "to": new}, for the audit trail.
func changedOutbreakCoreFields(before, after models.Outbreak) map[string]any {
	return changedFields(outbreakCoreFields(before), outbreakCoreFields(after))
}

// changedFields lists each field whose value differs between two versions as
// {"from": old, "to": new}, for the audit trail.
func changedFields(was, now map[string]any) map[string]any {
	changes := map[string]any{}
	for field, value := range now {
		from, to := auditComparable(was[field]), auditComparable(value)
		a, _ := json.Marshal(from)
		b, _ := json.Marshal(to)
		if !bytes.Equal(a, b) {
			changes[field] = map[string]any{"from": from, "to": to}
		}
	}
	return changes
}

// auditComparable dereferences pointers and puts times in UTC, so two equal
// values compare equal however the database driver returned them.
func auditComparable(value any) any {
	switch v := value.(type) {
	case *uuid.UUID:
		if v == nil {
			return nil
		}
		return *v
	case *time.Time:
		if v == nil {
			return nil
		}
		return v.UTC()
	case time.Time:
		return v.UTC()
	}
	return value
}

// correctionReason is the reason given when a correction was created, which
// is recorded on its correction_created audit entry.
func correctionReason(db *gorm.DB, kind string, correctionID uuid.UUID) string {
	var audit models.AuditLog
	if err := db.Where("entity_type = ? AND entity_id = ? AND action = ?", kind, correctionID.String(), kind+".correction_created").Order("created_at DESC").First(&audit).Error; err != nil {
		return ""
	}
	var meta struct {
		Reason string `json:"reason"`
	}
	if json.Unmarshal([]byte(audit.MetadataJSON), &meta) != nil {
		return ""
	}
	return strings.TrimSpace(meta.Reason)
}
func updateAdminDTO(r models.OutbreakUpdate) OutbreakUpdateAdminDTO {
	return OutbreakUpdateAdminDTO{r.ID, r.OutbreakID, r.Title, r.Summary, r.Status, r.PublishedAt, r.AuthorID, r.ReviewedBy, r.ReviewedAt, r.ApprovedBy, r.ApprovedAt, r.WithdrawnAt, r.WithdrawalReason, r.SupersedesID, r.LockVersion, r.CreatedAt, r.UpdatedAt}
}
func resourceAdminDTO(r models.OutbreakResource) OutbreakResourceAdminDTO {
	return OutbreakResourceAdminDTO{ID: r.ID, OutbreakID: r.OutbreakID, Title: r.Title, Description: r.Description, IssuingOrganization: r.IssuingAuthority, ResourceType: r.ResourceType, DocumentKind: r.DocumentKind, URL: r.URL, AssetURL: r.AssetURL, SortOrder: r.SortOrder, Status: r.Status, PublishedAt: r.PublishedAt, AuthorID: r.AuthorID, ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt, ApprovedBy: r.ApprovedBy, ApprovedAt: r.ApprovedAt, WithdrawnAt: r.WithdrawnAt, WithdrawalReason: r.WithdrawalReason, SupersedesID: r.SupersedesID, LockVersion: r.LockVersion, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func sameUUID(a, b *uuid.UUID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// reportContentFields are a report's editorial columns: what a draft save
// writes and what an approved correction replaces on the published report.
func reportContentFields(r models.SituationReport) map[string]any {
	return map[string]any{"outbreak_id": r.OutbreakID, "region_id": r.RegionID, "district_id": r.DistrictID, "title": r.Title, "geographic_area": r.GeographicArea, "summary": r.Summary, "source_organization": r.SourceOrganization, "publication_date": r.PublicationDate, "standalone_allowed": r.StandaloneAllowed, "source_url": r.SourceURL, "source_reference": r.SourceReference, "effective_at": r.EffectiveAt, "data_as_of": r.DataAsOf, "last_verified_at": r.LastVerifiedAt, "key_highlights": r.KeyHighlights, "metrics": r.Metrics}
}

func copyReportContent(dst *models.SituationReport, src models.SituationReport) {
	dst.OutbreakID, dst.RegionID, dst.DistrictID, dst.StandaloneAllowed = src.OutbreakID, src.RegionID, src.DistrictID, src.StandaloneAllowed
	dst.Title, dst.GeographicArea, dst.Summary, dst.SourceOrganization = src.Title, src.GeographicArea, src.Summary, src.SourceOrganization
	dst.PublicationDate, dst.SourceURL, dst.SourceReference = src.PublicationDate, src.SourceURL, src.SourceReference
	dst.EffectiveAt, dst.DataAsOf, dst.LastVerifiedAt = src.EffectiveAt, src.DataAsOf, src.LastVerifiedAt
	dst.KeyHighlights, dst.Metrics = src.KeyHighlights, src.Metrics
}

func reportAdminDTO(r models.SituationReport) SituationReportAdminDTO {
	return SituationReportAdminDTO{ID: r.ID, OutbreakID: r.OutbreakID, RegionID: r.RegionID, DistrictID: r.DistrictID, Title: r.Title, GeographicArea: r.GeographicArea, Summary: r.Summary, SourceOrganization: r.SourceOrganization, PublicationDate: r.PublicationDate, Status: r.Status, ReportAssetURL: r.ReportAssetURL, ReportAssetID: r.ReportAssetID, StandaloneAllowed: r.StandaloneAllowed, AuthorID: r.AuthorID, PublishedAt: r.PublishedAt, SubmittedBy: r.SubmittedBy, SubmittedAt: r.SubmittedAt, ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt, ApprovedBy: r.ApprovedBy, ApprovedAt: r.ApprovedAt, WithdrawnAt: r.WithdrawnAt, WithdrawalReason: r.WithdrawalReason, CorrectionReason: r.CorrectionReason, SupersedesID: r.SupersedesID, SourceURL: r.SourceURL, SourceReference: r.SourceReference, EffectiveAt: r.EffectiveAt, DataAsOf: r.DataAsOf, LastVerifiedAt: r.LastVerifiedAt, LockVersion: r.LockVersion, KeyHighlights: decodeHighlights(r.KeyHighlights), Metrics: decodeMetrics(r.Metrics), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
func auditOutbreak(tx *gorm.DB, a OutbreakActor, action, kind string, id uuid.UUID, meta any) error {
	if meta == nil {
		meta = map[string]any{}
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return tx.Create(&models.AuditLog{ActorID: a.ID.String(), Action: action, EntityType: kind, EntityID: id.String(), MetadataJSON: string(b), IPAddress: a.IP}).Error
}
func (s OutbreakAdminService) ensureOutbreak(id uuid.UUID) error {
	var n int64
	if err := s.DB.Model(&models.Outbreak{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func applyUpdate(r *models.OutbreakUpdate, in ChildContentInput) {
	if in.Title != nil {
		r.Title = strings.TrimSpace(*in.Title)
	}
	if in.Summary != nil {
		r.Summary = strings.TrimSpace(*in.Summary)
	}
}
func applyResource(r *models.OutbreakResource, in ChildContentInput) {
	if in.Title != nil {
		r.Title = strings.TrimSpace(*in.Title)
	}
	if in.ResourceType != nil {
		r.ResourceType = strings.TrimSpace(*in.ResourceType)
	}
	if in.DocumentKind != nil {
		r.DocumentKind = strings.ToLower(strings.TrimSpace(*in.DocumentKind))
	}
	if in.Description != nil {
		r.Description = strings.TrimSpace(*in.Description)
	}
	if in.IssuingOrganization != nil {
		r.IssuingAuthority = strings.TrimSpace(*in.IssuingOrganization)
	}
	if in.URL != nil {
		r.URL = strings.TrimSpace(*in.URL)
	}
	if in.AssetURL != nil {
		r.AssetURL = strings.TrimSpace(*in.AssetURL)
	}
	if in.SortOrder != nil {
		r.SortOrder = *in.SortOrder
	}
}
func applyReport(r *models.SituationReport, in SituationReportInput) error {
	if in.OutbreakID != nil {
		r.OutbreakID = in.OutbreakID
	}
	if in.RegionID != nil {
		r.RegionID = in.RegionID
	}
	if in.DistrictID != nil {
		r.DistrictID = in.DistrictID
	}
	if in.Title != nil {
		r.Title = strings.TrimSpace(*in.Title)
	}
	if in.GeographicArea != nil {
		r.GeographicArea = strings.TrimSpace(*in.GeographicArea)
	}
	if in.Summary != nil {
		r.Summary = strings.TrimSpace(*in.Summary)
	}
	if in.SourceOrganization != nil {
		r.SourceOrganization = strings.TrimSpace(*in.SourceOrganization)
	}
	if in.PublicationDate != nil {
		r.PublicationDate = *in.PublicationDate
	}
	if in.StandaloneAllowed != nil {
		r.StandaloneAllowed = *in.StandaloneAllowed
	}
	if in.SourceURL != nil {
		r.SourceURL = strings.TrimSpace(*in.SourceURL)
	}
	if in.SourceReference != nil {
		r.SourceReference = strings.TrimSpace(*in.SourceReference)
	}
	if in.EffectiveAt != nil {
		r.EffectiveAt = in.EffectiveAt
	}
	if in.DataAsOf != nil {
		r.DataAsOf = in.DataAsOf
	}
	if in.LastVerifiedAt != nil {
		r.LastVerifiedAt = in.LastVerifiedAt
	}
	if in.KeyHighlights != nil {
		value, err := encodeHighlights(*in.KeyHighlights)
		if err != nil {
			return err
		}
		r.KeyHighlights = datatypes.JSON(value)
	}
	if in.Metrics != nil {
		value, err := encodeMetrics(*in.Metrics)
		if err != nil {
			return err
		}
		r.Metrics = datatypes.JSON(value)
	}
	return nil
}
func (s OutbreakAdminService) validateDraftReport(r models.SituationReport) error {
	return s.validateReportFields(r, false)
}
func (s OutbreakAdminService) validatePublishReport(r models.SituationReport) error {
	return s.validateReportForPublishing(r, r.ID)
}

// validateReportForPublishing checks r as it would be published, with the
// attachments of report attachmentsOf (a correction's, when applying one).
func (s OutbreakAdminService) validateReportForPublishing(r models.SituationReport, attachmentsOf uuid.UUID) error {
	if err := s.validateReportFields(r, true); err != nil {
		return err
	}
	if r.OutbreakID != nil {
		var o models.Outbreak
		if err := s.DB.First(&o, "id = ?", *r.OutbreakID).Error; err != nil {
			return err
		}
		if !publicOutbreakStatus(o.Status) || o.PublishedAt == nil || o.WithdrawnAt != nil {
			return invalidField(fmt.Sprintf("The related outbreak isn't published (it is %s). Publish it first, or choose another outbreak.", outbreakStatusLabel(o.Status)), "outbreak_id")
		}
	}
	var attachments []models.SituationReportAttachment
	if err := s.DB.Where("situation_report_id = ?", attachmentsOf).Order("sort_order, id").Find(&attachments).Error; err != nil {
		return err
	}
	// Reports from before attachments carry their own uploaded PDF instead.
	if len(attachments) == 0 && r.ReportAssetID == nil && strings.TrimSpace(r.ReportAssetURL) == "" {
		return invalid("Attach at least one document before publishing.")
	}
	for _, attachment := range attachments {
		if _, err := s.validatePublishedDocument(attachment.URL, attachment.DocumentKind); err != nil {
			var validation *OutbreakValidationError
			if errors.As(err, &validation) {
				return invalidf("The attachment %q no longer points to a published document. Edit or remove it.", attachment.Title)
			}
			return err
		}
	}
	return nil
}
func listAdminChildren[T any](db *gorm.DB, id uuid.UUID, p PageInput, mapRow func(models.OutbreakUpdate) T) (*PageResult[T], error) {
	p = p.Normalize(20, 100)
	q := db.Model(&models.OutbreakUpdate{}).Where("outbreak_id = ?", id)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []models.OutbreakUpdate
	if err := q.Order("created_at DESC,id DESC").Offset(p.Offset()).Limit(p.PerPage).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]T, len(rows))
	for i := range rows {
		items[i] = mapRow(rows[i])
	}
	return NewPageResult(items, p, total), nil
}
func (s OutbreakAdminService) deleteChild(a OutbreakActor, parent, id uuid.UUID, lock int, kind string, model any) error {
	var status string
	if err := s.DB.Model(model).Select("status").Where("id = ? AND outbreak_id = ?", id, parent).Scan(&status).Error; err != nil {
		return err
	}
	if status == "published" || status == "withdrawn" {
		return ErrOutbreakImmutable
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Where("id = ? AND outbreak_id = ? AND lock_version = ?", id, parent, lock).Delete(model)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		return auditOutbreak(tx, a, kind+".deleted", kind, id, nil)
	})
}
func (s OutbreakAdminService) transitionChild(a OutbreakActor, parent, id uuid.UUID, action string, in TransitionInput, kind string, model any) error {
	return s.transitionChildWithHook(a, parent, id, action, in, kind, model, nil)
}

func (s OutbreakAdminService) transitionChildWithHook(a OutbreakActor, parent, id uuid.UUID, action string, in TransitionInput, kind string, model any, hook func(*gorm.DB) error) error {
	var row struct {
		Status      string
		AuthorID    *uuid.UUID
		ApprovedAt  *time.Time
		LockVersion int
	}
	if err := s.DB.Model(model).Where("id = ? AND outbreak_id = ?", id, parent).First(&row).Error; err != nil {
		return err
	}
	if row.LockVersion != in.LockVersion {
		return ErrOutbreakConflict
	}
	now := time.Now()
	label := childLabel(kind)
	updates := map[string]any{"lock_version": gorm.Expr("lock_version + 1")}
	switch action {
	case "submit":
		if row.Status != "draft" {
			return invalidf("Only a draft %s can be submitted for review. This %s is %s.", label, label, outbreakStatusLabel(row.Status))
		}
		updates["status"] = "pending_review"
	case "approve":
		if row.Status != "pending_review" {
			return invalidf("Only a submitted %s can be approved. This %s is %s.", label, label, outbreakStatusLabel(row.Status))
		}
		if row.AuthorID != nil && *row.AuthorID == a.ID {
			return invalidf("You created this %s, so a different reviewer has to approve it.", label)
		}
		updates["reviewed_by"] = a.ID
		updates["reviewed_at"] = now
		updates["approved_by"] = a.ID
		updates["approved_at"] = now
	case "publish":
		if row.Status != "pending_review" {
			return invalidf("Only a submitted %s can be published. This %s is %s.", label, label, outbreakStatusLabel(row.Status))
		}
		if row.ApprovedAt == nil {
			return invalidf("This %s has to be approved before it can be published.", label)
		}
		if _, err := (OutbreakService{DB: s.DB}).Get(parent); err != nil {
			var outbreak models.Outbreak
			if lookupErr := s.DB.Select("status").First(&outbreak, "id = ?", parent).Error; lookupErr != nil {
				return invalidf("The outbreak this %s belongs to could not be found.", label)
			}
			return invalidf("This %s can't be published yet because its outbreak isn't published (the outbreak is %s). Publish the outbreak first, then publish this %s.", label, outbreakStatusLabel(outbreak.Status), label)
		}
		updates["status"] = "published"
		updates["published_at"] = now
	case "withdraw":
		if row.Status != "published" {
			return invalidf("Only a published %s can be withdrawn. This %s is %s.", label, label, outbreakStatusLabel(row.Status))
		}
		if strings.TrimSpace(in.Reason) == "" {
			return invalidf("A reason is required to withdraw this %s.", label)
		}
		updates["status"] = "withdrawn"
		updates["withdrawn_at"] = now
		updates["withdrawal_reason"] = strings.TrimSpace(in.Reason)
	default:
		return invalidf("%q is not a supported action for a %s.", action, label)
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(model).Where("id = ? AND outbreak_id = ? AND lock_version = ?", id, parent, in.LockVersion).Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return ErrOutbreakConflict
		}
		if err := auditOutbreak(tx, a, kind+"."+action, kind, id, map[string]any{"reason": in.Reason}); err != nil {
			return err
		}
		if hook != nil {
			return hook(tx)
		}
		return nil
	})
}

// retireSuperseded withdraws the published original of an outbreak child
// (kind "outbreak_resource" or "outbreak_update") once its correction is
// published, so only the corrected version stays public.
func retireSuperseded(tx *gorm.DB, actor OutbreakActor, kind string, model any, outbreakID, correctionID, originalID uuid.UUID) error {
	reason := "superseded by approved correction"
	if given := correctionReason(tx, kind, correctionID); given != "" {
		reason += ": " + given
	}
	r := tx.Model(model).Where("id = ? AND outbreak_id = ? AND status = ?", originalID, outbreakID, "published").Updates(map[string]any{"status": "withdrawn", "withdrawn_at": time.Now(), "withdrawal_reason": reason, "lock_version": gorm.Expr("lock_version + 1")})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return nil
	}
	return auditOutbreak(tx, actor, kind+".superseded", kind, originalID, map[string]any{"superseded_by": correctionID})
}
