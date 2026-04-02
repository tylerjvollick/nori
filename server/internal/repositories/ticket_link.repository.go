package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TicketLinkRepository struct {
	db *gorm.DB
}

func NewTicketLinkRepository(db *gorm.DB) *TicketLinkRepository {
	return &TicketLinkRepository{db: db}
}

// Create inserts a new ticket link.
func (r *TicketLinkRepository) Create(link *models.TicketLink) error {
	return r.db.Create(link).Error
}

// GetByID returns a single ticket link by its primary key.
func (r *TicketLinkRepository) GetByID(id uuid.UUID) (*models.TicketLink, error) {
	var link models.TicketLink
	err := r.db.First(&link, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket link not found")
		}
		return nil, err
	}
	return &link, nil
}

// GetByTicketID returns all links where the given ticket is either the source
// or the target. This supports cross-space links — no SpaceID filtering.
func (r *TicketLinkRepository) GetByTicketID(ticketID uuid.UUID) ([]models.TicketLink, error) {
	var links []models.TicketLink
	err := r.db.Where("source_ticket_id = ? OR target_ticket_id = ?", ticketID, ticketID).
		Order("created_at ASC").
		Find(&links).Error
	return links, err
}

// GetBySourceTicketID returns all links originating from the given ticket.
func (r *TicketLinkRepository) GetBySourceTicketID(ticketID uuid.UUID) ([]models.TicketLink, error) {
	var links []models.TicketLink
	err := r.db.Where("source_ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&links).Error
	return links, err
}

// GetByTargetTicketID returns all links pointing to the given ticket.
func (r *TicketLinkRepository) GetByTargetTicketID(ticketID uuid.UUID) ([]models.TicketLink, error) {
	var links []models.TicketLink
	err := r.db.Where("target_ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&links).Error
	return links, err
}

// Delete removes a ticket link by ID.
func (r *TicketLinkRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.TicketLink{}, "id = ?", id).Error
}

// DeleteByTicketID removes all links where the given ticket is source or target.
func (r *TicketLinkRepository) DeleteByTicketID(ticketID uuid.UUID) error {
	return r.db.Where("source_ticket_id = ? OR target_ticket_id = ?", ticketID, ticketID).
		Delete(&models.TicketLink{}).Error
}
