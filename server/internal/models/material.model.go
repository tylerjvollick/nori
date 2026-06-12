package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// MaterialCategory represents the type of material in inventory.
type MaterialCategory string

const (
	MaterialCategoryLumber     MaterialCategory = "lumber"
	MaterialCategoryHardware   MaterialCategory = "hardware"
	MaterialCategoryFinish     MaterialCategory = "finish"
	MaterialCategoryConsumable MaterialCategory = "consumable"
	MaterialCategoryOther      MaterialCategory = "other"
)

type Material struct {
	ID               uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID          uuid.UUID        `gorm:"type:uuid;not null" json:"spaceId"`
	Name             string           `gorm:"not null" json:"name"`
	Category         MaterialCategory `gorm:"type:varchar(50);not null;default:'other'" json:"category"`
	Unit             string           `gorm:"type:varchar(50);not null" json:"unit"`
	CurrentStock     decimal.Decimal  `gorm:"type:numeric(12,4);not null;default:0" json:"currentStock"`
	ReorderThreshold decimal.Decimal  `gorm:"type:numeric(12,4);not null;default:0" json:"reorderThreshold"`
	ReorderQuantity  decimal.Decimal  `gorm:"type:numeric(12,4);not null;default:0" json:"reorderQuantity"`
	Location         *string          `json:"location,omitempty"`
	UnitCost         *decimal.Decimal `gorm:"type:numeric(12,4)" json:"unitCost,omitempty"`
	Supplier         *string          `json:"supplier,omitempty"`
	SKU              *string          `gorm:"column:sku" json:"sku,omitempty"`
	IsActive         bool             `gorm:"not null;default:true" json:"isActive"`
	CreatedAt        time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt        time.Time        `gorm:"default:now()" json:"updatedAt"`
	DeletedAt        gorm.DeletedAt   `gorm:"index" json:"-"`

	// Relations
	Space *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

func (Material) TableName() string {
	return "material"
}
