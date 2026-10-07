package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"

	"mediguide/internal/models"

	"github.com/google/uuid"
)

const (
	maxOutbreakMetrics    = 40
	maxReportHighlights   = 20
	maxHighlightLength    = 500
	maxSourceReferenceLen = 500
)

// OutbreakValidationError carries a reason an administrator can act on. It
// unwraps to ErrOutbreakInvalid, so existing errors.Is checks keep working.
// Fields, when set, name the form fields to fix so they can be highlighted.
type OutbreakValidationError struct {
	Message string
	Fields  []OutbreakFieldError
}

// OutbreakFieldError ties a reason to one form field, named by its JSON key
// (for example "effective_at").
type OutbreakFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *OutbreakValidationError) Error() string { return e.Message }
func (e *OutbreakValidationError) Unwrap() error { return ErrOutbreakInvalid }

func invalid(message string) error { return &OutbreakValidationError{Message: message} }
func invalidf(format string, args ...any) error {
	return &OutbreakValidationError{Message: fmt.Sprintf(format, args...)}
}

// invalidField is invalid for a reason that concerns the given fields. A rule
// spanning several fields (such as two dates out of order) names them all.
func invalidField(message string, fields ...string) error {
	return withFields(invalid(message), fields...)
}

// withFields names the fields a validation error concerns, unless it already
// names some. Any other error is returned unchanged.
func withFields(err error, fields ...string) error {
	var validation *OutbreakValidationError
	if !errors.As(err, &validation) || len(validation.Fields) > 0 {
		return err
	}
	named := make([]OutbreakFieldError, len(fields))
	for i, field := range fields {
		named[i] = OutbreakFieldError{Field: field, Message: validation.Message}
	}
	return &OutbreakValidationError{Message: validation.Message, Fields: named}
}

var outbreakMetricKey = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// OutbreakMetric is the only supported outbreak/report metric transport shape.
// NumericValue is optional because some public-health metrics are categorical.
type OutbreakMetric struct {
	Key             string    `json:"key"`
	Label           string    `json:"label"`
	Value           string    `json:"value,omitempty"`
	NumericValue    *float64  `json:"numeric_value,omitempty"`
	Unit            string    `json:"unit,omitempty"`
	AsOf            time.Time `json:"as_of"`
	SourceReference string    `json:"source_reference"`
	SortOrder       int       `json:"sort_order"`
}

func validateOutbreakMetrics(metrics []OutbreakMetric) error {
	if len(metrics) > maxOutbreakMetrics {
		return invalidf("At most %d metrics are allowed.", maxOutbreakMetrics)
	}
	seen := make(map[string]struct{}, len(metrics))
	orders := make(map[int]struct{}, len(metrics))
	for i := range metrics {
		metric := &metrics[i]
		name := fmt.Sprintf("Metric %d", i+1)
		metric.Key = strings.ToLower(strings.TrimSpace(metric.Key))
		metric.Label = strings.TrimSpace(metric.Label)
		metric.Value = strings.TrimSpace(metric.Value)
		metric.Unit = strings.TrimSpace(metric.Unit)
		metric.SourceReference = strings.TrimSpace(metric.SourceReference)
		switch {
		case metric.Key == "":
			return invalid(name + ": key is required.")
		case !outbreakMetricKey.MatchString(metric.Key):
			return invalid(name + ": key must be lowercase letters, numbers or underscores, start with a letter and be 2–64 characters (for example confirmed_cases).")
		case metric.Label == "":
			return invalid(name + ": label is required.")
		case len(metric.Label) > 120:
			return invalid(name + ": label must be 120 characters or fewer.")
		case len(metric.Value) > 120:
			return invalid(name + ": value must be 120 characters or fewer.")
		case len(metric.Unit) > 40:
			return invalid(name + ": unit must be 40 characters or fewer.")
		case metric.AsOf.IsZero():
			return invalid(name + ": the 'as of' date is required.")
		case metric.AsOf.After(time.Now().UTC().Add(5 * time.Minute)):
			return invalid(name + ": the 'as of' date can't be in the future.")
		case metric.SourceReference == "":
			return invalid(name + ": source is required.")
		case len(metric.SourceReference) > maxSourceReferenceLen:
			return invalidf("%s: source must be %d characters or fewer.", name, maxSourceReferenceLen)
		case metric.SortOrder < 0 || metric.SortOrder > 10_000:
			return invalid(name + ": sort order must be between 0 and 10,000.")
		case metric.Value == "" && metric.NumericValue == nil:
			return invalid(name + ": a value is required.")
		case metric.NumericValue != nil && (math.IsNaN(*metric.NumericValue) || math.IsInf(*metric.NumericValue, 0)):
			return invalid(name + ": the numeric value must be a real number.")
		}
		if _, exists := seen[metric.Key]; exists {
			return invalidf("%s: the key %q is used by another metric. Keys must be unique.", name, metric.Key)
		}
		if _, exists := orders[metric.SortOrder]; exists {
			return invalid(name + ": its sort order is used by another metric.")
		}
		seen[metric.Key] = struct{}{}
		orders[metric.SortOrder] = struct{}{}
	}
	return nil
}

