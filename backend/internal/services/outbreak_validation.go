package services

import (
	"encoding/json"
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
type OutbreakValidationError struct{ Message string }

func (e *OutbreakValidationError) Error() string { return e.Message }
func (e *OutbreakValidationError) Unwrap() error { return ErrOutbreakInvalid }

func invalid(message string) error { return &OutbreakValidationError{Message: message} }
func invalidf(format string, args ...any) error {
	return &OutbreakValidationError{Message: fmt.Sprintf(format, args...)}
}

var (
	outbreakMetricKey        = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	managedOutbreakAssetPath = regexp.MustCompile(`^/api/public/situation-reports/[0-9a-fA-F-]{36}/asset$`)
)

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
			return invalid("The selected district no longer exists. Pick another district.")
		}
		if regionID != nil && district.RegionID != *regionID {
			return invalid("The selected district doesn't belong to the selected region.")
		}
		return nil
	}
	if regionID != nil {
		var count int64
		if err := s.DB.Model(&models.Region{}).Where("id = ?", *regionID).Count(&count).Error; err != nil || count != 1 {
			return invalid("The selected region no longer exists. Pick another region.")
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
		parsed, err := url.ParseRequestURI(strings.TrimSpace(row.URL))
		parts := []string{}
		if err == nil && !parsed.IsAbs() && parsed.Host == "" && parsed.RawQuery == "" && parsed.Fragment == "" {
			parts = strings.Split(strings.Trim(parsed.Path, "/"), "/")
		}
		if len(parts) != 3 || parts[0] != "public" || parts[1] != "guidelines" {
			return invalid("Select a published guideline.")
		}
		if _, err := uuid.Parse(parts[2]); err != nil {
			return invalid("Select a published guideline.")
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
	case "managed_document", "downloadable_asset":
		asset := strings.TrimSpace(row.AssetURL)
		if asset == "" {
			return invalid("An asset path or approved https:// link is required for this resource type.")
		}
		if strings.TrimSpace(row.URL) != "" {
			return invalid("This resource type uses an asset path, not a URL. Clear the URL field.")
		}
		if !managedOutbreakAssetPath.MatchString(asset) {
			if err := validateSourceURL("Asset path", asset, s.AllowedExternalHosts); err != nil {
				return err
			}
		}
	default:
		return invalid("This resource type isn't supported.")
	}
	return nil
}

func (s OutbreakAdminService) validateOutbreakFields(row models.Outbreak, publishing bool) error {
	switch {
	case strings.TrimSpace(row.Title) == "":
		return invalid("Title is required.")
	case len(row.Title) > 240:
		return invalid("Title must be 240 characters or fewer.")
	case row.DiseaseID == nil:
		return invalid("Disease is required.")
	case len(row.GeographicArea) > 240:
		return invalid("Geographic coverage must be 240 characters or fewer.")
	case len(row.Summary) > 10_000:
		return invalid("Summary must be 10,000 characters or fewer.")
	case len(row.SourceOrganization) > 240:
		return invalid("Source organization must be 240 characters or fewer.")
	case len(row.SourceReference) > maxSourceReferenceLen:
		return invalidf("Source reference must be %d characters or fewer.", maxSourceReferenceLen)
	case !validOutbreakValue(row.VisualTone, "info", "warning", "critical", "success", "neutral"):
		return invalid("Visual tone must be neutral, info, warning, critical or success.")
	}
	if err := s.validateGeography(row.RegionID, row.DistrictID); err != nil {
		return err
	}
	if err := validateSourceURL("Source URL", row.SourceURL, s.AllowedExternalHosts); err != nil {
		return err
	}
	if err := validateOutbreakMetrics(decodeMetrics(row.Metrics)); err != nil {
		return err
	}
	switch {
	case row.StartDate != nil && row.LastUpdate.Before(*row.StartDate):
		return invalid("Last update can't be earlier than the start date.")
	case row.DataAsOf != nil && row.StartDate != nil && row.DataAsOf.Before(*row.StartDate):
		return invalid("Data as of can't be earlier than the start date.")
	case row.EffectiveAt != nil && row.StartDate != nil && row.EffectiveAt.Before(*row.StartDate):
		return invalid("Effective at can't be earlier than the start date.")
	case row.LastVerifiedAt != nil && row.LastVerifiedAt.After(time.Now().UTC().Add(5*time.Minute)):
		return invalid("Last verified can't be in the future.")
	case row.DataAsOf != nil && row.LastVerifiedAt != nil && row.LastVerifiedAt.Before(*row.DataAsOf):
		return invalid("Last verified can't be earlier than Data as of.")
	}
	if publishing {
		var missing []string
		if strings.TrimSpace(row.GeographicArea) == "" {
			missing = append(missing, "Geographic coverage")
		}
		if strings.TrimSpace(row.SourceOrganization) == "" {
			missing = append(missing, "Source organization")
		}
		if strings.TrimSpace(row.SourceReference) == "" {
			missing = append(missing, "Source reference")
		}
		if row.EffectiveAt == nil {
			missing = append(missing, "Effective at")
		}
		if row.LastVerifiedAt == nil {
			missing = append(missing, "Last verified")
		}
		if len(missing) > 0 {
			return invalid("Before publishing, fill in and save: " + strings.Join(missing, ", ") + ".")
		}
	}
	return nil
}

func (s OutbreakAdminService) validateReportFields(row models.SituationReport, publishing bool) error {
	if strings.TrimSpace(row.Title) == "" || len(row.Title) > 240 || len(row.GeographicArea) > 240 || len(row.Summary) > 10_000 || len(row.SourceOrganization) > 240 || len(row.SourceReference) > maxSourceReferenceLen {
		return ErrOutbreakInvalid
	}
	if err := s.validateGeography(row.RegionID, row.DistrictID); err != nil {
		return err
	}
	if err := validateSourceURL("Source URL", row.SourceURL, s.AllowedExternalHosts); err != nil {
		return err
	}
	if err := validateHighlights(decodeHighlights(row.KeyHighlights)); err != nil {
		return err
	}
	if err := validateOutbreakMetrics(decodeMetrics(row.Metrics)); err != nil {
		return err
	}
	if row.OutbreakID == nil && !row.StandaloneAllowed {
		return ErrOutbreakInvalid
	}
	if row.OutbreakID != nil {
		var count int64
		if err := s.DB.Model(&models.Outbreak{}).Where("id = ?", *row.OutbreakID).Count(&count).Error; err != nil || count != 1 {
			return ErrOutbreakInvalid
		}
	}
	if !row.PublicationDate.IsZero() && row.EffectiveAt != nil && row.PublicationDate.Before(*row.EffectiveAt) || row.DataAsOf != nil && row.LastVerifiedAt != nil && row.LastVerifiedAt.Before(*row.DataAsOf) || row.LastVerifiedAt != nil && row.LastVerifiedAt.After(time.Now().UTC().Add(5*time.Minute)) {
		return ErrOutbreakInvalid
	}
	if publishing && (row.PublicationDate.IsZero() || row.PublicationDate.After(time.Now().UTC().Add(24*time.Hour)) || strings.TrimSpace(row.GeographicArea) == "" || strings.TrimSpace(row.SourceOrganization) == "" || strings.TrimSpace(row.SourceReference) == "" || row.EffectiveAt == nil || row.LastVerifiedAt == nil || row.ReportAssetID == nil && strings.TrimSpace(row.ReportAssetURL) == "") {
		return ErrOutbreakInvalid
	}
	return nil
}
