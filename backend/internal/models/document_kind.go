package models

// DocumentKind classifies guideline documents and outbreak documents, for
// example Guideline, Form or SOP. Guideline documents reference a kind by ID;
// outbreak documents reference it by Slug, which clients use as a stable code
// and which therefore never changes after creation.
type DocumentKind struct {
	Base
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	SortOrder   int     `json:"sort_order"`
	Status      string  `json:"status"`
	// PublishAsUploaded kinds (for example Form) keep the uploaded file as the
	// published document: no extraction into editable Markdown or blocks.
	PublishAsUploaded bool `gorm:"not null;default:false" json:"publish_as_uploaded"`
	// PublishAsLink kinds (for example Link) publish an external https URL
	// instead of an uploaded file or extracted content.
	PublishAsLink          bool  `gorm:"not null;default:false" json:"publish_as_link"`
	GuidelineDocumentCount int64 `gorm:"->;-:migration" json:"guideline_document_count"`
	OutbreakResourceCount  int64 `gorm:"->;-:migration" json:"outbreak_resource_count"`
}

func (DocumentKind) TableName() string { return "document_kinds" }
