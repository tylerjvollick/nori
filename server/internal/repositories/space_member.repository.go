package repositories

import (
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SpaceMemberRepository struct {
	db *gorm.DB
}

func NewSpaceMemberRepository(db *gorm.DB) *SpaceMemberRepository {
	return &SpaceMemberRepository{db: db}
}

// Create creates a new space member
func (r *SpaceMemberRepository) Create(spaceMember *models.SpaceMember) error {
	return r.db.Create(spaceMember).Error
}

// Delete deletes a space member by user ID and space ID
func (r *SpaceMemberRepository) Delete(userID, spaceID uuid.UUID) error {
	return r.db.Where("user_id = ? AND space_id = ?", userID, spaceID).Delete(&models.SpaceMember{}).Error
}

// GetByUserAndSpace retrieves a space member by user ID and space ID
func (r *SpaceMemberRepository) GetByUserAndSpace(userID, spaceID uuid.UUID) (*models.SpaceMember, error) {
	var spaceMember models.SpaceMember
	err := r.db.Where("user_id = ? AND space_id = ?", userID, spaceID).First(&spaceMember).Error
	if err != nil {
		return nil, err
	}
	return &spaceMember, nil
}

// GetByUser retrieves all spaces for a given user
func (r *SpaceMemberRepository) GetByUser(userID uuid.UUID) ([]models.Space, error) {
	var spaces []models.Space
	err := r.db.
		Joins("INNER JOIN space_member ON space_member.space_id = spaces.id").
		Where("space_member.user_id = ?", userID).
		Order("space_member.created_at DESC").
		Find(&spaces).Error
	if err != nil {
		return nil, err
	}
	return spaces, nil
}

// GetBySpace retrieves all members of a given space
func (r *SpaceMemberRepository) GetBySpace(spaceID uuid.UUID) ([]models.SpaceMember, error) {
	var members []models.SpaceMember
	err := r.db.
		Preload("User").
		Where("space_id = ?", spaceID).
		Order("created_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}