func validateHighlights(values []string) error {
	if len(values) > maxReportHighlights {
		return ErrOutbreakInvalid
	}
	seen := make(map[string]struct{}, len(values))
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
		key := strings.ToLower(values[i])
		if values[i] == "" || len(values[i]) > maxHighlightLength {
			return ErrOutbreakInvalid
		}
		if _, exists := seen[key]; exists {
			return ErrOutbreakInvalid
		}
		seen[key] = struct{}{}
	}
	return nil
}

func encodeMetrics(values []OutbreakMetric) ([]byte, error) {
	if values == nil {
		values = []OutbreakMetric{}
	}
	if err := validateOutbreakMetrics(values); err != nil {
		return nil, err
	}
	return json.Marshal(values)
}

func decodeMetrics(value []byte) []OutbreakMetric {
	result := []OutbreakMetric{}
	if len(value) == 0 || json.Unmarshal(value, &result) != nil {
		return []OutbreakMetric{}
	}
	return result
}

func encodeHighlights(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	if err := validateHighlights(values); err != nil {
		return nil, err
	}
	return json.Marshal(values)
}

func decodeHighlights(value []byte) []string {
	result := []string{}
	if len(value) == 0 || json.Unmarshal(value, &result) != nil {
		return []string{}
	}
	return result
}

func (s OutbreakAdminService) validateGeography(regionID, districtID *uuid.UUID) error {
	if districtID != nil {
		var district models.District
		if err := s.DB.Select("id", "region_id").First(&district, "id = ?", *districtID).Error; err != nil {
			return invalidField("The selected district no longer exists. Pick another district.", "district_id")
		}
		if regionID != nil && district.RegionID != *regionID {
			return invalidField("The selected district doesn't belong to the selected region.", "district_id", "region_id")
		}
		return nil
	}
	if regionID != nil {
		var count int64
		if err := s.DB.Model(&models.Region{}).Where("id = ?", *regionID).Count(&count).Error; err != nil || count != 1 {
			return invalidField("The selected region no longer exists. Pick another region.", "region_id")
		}
	}
	return nil
}

// validateSourceURL explains why a link was rejected. label names the field in
// the message, for example "Source URL".
func validateSourceURL(label, value string, allowedHosts []string) error {
	value = strings.TrimSpace(value)
	if value == "" || validApprovedHTTPSURL(value, allowedHosts, true) {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" || parsed.User != nil {
		return invalidf("%s must be a full link starting with https://.", label)
	}
	if len(allowedHosts) > 0 {
		host := strings.ToLower(parsed.Hostname())
		approved := false
		for _, allowed := range allowedHosts {
			if host == strings.ToLower(strings.TrimSpace(allowed)) {
				approved = true
				break
			}
		}
		if !approved {
			return invalidf("%s: %s is not an approved domain. Approved domains: %s.", label, host, strings.Join(allowedHosts, ", "))
		}
	}
	return invalid(label + " can't contain reserved query parameters.")
}

