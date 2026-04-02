package models

import "time"

type SOPSubStep struct {
	ID           int       `gorm:"primaryKey;autoIncrement" json:"id"`
	SOPStepID    int       `gorm:"not null" json:"sopStepId"`
	DisplayOrder int       `gorm:"not null" json:"displayOrder"`
	Title        string    `gorm:"not null" json:"title"`
	Instructions *string   `json:"instructions,omitempty"`
	CreatedAt    time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"default:now()" json:"updatedAt"`

	// Relations
	SOPStep *SOPStep       `gorm:"foreignKey:SOPStepID" json:"sopStep,omitempty"`
	Media   []SOPStepMedia `gorm:"foreignKey:SOPSubStepID" json:"media,omitempty"`
}

func (SOPSubStep) TableName() string {
	return "sop_sub_step"
}
