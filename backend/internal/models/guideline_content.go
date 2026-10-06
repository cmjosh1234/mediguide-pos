package models

import "github.com/google/uuid"

type GuidelineCategory struct {
	Base
	ParentCategoryID *uuid.UUID `json:"parent_category_id,omitempty"`
	Name             string     `json:"name"`
	Slug             *string    `json:"slug,omitempty"`
	Description      *string    `json:"description,omitempty"`
	SortOrder        int        `json:"sort_order"`
	Status           string     `json:"status"`
	Color            *string    `json:"color,omitempty"`
	Icon             *string    `json:"icon,omitempty"`
	ParentName       string     `gorm:"->" json:"parent_name,omitempty"`
}

func (GuidelineCategory) TableName() string { return "guideline_categories" }

type GuidelineTag struct {
	Base
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func (GuidelineTag) TableName() string { return "guideline_tags" }

type Abbreviation struct {
	Base
	Abbreviation string     `json:"abbreviation"`
	Meaning      string     `json:"meaning"`
	Description  *string    `json:"description,omitempty"`
	CommonUsage  bool       `json:"common_usage"`
	Categories   StringList `gorm:"column:category_json;type:jsonb" json:"categories" swaggertype:"array,string"`
	Tags         StringList `gorm:"column:tags_json;type:jsonb" json:"tags" swaggertype:"array,string"`
	UsageCount   int64      `json:"usage_count"`
}

func (Abbreviation) TableName() string { return "abbreviations" }

type GuidelineIndexEntry struct {
	Base
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Title       string     `json:"title"`
	SortOrder   int        `json:"sort_order"`
	Description *string    `json:"description,omitempty"`
	Level       int        `json:"level"`
	HasChildren bool       `json:"has_children"`
	ParentTitle string     `gorm:"->" json:"parent_title,omitempty"`
}

func (GuidelineIndexEntry) TableName() string { return "guideline_index" }
