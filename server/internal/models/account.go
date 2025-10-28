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
	ID                        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name                      *string
	SyncContactAndBillingInfo bool `gorm:"not null"` // required field

	ContactFirstName   *string
	ContactLastName    *string
	ContactPhoneNumber *string
	ContactPhoneExt    *string
	ContactEmail       *string
	ContactAddress     *string
	ContactCity        *string
	ContactState       *string
	ContactZip         *string

	BillingFirstName   *string
	BillingLastName    *string
	BillingPhoneNumber *string
	BillingPhoneExt    *string
	BillingEmail       *string
	BillingAddress     *string
	BillingCity        *string
	BillingState       *string
	BillingZip         *string

	Plan Plan `gorm:"type:text;default:'trial'"` // default to "trial"

	// Foreign key relation
	CreatedByUserID uuid.UUID `gorm:"type:uuid;not null"`
	CreatedByUser   User      `gorm:"foreignKey:CreatedByUserID"`
}
