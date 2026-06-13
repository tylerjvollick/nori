package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskMedia is a photo or video captured during task execution and attached to
// a Task. Media files are stored on disk (see internal/storage) and FilePath
// holds the public URL path used to serve the file.
type TaskMedia struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID       string     `gorm:"type:varchar(255);not null" json:"taskId"`
	FilePath     string     `gorm:"not null" json:"filePath"`
	FileName     string     `gorm:"not null" json:"fileName"`
	MimeType     string     `gorm:"not null" json:"mimeType"`
	FileSize     int64      `gorm:"not null;default:0" json:"fileSize"`
	Duration     *int       `json:"duration,omitempty"`
	DisplayOrder int        `gorm:"not null;default:0" json:"displayOrder"`
	CapturedByID *uuid.UUID `gorm:"type:uuid" json:"capturedById,omitempty"`
	CreatedAt    time.Time  `gorm:"default:now()" json:"createdAt"`

	// Relations
	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

// TableName pins the table name so GORM does not pluralize to "task_medias".
func (TaskMedia) TableName() string {
	return "task_media"
}
