package models

import (
	"time"

	"github.com/google/uuid"
)

// TicketStepStatus represents the execution state of a ticket step.
type TicketStepStatus string

const (
	TicketStepStatusPending    TicketStepStatus = "pending"
	TicketStepStatusInProgress TicketStepStatus = "in_progress"
	TicketStepStatusPaused     TicketStepStatus = "paused"
	TicketStepStatusCompleted  TicketStepStatus = "completed"
	TicketStepStatusSkipped    TicketStepStatus = "skipped"
)

// TicketStep is a live execution step within a ticket. When a ticket is created
// with a linked SOP, the SOP's steps are copied as TicketSteps. Steps can also
// be added ad-hoc during execution (first-time capture mode).
type TicketStep struct {
	ID                uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TicketID          uuid.UUID        `gorm:"type:uuid;not null" json:"ticketId"`
	SOPStepID         *int             `json:"sopStepId,omitempty"`
	StationID         *uuid.UUID       `gorm:"type:uuid" json:"stationId,omitempty"`
	DisplayOrder      int              `gorm:"not null;default:0" json:"displayOrder"`
	Title             string           `gorm:"not null" json:"title"`
	Instructions      *string          `json:"instructions,omitempty"`
	Status            TicketStepStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	AssignedToID      *uuid.UUID       `gorm:"type:uuid" json:"assignedToId,omitempty"`
	StartedAt         *time.Time       `json:"startedAt,omitempty"`
	PausedAt          *time.Time       `json:"pausedAt,omitempty"`
	CompletedAt       *time.Time       `json:"completedAt,omitempty"`
	ActualTimeSeconds int              `gorm:"not null;default:0" json:"actualTimeSeconds"`
	DeviationNotes    *string          `json:"deviationNotes,omitempty"`
	CreatedAt         time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt         time.Time        `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Ticket     *Ticket         `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	SOPStep    *SOPStep        `gorm:"foreignKey:SOPStepID" json:"sopStep,omitempty"`
	AssignedTo *User           `gorm:"foreignKey:AssignedToID" json:"assignedTo,omitempty"`
	SubSteps   []TicketSubStep `gorm:"foreignKey:TicketStepID" json:"subSteps,omitempty"`
}

func (TicketStep) TableName() string {
	return "ticket_step"
}
