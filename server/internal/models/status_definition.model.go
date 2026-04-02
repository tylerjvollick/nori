package models

import (
	"time"

	"github.com/google/uuid"
)

// StatusCategory groups statuses for board layout and metric computation.
type StatusCategory string

const (
	StatusCategoryTodo       StatusCategory = "todo"
	StatusCategoryInProgress StatusCategory = "in_progress"
	StatusCategoryDone       StatusCategory = "done"
)

type StatusDefinition struct {
	ID           uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TicketTypeID uuid.UUID      `gorm:"type:uuid;not null" json:"ticketTypeId"`
	Name         string         `gorm:"not null" json:"name"`
	DisplayOrder int            `gorm:"not null;default:0" json:"displayOrder"`
	Category     StatusCategory `gorm:"type:varchar(50);not null;default:'todo'" json:"category"`
	Color        *string        `json:"color,omitempty"`
	IsDefault    bool           `gorm:"not null;default:false" json:"isDefault"`
	IsTerminal   bool           `gorm:"not null;default:false" json:"isTerminal"`
	CreatedAt    time.Time      `gorm:"default:now()" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"default:now()" json:"updatedAt"`

	// Relations
	TicketType *TicketType `gorm:"foreignKey:TicketTypeID" json:"ticketType,omitempty"`
}

func (StatusDefinition) TableName() string {
	return "status_definition"
}
