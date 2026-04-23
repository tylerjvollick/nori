package repositories

import (
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SpaceRepository struct {
	db *gorm.DB
}

func NewSpaceRepository(db *gorm.DB) *SpaceRepository {
	return &SpaceRepository{db: db}
}

// Create creates a new space
func (r *SpaceRepository) Create(space *models.Space) error {
	return r.db.Create(space).Error
}

// FindByID retrieves a space by its ID
func (r *SpaceRepository) FindByID(id uuid.UUID) (*models.Space, error) {
	var space models.Space
	err := r.db.Where("id = ?", id).First(&space).Error
	if err != nil {
		return nil, err
	}
	return &space, nil
}

// FindByAccountID retrieves all spaces for a given account
func (r *SpaceRepository) FindByAccountID(accountID uuid.UUID) ([]models.Space, error) {
	var spaces []models.Space
	err := r.db.Where("account_id = ?", accountID).Order("is_default DESC, name ASC").Find(&spaces).Error
	if err != nil {
		return nil, err
	}
	return spaces, nil
}

// FindDefaultByAccountID retrieves the default space for an account
func (r *SpaceRepository) FindDefaultByAccountID(accountID uuid.UUID) (*models.Space, error) {
	var space models.Space
	err := r.db.Where("account_id = ? AND is_default = ?", accountID, true).First(&space).Error
	if err != nil {
		return nil, err
	}
	return &space, nil
}

// Update updates a space
func (r *SpaceRepository) Update(space *models.Space) error {
	return r.db.Save(space).Error
}

// Delete deletes a space by ID
func (r *SpaceRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Space{}, "id = ?", id).Error
}

// CountByAccountID counts spaces for an account
func (r *SpaceRepository) CountByAccountID(accountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Space{}).Where("account_id = ?", accountID).Count(&count).Error
	return count, err
}

// FindBySlugAndAccountID retrieves a space by slug within an account
func (r *SpaceRepository) FindBySlugAndAccountID(slug string, accountID uuid.UUID) (*models.Space, error) {
	var space models.Space
	err := r.db.Where("slug = ? AND account_id = ?", slug, accountID).First(&space).Error
	if err != nil {
		return nil, err
	}
	return &space, nil
}

// SlugExistsInAccount checks if a slug already exists for the given account
func (r *SpaceRepository) SlugExistsInAccount(slug string, accountID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.Space{}).Where("slug = ? AND account_id = ?", slug, accountID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
