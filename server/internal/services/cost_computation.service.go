package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// CostBreakdown holds the aggregate cost information for a ticket, broken down
// by cost type.
type CostBreakdown struct {
	TicketID     uuid.UUID          `json:"ticketId"`
	TotalCost    decimal.Decimal    `json:"totalCost"`
	LaborCost    decimal.Decimal    `json:"laborCost"`
	MaterialCost decimal.Decimal    `json:"materialCost"`
	OtherCost    decimal.Decimal    `json:"otherCost"`
	Entries      []models.CostEntry `json:"entries"`
}

// CostComputationService handles cost-related logic: auto-generating labor
// CostEntries from completed TicketSteps × labor rate, creating material cost
// entries from BOM usage, and computing aggregate costs per ticket.
type CostComputationService struct {
	costEntryRepo  *repositories.CostEntryRepository
	ticketStepRepo *repositories.TicketStepRepository
	spaceRepo      *repositories.SpaceRepository
	bomItemRepo    *repositories.BOMItemRepository
	ticketRepo     *repositories.TicketRepository
	activitySvc    *ActivityLoggingService
}

func NewCostComputationService(
	costEntryRepo *repositories.CostEntryRepository,
	ticketStepRepo *repositories.TicketStepRepository,
	spaceRepo *repositories.SpaceRepository,
	bomItemRepo *repositories.BOMItemRepository,
	ticketRepo *repositories.TicketRepository,
	activitySvc *ActivityLoggingService,
) *CostComputationService {
	return &CostComputationService{
		costEntryRepo:  costEntryRepo,
		ticketStepRepo: ticketStepRepo,
		spaceRepo:      spaceRepo,
		bomItemRepo:    bomItemRepo,
		ticketRepo:     ticketRepo,
		activitySvc:    activitySvc,
	}
}

// GenerateLaborCosts creates labor CostEntries for all completed steps of a
// ticket using the space's default labor rate. Existing labor cost entries for
// the ticket are deleted first to allow idempotent re-generation. Returns the
// generated entries.
//
// The labor rate is expressed in cost-per-hour. Each step's ActualTimeSeconds
// is converted to hours and multiplied by the rate.
func (s *CostComputationService) GenerateLaborCosts(ticketID uuid.UUID, userID uuid.UUID) ([]models.CostEntry, error) {
	// 1. Load the ticket to get SpaceID and validate existence
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	// 2. Load the space to get the default labor rate
	space, err := s.spaceRepo.FindByID(ticket.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	if space.DefaultLaborRate == nil || space.DefaultLaborRate.IsZero() {
		return nil, fmt.Errorf("space %q has no default labor rate configured", space.Name)
	}
	laborRate := *space.DefaultLaborRate

	// 3. Delete existing auto-generated labor entries for this ticket
	existingLabor, err := s.costEntryRepo.GetByTicketIDAndType(ticketID, models.CostTypeLabor)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing labor costs: %w", err)
	}
	for _, entry := range existingLabor {
		if err := s.costEntryRepo.Delete(entry.ID); err != nil {
			log.Printf("Failed to delete old labor cost entry %s: %v", entry.ID, err)
		}
	}

	// 4. Get all completed steps for this ticket
	steps, err := s.ticketStepRepo.GetByTicketID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket steps: %w", err)
	}

	// 5. Generate a labor CostEntry for each completed step with time > 0
	secondsPerHour := decimal.NewFromInt(3600)
	unit := "hours"
	var generated []models.CostEntry

	for _, step := range steps {
		if step.Status != models.TicketStepStatusCompleted {
			continue
		}
		if step.ActualTimeSeconds <= 0 {
			continue
		}

		hours := decimal.NewFromInt(int64(step.ActualTimeSeconds)).Div(secondsPerHour)
		amount := hours.Mul(laborRate)

		entry := models.CostEntry{
			TicketID:    ticketID,
			CostType:    models.CostTypeLabor,
			Description: fmt.Sprintf("Labor: %s (%s)", step.Title, hours.StringFixed(2)+" hrs"),
			Amount:      amount,
			Quantity:    &hours,
			Unit:        &unit,
			UnitCost:    &laborRate,
			CreatedByID: userID,
		}

		if err := s.costEntryRepo.Create(&entry); err != nil {
			log.Printf("Failed to create labor cost entry for step %s: %v", step.ID, err)
			continue
		}

		generated = append(generated, entry)
	}

	return generated, nil
}

