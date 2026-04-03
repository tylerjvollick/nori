package dtos

import (
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

type AuthDTO struct {
	User          models.User
	AccountID     uuid.UUID
	ActiveSpaceID *uuid.UUID
}

// MeResponseDTO is the response for GET /auth/me.
type MeResponseDTO struct {
	ID               uuid.UUID          `json:"id"`
	Email            string             `json:"email"`
	FirstName        *string            `json:"firstName,omitempty"`
	LastName         *string            `json:"lastName,omitempty"`
	Role             *models.Role       `json:"role"`
	ActiveSpaceID    *uuid.UUID         `json:"activeSpaceId,omitempty"`
	AccessibleSpaces []SpaceResponseDTO `json:"accessibleSpaces"`
}
