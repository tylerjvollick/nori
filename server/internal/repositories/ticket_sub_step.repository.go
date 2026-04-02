package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TicketSubStepRepository struct {
	db *gorm.DB
}

func NewTicketSubStepRepository(db *gorm.DB) *TicketSubStepRepository {
	return &TicketSubStepRepository{db: db}
}

// Create inserts a new ticket sub-step.
func (r *TicketSubStepRepository) Create(subStep *models.TicketSubStep) error {
	return r.db.Create(subStep).Error
}

// CreateBatch inserts multiple ticket sub-steps at once.
func (r *TicketSubStepRepository) CreateBatch(subSteps []models.TicketSubStep) error {
	if len(subSteps) == 0 {
		return nil
	}
	return r.db.Create(&subSteps).Error
}

// GetByID returns a single ticket sub-step by its primary key.
func (r *TicketSubStepRepository) GetByID(id uuid.UUID) (*models.TicketSubStep, error) {
	var subStep models.TicketSubStep
	err := r.db.First(&subStep, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket sub-step not found")
		}
		return nil, err
	}
	return &subStep, nil
}

// GetByTicketStepID returns all sub-steps for a ticket step, ordered by display_order.
func (r *TicketSubStepRepository) GetByTicketStepID(ticketStepID uuid.UUID) ([]models.TicketSubStep, error) {
	var subSteps []models.TicketSubStep
	err := r.db.Where("ticket_step_id = ?", ticketStepID).
		Order("display_order ASC").
		Find(&subSteps).Error
	return subSteps, err
}

// Update saves changes to an existing ticket sub-step.
func (r *TicketSubStepRepository) Update(subStep *models.TicketSubStep) error {
	return r.db.Save(subStep).Error
}

// Delete removes a ticket sub-step by ID.
func (r *TicketSubStepRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.TicketSubStep{}, "id = ?", id).Error
}

// DeleteByTicketStepID removes all sub-steps for a given ticket step.
func (r *TicketSubStepRepository) DeleteByTicketStepID(ticketStepID uuid.UUID) error {
	return r.db.Where("ticket_step_id = ?", ticketStepID).Delete(&models.TicketSubStep{}).Error
}

// ToggleComplete flips the is_completed flag on a ticket sub-step.
func (r *TicketSubStepRepository) ToggleComplete(id uuid.UUID) (*models.TicketSubStep, error) {
	var subStep models.TicketSubStep
	err := r.db.First(&subStep, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket sub-step not found")
		}
		return nil, err
	}

	subStep.IsCompleted = !subStep.IsCompleted
	if err := r.db.Save(&subStep).Error; err != nil {
		return nil, err
	}
	return &subStep, nil
}

// Reorder updates the display_order of all sub-steps for a given ticket step.
// The ids slice defines the new order — ids[0] gets display_order 0, etc.
func (r *TicketSubStepRepository) Reorder(ticketStepID uuid.UUID, ids []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			err := tx.Model(&models.TicketSubStep{}).
				Where("id = ? AND ticket_step_id = ?", id, ticketStepID).
				Update("display_order", i).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMaxDisplayOrder returns the highest display_order for sub-steps in a given ticket step.
func (r *TicketSubStepRepository) GetMaxDisplayOrder(ticketStepID uuid.UUID) (int, error) {
	var maxOrder int
	err := r.db.Model(&models.TicketSubStep{}).
		Where("ticket_step_id = ?", ticketStepID).
		Select("COALESCE(MAX(display_order), -1)").
		Scan(&maxOrder).Error
	return maxOrder, err
}
