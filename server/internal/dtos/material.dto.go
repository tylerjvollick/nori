package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tylerjvollick/nori/internal/models"
)

// CreateMaterialRequest is the JSON body for
// POST /api/v1/spaces/:spaceId/materials.
type CreateMaterialRequest struct {
	Name     string           `json:"name"`
	Category *string          `json:"category,omitempty"`
	Unit     string           `json:"unit"`
	Supplier *string          `json:"supplier,omitempty"`
	SKU      *string          `json:"sku,omitempty"`
	Location *string          `json:"location,omitempty"`
	UnitCost *decimal.Decimal `json:"unitCost,omitempty"`
}

// UpdateMaterialRequest is the JSON body for
// PUT /api/v1/spaces/:spaceId/materials/:id. All fields are optional;
// only provided fields are updated.
type UpdateMaterialRequest struct {
	Name     *string          `json:"name,omitempty"`
	Category *string          `json:"category,omitempty"`
	Unit     *string          `json:"unit,omitempty"`
	Supplier *string          `json:"supplier,omitempty"`
	SKU      *string          `json:"sku,omitempty"`
	Location *string          `json:"location,omitempty"`
	UnitCost *decimal.Decimal `json:"unitCost,omitempty"`
	IsActive *bool            `json:"isActive,omitempty"`
}

// MaterialResponse is the JSON representation of a material.
type MaterialResponse struct {
	ID           uuid.UUID        `json:"id"`
	SpaceID      uuid.UUID        `json:"spaceId"`
	Name         string           `json:"name"`
	Category     string           `json:"category"`
	Unit         string           `json:"unit"`
	Supplier     *string          `json:"supplier,omitempty"`
	SKU          *string          `json:"sku,omitempty"`
	Location     *string          `json:"location,omitempty"`
	UnitCost     *decimal.Decimal `json:"unitCost,omitempty"`
	CurrentStock decimal.Decimal  `json:"currentStock"`
	IsActive     bool             `json:"isActive"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// MaterialResponseFromModel converts a models.Material to a MaterialResponse.
func MaterialResponseFromModel(m *models.Material) MaterialResponse {
	return MaterialResponse{
		ID:           m.ID,
		SpaceID:      m.SpaceID,
		Name:         m.Name,
		Category:     string(m.Category),
		Unit:         m.Unit,
		Supplier:     m.Supplier,
		SKU:          m.SKU,
		Location:     m.Location,
		UnitCost:     m.UnitCost,
		CurrentStock: m.CurrentStock,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
