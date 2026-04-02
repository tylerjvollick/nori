package models

import (
	"github.com/google/uuid"
)

// Account represents the account entity
type Plan string

const (
	Trial      Plan = "trial"
	Paid       Plan = "paid"
	Enterprise Plan = "enterprise"
)

type Account struct {
	ID   uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Name *string   `json:"name,omitempty"`
	Plan Plan      `gorm:"type:text;default:'trial'" json:"plan"` // default to "trial"

	// Foreign key relation
	CreatedByUserID uuid.UUID `gorm:"type:uuid;not null" json:"createdByUserId"`
	CreatedByUser   User      `gorm:"foreignKey:CreatedByUserID" json:"createdByUser,omitempty"`
}
