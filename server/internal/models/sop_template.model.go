package models

import (
	"time"

	"github.com/google/uuid"
)

type SOPTemplate struct {
	ID               int        `gorm:"primaryKey;autoIncrement" json:"id"`
	SpaceID          *uuid.UUID `gorm:"type:uuid" json:"spaceId,omitempty"`
	SOPCategoryID    *uuid.UUID `gorm:"type:uuid" json:"sopCategoryId,omitempty"`
	Name             string     `gorm:"not null" json:"name"`
	CurrentVersionID *int       `json:"currentVersionId,omitempty"`
	CreatedBy        uuid.UUID  `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt        time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space          *Space       `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	SOPCategory    *SOPCategory `gorm:"foreignKey:SOPCategoryID" json:"sopCategory,omitempty"`
	CurrentVersion *SOPVersion  `gorm:"foreignKey:CurrentVersionID" json:"currentVersion,omitempty"`
	Versions       []SOPVersion `gorm:"foreignKey:SOPTemplateID" json:"versions,omitempty"`
	Creator        *User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Tags           []Tag        `gorm:"many2many:sop_template_tag;joinForeignKey:SOPTemplateID;joinReferences:TagID" json:"tags,omitempty"`
}

func (SOPTemplate) TableName() string {
	return "sop_template"
}
