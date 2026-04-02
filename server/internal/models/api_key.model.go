package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKey struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	AccountID   uuid.UUID  `gorm:"type:uuid;not null" json:"accountId"`
	Name        string     `gorm:"type:text;not null" json:"name"`
	KeyHash     string     `gorm:"type:text;not null;uniqueIndex" json:"-"` // Never expose hash in JSON
	LastUsedAt  *time.Time `gorm:"type:timestamptz" json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time `gorm:"type:timestamptz" json:"expiresAt,omitempty"`
	IsActive    bool       `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"createdAt"`
	CreatedByID uuid.UUID  `gorm:"type:uuid;not null" json:"createdById"`

	// Optional: GORM relations
	Account   *Account `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	CreatedBy *User    `gorm:"foreignKey:CreatedByID" json:"createdBy,omitempty"`
}

// BeforeCreate hook to ensure ID is set
func (ak *APIKey) BeforeCreate(tx *gorm.DB) (err error) {
	if ak.ID == uuid.Nil {
		ak.ID = uuid.New()
	}
	return
}
