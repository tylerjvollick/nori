package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type ActivityEntryRepository struct {
	db *gorm.DB
}

func NewActivityEntryRepository(db *gorm.DB) *ActivityEntryRepository {
	return &ActivityEntryRepository{db: db}
}

// Create inserts a new activity entry.
func (r *ActivityEntryRepository) Create(entry *models.ActivityEntry) error {
	return r.db.Create(entry).Error
}

// GetByID returns a single activity entry by its primary key.
func (r *ActivityEntryRepository) GetByID(id uuid.UUID) (*models.ActivityEntry, error) {
	var entry models.ActivityEntry
	err := r.db.First(&entry, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("activity entry not found")
		}
		return nil, err
	}
	return &entry, nil
}

// GetByTaskID returns all activity entries for a task, ordered chronologically.
func (r *ActivityEntryRepository) GetByTaskID(taskID string) ([]models.ActivityEntry, error) {
	var entries []models.ActivityEntry
	err := r.db.Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&entries).Error
	return entries, err
}

// Delete removes an activity entry by ID.
func (r *ActivityEntryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.ActivityEntry{}, "id = ?", id).Error
}
