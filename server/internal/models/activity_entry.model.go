package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ActivityEntryType represents the kind of activity logged on a ticket.
type ActivityEntryType string

const (
	ActivityEntryTypeStatusChange     ActivityEntryType = "status_change"
	ActivityEntryTypeStepStarted      ActivityEntryType = "step_started"
	ActivityEntryTypeStepCompleted    ActivityEntryType = "step_completed"
	ActivityEntryTypeStepPaused       ActivityEntryType = "step_paused"
	ActivityEntryTypeStepResumed      ActivityEntryType = "step_resumed"
	ActivityEntryTypeStepSkipped      ActivityEntryType = "step_skipped"
	ActivityEntryTypeComment          ActivityEntryType = "comment"
	ActivityEntryTypeInterruption     ActivityEntryType = "interruption"
	ActivityEntryTypeAssignmentChange ActivityEntryType = "assignment_change"
	ActivityEntryTypeLinkAdded        ActivityEntryType = "link_added"
	ActivityEntryTypeSOPEdited        ActivityEntryType = "sop_edited"
	ActivityEntryTypeCostLogged       ActivityEntryType = "cost_logged"
	ActivityEntryTypeTicketCreated    ActivityEntryType = "ticket_created"
)

// JSONB is a custom type for storing arbitrary JSON data as PostgreSQL JSONB.
type JSONB json.RawMessage

// Scan implements the sql.Scanner interface for reading JSONB from the database.
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		*j = nil
		return nil
	}
	result := make(JSONB, len(bytes))
	copy(result, bytes)
	*j = result
	return nil
}

// Value implements the driver.Valuer interface for writing JSONB to the database.
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

// MarshalJSON returns the raw JSON bytes.
func (j JSONB) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

// UnmarshalJSON sets the JSONB value from raw JSON bytes.
func (j *JSONB) UnmarshalJSON(data []byte) error {
	if data == nil || string(data) == "null" {
		*j = nil
		return nil
	}
	result := make(JSONB, len(data))
	copy(result, data)
	*j = result
	return nil
}

// ActivityEntry is a chronological log of everything that happens on a ticket.
// It tells the full story of a ticket's life: step transitions, interruptions,
// comments, status changes. This is the "activity tab" on a ticket.
type ActivityEntry struct {
	ID              uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TicketID        uuid.UUID         `gorm:"type:uuid;not null" json:"ticketId"`
	UserID          uuid.UUID         `gorm:"type:uuid;not null" json:"userId"`
	EntryType       ActivityEntryType `gorm:"type:varchar(50);not null" json:"entryType"`
	Description     string            `gorm:"type:text;not null" json:"description"`
	LinkedTicketID  *uuid.UUID        `gorm:"type:uuid" json:"linkedTicketId,omitempty"`
	TicketStepID    *uuid.UUID        `gorm:"type:uuid" json:"ticketStepId,omitempty"`
	DurationSeconds *int              `json:"durationSeconds,omitempty"`
	OldValue        *string           `gorm:"type:text" json:"oldValue,omitempty"`
	NewValue        *string           `gorm:"type:text" json:"newValue,omitempty"`
	Metadata        *JSONB            `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt       time.Time         `gorm:"default:now()" json:"createdAt"`

	// Relations
	Ticket       *Ticket     `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	User         *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	LinkedTicket *Ticket     `gorm:"foreignKey:LinkedTicketID" json:"linkedTicket,omitempty"`
	TicketStep   *TicketStep `gorm:"foreignKey:TicketStepID" json:"ticketStep,omitempty"`
}

func (ActivityEntry) TableName() string {
	return "activity_entry"
}
