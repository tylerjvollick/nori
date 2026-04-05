package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type CostEntryRepository struct {
	db *gorm.DB
}

func NewCostEntryRepository(db *gorm.DB) *CostEntryRepository {
	return &CostEntryRepository{db: db}
}

// Create inserts a new cost entry.
func (r *CostEntryRepository) Create(entry *models.CostEntry) error {
	return r.db.Create(entry).Error
}

// GetByID returns a single cost entry by its primary key.
func (r *CostEntryRepository) GetByID(id uuid.UUID) (*models.CostEntry, error) {
	var entry models.CostEntry
	err := r.db.First(&entry, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cost entry not found")
		}
		return nil, err
	}
	return &entry, nil
}

// Delete removes a cost entry by ID.
func (r *CostEntryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.CostEntry{}, "id = ?", id).Error
}
