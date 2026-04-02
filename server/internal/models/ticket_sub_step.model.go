package models

import (
	"time"

	"github.com/google/uuid"
)

// TicketSubStep is a detail step within a ticket step. Not individually timed.
// These are checklist-style items for documentation and execution guidance.
// Copied from SOP sub-steps when the ticket is created.
type TicketSubStep struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TicketStepID uuid.UUID `gorm:"type:uuid;not null" json:"ticketStepId"`
	SOPSubStepID *int      `json:"sopSubStepId,omitempty"`
	DisplayOrder int       `gorm:"not null;default:0" json:"displayOrder"`
	Title        string    `gorm:"not null" json:"title"`
	Instructions *string   `json:"instructions,omitempty"`
	IsCompleted  bool      `gorm:"not null;default:false" json:"isCompleted"`
	CreatedAt    time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"default:now()" json:"updatedAt"`

	// Relations
	TicketStep *TicketStep `gorm:"foreignKey:TicketStepID" json:"ticketStep,omitempty"`
}

func (TicketSubStep) TableName() string {
	return "ticket_sub_step"
}
