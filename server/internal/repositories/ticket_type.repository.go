package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TicketTypeRepository struct {
	db *gorm.DB
}

func NewTicketTypeRepository(db *gorm.DB) *TicketTypeRepository {
	return &TicketTypeRepository{db: db}
}

func (r *TicketTypeRepository) Create(ticketType *models.TicketType) error {
	return r.db.Create(ticketType).Error
}

func (r *TicketTypeRepository) GetByID(id uuid.UUID) (*models.TicketType, error) {
	var ticketType models.TicketType
	err := r.db.First(&ticketType, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket type not found")
		}
		return nil, err
	}
	return &ticketType, nil
}

func (r *TicketTypeRepository) GetBySpaceID(spaceID uuid.UUID) ([]models.TicketType, error) {
	var ticketTypes []models.TicketType
	err := r.db.Where("space_id = ?", spaceID).
		Order("display_order ASC").
		Find(&ticketTypes).Error
	return ticketTypes, err
}

func (r *TicketTypeRepository) Update(ticketType *models.TicketType) error {
	return r.db.Save(ticketType).Error
}

func (r *TicketTypeRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.TicketType{}, "id = ?", id).Error
}

// Reorder updates the display_order of all ticket types for a given space.
// The ids slice defines the new order — ids[0] gets display_order 0, etc.
func (r *TicketTypeRepository) Reorder(spaceID uuid.UUID, ids []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			err := tx.Model(&models.TicketType{}).
				Where("id = ? AND space_id = ?", id, spaceID).
				Update("display_order", i).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetMaxDisplayOrder returns the highest display_order for ticket types in a given space.
func (r *TicketTypeRepository) GetMaxDisplayOrder(spaceID uuid.UUID) (int, error) {
	var maxOrder int
	err := r.db.Model(&models.TicketType{}).
		Where("space_id = ?", spaceID).
		Select("COALESCE(MAX(display_order), -1)").
		Scan(&maxOrder).Error
	return maxOrder, err
}
