package models

import (
	"time"

	"github.com/google/uuid"
)

type TicketType struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID              uuid.UUID `gorm:"type:uuid;not null" json:"spaceId"`
	Name                 string    `gorm:"not null" json:"name"`
	Description          *string   `json:"description,omitempty"`
	Icon                 *string   `json:"icon,omitempty"`
	Color                *string   `json:"color,omitempty"`
	DefaultSOPTemplateID *int      `json:"defaultSopTemplateId,omitempty"`
	IsActive             bool      `gorm:"not null;default:true" json:"isActive"`
	DisplayOrder         int       `gorm:"not null;default:0" json:"displayOrder"`
	CreatedAt            time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt            time.Time `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space              *Space             `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	DefaultSOPTemplate *SOPTemplate       `gorm:"foreignKey:DefaultSOPTemplateID" json:"defaultSopTemplate,omitempty"`
	StatusDefinitions  []StatusDefinition `gorm:"foreignKey:TicketTypeID" json:"statusDefinitions,omitempty"`
}

func (TicketType) TableName() string {
	return "ticket_type"
}
