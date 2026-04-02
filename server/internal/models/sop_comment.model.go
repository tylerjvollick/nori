package models

import (
	"time"

	"github.com/google/uuid"
)

// SOPComment represents a comment or suggested edit on an SOP. Comments can
// target the overall SOP, a specific step, or a specific sub-step. The
// IsSuggestion flag indicates a proposed change (like GitHub suggested edits).
type SOPComment struct {
	ID            uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SOPTemplateID int       `gorm:"not null" json:"sopTemplateId"`
	SOPStepID     *int      `json:"sopStepId,omitempty"`
	SOPSubStepID  *int      `json:"sopSubStepId,omitempty"`
	UserID        uuid.UUID `gorm:"type:uuid;not null" json:"userId"`
	Body          string    `gorm:"type:text;not null" json:"body"`
	IsSuggestion  bool      `gorm:"default:false" json:"isSuggestion"`
	IsResolved    bool      `gorm:"default:false" json:"isResolved"`
	CreatedAt     time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"default:now()" json:"updatedAt"`

	// Relations
	SOPTemplate *SOPTemplate `gorm:"foreignKey:SOPTemplateID" json:"sopTemplate,omitempty"`
	SOPStep     *SOPStep     `gorm:"foreignKey:SOPStepID" json:"sopStep,omitempty"`
	SOPSubStep  *SOPSubStep  `gorm:"foreignKey:SOPSubStepID" json:"sopSubStep,omitempty"`
	User        *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (SOPComment) TableName() string {
	return "sop_comment"
}
