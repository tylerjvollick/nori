package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

// TicketLifecycleService manages the core lifecycle of tickets: creation with
// default status and SOP step copying, status transitions with validation,
// and automatic activity logging.
type TicketLifecycleService struct {
	db             *gorm.DB
	ticketRepo     *repositories.TicketRepository
	ticketTypeRepo *repositories.TicketTypeRepository
	statusDefRepo  *repositories.StatusDefinitionRepository
	ticketStepRepo *repositories.TicketStepRepository
	activitySvc    *ActivityLoggingService
	ticketSOPSvc   *TicketSOPService
}

func NewTicketLifecycleService(
	db *gorm.DB,
	ticketRepo *repositories.TicketRepository,
	ticketTypeRepo *repositories.TicketTypeRepository,
	statusDefRepo *repositories.StatusDefinitionRepository,
	ticketStepRepo *repositories.TicketStepRepository,
	activitySvc *ActivityLoggingService,
	ticketSOPSvc *TicketSOPService,
) *TicketLifecycleService {
	return &TicketLifecycleService{
		db:             db,
		ticketRepo:     ticketRepo,
		ticketTypeRepo: ticketTypeRepo,
		statusDefRepo:  statusDefRepo,
		ticketStepRepo: ticketStepRepo,
		activitySvc:    activitySvc,
		ticketSOPSvc:   ticketSOPSvc,
	}
}

// CreateTicketInput contains the parameters for creating a new ticket.
type CreateTicketInput struct {
	SpaceID        uuid.UUID
	TicketTypeID   uuid.UUID
	Title          string
	Description    *string
	ParentTicketID *uuid.UUID
	SOPTemplateID  *int
	CustomerID     *uuid.UUID
	AssignedToID   *uuid.UUID
	Priority       int
	DueDate        *time.Time
	CreatedByID    uuid.UUID
}

// CreateTicket creates a new ticket with the default status for its ticket type.
// If SOPTemplateID is provided, the current SOP version's steps are copied as
// TicketSteps. An activity entry is automatically logged for the creation.
func (s *TicketLifecycleService) CreateTicket(input CreateTicketInput) (*models.Ticket, error) {
	// 1. Validate the ticket type exists
	ticketType, err := s.ticketTypeRepo.GetByID(input.TicketTypeID)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket type: %w", err)
	}

	// 2. Get the default status for this ticket type
	defaultStatus, err := s.statusDefRepo.GetDefaultByTicketTypeID(input.TicketTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default status for ticket type %q: %w", ticketType.Name, err)
	}

	// 3. Validate parent ticket constraint (one level only)
	if input.ParentTicketID != nil {
		parent, err := s.ticketRepo.GetByID(*input.ParentTicketID)
		if err != nil {
			return nil, fmt.Errorf("parent ticket not found: %w", err)
		}
		if parent.ParentTicketID != nil {
			return nil, fmt.Errorf("cannot nest tickets more than one level: parent ticket %s already has a parent", parent.TicketNumber)
		}
	}

	// 4. Resolve the SOP template and current version via TicketSOPService
	sopResolution, err := s.ticketSOPSvc.ResolveSOPVersion(input.SOPTemplateID, ticketType)
	if err != nil {
		log.Println("Failed to resolve SOP version:", err)
		return nil, fmt.Errorf("failed to resolve SOP version: %w", err)
	}

	var sopTemplateID *int
	var sopVersionID *int
	if sopResolution != nil {
		sopTemplateID = &sopResolution.SOPTemplateID
		sopVersionID = &sopResolution.SOPVersionID
	}

	// 5. Create the ticket
	ticket := &models.Ticket{
		SpaceID:        input.SpaceID,
		TicketTypeID:   input.TicketTypeID,
		ParentTicketID: input.ParentTicketID,
		StatusID:       defaultStatus.ID,
		SOPTemplateID:  sopTemplateID,
		SOPVersionID:   sopVersionID,
		CustomerID:     input.CustomerID,
		AssignedToID:   input.AssignedToID,
		Title:          input.Title,
		Description:    input.Description,
		Priority:       input.Priority,
		DueDate:        input.DueDate,
		CreatedByID:    input.CreatedByID,
	}

	if err := s.ticketRepo.Create(ticket); err != nil {
		log.Println("Failed to create ticket:", err)
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// 6. Copy SOP steps into ticket steps if we have a version
	if sopVersionID != nil {
		if err := s.ticketSOPSvc.CopySOPStepsToTicketNonTx(ticket.ID, *sopVersionID); err != nil {
			log.Println("Failed to copy SOP steps to ticket:", err)
			return nil, fmt.Errorf("failed to copy SOP steps to ticket: %w", err)
		}
	}

	// 7. Log the creation activity
	s.activitySvc.LogTicketCreated(ticket.ID, input.CreatedByID, ticket.TicketNumber, sopTemplateID != nil, defaultStatus.Name)

	// 8. Reload the ticket with relations for the response
	loaded, err := s.ticketRepo.GetByIDWithRelations(ticket.ID)
	if err != nil {
		return ticket, nil // Return the ticket even if reload fails
	}
	return loaded, nil
}

