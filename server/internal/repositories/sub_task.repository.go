package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// SubTaskRepository handles database operations for SubTask records.
type SubTaskRepository struct {
	db *gorm.DB
}

// NewSubTaskRepository creates a new SubTaskRepository.
func NewSubTaskRepository(db *gorm.DB) *SubTaskRepository {
	return &SubTaskRepository{db: db}
}

// WithTx returns a new SubTaskRepository that uses the given transaction handle.
func (r *SubTaskRepository) WithTx(tx *gorm.DB) *SubTaskRepository {
	return &SubTaskRepository{db: tx}
}

// Create inserts a new SubTask record. If DisplayOrder is not set, the sub-task
// is appended after the current maximum display order for the parent task.
func (r *SubTaskRepository) Create(s *models.SubTask) error {
	return r.db.Create(s).Error
}

// GetByID returns the SubTask with the given UUID, or an error if not found.
func (r *SubTaskRepository) GetByID(id uuid.UUID) (*models.SubTask, error) {
	var s models.SubTask
	err := r.db.Where("id = ?", id).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListByTaskID returns all sub-tasks for the given task, ordered by display_order asc.
func (r *SubTaskRepository) ListByTaskID(taskID string) ([]models.SubTask, error) {
	var subtasks []models.SubTask
	err := r.db.Where("task_id = ?", taskID).Order("display_order asc").Find(&subtasks).Error
	return subtasks, err
}

// GetMaxDisplayOrder returns the current maximum display_order for subtasks of
// the given task, or -1 if there are no sub-tasks yet.
func (r *SubTaskRepository) GetMaxDisplayOrder(taskID string) (int, error) {
	var max *int
	err := r.db.Model(&models.SubTask{}).
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

// Update applies non-nil fields in the update map to the sub-task record.
func (r *SubTaskRepository) Update(id uuid.UUID, updates map[string]interface{}) error {
	result := r.db.Model(&models.SubTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("sub-task %q not found", id)
	}
	return nil
}

// Delete removes the SubTask with the given UUID.
func (r *SubTaskRepository) Delete(id uuid.UUID) error {
	result := r.db.Where("id = ?", id).Delete(&models.SubTask{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("sub-task %q not found", id)
	}
	return nil
}

// Reorder updates display_order for each sub-task in the given ordered ID slice.
// All IDs must belong to the same task; the caller is responsible for validation.
func (r *SubTaskRepository) Reorder(taskID string, ids []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			result := tx.Model(&models.SubTask{}).
				Where("id = ? AND task_id = ?", id, taskID).
				Update("display_order", i)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("sub-task %q not found in task %q", id, taskID)
			}
		}
		return nil
	})
}
