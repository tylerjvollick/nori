package models

import (
	"time"

	"github.com/google/uuid"
)

// TimeEntry records a work session on a task. A session accumulates elapsed
// time and supports pause/resume. An entry with a nil EndedAt is still active
// (running or paused). ElapsedSecs holds total accumulated seconds; when
// running (!IsPaused && EndedAt==nil), live elapsed = ElapsedSecs + (now - ResumedAt).
type TimeEntry struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID     string     `gorm:"type:varchar(255);not null" json:"taskId"`
	SpaceID    uuid.UUID  `gorm:"type:uuid;not null" json:"spaceId"`
	LoggedByID uuid.UUID  `gorm:"type:uuid;not null" json:"loggedById"`
	StartedAt  time.Time  `gorm:"not null" json:"startedAt"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
	ElapsedSecs int       `gorm:"not null;default:0" json:"elapsedSecs"`
	IsPaused   bool       `gorm:"not null;default:false" json:"isPaused"`
	PausedAt   *time.Time `json:"pausedAt,omitempty"`
	ResumedAt  *time.Time `json:"resumedAt,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
	CreatedAt  time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Task     *Task  `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Space    *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	LoggedBy *User  `gorm:"foreignKey:LoggedByID" json:"loggedBy,omitempty"`
}

func (TimeEntry) TableName() string {
	return "time_entry"
}
