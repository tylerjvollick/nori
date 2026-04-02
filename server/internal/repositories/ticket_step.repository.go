package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TicketStepRepository struct {
	db *gorm.DB
}

func NewTicketStepRepository(db *gorm.DB) *TicketStepRepository {
	return &TicketStepRepository{db: db}
}

// Create inserts a new ticket step.
func (r *TicketStepRepository) Create(step *models.TicketStep) error {
	return r.db.Create(step).Error
}

// CreateBatch inserts multiple ticket steps at once.
func (r *TicketStepRepository) CreateBatch(steps []models.TicketStep) error {
	if len(steps) == 0 {
		return nil
	}
	return r.db.Create(&steps).Error
}

// CreateBatchWithTx inserts multiple ticket steps within an existing transaction.
func (r *TicketStepRepository) CreateBatchWithTx(tx *gorm.DB, steps []models.TicketStep) error {
	if len(steps) == 0 {
		return nil
	}
	return tx.Create(&steps).Error
}

// GetByID returns a single ticket step by its primary key.
func (r *TicketStepRepository) GetByID(id uuid.UUID) (*models.TicketStep, error) {
	var step models.TicketStep
	err := r.db.First(&step, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket step not found")
		}
		return nil, err
	}
	return &step, nil
}

// GetByTicketID returns all steps for a ticket, ordered by display_order.
func (r *TicketStepRepository) GetByTicketID(ticketID uuid.UUID) ([]models.TicketStep, error) {
	var steps []models.TicketStep
	err := r.db.Where("ticket_id = ?", ticketID).
		Order("display_order ASC").
		Find(&steps).Error
	return steps, err
}

// GetByTicketIDWithSubSteps returns all steps for a ticket with sub-steps preloaded.
func (r *TicketStepRepository) GetByTicketIDWithSubSteps(ticketID uuid.UUID) ([]models.TicketStep, error) {
	var steps []models.TicketStep
	err := r.db.Where("ticket_id = ?", ticketID).
		Preload("SubSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Order("display_order ASC").
		Find(&steps).Error
	return steps, err
}

// Update saves changes to an existing ticket step.
func (r *TicketStepRepository) Update(step *models.TicketStep) error {
	return r.db.Save(step).Error
}

// Delete removes a ticket step by ID.
func (r *TicketStepRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.TicketStep{}, "id = ?", id).Error
}

// DeleteByTicketID removes all steps for a given ticket.
func (r *TicketStepRepository) DeleteByTicketID(ticketID uuid.UUID) error {
	return r.db.Where("ticket_id = ?", ticketID).Delete(&models.TicketStep{}).Error
}

// Reorder updates the display_order of all steps for a given ticket.
// The ids slice defines the new order — ids[0] gets display_order 0, etc.
func (r *TicketStepRepository) Reorder(ticketID uuid.UUID, ids []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			err := tx.Model(&models.TicketStep{}).
				Where("id = ? AND ticket_id = ?", id, ticketID).
				Update("display_order", i).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMaxDisplayOrder returns the highest display_order for steps in a given ticket.
func (r *TicketStepRepository) GetMaxDisplayOrder(ticketID uuid.UUID) (int, error) {
	var maxOrder int
	err := r.db.Model(&models.TicketStep{}).
		Where("ticket_id = ?", ticketID).
		Select("COALESCE(MAX(display_order), -1)").
		Scan(&maxOrder).Error
	return maxOrder, err
}

// CopyFromSOPVersion copies all SOPSteps (and their SOPSubSteps) from the given
// SOP version into TicketSteps (and TicketSubSteps) for the specified ticket.
// This is used when a ticket is created with a linked SOP.
func (r *TicketStepRepository) CopyFromSOPVersion(ticketID uuid.UUID, versionID int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Fetch SOP steps with sub-steps
		var sopSteps []models.SOPStep
		err := tx.Where("sop_version_id = ?", versionID).
			Preload("SubSteps", func(db *gorm.DB) *gorm.DB {
				return db.Order("display_order ASC")
			}).
			Order(`"order" ASC`).
			Find(&sopSteps).Error
		if err != nil {
			return fmt.Errorf("fetching SOP steps: %w", err)
		}

		for i, sopStep := range sopSteps {
			ticketStep := models.TicketStep{
				TicketID:     ticketID,
				SOPStepID:    &sopStep.ID,
				StationID:    sopStep.StationID,
				DisplayOrder: i,
				Title:        sopStep.Title,
				Instructions: sopStep.Instructions,
				Status:       models.TicketStepStatusPending,
			}

			if err := tx.Create(&ticketStep).Error; err != nil {
				return fmt.Errorf("creating ticket step %d: %w", i, err)
			}

			// Copy sub-steps
			for j, sopSubStep := range sopStep.SubSteps {
				subStepID := sopSubStep.ID
				ticketSubStep := models.TicketSubStep{
					TicketStepID: ticketStep.ID,
					SOPSubStepID: &subStepID,
					DisplayOrder: j,
					Title:        sopSubStep.Title,
					Instructions: sopSubStep.Instructions,
					IsCompleted:  false,
				}
				if err := tx.Create(&ticketSubStep).Error; err != nil {
					return fmt.Errorf("creating ticket sub-step %d.%d: %w", i, j, err)
				}
			}
		}

		return nil
	})
}