// GenerateMaterialCosts creates material CostEntries for a ticket based on the
// BOM items of its linked SOP version. Existing material cost entries for the
// ticket are deleted first to allow idempotent re-generation. Each BOM item
// with a UnitCost produces a cost entry of Amount = Quantity × UnitCost.
func (s *CostComputationService) GenerateMaterialCosts(ticketID uuid.UUID, userID uuid.UUID) ([]models.CostEntry, error) {
	// 1. Load the ticket to get SOPVersionID
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	if ticket.SOPVersionID == nil {
		return nil, fmt.Errorf("ticket %s has no linked SOP version", ticket.TicketNumber)
	}

	// 2. Delete existing material entries for this ticket
	existingMaterial, err := s.costEntryRepo.GetByTicketIDAndType(ticketID, models.CostTypeMaterial)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing material costs: %w", err)
	}
	for _, entry := range existingMaterial {
		if err := s.costEntryRepo.Delete(entry.ID); err != nil {
			log.Printf("Failed to delete old material cost entry %s: %v", entry.ID, err)
		}
	}

	// 3. Get BOM items for the SOP version
	bomItems, err := s.bomItemRepo.GetByVersionID(*ticket.SOPVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get BOM items: %w", err)
	}

	// 4. Generate a material CostEntry for each BOM item with a unit cost
	var generated []models.CostEntry

	for _, bom := range bomItems {
		if bom.UnitCost == nil || bom.UnitCost.IsZero() {
			continue
		}

		amount := bom.Quantity.Mul(*bom.UnitCost)
		quantity := bom.Quantity
		unit := bom.Unit
		unitCost := *bom.UnitCost

		entry := models.CostEntry{
			TicketID:    ticketID,
			CostType:    models.CostTypeMaterial,
			Description: fmt.Sprintf("Material: %s (%s %s)", bom.Name, bom.Quantity.StringFixed(2), bom.Unit),
			Amount:      amount,
			Quantity:    &quantity,
			Unit:        &unit,
			UnitCost:    &unitCost,
			MaterialID:  bom.MaterialID,
			CreatedByID: userID,
		}

		if err := s.costEntryRepo.Create(&entry); err != nil {
			log.Printf("Failed to create material cost entry for BOM item %d: %v", bom.ID, err)
			continue
		}

		generated = append(generated, entry)
	}

	return generated, nil
}

// GetCostBreakdown returns the aggregate cost for a ticket, broken down by
// type (labor, material, other). "Other" aggregates consumable, marketing,
// and other cost types.
func (s *CostComputationService) GetCostBreakdown(ticketID uuid.UUID) (*CostBreakdown, error) {
	entries, err := s.costEntryRepo.GetByTicketID(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost entries: %w", err)
	}

	result := computeBreakdown(ticketID, entries)
	return &result, nil
}

// computeBreakdown aggregates cost entries into a CostBreakdown. Extracted for
// testability — the logic is pure and does not require a database.
func computeBreakdown(ticketID uuid.UUID, entries []models.CostEntry) CostBreakdown {
	breakdown := CostBreakdown{
		TicketID:     ticketID,
		TotalCost:    decimal.Zero,
		LaborCost:    decimal.Zero,
		MaterialCost: decimal.Zero,
		OtherCost:    decimal.Zero,
		Entries:      entries,
	}

	for _, entry := range entries {
		breakdown.TotalCost = breakdown.TotalCost.Add(entry.Amount)

		switch entry.CostType {
		case models.CostTypeLabor:
			breakdown.LaborCost = breakdown.LaborCost.Add(entry.Amount)
		case models.CostTypeMaterial:
			breakdown.MaterialCost = breakdown.MaterialCost.Add(entry.Amount)
		default:
			// consumable, marketing, other → all roll up to OtherCost
			breakdown.OtherCost = breakdown.OtherCost.Add(entry.Amount)
		}
	}

	return breakdown
}

// GetTotalCost returns just the total cost for a ticket as a convenience.
func (s *CostComputationService) GetTotalCost(ticketID uuid.UUID) (decimal.Decimal, error) {
	breakdown, err := s.GetCostBreakdown(ticketID)
	if err != nil {
		return decimal.Zero, err
	}
	return breakdown.TotalCost, nil
}
