package services

import (
	"strings"
	"time"

	"mediguide/internal/models"

	"gorm.io/gorm"
)

func routineNotification(priority, category string) bool {
	return priority != "high" && priority != "urgent" && category != "emergency_alerts"
}

func nextNotificationDigest(at time.Time, settings models.NotificationPreferenceSettings) time.Time {
	location, err := time.LoadLocation(settings.QuietHoursTimezone)
	if err != nil {
		location = time.UTC
	}
	clock, err := time.Parse("15:04", settings.DigestTime)
	if err != nil {
		clock, _ = time.Parse("15:04", "18:00")
	}
	local := at.In(location)
	next := time.Date(local.Year(), local.Month(), local.Day(), clock.Hour(), clock.Minute(), 0, 0, location)
	if !next.After(local) {
		next = next.AddDate(0, 0, 1)
	}
	return afterNotificationQuietHours(next.UTC(), settings)
}

// Urgent alerts remain category/channel opt-ins, but are not restricted by a
// user's routine interests. National outbreak notices match every region.
func notificationMatchesInterests(db *gorm.DB, settings models.NotificationPreferenceSettings, action NotificationAction, priority string) (bool, error) {
	if priority == "urgent" {
		return true, nil
	}
	interests := decodeNotificationInterests(settings.InterestsJSON)
	if action.ResourceID == nil {
		return true, nil
	}
	switch action.Type {
	case NotificationActionGuideline:
		if containsNotificationString(interests.FollowedGuidelineIDs, *action.ResourceID) {
			return true, nil
		}
		if interests.FollowedOnly {
			return false, nil
		}
		if len(interests.ProgramAreas) > 0 {
			var document models.GuidelineDocument
			if err := db.Select("program_area").First(&document, "id = ?", *action.ResourceID).Error; err != nil {
				return false, err
			}
			return containsNotificationString(normalizedAudienceStrings(interests.ProgramAreas), strings.ToLower(strings.TrimSpace(document.ProgramArea))), nil
		}
	case NotificationActionOutbreak:
		if len(interests.RegionIDs) > 0 {
			var outbreak models.Outbreak
			if err := db.Select("region_id").First(&outbreak, "id = ?", *action.ResourceID).Error; err != nil {
				return false, err
			}
			return outbreak.RegionID == nil || containsNotificationString(interests.RegionIDs, outbreak.RegionID.String()), nil
		}
	case NotificationActionSituationReport:
		if len(interests.RegionIDs) > 0 {
			var report models.SituationReport
			if err := db.Select("region_id").First(&report, "id = ?", *action.ResourceID).Error; err != nil {
				return false, err
			}
			return report.RegionID == nil || containsNotificationString(interests.RegionIDs, report.RegionID.String()), nil
		}
	}
	return true, nil
}
