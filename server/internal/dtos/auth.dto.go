package dtos

import (
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
)

type AuthDTO struct {
	User      models.User
	AccountID uuid.UUID
}
