package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tylerjvollick/nori/internal/models"
)

// CreateTaskMaterialRequest is the JSON body for
// POST /api/v1/spaces/:spaceId/tasks/:taskId/materials.
type CreateTaskMaterialRequest struct {
	MaterialID      uuid.UUID       `json:"materialId"`
	QuantityPerUnit decimal.Decimal `json:"quantityPerUnit"`
	Notes           *string         `json:"notes,omitempty"`
}

// UpdateTaskMaterialRequest is the JSON body for
// PUT /api/v1/spaces/:spaceId/tasks/:taskId/materials/:id. All fields optional.
type UpdateTaskMaterialRequest struct {
	QuantityPerUnit *decimal.Decimal `json:"quantityPerUnit,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
}

// TaskMaterialResponse is the JSON representation of a task material.
type TaskMaterialResponse struct {
	ID                  uuid.UUID         `json:"id"`
	TaskID              string            `json:"taskId"`
	MaterialID          uuid.UUID         `json:"materialId"`
	QuantityPerUnit     decimal.Decimal   `json:"quantityPerUnit"`
	SnapshottedUnitCost *decimal.Decimal  `json:"snapshottedUnitCost,omitempty"`
	Notes               *string           `json:"notes,omitempty"`
	Material            *MaterialResponse `json:"material,omitempty"`
	CreatedAt           time.Time         `json:"createdAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
}

// TaskMaterialResponseFromModel converts a models.TaskMaterial to its DTO.
func TaskMaterialResponseFromModel(tm *models.TaskMaterial) TaskMaterialResponse {
	resp := TaskMaterialResponse{
		ID:                  tm.ID,
		TaskID:              tm.TaskID,
		MaterialID:          tm.MaterialID,
		QuantityPerUnit:     tm.QuantityPerUnit,
		SnapshottedUnitCost: tm.SnapshottedUnitCost,
		Notes:               tm.Notes,
		CreatedAt:           tm.CreatedAt,
		UpdatedAt:           tm.UpdatedAt,
	}
	if tm.Material != nil {
		m := MaterialResponseFromModel(tm.Material)
		resp.Material = &m
	}
	return resp
}
