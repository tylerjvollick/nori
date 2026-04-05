package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CostType represents the category of a cost entry.
type CostType string

const (
	CostTypeLabor      CostType = "labor"
	CostTypeMaterial   CostType = "material"
	CostTypeConsumable CostType = "consumable"
	CostTypeMarketing  CostType = "marketing"
	CostTypeOther      CostType = "other"
)

// CostEntry tracks costs. Supports labor, materials, consumables,
// marketing, and other cost types. Labor costs can be auto-computed from
// TimeEvents × labor rate, or entered manually.
type CostEntry struct {
	ID          uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	CostType    CostType         `gorm:"type:varchar(20);not null" json:"costType"`
	Description string           `gorm:"type:text;not null" json:"description"`
	Amount      decimal.Decimal  `gorm:"type:numeric(12,4);not null" json:"amount"`
	Quantity    *decimal.Decimal `gorm:"type:numeric(12,4)" json:"quantity,omitempty"`
	Unit        *string          `gorm:"type:varchar(50)" json:"unit,omitempty"`
	UnitCost    *decimal.Decimal `gorm:"type:numeric(12,4)" json:"unitCost,omitempty"`
	MaterialID  *uuid.UUID       `gorm:"type:uuid" json:"materialId,omitempty"`
	TimeEventID *uuid.UUID       `gorm:"type:uuid" json:"timeEventId,omitempty"`
	CreatedByID uuid.UUID        `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt   time.Time        `gorm:"default:now()" json:"createdAt"`

	// Relations
	Material  *Material  `gorm:"foreignKey:MaterialID" json:"material,omitempty"`
	TimeEvent *TimeEvent `gorm:"foreignKey:TimeEventID" json:"timeEvent,omitempty"`
	CreatedBy *User      `gorm:"foreignKey:CreatedByID" json:"createdBy,omitempty"`
}

func (CostEntry) TableName() string {
	return "cost_entry"
}