// validatePublishedDocument checks that rawURL (/public/guidelines/{id}) names
// a published guideline-library document of the given document kind, and
// returns the document's ID.
func (s OutbreakAdminService) validatePublishedDocument(rawURL, kindSlug string) (uuid.UUID, error) {
	documentID, ok := guidelineDocumentID(rawURL)
	if !ok {
		return uuid.Nil, invalid("Select a published document.")
	}
	var kinds []string
	if err := s.DB.Table("guideline_documents AS gd").
		Joins("JOIN guideline_versions AS gv ON gv.id = gd.current_version_id AND gv.document_id = gd.id").
		Joins("JOIN document_kinds AS dk ON dk.id = gd.document_kind_id").
		Where("gd.id = ? AND gd.deleted_at IS NULL AND gv.deleted_at IS NULL AND LOWER(gv.status) = ?", documentID, "published").
		Pluck("dk.slug", &kinds).Error; err != nil {
		return uuid.Nil, err
	}
	if len(kinds) == 0 {
		return uuid.Nil, invalid("Select a published document.")
	}
	if kinds[0] != kindSlug {
		return uuid.Nil, invalid("The selected document doesn't belong to the chosen document type.")
	}
	return documentID, nil
}

// guidelineDocumentID reads the document ID from a /public/guidelines/{id} link.
func guidelineDocumentID(rawURL string) (uuid.UUID, bool) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return uuid.Nil, false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "public" || parts[1] != "guidelines" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(parts[2])
	return id, err == nil
}

func (s OutbreakAdminService) validateResource(row models.OutbreakResource) error {
	switch {
	case strings.TrimSpace(row.Title) == "":
		return invalid("Title is required.")
	case len(row.Title) > 240:
		return invalid("Title must be 240 characters or fewer.")
	case len(row.Description) > 4000:
		return invalid("Description must be 4,000 characters or fewer.")
	case len(row.IssuingAuthority) > 240:
		return invalid("Issuing organization must be 240 characters or fewer.")
	case row.SortOrder < 0 || row.SortOrder > 10_000:
		return invalid("Sort order must be between 0 and 10,000.")
	}
	kind := strings.TrimSpace(row.ResourceType)
	switch kind {
	case "guideline":
		if _, err := s.validatePublishedDocument(row.URL, row.DocumentKind); err != nil {
			return err
		}
	case "situation_report":
		parsed, err := url.ParseRequestURI(strings.TrimSpace(row.URL))
		parts := []string{}
		if err == nil && !parsed.IsAbs() && parsed.Host == "" && parsed.RawQuery == "" && parsed.Fragment == "" {
			parts = strings.Split(strings.Trim(parsed.Path, "/"), "/")
		}
		if len(parts) != 2 || parts[0] != "situation-reports" {
			return invalid("Select a published situation report.")
		}
		if _, err := uuid.Parse(parts[1]); err != nil {
			return invalid("Select a published situation report.")
		}
	case "internal_route":
		if !validNotificationInternalRoute(strings.TrimSpace(row.URL)) {
			return invalid("Enter an allowed in-app route, for example /outbreak-hub. Other routes aren't permitted.")
		}
	case "approved_external_url", "official_statement", "official_update", "link":
		if err := validateSourceURL("Resource URL", row.URL, s.AllowedExternalHosts); err != nil {
			return err
		}
	default:
		return invalid("This resource type isn't supported.")
	}
	return nil
}

