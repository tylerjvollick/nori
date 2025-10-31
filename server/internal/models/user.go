package models

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/google/uuid"
)

// RecentSpaces is a custom type for storing recent space IDs as JSONB
type RecentSpaces []uuid.UUID

// Scan implements the sql.Scanner interface
func (rs *RecentSpaces) Scan(value interface{}) error {
	if value == nil {
		*rs = []uuid.UUID{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		*rs = []uuid.UUID{}
		return nil
	}
	return json.Unmarshal(bytes, rs)
}

// Value implements the driver.Valuer interface
func (rs RecentSpaces) Value() (driver.Value, error) {
	if len(rs) == 0 {
		return "[]", nil
	}
	return json.Marshal(rs)
}

// Define your enum type in Go
type GlobalRole string

const (
	GlobalRoleAdmin  GlobalRole = "ADMIN"
	GlobalRoleUser   GlobalRole = "USER"
	GlobalRoleViewer GlobalRole = "VIEWER"
	// Add other roles as needed
)

type User struct {
	ID               uuid.UUID   `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	GlobalRole       *GlobalRole `gorm:"type:enum('ADMIN','USER','VIEWER');default:NULL"` // nullable enum
	Email            string      `gorm:"not null;uniqueIndex"`
	Password         *string     `gorm:"type:text"` // optional
	FirstName        *string
	LastName         *string
	DefaultAccountID *uuid.UUID   `gorm:"type:uuid"` // foreign key to Account
	RecentSpaces     RecentSpaces `gorm:"type:jsonb;default:'[]'"`

	// Optional: GORM relation
	DefaultAccount *Account `gorm:"foreignKey:DefaultAccountID"`
}
