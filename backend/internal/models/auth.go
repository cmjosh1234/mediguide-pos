package models

import "encoding/json"

type User struct {
	Base
	Name              string     `gorm:"not null" json:"name"`
	Email             string     `gorm:"uniqueIndex;not null" json:"email"`
	Phone             string     `json:"phone"`
	AlternativePhone  *string    `json:"alternative_phone,omitempty"`
	PasswordHash      string     `gorm:"not null" json:"-"`
	FacilityID        *string    `json:"facility_id,omitempty"`
	IsActive          bool       `gorm:"default:true" json:"is_active"`
	Address           *string    `json:"address,omitempty"`
	City              *string    `json:"city,omitempty"`
	Country           *string    `json:"country,omitempty"`
	PostalCode        *string    `json:"postal_code,omitempty"`
	LicenseNumber     *string    `json:"license_number,omitempty"`
	Organization      *string    `json:"organization,omitempty"`
	Department        *string    `json:"department,omitempty"`
	JobTitle          *string    `json:"job_title,omitempty"`
	PreferredLanguage *string    `json:"preferred_language,omitempty"`
	Timezone          *string    `json:"timezone,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	Specialization    StringList `gorm:"type:jsonb;column:specialization_json" json:"specialization,omitempty" swaggertype:"array,string"`
	Avatar            *string    `json:"avatar,omitempty"`
	EmailVerified     bool       `gorm:"default:false" json:"email_verified"`
	Verified          bool       `gorm:"default:false" json:"verified"`
	Status            string     `json:"status"`
	Roles             []Role     `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

type Role struct {
	Base
	Name            string          `gorm:"uniqueIndex;not null" json:"name"`
	RoleKey         *string         `gorm:"column:role_key" json:"role_key,omitempty"`
	Description     string          `json:"description"`
	PermissionsJSON json.RawMessage `gorm:"type:jsonb;column:permissions_json" json:"permissions_json,omitempty" swaggertype:"object"`
	IsActive        bool            `gorm:"default:true;column:is_active" json:"is_active"`
	Permissions     []Permission    `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

type Permission struct {
	Base
	Code string `gorm:"uniqueIndex;not null" json:"code"`
	Name string `json:"name"`
}
