package models

import (
	"time"

	"github.com/google/uuid"
)

type SOPTemplate struct {
	ID               int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"not null" json:"name"`
	CurrentVersionID *int      `json:"currentVersionId,omitempty"`
	CreatedBy        uuid.UUID `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt        time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"default:now()" json:"updatedAt"`

	// Relations
	CurrentVersion *SOPTemplateVersion  `gorm:"foreignKey:CurrentVersionID" json:"currentVersion,omitempty"`
	Versions       []SOPTemplateVersion `gorm:"foreignKey:SOPTemplateID" json:"versions,omitempty"`
	Creator        *User                `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (SOPTemplate) TableName() string {
	return "sop_template"
}
