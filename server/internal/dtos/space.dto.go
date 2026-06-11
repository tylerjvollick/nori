package dtos

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateSpaceDTO represents the request to create a new space
type CreateSpaceDTO struct {
	Name     string  `json:"name" binding:"required"`
	Slug     *string `json:"slug,omitempty"`     // Optional: 2-5 uppercase chars. Auto-generated if not provided.
	Template *string `json:"template,omitempty"` // Optional: "woodworking_shop", "sales"
}

// UpdateSpaceDTO represents the request to update a space.
// DefaultLaborRate uses Nullable: absent leaves the rate unchanged,
// explicit null clears it.
type UpdateSpaceDTO struct {
	Name             *string                   `json:"name,omitempty"`
	Slug             *string                   `json:"slug,omitempty"`
	DefaultLaborRate Nullable[decimal.Decimal] `json:"defaultLaborRate"`
}

// SpaceResponseDTO represents a space in the response
type SpaceResponseDTO struct {
	ID               uuid.UUID        `json:"id"`
	Name             string           `json:"name"`
	Slug             string           `json:"slug"`
	AccountID        uuid.UUID        `json:"accountId"`
	IsDefault        bool             `json:"isDefault"`
	DefaultLaborRate *decimal.Decimal `json:"defaultLaborRate,omitempty"`
	CreatedAt        string           `json:"createdAt"`
	UpdatedAt        string           `json:"updatedAt"`
}

// RecordSpaceVisitDTO represents the request to record a space visit
type RecordSpaceVisitDTO struct {
	SpaceID uuid.UUID `json:"spaceId" binding:"required"`
}

// SpaceMemberUserDTO is a safe user representation for space member listings
// (no password hash or sensitive fields).
type SpaceMemberUserDTO struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName *string   `json:"firstName,omitempty"`
	LastName  *string   `json:"lastName,omitempty"`
}

// SpaceMemberDTO represents a space member in API responses.
type SpaceMemberDTO struct {
	ID        uuid.UUID          `json:"id"`
	UserID    uuid.UUID          `json:"userId"`
	SpaceID   uuid.UUID          `json:"spaceId"`
	CreatedAt string             `json:"createdAt"`
	User      SpaceMemberUserDTO `json:"user"`
}
