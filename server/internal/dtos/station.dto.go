package dtos

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tylerjvollick/nori/internal/models"
)

// StationResponse is the JSON representation of a station with WIP count.
type StationResponse struct {
	ID           uuid.UUID        `json:"id"`
	Name         string           `json:"name"`
	Description  *string          `json:"description,omitempty"`
	DisplayOrder int              `json:"displayOrder"`
	WIPLimit     int              `json:"wipLimit"`
	WIPCount     int              `json:"wipCount"`
	BufferSize   int              `json:"bufferSize"`
	CostsHour    *decimal.Decimal `json:"costsHour,omitempty"`
	IsActive     bool             `json:"isActive"`
}

// StationResponseFromModel converts a models.Station to a StationResponse DTO.
// wipCount must be supplied externally (it comes from a separate query).
func StationResponseFromModel(s *models.Station, wipCount int) StationResponse {
	return StationResponse{
		ID:           s.ID,
		Name:         s.Name,
		Description:  s.Description,
		DisplayOrder: s.DisplayOrder,
		WIPLimit:     s.WIPLimit,
		WIPCount:     wipCount,
		BufferSize:   s.BufferSize,
		CostsHour:    s.CostsHour,
		IsActive:     s.IsActive,
	}
}

// CreateStationRequest is the JSON body for POST /api/v1/stations.
type CreateStationRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	WIPLimit    *int    `json:"wipLimit,omitempty"`
	BufferSize  *int    `json:"bufferSize,omitempty"`
}

// UpdateStationRequest is the JSON body for PUT /api/v1/stations/:id.
// CostsHour uses Nullable: absent leaves the rate unchanged, explicit null
// clears it.
type UpdateStationRequest struct {
	Name        *string                   `json:"name,omitempty"`
	Description *string                   `json:"description,omitempty"`
	WIPLimit    *int                      `json:"wipLimit,omitempty"`
	BufferSize  *int                      `json:"bufferSize,omitempty"`
	CostsHour   Nullable[decimal.Decimal] `json:"costsHour"`
	IsActive    *bool                     `json:"isActive,omitempty"`
}
