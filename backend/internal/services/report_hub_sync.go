package services

import (
	"mediguide/internal/models"

	"gorm.io/gorm"
)

// situationReportsPillar is the hub section that lists an outbreak's reports
// (see ConfigureOutbreakHub).
const situationReportsPillar = "situation-reports"

// syncReportIntoOutbreakHubs adds a newly published report at the top of the
// "Situation reports" section of every hub assigned to its outbreak. Items
// carry no label or description of their own, so hubs show the report's
// current title and summary. Corrections keep the report's ID and outbreak
// (applyReportCorrection), so its hub items stay as they are.
func syncReportIntoOutbreakHubs(tx *gorm.DB, actor OutbreakActor, report models.SituationReport) error {
	if report.OutbreakID == nil {
		return nil
	}
	var pillars []models.ContentPillar
	if err := tx.Model(&models.ContentPillar{}).
		Joins("JOIN content_hub_outbreaks cho ON cho.content_hub_id = content_pillars.hub_id").
		Joins("JOIN content_hubs h ON h.id = content_pillars.hub_id AND h.deleted_at IS NULL").
		Where("cho.outbreak_id = ? AND content_pillars.slug = ?", *report.OutbreakID, situationReportsPillar).
		Find(&pillars).Error; err != nil {
		return err
	}
	for _, pillar := range pillars {
		var listed int64
		if err := tx.Model(&models.ContentPillarItem{}).Where("pillar_id = ? AND content_type = ? AND content_id = ?", pillar.ID, models.ContentDiseaseSituationReport, report.ID).Count(&listed).Error; err != nil {
			return err
		}
		if listed > 0 {
			continue
		}
		// Newest first: just above the current top item.
		var top struct{ SortOrder *int }
		if err := tx.Model(&models.ContentPillarItem{}).Select("MIN(sort_order) AS sort_order").Where("pillar_id = ?", pillar.ID).Scan(&top).Error; err != nil {
			return err
		}
		order := 0
		if top.SortOrder != nil && *top.SortOrder >= 10 {
			order = *top.SortOrder - 10
		}
		id := report.ID
		item := models.ContentPillarItem{PillarID: pillar.ID, ContentType: models.ContentDiseaseSituationReport, ContentID: &id, SortOrder: order, Status: models.ContentPillarItemStatusActive, CreatedBy: &actor.ID, LockVersion: 1}
		if err := tx.Create(&item).Error; err != nil {
			return mapContentHubConstraint(err)
		}
		if err := auditContentHub(tx, ContentHubActor(actor), "content_pillar_item.create", "content_pillar_item", item.ID, item); err != nil {
			return err
		}
	}
	return nil
}
