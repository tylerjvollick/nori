package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// SpaceTemplate is an identifier for a preset space configuration.
type SpaceTemplate string

const (
	SpaceTemplateWoodworkingShop SpaceTemplate = "woodworking_shop"
	SpaceTemplateSales           SpaceTemplate = "sales"
)

// statusDef is a short-hand used to declare seed status definitions.
type statusDef struct {
	Name       string
	Category   models.StatusCategory
	IsDefault  bool
	IsTerminal bool
}

// ticketTypeSeed groups a ticket type name with its statuses.
type ticketTypeSeed struct {
	Name     string
	Statuses []statusDef
}

// woodworkingShopSOPCategories defines the default SOP categories for a
// woodworking shop space (task 6.3).
var woodworkingShopSOPCategories = []string{
	"Techniques",
	"Maintenance",
	"Setup",
	"Products",
}

// woodworkingShopTicketTypes defines the default ticket types and statuses for
// a woodworking shop space (task 6.1).
var woodworkingShopTicketTypes = []ticketTypeSeed{
	{
		Name: "Build",
		Statuses: []statusDef{
			{Name: "Queued", Category: models.StatusCategoryTodo, IsDefault: true},
			{Name: "In Progress", Category: models.StatusCategoryInProgress},
			{Name: "QC", Category: models.StatusCategoryInProgress},
			{Name: "Done", Category: models.StatusCategoryDone, IsTerminal: true},
		},
	},
	{
		Name: "Maintenance",
		Statuses: []statusDef{
			{Name: "Open", Category: models.StatusCategoryTodo, IsDefault: true},
			{Name: "In Progress", Category: models.StatusCategoryInProgress},
			{Name: "Done", Category: models.StatusCategoryDone, IsTerminal: true},
		},
	},
	{
		Name: "Prep",
		Statuses: []statusDef{
			{Name: "Todo", Category: models.StatusCategoryTodo, IsDefault: true},
			{Name: "In Progress", Category: models.StatusCategoryInProgress},
			{Name: "Done", Category: models.StatusCategoryDone, IsTerminal: true},
		},
	},
}

// salesTicketTypes defines the default ticket types and statuses for a sales
// space (task 6.2).
var salesTicketTypes = []ticketTypeSeed{
	{
		Name: "Lead",
		Statuses: []statusDef{
			{Name: "New", Category: models.StatusCategoryTodo, IsDefault: true},
			{Name: "Contacted", Category: models.StatusCategoryInProgress},
			{Name: "Meeting", Category: models.StatusCategoryInProgress},
			{Name: "Proposal", Category: models.StatusCategoryInProgress},
			{Name: "Won", Category: models.StatusCategoryDone, IsTerminal: true},
			{Name: "Lost", Category: models.StatusCategoryDone, IsTerminal: true},
		},
	},
}

// SpaceTemplateService applies pre-configured seed data (ticket types, status
// definitions, SOP categories, etc.) to a newly created space based on a
// template identifier.
type SpaceTemplateService struct {
	ticketTypeRepo       *repositories.TicketTypeRepository
	statusDefinitionRepo *repositories.StatusDefinitionRepository
	sopCategoryRepo      *repositories.SOPCategoryRepository
}

// NewSpaceTemplateService creates a new SpaceTemplateService.
func NewSpaceTemplateService(
	ticketTypeRepo *repositories.TicketTypeRepository,
	statusDefinitionRepo *repositories.StatusDefinitionRepository,
	sopCategoryRepo *repositories.SOPCategoryRepository,
) *SpaceTemplateService {
	return &SpaceTemplateService{
		ticketTypeRepo:       ticketTypeRepo,
		statusDefinitionRepo: statusDefinitionRepo,
		sopCategoryRepo:      sopCategoryRepo,
	}
}

// ApplyTemplate seeds the given space with default data for the requested
// template. Returns an error if the template is unknown.
func (s *SpaceTemplateService) ApplyTemplate(spaceID uuid.UUID, template SpaceTemplate) error {
	switch template {
	case SpaceTemplateWoodworkingShop:
		if err := s.seedTicketTypes(spaceID, woodworkingShopTicketTypes); err != nil {
			return err
		}
		return s.seedSOPCategories(spaceID, woodworkingShopSOPCategories)
	case SpaceTemplateSales:
		return s.seedTicketTypes(spaceID, salesTicketTypes)
	default:
		return fmt.Errorf("unknown space template: %s", template)
	}
}

// IsValidTemplate returns true when the template string is recognised.
func IsValidTemplate(t string) bool {
	switch SpaceTemplate(t) {
	case SpaceTemplateWoodworkingShop, SpaceTemplateSales:
		return true
	default:
		return false
	}
}

// seedTicketTypes creates ticket types and their status definitions for a space.
func (s *SpaceTemplateService) seedTicketTypes(spaceID uuid.UUID, seeds []ticketTypeSeed) error {
	now := time.Now()

	for i, seed := range seeds {
		tt := &models.TicketType{
			ID:           uuid.New(),
			SpaceID:      spaceID,
			Name:         seed.Name,
			IsActive:     true,
			DisplayOrder: i,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := s.ticketTypeRepo.Create(tt); err != nil {
			return fmt.Errorf("create ticket type %q: %w", seed.Name, err)
		}

		for j, st := range seed.Statuses {
			sd := &models.StatusDefinition{
				ID:           uuid.New(),
				TicketTypeID: tt.ID,
				Name:         st.Name,
				DisplayOrder: j,
				Category:     st.Category,
				IsDefault:    st.IsDefault,
				IsTerminal:   st.IsTerminal,
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			if err := s.statusDefinitionRepo.Create(sd); err != nil {
				return fmt.Errorf("create status %q for ticket type %q: %w", st.Name, seed.Name, err)
			}
		}
	}

	return nil
}

// seedSOPCategories creates root-level SOP categories for a space.
func (s *SpaceTemplateService) seedSOPCategories(spaceID uuid.UUID, names []string) error {
	now := time.Now()

	for i, name := range names {
		cat := &models.SOPCategory{
			ID:           uuid.New(),
			SpaceID:      spaceID,
			Name:         name,
			DisplayOrder: i,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := s.sopCategoryRepo.Create(cat); err != nil {
			return fmt.Errorf("create SOP category %q: %w", name, err)
		}
	}

	return nil
}
