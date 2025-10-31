package models

import (
	"time"

	"github.com/google/uuid"
)

type SOPTemplateVersion struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	SOPTemplateID int       `gorm:"not null" json:"sopTemplateId"`
	VersionNumber int       `gorm:"not null" json:"versionNumber"`
	Description   *string   `json:"description,omitempty"`
	Materials     []string  `gorm:"type:text[]" json:"materials,omitempty"`
	Equipment     []string  `gorm:"type:text[]" json:"equipment,omitempty"`
	CreatedBy     uuid.UUID `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt     time.Time `gorm:"default:now()" json:"createdAt"`
	ChangeSummary *string   `json:"changeSummary,omitempty"`
	IsActive      bool      `gorm:"default:true" json:"isActive"`

	// Relations
	SOPTemplate *SOPTemplate `gorm:"foreignKey:SOPTemplateID" json:"sopTemplate,omitempty"`
	Steps       []SOPStep    `gorm:"foreignKey:SOPTemplateVersionID" json:"steps,omitempty"`
	Creator     *User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (SOPTemplateVersion) TableName() string {
	return "sop_template_version"
}
