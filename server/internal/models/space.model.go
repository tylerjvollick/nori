package models

import (
	"time"

	"github.com/google/uuid"
)

// Space represents a workspace/department within an account
type Space struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	AccountID uuid.UUID `gorm:"type:uuid;not null" json:"accountId"`
	IsDefault bool      `gorm:"not null;default:false" json:"isDefault"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updatedAt"`

	// Foreign key relation
	Account Account `gorm:"foreignKey:AccountID" json:"-"`
}
