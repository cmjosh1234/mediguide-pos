package services

import (
	"errors"
	"strings"

	"mediguide/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrToolCategoryInvalid  = errors.New("invalid tool category")
	ErrToolCategoryConflict = errors.New("a tool category with this slug already exists")
	ErrToolCategoryInUse    = errors.New("tool category is assigned to tools; reassign them before deleting it")
)

type ToolCategoryService struct {
	DB *gorm.DB
}

type ToolCategoryQuery struct {
	Page   PageInput
	Search string
	Status string
	Sort   string
	Order  string
}

type ToolCategoryInput struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	Status      *string `json:"status"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
}

func (s ToolCategoryService) base() *gorm.DB {
	return s.DB.Table("tool_categories tc").
		Select("tc.*, (SELECT COUNT(*) FROM calculators c WHERE c.category_id = tc.id AND c.deleted_at IS NULL) AS tool_count").
		Where("tc.deleted_at IS NULL")
}

func (s ToolCategoryService) List(in ToolCategoryQuery) (*PageResult[models.ToolCategory], error) {
	p := in.Page.Normalize(50, 100)
	q := s.base()
	if in.Status != "" {
		if !oneOf(in.Status, "active", "inactive") {
			return nil, ErrToolCategoryInvalid
		}
		q = q.Where("tc.status = ?", in.Status)
	}
	if search := strings.TrimSpace(in.Search); search != "" {
		like := "%" + search + "%"
		q = q.Where("LOWER(tc.name) LIKE LOWER(?) OR LOWER(tc.slug) LIKE LOWER(?) OR LOWER(COALESCE(tc.description,'')) LIKE LOWER(?)", like, like, like)
	}
	return pageHelp[models.ToolCategory](q, p, map[string]string{"name": "tc.name", "sort_order": "tc.sort_order", "created_at": "tc.created_at", "updated_at": "tc.updated_at"}, in.Sort, in.Order, "tc.sort_order ASC, tc.name ASC")
}

func (s ToolCategoryService) Get(id uuid.UUID) (*models.ToolCategory, error) {
	var item models.ToolCategory
	if err := s.base().Where("tc.id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s ToolCategoryService) Save(id *uuid.UUID, in ToolCategoryInput) (*models.ToolCategory, error) {
	item := models.ToolCategory{Status: "active"}
	if id != nil {
		existing, err := s.Get(*id)
		if err != nil {
			return nil, err
		}
		item = *existing
	}
	if in.Name != nil {
		item.Name = strings.TrimSpace(*in.Name)
	}
	if in.Slug != nil {
		item.Slug = strings.TrimSpace(*in.Slug)
	}
	if item.Slug == "" {
		item.Slug = taxonomySlugify(item.Name)
	}
	item.Description = mergeOptionalString(item.Description, in.Description)
	item.Color = mergeOptionalString(item.Color, in.Color)
	item.Icon = mergeOptionalString(item.Icon, in.Icon)
	if in.SortOrder != nil {
		item.SortOrder = *in.SortOrder
	}
	if in.Status != nil {
		item.Status = strings.TrimSpace(*in.Status)
	}
	if item.Name == "" || !validTaxonomySlug(item.Slug) || !oneOf(item.Status, "active", "inactive") || item.SortOrder < 0 {
		return nil, ErrToolCategoryInvalid
	}
	duplicate := s.DB.Model(&models.ToolCategory{}).Where("slug = ?", item.Slug)
	if id != nil {
		duplicate = duplicate.Where("id <> ?", *id)
	}
	var count int64
	if err := duplicate.Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrToolCategoryConflict
	}
	if err := s.DB.Save(&item).Error; err != nil {
		return nil, err
	}
	return s.Get(item.ID)
}

func (s ToolCategoryService) Delete(id uuid.UUID) error {
	var count int64
	if err := s.DB.Model(&models.Calculator{}).Where("category_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrToolCategoryInUse
	}
	return deleteExisting(s.DB, &models.ToolCategory{}, id)
}
