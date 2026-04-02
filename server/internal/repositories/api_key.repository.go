package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create creates a new API key
func (r *APIKeyRepository) Create(apiKey *models.APIKey) error {
	return r.db.Create(apiKey).Error
}

// GetByKeyHash retrieves an API key by its hash
func (r *APIKeyRepository) GetByKeyHash(keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := r.db.Where("key_hash = ?", keyHash).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// GetByAccount retrieves all API keys for a given account
func (r *APIKeyRepository) GetByAccount(accountID uuid.UUID) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	err := r.db.
		Where("account_id = ?", accountID).
		Order("created_at DESC").
		Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

// Deactivate sets the IsActive flag to false for an API key
func (r *APIKeyRepository) Deactivate(id uuid.UUID) error {
	return r.db.Model(&models.APIKey{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// Delete permanently removes an API key
func (r *APIKeyRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.APIKey{}).Error
}

// UpdateLastUsed updates the LastUsedAt timestamp for an API key
func (r *APIKeyRepository) UpdateLastUsed(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
}
