package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

// TicketSOPService handles the creation of ticket steps from SOP templates.
// It encapsulates the logic for resolving which SOP version to use, snapshotting
// the version, and copying SOPSteps/SOPSubSteps into TicketSteps/TicketSubSteps.
//
// This service is used by TicketLifecycleService during ticket creation and can
// also be called independently when re-linking an SOP to an existing ticket.
type TicketSOPService struct {
	db              *gorm.DB
	sopTemplateRepo *repositories.SOPTemplateRepository
	ticketStepRepo  *repositories.TicketStepRepository
}

func NewTicketSOPService(
	db *gorm.DB,
	sopTemplateRepo *repositories.SOPTemplateRepository,
	ticketStepRepo *repositories.TicketStepRepository,
) *TicketSOPService {
	return &TicketSOPService{
		db:              db,
		sopTemplateRepo: sopTemplateRepo,
		ticketStepRepo:  ticketStepRepo,
	}
}

// ResolveSOPVersionResult holds the resolved SOP template and version IDs.
type ResolveSOPVersionResult struct {
	SOPTemplateID int
	SOPVersionID  int
}

// ResolveSOPVersion determines which SOP template and version to use for a ticket.
// If an explicit sopTemplateID is provided, it takes priority. Otherwise, the
// ticket type's DefaultSOPTemplateID is used. Returns nil if no SOP is applicable.
//
// The version is always the template's CurrentVersionID — the "snapshot" of the
// SOP at ticket creation time.
func (s *TicketSOPService) ResolveSOPVersion(
	sopTemplateID *int,
	ticketType *models.TicketType,
) (*ResolveSOPVersionResult, error) {
	// Determine the effective template ID
	effectiveTemplateID := sopTemplateID
	if effectiveTemplateID == nil && ticketType.DefaultSOPTemplateID != nil {
		effectiveTemplateID = ticketType.DefaultSOPTemplateID
	}

	if effectiveTemplateID == nil {
		return nil, nil
	}

	// Look up the template and its current version
	template, err := s.sopTemplateRepo.GetByID(*effectiveTemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up SOP template %d: %w", *effectiveTemplateID, err)
	}

	if template.CurrentVersionID == nil {
		log.Printf("SOP template %d has no current version; skipping step copy", *effectiveTemplateID)
		return nil, nil
	}

	return &ResolveSOPVersionResult{
		SOPTemplateID: *effectiveTemplateID,
		SOPVersionID:  *template.CurrentVersionID,
	}, nil
}

// CopySOPStepsToTicket copies all steps and sub-steps from the given SOP version
// into the ticket as TicketSteps and TicketSubSteps. This must be called within
// a transaction (pass a *gorm.DB transaction handle) so the step creation is
// atomic with the ticket creation.
//
// The copy process:
//  1. Fetches SOPSteps for the version with SubSteps preloaded.
//  2. For each SOPStep, creates a TicketStep preserving StationID, Title,
//     Instructions, and linking back via SOPStepID.
//  3. For each SOPSubStep, creates a TicketSubStep preserving Title,
//     Instructions, and linking back via SOPSubStepID.
//  4. DisplayOrder is assigned sequentially (0-based) from the SOP ordering.
func (s *TicketSOPService) CopySOPStepsToTicket(tx *gorm.DB, ticketID uuid.UUID, versionID int) error {
	// Fetch SOP steps with sub-steps
	var sopSteps []models.SOPStep
	err := tx.Where("sop_version_id = ?", versionID).
		Preload("SubSteps", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Order(`"order" ASC`).
		Find(&sopSteps).Error
	if err != nil {
		return fmt.Errorf("fetching SOP steps for version %d: %w", versionID, err)
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
			return fmt.Errorf("creating ticket step %d (%q): %w", i, sopStep.Title, err)
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
				return fmt.Errorf("creating ticket sub-step %d.%d (%q): %w", i, j, sopSubStep.Title, err)
			}
		}
	}

	return nil
}

// CopySOPStepsToTicketNonTx copies SOP steps to a ticket using an internal
// transaction. Use this when you don't have an existing transaction context.
func (s *TicketSOPService) CopySOPStepsToTicketNonTx(ticketID uuid.UUID, versionID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.CopySOPStepsToTicket(tx, ticketID, versionID)
	})
}

// ValidateParentChild validates the one-level nesting constraint for parent/child
// tickets. A ticket cannot be a child of a ticket that already has a parent.
func ValidateParentChild(parentTicket *models.Ticket) error {
	if parentTicket.ParentTicketID != nil {
		return fmt.Errorf(
			"cannot nest tickets more than one level: parent ticket %s already has a parent",
			parentTicket.TicketNumber,
		)
	}
	return nil
}

// sopCopyDescription generates a human-readable description for step copy activity.
func sopCopyDescription(stepCount int) string {
	if stepCount == 1 {
		return "Copied 1 step from linked SOP"
	}
	return fmt.Sprintf("Copied %d steps from linked SOP", stepCount)
}
