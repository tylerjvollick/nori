package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// BOMItem represents a line item on a bill of materials.
// It links materials (or ad-hoc items) with quantity, unit, and cost info.
// BOMItems belong to a RecipeVersion and optionally reference a step task.
type BOMItem struct {
	ID              int              `gorm:"primaryKey;autoIncrement" json:"id"`
	RecipeVersionID int              `gorm:"not null" json:"recipeVersionId"`
	MaterialID      *uuid.UUID       `gorm:"type:uuid" json:"materialId,omitempty"`
	Name            string           `gorm:"not null" json:"name"`
	Quantity        decimal.Decimal  `gorm:"type:numeric(12,4);not null" json:"quantity"`
	Unit            string           `gorm:"type:varchar(50);not null" json:"unit"`
	UnitCost        *decimal.Decimal `gorm:"type:numeric(12,4)" json:"unitCost,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	CreatedAt       time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt       time.Time        `gorm:"default:now()" json:"updatedAt"`

	// Relations
	RecipeVersion *RecipeVersion `gorm:"foreignKey:RecipeVersionID" json:"recipeVersion,omitempty"`
	Material      *Material      `gorm:"foreignKey:MaterialID" json:"material,omitempty"`
}

func (BOMItem) TableName() string {
	return "bom_item"
}
