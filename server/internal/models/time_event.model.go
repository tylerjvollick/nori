package models

import (
	"time"

	"github.com/google/uuid"
)

// TimeEventType represents the kind of time event.
type TimeEventType string

const (
	TimeEventTypeCheckIn  TimeEventType = "check_in"
	TimeEventTypeCheckOut TimeEventType = "check_out"
	TimeEventTypePause    TimeEventType = "pause"
	TimeEventTypeResume   TimeEventType = "resume"
)

// TimeEventSource represents how the time event was created.
type TimeEventSource string

const (
	TimeEventSourceManual TimeEventSource = "manual"
	TimeEventSourceWeb    TimeEventSource = "web"
	TimeEventSourceCLI    TimeEventSource = "cli"
	TimeEventSourceTap    TimeEventSource = "tap"
	TimeEventSourceSensor TimeEventSource = "sensor"
	TimeEventSourceAPI    TimeEventSource = "api"
	TimeEventSourceSystem TimeEventSource = "system"
)

// TimeEvent is an append-only, source-agnostic time log. Every time-relevant
// action (starting a step, checking in at a station, pausing work) creates a
// TimeEvent. These records are immutable — corrections are new events with
// backdated timestamps.
type TimeEvent struct {
	ID           uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID      uuid.UUID       `gorm:"type:uuid;not null" json:"spaceId"`
	UserID       uuid.UUID       `gorm:"type:uuid;not null" json:"userId"`
	TicketID     *uuid.UUID      `gorm:"type:uuid" json:"ticketId,omitempty"`
	TicketStepID *uuid.UUID      `gorm:"type:uuid" json:"ticketStepId,omitempty"`
	StationID    *uuid.UUID      `gorm:"type:uuid" json:"stationId,omitempty"`
	EventType    TimeEventType   `gorm:"type:varchar(20);not null" json:"eventType"`
	Source       TimeEventSource `gorm:"type:varchar(20);not null" json:"source"`
	Timestamp    time.Time       `gorm:"not null" json:"timestamp"`
	Notes        *string         `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt    time.Time       `gorm:"default:now()" json:"createdAt"`

	// Relations
	Space      *Space      `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	User       *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Ticket     *Ticket     `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
	TicketStep *TicketStep `gorm:"foreignKey:TicketStepID" json:"ticketStep,omitempty"`
	Station    *Station    `gorm:"foreignKey:StationID" json:"station,omitempty"`
}

func (TimeEvent) TableName() string {
	return "time_event"
}
