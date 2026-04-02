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

// GetByTicketID returns all activity entries for a ticket in chronological order.
func (r *ActivityEntryRepository) GetByTicketID(ticketID uuid.UUID) ([]models.ActivityEntry, error) {
	var entries []models.ActivityEntry
	err := r.db.Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&entries).Error
	return entries, err
}

// GetByTicketStepID returns all activity entries for a specific ticket step.
func (r *ActivityEntryRepository) GetByTicketStepID(ticketStepID uuid.UUID) ([]models.ActivityEntry, error) {
	var entries []models.ActivityEntry
	err := r.db.Where("ticket_step_id = ?", ticketStepID).
		Order("created_at ASC").
		Find(&entries).Error
	return entries, err
}

// GetByUserID returns all activity entries created by a specific user.
func (r *ActivityEntryRepository) GetByUserID(userID uuid.UUID) ([]models.ActivityEntry, error) {
	var entries []models.ActivityEntry
	err := r.db.Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&entries).Error
	return entries, err
}

// GetByTicketIDAndType returns activity entries for a ticket filtered by entry type.
func (r *ActivityEntryRepository) GetByTicketIDAndType(ticketID uuid.UUID, entryType models.ActivityEntryType) ([]models.ActivityEntry, error) {
	var entries []models.ActivityEntry
	err := r.db.Where("ticket_id = ? AND entry_type = ?", ticketID, entryType).
		Order("created_at ASC").
		Find(&entries).Error
	return entries, err
}

// Delete removes an activity entry by ID.
func (r *ActivityEntryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.ActivityEntry{}, "id = ?", id).Error
}

// DeleteByTicketID removes all activity entries for a ticket.
func (r *ActivityEntryRepository) DeleteByTicketID(ticketID uuid.UUID) error {
	return r.db.Where("ticket_id = ?", ticketID).
		Delete(&models.ActivityEntry{}).Error
}
