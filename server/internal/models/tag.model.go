package models

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID   uuid.UUID `gorm:"type:uuid;not null" json:"spaceId"`
	Name      string    `gorm:"not null" json:"name"`
	Color     *string   `json:"color,omitempty"`
	CreatedAt time.Time `gorm:"default:now()" json:"createdAt"`

	// Relations
	Space *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

func (Tag) TableName() string {
	return "tag"
}

// SOPTemplateTag is the join table linking SOPTemplates to Tags.
type SOPTemplateTag struct {
	SOPTemplateID int       `gorm:"not null;primaryKey" json:"sopTemplateId"`
	TagID         uuid.UUID `gorm:"type:uuid;not null;primaryKey" json:"tagId"`

	// Relations
	SOPTemplate *SOPTemplate `gorm:"foreignKey:SOPTemplateID" json:"sopTemplate,omitempty"`
	Tag         *Tag         `gorm:"foreignKey:TagID" json:"tag,omitempty"`
}

func (SOPTemplateTag) TableName() string {
	return "sop_template_tag"
}

// TicketTag is the join table linking Tickets to Tags.
// Note: Ticket FK constraint is deferred until the Ticket model is created in Phase 3.
type TicketTag struct {
	TicketID uuid.UUID `gorm:"type:uuid;not null;primaryKey" json:"ticketId"`
	TagID    uuid.UUID `gorm:"type:uuid;not null;primaryKey" json:"tagId"`

	// Relations (Tag only — Ticket relation added when Ticket model exists)
	Tag *Tag `gorm:"foreignKey:TagID" json:"tag,omitempty"`
}

func (TicketTag) TableName() string {
	return "ticket_tag"
}
