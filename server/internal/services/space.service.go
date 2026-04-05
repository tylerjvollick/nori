package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

type SpaceService struct {
	spaceRepository *repositories.SpaceRepository
	userRepository  *repositories.UserRepository
}

func NewSpaceService(
	spaceRepository *repositories.SpaceRepository,
	userRepository *repositories.UserRepository,
	spaceTemplateService interface{},
) *SpaceService {
	return &SpaceService{
		spaceRepository: spaceRepository,
		userRepository:  userRepository,
	}
}

// CreateSpace creates a new space for an account
func (s *SpaceService) CreateSpace(accountID uuid.UUID, dto *dtos.CreateSpaceDTO) (*models.Space, error) {
	space := &models.Space{
		ID:        uuid.New(),
		Name:      dto.Name,
		AccountID: accountID,
		IsDefault: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.spaceRepository.Create(space); err != nil {
		return nil, err
	}

	return space, nil
}

// CreateDefaultSpace creates a default space for an account (called during registration)
func (s *SpaceService) CreateDefaultSpace(accountID uuid.UUID) (*models.Space, error) {
	space := &models.Space{
		ID:        uuid.New(),
		Name:      "Default",
		AccountID: accountID,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.spaceRepository.Create(space); err != nil {
		return nil, err
	}

	return space, nil
}

// GetSpaceByID retrieves a space by ID and verifies it belongs to the user's account
func (s *SpaceService) GetSpaceByID(spaceID uuid.UUID, accountID uuid.UUID) (*models.Space, error) {
	space, err := s.spaceRepository.FindByID(spaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("space not found")
		}
		return nil, err
	}

	// Verify the space belongs to the user's account
	if space.AccountID != accountID {
		return nil, fmt.Errorf("unauthorized access to space")
	}

	return space, nil
}

// GetSpacesByAccountID retrieves all spaces for an account
func (s *SpaceService) GetSpacesByAccountID(accountID uuid.UUID) ([]models.Space, error) {
	return s.spaceRepository.FindByAccountID(accountID)
}

// GetRecentSpaces retrieves the user's recently visited spaces
func (s *SpaceService) GetRecentSpaces(userID uuid.UUID, accountID uuid.UUID) ([]models.Space, error) {
	// Get user's recent space IDs
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// If no recent spaces, return default space
	if len(user.RecentSpaces) == 0 {
		defaultSpace, err := s.spaceRepository.FindDefaultByAccountID(accountID)
		if err != nil {
			return []models.Space{}, nil
		}
		return []models.Space{*defaultSpace}, nil
	}

	// Fetch spaces by IDs and filter by account
	spaces := []models.Space{}
	for _, spaceID := range user.RecentSpaces {
		space, err := s.spaceRepository.FindByID(spaceID)
		if err != nil {
			continue // Skip if space not found
		}
		// Only include spaces from the current account
		if space.AccountID == accountID {
			spaces = append(spaces, *space)
		}
	}

	return spaces, nil
}

// UpdateSpace updates a space
func (s *SpaceService) UpdateSpace(spaceID uuid.UUID, accountID uuid.UUID, dto *dtos.UpdateSpaceDTO) (*models.Space, error) {
	space, err := s.GetSpaceByID(spaceID, accountID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if dto.Name != nil {
		space.Name = *dto.Name
	}

	space.UpdatedAt = time.Now()

	if err := s.spaceRepository.Update(space); err != nil {
		return nil, err
	}

	return space, nil
}

// DeleteSpace deletes a space
func (s *SpaceService) DeleteSpace(spaceID uuid.UUID, accountID uuid.UUID) error {
	space, err := s.GetSpaceByID(spaceID, accountID)
	if err != nil {
		return err
	}

	// Prevent deletion of default space
	if space.IsDefault {
		return fmt.Errorf("cannot delete default space")
	}

	// Check if it's the last space
	count, err := s.spaceRepository.CountByAccountID(accountID)
	if err != nil {
		return err
	}
	if count <= 1 {
		return fmt.Errorf("cannot delete the last space")
	}

	return s.spaceRepository.Delete(spaceID)
}

// RecordSpaceVisit records a visit to a space in the user's recent spaces
func (s *SpaceService) RecordSpaceVisit(userID uuid.UUID, spaceID uuid.UUID) error {
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Remove space if it already exists in the list
	newRecentSpaces := []uuid.UUID{}
	for _, id := range user.RecentSpaces {
		if id != spaceID {
			newRecentSpaces = append(newRecentSpaces, id)
		}
	}

	// Add space to the beginning
	newRecentSpaces = append([]uuid.UUID{spaceID}, newRecentSpaces...)

	// Keep only the last 5 spaces
	if len(newRecentSpaces) > 5 {
		newRecentSpaces = newRecentSpaces[:5]
	}

	return s.userRepository.UpdateRecentSpaces(userID, newRecentSpaces)
}