// TransitionStatus changes a ticket's status to a new status, validating that
// the new status belongs to the same ticket type. An activity entry is
// automatically logged for the transition. If the new status is terminal,
// CompletedAt is set. If the status category changes to in_progress and
// StartedAt is nil, StartedAt is set.
func (s *TicketLifecycleService) TransitionStatus(
	ticketID uuid.UUID,
	newStatusID uuid.UUID,
	userID uuid.UUID,
) (*models.Ticket, error) {
	// 1. Load the ticket
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	// 2. Load the old status for activity logging
	oldStatus, err := s.statusDefRepo.GetByID(ticket.StatusID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current status: %w", err)
	}

	// 3. Load and validate the new status belongs to the same ticket type
	newStatus, err := s.statusDefRepo.GetByID(newStatusID)
	if err != nil {
		return nil, fmt.Errorf("invalid target status: %w", err)
	}

	if newStatus.TicketTypeID != ticket.TicketTypeID {
		return nil, fmt.Errorf(
			"status %q belongs to a different ticket type; expected type %s",
			newStatus.Name, ticket.TicketTypeID,
		)
	}

	// 4. No-op if the status hasn't changed
	if ticket.StatusID == newStatusID {
		return ticket, nil
	}

	// 5. Update the ticket's status and lifecycle timestamps
	ticket.StatusID = newStatusID
	now := time.Now()

	// Set StartedAt on first transition to an in_progress status
	if newStatus.Category == models.StatusCategoryInProgress && ticket.StartedAt == nil {
		ticket.StartedAt = &now
	}

	// Set CompletedAt when moving to a terminal status
	if newStatus.IsTerminal {
		ticket.CompletedAt = &now
	} else {
		// Clear CompletedAt if moving back from a terminal status
		ticket.CompletedAt = nil
	}

	if err := s.ticketRepo.Update(ticket); err != nil {
		log.Println("Failed to update ticket status:", err)
		return nil, fmt.Errorf("failed to update ticket status: %w", err)
	}

	// 6. Log the status change activity
	s.activitySvc.LogStatusChange(ticket.ID, userID, oldStatus.Name, newStatus.Name)

	// 7. Reload with relations
	loaded, err := s.ticketRepo.GetByIDWithRelations(ticket.ID)
	if err != nil {
		return ticket, nil
	}
	return loaded, nil
}

// statusChangeDescription generates a human-readable description for a status transition.
// Delegates to the central ActivityLoggingService description generator.
// Kept for backward compatibility with existing tests.
func statusChangeDescription(oldName, newName string) string {
	return statusChangeDesc(oldName, newName)
}

// ticketCreationDescription generates a human-readable description for ticket creation.
// Delegates to the central ActivityLoggingService description generator.
// Kept for backward compatibility with existing tests.
func ticketCreationDescription(ticketNumber string, hasSOP bool) string {
	return ticketCreatedDescription(ticketNumber, hasSOP)
}
