package models

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID   uuid.UUID `gorm:"type:uuid;not null" json:"spaceId"`
	Name      string    `gorm:"not null" json:"name"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Address   *string   `json:"address,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `gorm:"default:now()" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

func (Customer) TableName() string {
	return "customer"
}
