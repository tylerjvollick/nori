package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Station struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID      uuid.UUID `gorm:"type:uuid;not null" json:"spaceId"`
	Name         string    `gorm:"not null" json:"name"`
	Description  *string   `json:"description,omitempty"`
	DisplayOrder int       `gorm:"not null;default:0" json:"displayOrder"`
	WIPLimit     int       `gorm:"column:wip_limit;not null;default:1" json:"wipLimit"`
	BufferSize   int       `gorm:"not null;default:0" json:"bufferSize"`
	// CostsHour is the hourly labor rate for work at this station. Falls
	// back to the space's default_labor_rate when null.
	CostsHour *decimal.Decimal `gorm:"type:numeric(12,4)" json:"costsHour,omitempty"`
	IsActive  bool             `gorm:"not null;default:true" json:"isActive"`
	CreatedAt time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt time.Time        `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

func (Station) TableName() string {
	return "station"
}
