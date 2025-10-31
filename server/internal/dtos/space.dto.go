package dtos

import "github.com/google/uuid"

// CreateSpaceDTO represents the request to create a new space
type CreateSpaceDTO struct {
	Name string `json:"name" binding:"required"`
}

// UpdateSpaceDTO represents the request to update a space
type UpdateSpaceDTO struct {
	Name *string `json:"name,omitempty"`
}

// SpaceResponseDTO represents a space in the response
type SpaceResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	AccountID uuid.UUID `json:"accountId"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
}

// RecordSpaceVisitDTO represents the request to record a space visit
type RecordSpaceVisitDTO struct {
	SpaceID uuid.UUID `json:"spaceId" binding:"required"`
}