func (s OutbreakAdminService) validateOutbreakFields(row models.Outbreak, publishing bool) error {
	switch {
	case strings.TrimSpace(row.Title) == "":
		return invalidField("Title is required.", "title")
	case len(row.Title) > 240:
		return invalidField("Title must be 240 characters or fewer.", "title")
	case row.DiseaseID == nil:
		return invalidField("Disease is required.", "disease_id")
	case len(row.GeographicArea) > 240:
		return invalidField("Geographic coverage must be 240 characters or fewer.", "geographic_area")
	case len(row.Summary) > 10_000:
		return invalidField("Summary must be 10,000 characters or fewer.", "summary")
	case len(row.SourceOrganization) > 240:
		return invalidField("Source organization must be 240 characters or fewer.", "source_organization")
	case len(row.SourceReference) > maxSourceReferenceLen:
		return invalidField(fmt.Sprintf("Source reference must be %d characters or fewer.", maxSourceReferenceLen), "source_reference")
	case !validOutbreakValue(row.VisualTone, "info", "warning", "critical", "success", "neutral"):
		return invalidField("Visual tone must be neutral, info, warning, critical or success.", "visual_tone")
	}
	if err := s.validateGeography(row.RegionID, row.DistrictID); err != nil {
		return err
	}
	if err := validateSourceURL("Source URL", row.SourceURL, s.AllowedExternalHosts); err != nil {
		return withFields(err, "source_url")
	}
	if err := validateOutbreakMetrics(decodeMetrics(row.Metrics)); err != nil {
		return err
	}
	switch {
	case row.StartDate != nil && row.LastUpdate.Before(*row.StartDate):
		return invalidField("Last update can't be earlier than the start date.", "last_update", "start_date")
	case row.DataAsOf != nil && row.StartDate != nil && row.DataAsOf.Before(*row.StartDate):
		return invalidField("Data as of can't be earlier than the start date.", "data_as_of", "start_date")
	case row.EffectiveAt != nil && row.StartDate != nil && row.EffectiveAt.Before(*row.StartDate):
		return invalidField("Effective at can't be earlier than the start date.", "effective_at", "start_date")
	case row.LastVerifiedAt != nil && row.LastVerifiedAt.After(time.Now().UTC().Add(5*time.Minute)):
		return invalidField("Last verified can't be in the future.", "last_verified_at")
	case row.DataAsOf != nil && row.LastVerifiedAt != nil && row.LastVerifiedAt.Before(*row.DataAsOf):
		return invalidField("Last verified can't be earlier than Data as of.", "last_verified_at", "data_as_of")
	}
	if publishing {
		var missing []string
		var fields []OutbreakFieldError
		need := func(empty bool, field, label string) {
			if empty {
				missing = append(missing, label)
				fields = append(fields, OutbreakFieldError{Field: field, Message: label + " is needed to publish."})
			}
		}
		need(strings.TrimSpace(row.GeographicArea) == "", "geographic_area", "Geographic coverage")
		need(strings.TrimSpace(row.SourceOrganization) == "", "source_organization", "Source organization")
		need(strings.TrimSpace(row.SourceReference) == "", "source_reference", "Source reference")
		need(row.EffectiveAt == nil, "effective_at", "Effective at")
		need(row.LastVerifiedAt == nil, "last_verified_at", "Last verified")
		if len(missing) > 0 {
			return &OutbreakValidationError{Message: "Before publishing, fill in and save: " + strings.Join(missing, ", ") + ".", Fields: fields}
		}
	}
	return nil
}

