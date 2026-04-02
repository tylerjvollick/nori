package models

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID             uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID        uuid.UUID  `gorm:"type:uuid;not null" json:"spaceId"`
	TicketTypeID   uuid.UUID  `gorm:"type:uuid;not null" json:"ticketTypeId"`
	ParentTicketID *uuid.UUID `gorm:"type:uuid" json:"parentTicketId,omitempty"`
	StatusID       uuid.UUID  `gorm:"type:uuid;not null" json:"statusId"`
	SOPTemplateID  *int       `json:"sopTemplateId,omitempty"`
	SOPVersionID   *int       `json:"sopVersionId,omitempty"`
	CustomerID     *uuid.UUID `gorm:"type:uuid" json:"customerId,omitempty"`
	AssignedToID   *uuid.UUID `gorm:"type:uuid" json:"assignedToId,omitempty"`
	Title          string     `gorm:"not null" json:"title"`
	Description    *string    `json:"description,omitempty"`
	TicketNumber   string     `gorm:"not null;uniqueIndex" json:"ticketNumber"`
	Priority       int        `gorm:"not null;default:0" json:"priority"`
	DueDate        *time.Time `json:"dueDate,omitempty"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	CreatedByID    uuid.UUID  `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt      time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space        *Space            `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	TicketType   *TicketType       `gorm:"foreignKey:TicketTypeID" json:"ticketType,omitempty"`
	ParentTicket *Ticket           `gorm:"foreignKey:ParentTicketID" json:"parentTicket,omitempty"`
	Status       *StatusDefinition `gorm:"foreignKey:StatusID" json:"status,omitempty"`
	SOPTemplate  *SOPTemplate      `gorm:"foreignKey:SOPTemplateID" json:"sopTemplate,omitempty"`
	SOPVersion   *SOPVersion       `gorm:"foreignKey:SOPVersionID" json:"sopVersion,omitempty"`
	Customer     *Customer         `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	AssignedTo   *User             `gorm:"foreignKey:AssignedToID" json:"assignedTo,omitempty"`
	CreatedBy    *User             `gorm:"foreignKey:CreatedByID" json:"createdBy,omitempty"`
	Children     []Ticket          `gorm:"foreignKey:ParentTicketID" json:"children,omitempty"`
	Steps        []TicketStep      `gorm:"foreignKey:TicketID" json:"steps,omitempty"`
	Tags         []Tag             `gorm:"many2many:ticket_tag;" json:"tags,omitempty"`
}

func (Ticket) TableName() string {
	return "ticket"
}
