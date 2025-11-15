package models

import (
	"time"
)

type SOPStepMedia struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	SOPStepID int       `gorm:"not null;index" json:"sopStepId"`
	UUID      string    `gorm:"type:varchar(255);not null;unique" json:"uuid"`
	FilePath  string    `gorm:"type:varchar(500);not null" json:"filePath"` // Relative path: sops/1/steps/5/uuid.jpg
	FileName  string    `gorm:"type:varchar(255);not null" json:"fileName"` // Original filename for download
	MimeType  string    `gorm:"type:varchar(100);not null" json:"mimeType"` // image/jpeg, image/png, video/mp4, etc.
	FileSize  int64     `gorm:"not null" json:"fileSize"`                   // In bytes
	Duration  *int      `gorm:"null" json:"duration,omitempty"`             // Duration in seconds (for videos)
	Order     string    `gorm:"type:varchar(50);not null;index" json:"order"` // Lexicographic ordering like steps
	CreatedAt time.Time `gorm:"default:now()" json:"createdAt"`

	// Relations
	SOPStep *SOPStep `gorm:"foreignKey:SOPStepID" json:"sopStep,omitempty"`
}

func (SOPStepMedia) TableName() string {
	return "sop_step_media"
}
