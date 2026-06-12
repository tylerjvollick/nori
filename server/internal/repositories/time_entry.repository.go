package repositories

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/models"
)

type TimeEntryRepository struct {
	db *gorm.DB
}

func NewTimeEntryRepository(db *gorm.DB) *TimeEntryRepository {
	return &TimeEntryRepository{db: db}
}

func (r *TimeEntryRepository) Create(entry *models.TimeEntry) error {
	return r.db.Create(entry).Error
}

func (r *TimeEntryRepository) GetByID(id uuid.UUID) (*models.TimeEntry, error) {
	var entry models.TimeEntry
	if err := r.db.Where("id = ?", id).First(&entry).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *TimeEntryRepository) Update(entry *models.TimeEntry) error {
	return r.db.Save(entry).Error
}

func (r *TimeEntryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.TimeEntry{}, "id = ?", id).Error
}

// GetByTaskID returns all time entries for a task, ordered by started_at.
func (r *TimeEntryRepository) GetByTaskID(taskID string) ([]models.TimeEntry, error) {
	var entries []models.TimeEntry
	if err := r.db.Where("task_id = ?", taskID).Order("started_at ASC").Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// GetRunningEntry returns the active (ended_at IS NULL) time entry for a task, or nil.
func (r *TimeEntryRepository) GetRunningEntry(taskID string) (*models.TimeEntry, error) {
	var entry models.TimeEntry
	err := r.db.Where("task_id = ? AND ended_at IS NULL", taskID).First(&entry).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetCompletedTasksWithNoEntries returns completed tasks in a space that have
// zero time entries (for manager review of unlogged work).
func (r *TimeEntryRepository) GetCompletedTasksWithNoEntries(spaceID uuid.UUID) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.
		Where("space_id = ? AND status = ?", spaceID, models.TaskStatusDone).
		Where("id NOT IN (SELECT DISTINCT task_id FROM time_entry WHERE space_id = ?)", spaceID).
		Find(&tasks).Error
	return tasks, err
}
