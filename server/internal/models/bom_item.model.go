package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BOMItem represents a line item on a bill of materials, tied to an SOP version.
// It links materials (or ad-hoc items) to the SOP with quantity, unit, and cost info.
type BOMItem struct {
	ID           int              `gorm:"primaryKey;autoIncrement" json:"id"`
	SOPVersionID int              `gorm:"not null" json:"sopVersionId"`
	MaterialID   *uuid.UUID       `gorm:"type:uuid" json:"materialId,omitempty"`
	Name         string           `gorm:"not null" json:"name"`
	Quantity     decimal.Decimal  `gorm:"type:numeric(12,4);not null" json:"quantity"`
	Unit         string           `gorm:"type:varchar(50);not null" json:"unit"`
	SOPStepID    *int             `json:"sopStepId,omitempty"`
	UnitCost     *decimal.Decimal `gorm:"type:numeric(12,4)" json:"unitCost,omitempty"`
	Notes        *string          `json:"notes,omitempty"`
	CreatedAt    time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt    time.Time        `gorm:"default:now()" json:"updatedAt"`

	// Relations
	SOPVersion *SOPVersion `gorm:"foreignKey:SOPVersionID" json:"sopVersion,omitempty"`
	Material   *Material   `gorm:"foreignKey:MaterialID" json:"material,omitempty"`
	SOPStep    *SOPStep    `gorm:"foreignKey:SOPStepID" json:"sopStep,omitempty"`
}

func (BOMItem) TableName() string {
	return "bom_item"
}
