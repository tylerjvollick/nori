package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type StatusDefinitionRepository struct {
	db *gorm.DB
}

func NewStatusDefinitionRepository(db *gorm.DB) *StatusDefinitionRepository {
	return &StatusDefinitionRepository{db: db}
}

func (r *StatusDefinitionRepository) Create(status *models.StatusDefinition) error {
	return r.db.Create(status).Error
}

func (r *StatusDefinitionRepository) GetByID(id uuid.UUID) (*models.StatusDefinition, error) {
	var status models.StatusDefinition
	err := r.db.First(&status, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("status definition not found")
		}
		return nil, err
	}
	return &status, nil
}

func (r *StatusDefinitionRepository) GetByTicketTypeID(ticketTypeID uuid.UUID) ([]models.StatusDefinition, error) {
	var statuses []models.StatusDefinition
	err := r.db.Where("ticket_type_id = ?", ticketTypeID).
		Order("display_order ASC").
		Find(&statuses).Error
	return statuses, err
}

func (r *StatusDefinitionRepository) GetDefaultByTicketTypeID(ticketTypeID uuid.UUID) (*models.StatusDefinition, error) {
	var status models.StatusDefinition
	err := r.db.Where("ticket_type_id = ? AND is_default = true", ticketTypeID).
		First(&status).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no default status defined for ticket type")
		}
		return nil, err
	}
	return &status, nil
}

func (r *StatusDefinitionRepository) Update(status *models.StatusDefinition) error {
	return r.db.Save(status).Error
}

func (r *StatusDefinitionRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.StatusDefinition{}, "id = ?", id).Error
}

// Reorder updates the display_order of all status definitions for a given ticket type.
// The ids slice defines the new order — ids[0] gets display_order 0, etc.
func (r *StatusDefinitionRepository) Reorder(ticketTypeID uuid.UUID, ids []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			err := tx.Model(&models.StatusDefinition{}).
				Where("id = ? AND ticket_type_id = ?", id, ticketTypeID).
				Update("display_order", i).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMaxDisplayOrder returns the highest display_order for status definitions in a given ticket type.
func (r *StatusDefinitionRepository) GetMaxDisplayOrder(ticketTypeID uuid.UUID) (int, error) {
	var maxOrder int
	err := r.db.Model(&models.StatusDefinition{}).
		Where("ticket_type_id = ?", ticketTypeID).
		Select("COALESCE(MAX(display_order), -1)").
		Scan(&maxOrder).Error
	return maxOrder, err
}

// HasTerminal checks whether at least one terminal status exists for the given ticket type.
func (r *StatusDefinitionRepository) HasTerminal(ticketTypeID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.StatusDefinition{}).
		Where("ticket_type_id = ? AND is_terminal = true", ticketTypeID).
		Count(&count).Error
	return count > 0, err
}

// SetDefault clears the current default and sets the given status as the default for its ticket type.
// This runs in a transaction to maintain the one-default invariant.
func (r *StatusDefinitionRepository) SetDefault(ticketTypeID uuid.UUID, statusID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Clear existing default
		err := tx.Model(&models.StatusDefinition{}).
			Where("ticket_type_id = ? AND is_default = true", ticketTypeID).
			Update("is_default", false).Error
		if err != nil {
			return err
		}
		// Set new default
		return tx.Model(&models.StatusDefinition{}).
			Where("id = ? AND ticket_type_id = ?", statusID, ticketTypeID).
			Update("is_default", true).Error
	})
}
