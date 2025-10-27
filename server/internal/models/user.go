package models

import (
	"github.com/google/uuid"
)

// Define your enum type in Go
type GlobalRole string

const (
	GlobalRoleAdmin  GlobalRole = "ADMIN"
	GlobalRoleUser   GlobalRole = "USER"
	GlobalRoleViewer GlobalRole = "VIEWER"
	// Add other roles as needed
)

type User struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	GlobalRole       *GlobalRole `gorm:"type:enum('ADMIN','USER','VIEWER');default:NULL"` // nullable enum
	Email            string      `gorm:"not null;uniqueIndex"`
	Password         *string     `gorm:"type:text"` // optional
	FirstName        *string
	LastName         *string
	DefaultAccountID *uuid.UUID `gorm:"type:uuid"` // foreign key to Account

	// Optional: GORM relation
	DefaultAccount   *Account `gorm:"foreignKey:DefaultAccountID"`
}


