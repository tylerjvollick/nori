package models

import (
	"time"

	"github.com/google/uuid"
)

type SOPVersion struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	SOPTemplateID int       `gorm:"not null" json:"sopTemplateId"`
	VersionNumber int       `gorm:"not null" json:"versionNumber"`
	Description   *string   `json:"description,omitempty"`
	CreatedBy     uuid.UUID `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt     time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"default:now()" json:"updatedAt"`
	ChangeSummary *string   `json:"changeSummary,omitempty"`
	IsActive      bool      `gorm:"default:true" json:"isActive"`

	// Relations
	SOPTemplate *SOPTemplate `gorm:"foreignKey:SOPTemplateID" json:"sopTemplate,omitempty"`
	Steps       []SOPStep    `gorm:"foreignKey:SOPVersionID" json:"steps,omitempty"`
	BOMItems    []BOMItem    `gorm:"foreignKey:SOPVersionID" json:"bomItems,omitempty"`
	Creator     *User        `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (SOPVersion) TableName() string {
	return "sop_version"
}
