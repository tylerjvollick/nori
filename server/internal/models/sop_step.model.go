package models

import (
	"time"

	"github.com/google/uuid"
)

type SOPStep struct {
	ID                   int        `gorm:"primaryKey;autoIncrement" json:"id"`
	SOPVersionID         int        `gorm:"not null" json:"sopVersionId"`
	StationID            *uuid.UUID `gorm:"type:uuid" json:"stationId,omitempty"`
	LinkedSOPTemplateID  *int       `json:"linkedSopTemplateId,omitempty"`
	Order                string     `gorm:"not null;index" json:"order"`
	Title                string     `gorm:"not null" json:"title"`
	Instructions         *string    `json:"instructions,omitempty"`
	EstimatedTimeMinutes *int       `json:"estimatedTimeMinutes,omitempty"`
	ImageURL             *string    `json:"imageUrl,omitempty"`
	VideoURL             *string    `json:"videoUrl,omitempty"`
	RequiresApproval     bool       `gorm:"default:false" json:"requiresApproval"`
	CreatedAt            time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt            time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	SOPVersion        *SOPVersion  `gorm:"foreignKey:SOPVersionID" json:"sopVersion,omitempty"`
	LinkedSOPTemplate *SOPTemplate `gorm:"foreignKey:LinkedSOPTemplateID" json:"linkedSopTemplate,omitempty"`
	SubSteps          []SOPSubStep `gorm:"foreignKey:SOPStepID" json:"subSteps,omitempty"`
}

func (SOPStep) TableName() string {
	return "sop_step"
}
