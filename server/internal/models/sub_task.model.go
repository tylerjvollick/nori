package models

import (
	"time"

	"github.com/google/uuid"
)

// SubTask is a sequential checklist item within a Task. Sub-tasks have no
// status, dependencies, station assignment, or time tracking — they are
// purely ordered steps with an optional description and attached images.
type SubTask struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID       string     `gorm:"type:varchar(255);not null" json:"taskId"`
	Title        string     `gorm:"not null" json:"title"`
	Description  *string    `json:"description,omitempty"`
	DisplayOrder int        `gorm:"not null;default:0" json:"displayOrder"`
	CreatedAt    time.Time  `gorm:"default:now()" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Task   *Task          `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Images []SubTaskImage `gorm:"foreignKey:SubTaskID" json:"images,omitempty"`
}

// SubTaskImage is an ordered image attached to a SubTask.
type SubTaskImage struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SubTaskID    uuid.UUID `gorm:"type:uuid;not null" json:"subTaskId"`
	ImageURL     string    `gorm:"not null" json:"imageUrl"`
	DisplayOrder int       `gorm:"not null;default:0" json:"displayOrder"`
	CreatedAt    time.Time `gorm:"default:now()" json:"createdAt"`

	// Relations
	SubTask *SubTask `gorm:"foreignKey:SubTaskID" json:"subTask,omitempty"`
}
