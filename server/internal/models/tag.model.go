package models

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID   uuid.UUID `gorm:"type:uuid;not null" json:"spaceId"`
	Name      string    `gorm:"not null" json:"name"`
	Color     *string   `json:"color,omitempty"`
	CreatedAt time.Time `gorm:"default:now()" json:"createdAt"`

	// Relations
	Space *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

func (Tag) TableName() string {
	return "tag"
}