func (s OutbreakAdminService) validateReportFields(row models.SituationReport, publishing bool) error {
	switch {
	case strings.TrimSpace(row.Title) == "":
		return invalidField("Title is required.", "title")
	case len(row.Title) > 240:
		return invalidField("Title must be 240 characters or fewer.", "title")
	case len(row.GeographicArea) > 240:
		return invalidField("Geographic area must be 240 characters or fewer.", "geographic_area")
	case len(row.Summary) > 10_000:
		return invalidField("Summary must be 10,000 characters or fewer.", "summary")
	case len(row.SourceOrganization) > 240:
		return invalidField("Source organization must be 240 characters or fewer.", "source_organization")
	case len(row.SourceReference) > maxSourceReferenceLen:
		return invalidField(fmt.Sprintf("Source reference must be %d characters or fewer.", maxSourceReferenceLen), "source_reference")
	}
	if err := s.validateGeography(row.RegionID, row.DistrictID); err != nil {
		return err
	}
	if err := validateSourceURL("Source URL", row.SourceURL, s.AllowedExternalHosts); err != nil {
		return withFields(err, "source_url")
	}
	if validateHighlights(decodeHighlights(row.KeyHighlights)) != nil {
		return invalidField(fmt.Sprintf("Highlights take one statement per line: at most %d lines of up to %d characters each, with no repeats.", maxReportHighlights, maxHighlightLength), "key_highlights")
	}
	if err := validateOutbreakMetrics(decodeMetrics(row.Metrics)); err != nil {
		return err
	}
	if row.OutbreakID == nil && !row.StandaloneAllowed {
		return invalidField("Choose the related outbreak, or tick \"Explicitly allow standalone report\".", "outbreak_id", "standalone_allowed")
	}
	if row.OutbreakID != nil {
		var count int64
		// Reports link to the live outbreak; a correction is removed once applied.
		if err := s.DB.Model(&models.Outbreak{}).Where("id = ? AND supersedes_id IS NULL", *row.OutbreakID).Count(&count).Error; err != nil || count != 1 {
			return invalidField("The selected outbreak is no longer available. Pick another one.", "outbreak_id")
		}
	}
	switch {
	case !row.PublicationDate.IsZero() && row.EffectiveAt != nil && row.PublicationDate.Before(*row.EffectiveAt):
		return invalidField("Publication date can't be earlier than Effective at.", "publication_date", "effective_at")
	case row.LastVerifiedAt != nil && row.LastVerifiedAt.After(time.Now().UTC().Add(5*time.Minute)):
		return invalidField("Last verified can't be in the future.", "last_verified_at")
	case row.DataAsOf != nil && row.LastVerifiedAt != nil && row.LastVerifiedAt.Before(*row.DataAsOf):
		return invalidField("Last verified can't be earlier than Data as of.", "last_verified_at", "data_as_of")
	}
	// Attachments are checked in validatePublishReport, which can read them.
	if publishing {
		if !row.PublicationDate.IsZero() && row.PublicationDate.After(time.Now().UTC().Add(24*time.Hour)) {
			return invalidField("Publication date can't be more than a day in the future.", "publication_date")
		}
		var missing []string
		var fields []OutbreakFieldError
		need := func(empty bool, field, label string) {
			if empty {
				missing = append(missing, label)
				fields = append(fields, OutbreakFieldError{Field: field, Message: label + " is needed to publish."})
			}
		}
		need(strings.TrimSpace(row.GeographicArea) == "", "geographic_area", "Geographic area")
		need(strings.TrimSpace(row.SourceOrganization) == "", "source_organization", "Source organization")
		need(strings.TrimSpace(row.SourceReference) == "", "source_reference", "Source reference")
		need(row.PublicationDate.IsZero(), "publication_date", "Publication date")
		need(row.EffectiveAt == nil, "effective_at", "Effective at")
		need(row.LastVerifiedAt == nil, "last_verified_at", "Last verified")
		if len(missing) > 0 {
			return &OutbreakValidationError{Message: "Before publishing, fill in and save: " + strings.Join(missing, ", ") + ".", Fields: fields}
		}
	}
	return nil
}

func validateReportAttachmentFields(row models.SituationReportAttachment) error {
	switch {
	case strings.TrimSpace(row.Title) == "":
		return invalid("Title is required.")
	case len(row.Title) > 240:
		return invalid("Title must be 240 characters or fewer.")
	case len(row.Description) > 4000:
		return invalid("Description must be 4,000 characters or fewer.")
	case len(row.IssuingOrganization) > 240:
		return invalid("Issuing organization must be 240 characters or fewer.")
	case row.SortOrder < 0 || row.SortOrder > 10_000:
		return invalid("Sort order must be between 0 and 10,000.")
	}
	return nil
}
