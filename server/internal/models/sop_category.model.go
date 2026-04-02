package models

import (
	"time"

	"github.com/google/uuid"
)

type SOPCategory struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID          uuid.UUID  `gorm:"type:uuid;not null" json:"spaceId"`
	ParentCategoryID *uuid.UUID `gorm:"type:uuid" json:"parentCategoryId,omitempty"`
	Name             string     `gorm:"not null" json:"name"`
	Description      *string    `json:"description,omitempty"`
	DisplayOrder     int        `gorm:"default:0" json:"displayOrder"`
	CreatedAt        time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space          *Space        `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	ParentCategory *SOPCategory  `gorm:"foreignKey:ParentCategoryID" json:"parentCategory,omitempty"`
	Children       []SOPCategory `gorm:"foreignKey:ParentCategoryID" json:"children,omitempty"`
	SOPTemplates   []SOPTemplate `gorm:"foreignKey:SOPCategoryID" json:"sopTemplates,omitempty"`
}

func (SOPCategory) TableName() string {
	return "sop_category"
}
