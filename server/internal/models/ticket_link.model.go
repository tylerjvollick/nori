package models

import (
	"time"

	"github.com/google/uuid"
)

// LinkType represents the relationship type between two tickets.
type LinkType string

const (
	LinkTypeCreatedFrom LinkType = "created_from"
	LinkTypeBlocks      LinkType = "blocks"
	LinkTypeBlockedBy   LinkType = "blocked_by"
	LinkTypeRelatesTo   LinkType = "relates_to"
)

// TicketLink represents a cross-ticket relationship, including across spaces.
// Parent/child relationships are modeled via Ticket.ParentTicketID, not TicketLink.
// TicketLink is for peer relationships (blocks, relates_to, created_from).
type TicketLink struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SourceTicketID uuid.UUID `gorm:"type:uuid;not null" json:"sourceTicketId"`
	TargetTicketID uuid.UUID `gorm:"type:uuid;not null" json:"targetTicketId"`
	LinkType       LinkType  `gorm:"type:varchar(50);not null" json:"linkType"`
	CreatedByID    uuid.UUID `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt      time.Time `gorm:"default:now()" json:"createdAt"`

	// Relations
	SourceTicket *Ticket `gorm:"foreignKey:SourceTicketID" json:"sourceTicket,omitempty"`
	TargetTicket *Ticket `gorm:"foreignKey:TargetTicketID" json:"targetTicket,omitempty"`
	CreatedBy    *User   `gorm:"foreignKey:CreatedByID" json:"createdBy,omitempty"`
}

func (TicketLink) TableName() string {
	return "ticket_link"
}
