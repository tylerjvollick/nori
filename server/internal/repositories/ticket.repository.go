package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

// Create inserts a new ticket. If TicketNumber is empty, it is auto-generated
// from the ticket type name and a per-type sequence counter.
func (r *TicketRepository) Create(ticket *models.Ticket) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if ticket.TicketNumber == "" {
			number, err := r.nextTicketNumber(tx, ticket.TicketTypeID)
			if err != nil {
				return fmt.Errorf("generating ticket number: %w", err)
			}
			ticket.TicketNumber = number
		}
		return tx.Create(ticket).Error
	})
}

// GetByID returns a single ticket by its primary key.
func (r *TicketRepository) GetByID(id uuid.UUID) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.First(&ticket, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, err
	}
	return &ticket, nil
}

// GetByIDWithRelations returns a ticket with its common relations preloaded.
func (r *TicketRepository) GetByIDWithRelations(id uuid.UUID) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.
		Preload("TicketType").
		Preload("Status").
		Preload("Customer").
		Preload("AssignedTo").
		Preload("CreatedBy").
		Preload("Tags").
		First(&ticket, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, err
	}
	return &ticket, nil
}

// TicketFilter holds optional filter criteria for listing tickets.
type TicketFilter struct {
	TicketTypeID   *uuid.UUID
	StatusID       *uuid.UUID
	ParentTicketID *uuid.UUID
	CustomerID     *uuid.UUID
	AssignedToID   *uuid.UUID
	IsTopLevel     bool // if true, only tickets with no parent
}

// GetBySpaceID returns tickets for a space, with optional filters.
func (r *TicketRepository) GetBySpaceID(spaceID uuid.UUID, filter *TicketFilter) ([]models.Ticket, error) {
	query := r.db.Where("space_id = ?", spaceID)

	if filter != nil {
		if filter.TicketTypeID != nil {
			query = query.Where("ticket_type_id = ?", *filter.TicketTypeID)
		}
		if filter.StatusID != nil {
			query = query.Where("status_id = ?", *filter.StatusID)
		}
		if filter.ParentTicketID != nil {
			query = query.Where("parent_ticket_id = ?", *filter.ParentTicketID)
		}
		if filter.CustomerID != nil {
			query = query.Where("customer_id = ?", *filter.CustomerID)
		}
		if filter.AssignedToID != nil {
			query = query.Where("assigned_to_id = ?", *filter.AssignedToID)
		}
		if filter.IsTopLevel {
			query = query.Where("parent_ticket_id IS NULL")
		}
	}

	var tickets []models.Ticket
	err := query.Order("priority ASC, created_at DESC").Find(&tickets).Error
	return tickets, err
}

// GetChildrenByParentID returns the child tickets of a given parent ticket.
func (r *TicketRepository) GetChildrenByParentID(parentID uuid.UUID) ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.Where("parent_ticket_id = ?", parentID).
		Order("priority ASC, created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

// Update saves changes to an existing ticket.
func (r *TicketRepository) Update(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}

// Delete removes a ticket by ID.
func (r *TicketRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Ticket{}, "id = ?", id).Error
}

// GetByTicketNumber returns a ticket by its human-readable ticket number.
func (r *TicketRepository) GetByTicketNumber(ticketNumber string) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.First(&ticket, "ticket_number = ?", ticketNumber).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ticket not found")
		}
		return nil, err
	}
	return &ticket, nil
}

// nextTicketNumber generates the next ticket number for a given ticket type.
// Format: "{TYPE_NAME_PREFIX}-{sequence}", e.g. "BUILD-001".
// Uses the ticket_number_sequence table with row-level locking to prevent duplicates.
func (r *TicketRepository) nextTicketNumber(tx *gorm.DB, ticketTypeID uuid.UUID) (string, error) {
	// Upsert the sequence row and increment atomically
	var lastNumber int
	err := tx.Raw(`
		INSERT INTO ticket_number_sequence (ticket_type_id, last_number)
		VALUES (?, 1)
		ON CONFLICT (ticket_type_id) DO UPDATE
		SET last_number = ticket_number_sequence.last_number + 1
		RETURNING last_number
	`, ticketTypeID).Scan(&lastNumber).Error
	if err != nil {
		return "", fmt.Errorf("incrementing sequence: %w", err)
	}

	// Look up the ticket type name to build the prefix
	var ticketType models.TicketType
	err = tx.Select("name").First(&ticketType, "id = ?", ticketTypeID).Error
	if err != nil {
		return "", fmt.Errorf("looking up ticket type: %w", err)
	}

	prefix := buildPrefix(ticketType.Name)
	return fmt.Sprintf("%s-%03d", prefix, lastNumber), nil
}

// buildPrefix creates a short uppercase prefix from a ticket type name.
// Examples: "Build" -> "BUILD", "Sales Lead" -> "SALES-LEAD", "3S" -> "3S".
func buildPrefix(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z':
			result = append(result, c-32) // to uppercase
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			result = append(result, c)
		case c == ' ' || c == '_':
			result = append(result, '-')
		}
	}
	return string(result)
}
