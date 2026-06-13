package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// TaskMediaRepository handles database operations for TaskMedia records.
type TaskMediaRepository struct {
	db *gorm.DB
}

// NewTaskMediaRepository creates a new TaskMediaRepository.
func NewTaskMediaRepository(db *gorm.DB) *TaskMediaRepository {
	return &TaskMediaRepository{db: db}
}

// WithTx returns a new TaskMediaRepository that uses the given transaction handle.
func (r *TaskMediaRepository) WithTx(tx *gorm.DB) *TaskMediaRepository {
	return &TaskMediaRepository{db: tx}
}

// Create inserts a new TaskMedia record.
func (r *TaskMediaRepository) Create(m *models.TaskMedia) error {
	return r.db.Create(m).Error
}

// GetByID returns the TaskMedia with the given UUID, or an error if not found.
func (r *TaskMediaRepository) GetByID(id uuid.UUID) (*models.TaskMedia, error) {
	var m models.TaskMedia
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// ListByTaskID returns all media for the given task, ordered by display_order asc.
func (r *TaskMediaRepository) ListByTaskID(taskID string) ([]models.TaskMedia, error) {
	var media []models.TaskMedia
	err := r.db.Where("task_id = ?", taskID).
		Order("display_order asc, created_at asc").
		Find(&media).Error
	return media, err
}

// GetMaxDisplayOrder returns the current maximum display_order for media of the
// given task, or -1 if there is no media yet.
func (r *TaskMediaRepository) GetMaxDisplayOrder(taskID string) (int, error) {
	var max *int
	err := r.db.Model(&models.TaskMedia{}).
		Select("MAX(display_order)").
		Where("task_id = ?", taskID).
		Scan(&max).Error
	if err != nil {
		return -1, err
	}
	if max == nil {
		return -1, nil
	}
	return *max, nil
}

// Delete removes the TaskMedia with the given UUID.
func (r *TaskMediaRepository) Delete(id uuid.UUID) error {
	result := r.db.Where("id = ?", id).Delete(&models.TaskMedia{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task media %q not found", id)
	}
	return nil
}
