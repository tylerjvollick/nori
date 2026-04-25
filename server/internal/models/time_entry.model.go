package models

import (
	"time"

	"github.com/google/uuid"
)

// TimeEntry records a discrete block of time spent on a task. An entry with a
// nil EndedAt represents a running timer. DurationSecs is computed from
// StartedAt/EndedAt, or manually set for manual entries.
type TimeEntry struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID       string     `gorm:"type:varchar(255);not null" json:"taskId"`
	SpaceID      uuid.UUID  `gorm:"type:uuid;not null" json:"spaceId"`
	LoggedByID   uuid.UUID  `gorm:"type:uuid;not null" json:"loggedById"`
	StartedAt    time.Time  `gorm:"not null" json:"startedAt"`
	EndedAt      *time.Time `json:"endedAt,omitempty"`
	DurationSecs *int       `json:"durationSecs,omitempty"`
	Notes        *string    `json:"notes,omitempty"`
	CreatedAt    time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Task     *Task  `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Space    *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	LoggedBy *User  `gorm:"foreignKey:LoggedByID" json:"loggedBy,omitempty"`
}

func (TimeEntry) TableName() string {
	return "time_entry"
}
