package models

import (
	"time"
)

type SOPStep struct {
	ID                   int       `gorm:"primaryKey;autoIncrement" json:"id"`
	SOPTemplateVersionID int       `gorm:"not null" json:"sopTemplateVersionId"`
	StepNumber           int       `gorm:"not null" json:"stepNumber"`
	Title                string    `gorm:"not null" json:"title"`
	Instructions         *string   `json:"instructions,omitempty"`
	EstimatedTimeMinutes *int      `json:"estimatedTimeMinutes,omitempty"`
	ImageURL             *string   `json:"imageUrl,omitempty"`
	VideoURL             *string   `json:"videoUrl,omitempty"`
	RequiresApproval     bool      `gorm:"default:false" json:"requiresApproval"`
	CreatedAt            time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt            time.Time `gorm:"default:now()" json:"updatedAt"`

	// Relations
	SOPTemplateVersion *SOPTemplateVersion `gorm:"foreignKey:SOPTemplateVersionID" json:"sopTemplateVersion,omitempty"`
}

func (SOPStep) TableName() string {
	return "sop_step"
}
