package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type UserAccount struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"userId"`
	AccountID uuid.UUID `gorm:"type:uuid;not null" json:"accountId"`
	Role      Role      `gorm:"type:enum('user','admin');default:'user'" json:"role"`

	// Optional: if you want associations
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Account Account `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

// BeforeCreate hook to ensure ID is set
func (ua *UserAccount) BeforeCreate(tx *gorm.DB) (err error) {
	if ua.ID == uuid.Nil {
		ua.ID = uuid.New()
	}
	return
}
