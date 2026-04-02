package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SpaceMember represents a user's membership in a space
// This is a many-to-many join table between User and Space
type SpaceMember struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"userId"`
	SpaceID   uuid.UUID `gorm:"type:uuid;not null" json:"spaceId"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"createdAt"`

	// Foreign key relations
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Space Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

// BeforeCreate hook to ensure ID is set
func (sm *SpaceMember) BeforeCreate(tx *gorm.DB) (err error) {
	if sm.ID == uuid.Nil {
		sm.ID = uuid.New()
	}
	return
}
