package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/utils"
	"gorm.io/gorm"
)

type SOPService struct {
	db           *gorm.DB
	templateRepo *repositories.SOPTemplateRepository
	versionRepo  *repositories.SOPTemplateVersionRepository
	stepRepo     *repositories.SOPStepRepository
}

func NewSOPService(
	db *gorm.DB,
	templateRepo *repositories.SOPTemplateRepository,
	versionRepo *repositories.SOPTemplateVersionRepository,
	stepRepo *repositories.SOPStepRepository,
) *SOPService {
	return &SOPService{
		db:           db,
		templateRepo: templateRepo,
		versionRepo:  versionRepo,
		stepRepo:     stepRepo,
	}
}

// CreateSOP creates a new SOP template with its first version and steps
func (s *SOPService) CreateSOP(dto *dtos.CreateSOPDTO, userID uuid.UUID) (*models.SOPTemplate, error) {
	var template *models.SOPTemplate

	// Use transaction to ensure all creates succeed or all fail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the SOP template (without current_version_id yet)
		template = &models.SOPTemplate{
			Name:      dto.Name,
			CreatedBy: userID,
		}

		if err := s.templateRepo.Create(template); err != nil {
			log.Println("Failed to create SOP template:", err)
			return fmt.Errorf("failed to create SOP template: %w", err)
		}

		// 2. Create the first version
		version := &models.SOPTemplateVersion{
			SOPTemplateID: template.ID,
			VersionNumber: 1,
			Status:        models.VersionStatusPublished,
			Description:   dto.Description,
			Materials:     dto.Materials,
			Equipment:     dto.Equipment,
			CreatedBy:     userID,
			ChangeSummary: dto.ChangeSummary,
			IsActive:      true,
		}

		if err := s.versionRepo.Create(version); err != nil {
			log.Println("Failed to create SOP version:", err)
			return fmt.Errorf("failed to create SOP version: %w", err)
		}

		// 3. Create the steps
		var steps []models.SOPStep
		for i, stepDTO := range dto.Steps {
			// Generate order value for each step
			order := stepDTO.Order
			if order == "" {
				// If no order provided, generate based on position
				orderMap := utils.RebalanceOrders(len(dto.Steps))
				order = orderMap[i]
			}

			step := models.SOPStep{
				SOPTemplateVersionID: version.ID,
				Order:                order,
				Title:                stepDTO.Title,
				Instructions:         stepDTO.Instructions,
				EstimatedTimeMinutes: stepDTO.EstimatedTimeMinutes,
				ImageURL:             stepDTO.ImageURL,
				VideoURL:             stepDTO.VideoURL,
				RequiresApproval:     stepDTO.RequiresApproval,
			}
			steps = append(steps, step)
		}

		if err := s.stepRepo.CreateBatch(steps); err != nil {
			log.Println("Failed to create SOP steps:", err)
			return fmt.Errorf("failed to create SOP steps: %w", err)
		}

		// 4. Update the template's current_version_id
		template.CurrentVersionID = &version.ID
		if err := s.templateRepo.Update(template); err != nil {
			log.Println("Failed to update template current version:", err)
			return fmt.Errorf("failed to update template current version: %w", err)
		}

		// Load the version with steps for the response
		version.Steps = steps
		template.CurrentVersion = version

		return nil
	})

	if err != nil {
		return nil, err
	}

	return template, nil
}

// UpdateSOP creates a new version of an existing SOP
func (s *SOPService) UpdateSOP(templateID int, dto *dtos.UpdateSOPDTO, userID uuid.UUID) (*models.SOPTemplate, error) {
	var template *models.SOPTemplate

	// Use transaction to ensure all creates succeed or all fail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the existing template
		existingTemplate, err := s.templateRepo.GetByID(templateID)
		if err != nil {
			return fmt.Errorf("failed to get SOP template: %w", err)
		}

		// 2. Get the latest version number
		latestVersionNumber, err := s.versionRepo.GetLatestVersionNumber(templateID)
		if err != nil {
			return fmt.Errorf("failed to get latest version number: %w", err)
		}

		// 3. Create the new version
		newVersion := &models.SOPTemplateVersion{
			SOPTemplateID: templateID,
			VersionNumber: latestVersionNumber + 1,
			Status:        models.VersionStatusPublished,
			Description:   dto.Description,
			Materials:     dto.Materials,
			Equipment:     dto.Equipment,
			CreatedBy:     userID,
			ChangeSummary: &dto.ChangeSummary,
			IsActive:      true,
		}

		if err := s.versionRepo.Create(newVersion); err != nil {
			log.Println("Failed to create new SOP version:", err)
			return fmt.Errorf("failed to create new SOP version: %w", err)
		}

		// 4. Create the steps for the new version
		var steps []models.SOPStep
		for i, stepDTO := range dto.Steps {
			// Generate order value for each step
			order := stepDTO.Order
			if order == "" {
				// If no order provided, generate based on position
				orderMap := utils.RebalanceOrders(len(dto.Steps))
				order = orderMap[i]
			}

			step := models.SOPStep{
				SOPTemplateVersionID: newVersion.ID,
				Order:                order,
				Title:                stepDTO.Title,
				Instructions:         stepDTO.Instructions,
				EstimatedTimeMinutes: stepDTO.EstimatedTimeMinutes,
				ImageURL:             stepDTO.ImageURL,
				VideoURL:             stepDTO.VideoURL,
				RequiresApproval:     stepDTO.RequiresApproval,
			}
			steps = append(steps, step)
		}

		if err := s.stepRepo.CreateBatch(steps); err != nil {
			log.Println("Failed to create SOP steps:", err)
			return fmt.Errorf("failed to create SOP steps: %w", err)
		}

		// 5. Update the template's current_version_id and optionally the name
		existingTemplate.CurrentVersionID = &newVersion.ID
		if dto.Name != nil {
			existingTemplate.Name = *dto.Name
		}

		if err := s.templateRepo.Update(existingTemplate); err != nil {
			log.Println("Failed to update template:", err)
			return fmt.Errorf("failed to update template: %w", err)
		}

		// Load the version with steps for the response
		newVersion.Steps = steps
		existingTemplate.CurrentVersion = newVersion
		template = existingTemplate

		return nil
	})

	if err != nil {
		return nil, err
	}

	return template, nil
}

// GetSOP gets an SOP template by ID with its latest version (draft if exists, otherwise current published version)
func (s *SOPService) GetSOP(templateID int) (*models.SOPTemplate, error) {
	template, err := s.templateRepo.GetByID(templateID)
	if err != nil {
		return nil, err
	}

	// Check if a draft exists for this template
	draft, err := s.versionRepo.GetDraftByTemplateID(templateID)
	if err == nil && draft != nil {
		// Load steps for the draft
		steps, err := s.stepRepo.GetByVersionID(draft.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load draft steps: %w", err)
		}
		draft.Steps = steps
		// Replace the current version with the draft
		template.CurrentVersion = draft
	}

	return template, nil
}

// GetAllSOPs gets all SOP templates
func (s *SOPService) GetAllSOPs() ([]models.SOPTemplate, error) {
	return s.templateRepo.GetAll()
}

// GetSOPVersions gets all versions of an SOP template
func (s *SOPService) GetSOPVersions(templateID int) ([]models.SOPTemplateVersion, error) {
	return s.versionRepo.GetByTemplateID(templateID)
}

// GetSOPVersion gets a specific version of an SOP template
func (s *SOPService) GetSOPVersion(versionID int) (*models.SOPTemplateVersion, error) {
	return s.versionRepo.GetByID(versionID)
}

// DeleteSOP deletes an SOP template and all its versions/steps (cascade)
func (s *SOPService) DeleteSOP(templateID int) error {
	return s.templateRepo.Delete(templateID)
}

// SaveDraft creates a new draft version for an existing SOP template
func (s *SOPService) SaveDraft(templateID int, dto *dtos.SaveDraftSOPDTO, userID uuid.UUID) (*models.SOPTemplateVersion, error) {
	var draft *models.SOPTemplateVersion

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the template exists
		_, err := s.templateRepo.GetByID(templateID)
		if err != nil {
			return fmt.Errorf("failed to get SOP template: %w", err)
		}

		// 2. Check if a draft already exists for this template (defensive check)
		existingDraft, err := s.versionRepo.GetDraftByTemplateID(templateID)
		if err != nil {
			return fmt.Errorf("failed to check for existing draft: %w", err)
		}
		if existingDraft != nil {
			return fmt.Errorf("a draft already exists for this SOP (ID: %d)", existingDraft.ID)
		}

		// 3. Get the latest published version number to determine next version number
		latestVersionNumber, err := s.versionRepo.GetLatestPublishedVersionNumber(templateID)
		if err != nil {
			return fmt.Errorf("failed to get latest version number: %w", err)
		}

		// 4. Create the draft version with next version number
		draft = &models.SOPTemplateVersion{
			SOPTemplateID: templateID,
			VersionNumber: latestVersionNumber + 1,
			Status:        models.VersionStatusDraft,
			Description:   dto.Description,
			Materials:     dto.Materials,
			Equipment:     dto.Equipment,
			CreatedBy:     userID,
			ChangeSummary: dto.ChangeSummary,
			IsActive:      true,
		}

		if err := s.versionRepo.Create(draft); err != nil {
			log.Println("Failed to create draft version:", err)
			return fmt.Errorf("failed to create draft version: %w", err)
		}

		// 5. Create the steps for the draft
		var steps []models.SOPStep
		for i, stepDTO := range dto.Steps {
			// Generate order value for each step
			order := stepDTO.Order
			if order == "" {
				// If no order provided, generate based on position
				orderMap := utils.RebalanceOrders(len(dto.Steps))
				order = orderMap[i]
			}

			step := models.SOPStep{
				SOPTemplateVersionID: draft.ID,
				Order:                order,
				Title:                stepDTO.Title,
				Instructions:         stepDTO.Instructions,
				EstimatedTimeMinutes: stepDTO.EstimatedTimeMinutes,
				ImageURL:             stepDTO.ImageURL,
				VideoURL:             stepDTO.VideoURL,
				RequiresApproval:     stepDTO.RequiresApproval,
			}
			steps = append(steps, step)
		}

		if err := s.stepRepo.CreateBatch(steps); err != nil {
			log.Println("Failed to create draft steps:", err)
			return fmt.Errorf("failed to create draft steps: %w", err)
		}

		draft.Steps = steps
		return nil
	})

	if err != nil {
		return nil, err
	}

	return draft, nil
}

// UpdateDraft updates an existing draft version
func (s *SOPService) UpdateDraft(draftID int, dto *dtos.SaveDraftSOPDTO, userID uuid.UUID) (*models.SOPTemplateVersion, error) {
	var draft *models.SOPTemplateVersion

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the existing draft
		existingDraft, err := s.versionRepo.GetByID(draftID)
		if err != nil {
			return fmt.Errorf("failed to get draft: %w", err)
		}

		// 2. Verify it's a draft
		if existingDraft.Status != models.VersionStatusDraft {
			return fmt.Errorf("version is not a draft")
		}

		// 3. Verify the user owns this draft
		if existingDraft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to update this draft")
		}

		// 4. Update the draft version metadata
		existingDraft.Description = dto.Description
		existingDraft.Materials = dto.Materials
		existingDraft.Equipment = dto.Equipment
		existingDraft.ChangeSummary = dto.ChangeSummary

		if err := s.versionRepo.UpdateWithTx(tx, existingDraft); err != nil {
			log.Println("Failed to update draft:", err)
			return fmt.Errorf("failed to update draft: %w", err)
		}

		// 5. Get existing steps
		existingSteps, err := s.stepRepo.GetByVersionID(existingDraft.ID)
		if err != nil {
			return fmt.Errorf("failed to get existing steps: %w", err)
		}

		// 6. Create maps for efficient lookup by order
		existingStepMap := make(map[string]*models.SOPStep)
		for i := range existingSteps {
			existingStepMap[existingSteps[i].Order] = &existingSteps[i]
		}

		dtoStepMap := make(map[string]bool)
		var updatedSteps []models.SOPStep

		// 7. Update or insert steps from DTO
		for i, stepDTO := range dto.Steps {
			order := stepDTO.Order
			if order == "" {
				// If no order provided, generate based on position
				orderMap := utils.RebalanceOrders(len(dto.Steps))
				order = orderMap[i]
			}
			dtoStepMap[order] = true

			if existingStep, exists := existingStepMap[order]; exists {
				// Update existing step
				existingStep.Title = stepDTO.Title
				existingStep.Instructions = stepDTO.Instructions
				existingStep.EstimatedTimeMinutes = stepDTO.EstimatedTimeMinutes
				existingStep.ImageURL = stepDTO.ImageURL
				existingStep.VideoURL = stepDTO.VideoURL
				existingStep.RequiresApproval = stepDTO.RequiresApproval

				if err := s.stepRepo.UpdateWithTx(tx, existingStep); err != nil {
					log.Println("Failed to update step:", err)
					return fmt.Errorf("failed to update step: %w", err)
				}
				updatedSteps = append(updatedSteps, *existingStep)
			} else {
				// Insert new step
				newStep := models.SOPStep{
					SOPTemplateVersionID: existingDraft.ID,
					Order:                order,
					Title:                stepDTO.Title,
					Instructions:         stepDTO.Instructions,
					EstimatedTimeMinutes: stepDTO.EstimatedTimeMinutes,
					ImageURL:             stepDTO.ImageURL,
					VideoURL:             stepDTO.VideoURL,
					RequiresApproval:     stepDTO.RequiresApproval,
				}

				if err := tx.Create(&newStep).Error; err != nil {
					log.Println("Failed to create new step:", err)
					return fmt.Errorf("failed to create new step: %w", err)
				}
				updatedSteps = append(updatedSteps, newStep)
			}
		}

		// 8. Delete steps that are no longer in the DTO
		for stepOrder, existingStep := range existingStepMap {
			if !dtoStepMap[stepOrder] {
				if err := s.stepRepo.DeleteWithTx(tx, existingStep.ID); err != nil {
					log.Println("Failed to delete step:", err)
					return fmt.Errorf("failed to delete step: %w", err)
				}
			}
		}

		existingDraft.Steps = updatedSteps
		draft = existingDraft
		return nil
	})

	if err != nil {
		return nil, err
	}

	return draft, nil
}

// PublishDraft publishes a draft version, making it the current version
func (s *SOPService) PublishDraft(draftID int, changeSummary string) (*models.SOPTemplate, error) {
	var template *models.SOPTemplate

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the draft
		draft, err := s.versionRepo.GetByID(draftID)
		if err != nil {
			return fmt.Errorf("failed to get draft: %w", err)
		}

		// 2. Verify it's a draft
		if draft.Status != models.VersionStatusDraft {
			return fmt.Errorf("version is not a draft")
		}

		// 3. Update the draft to published (using transaction)
		draft.Status = models.VersionStatusPublished
		summary := changeSummary
		draft.ChangeSummary = &summary

		if err := s.versionRepo.UpdateWithTx(tx, draft); err != nil {
			log.Println("Failed to publish draft:", err)
			return fmt.Errorf("failed to publish draft: %w", err)
		}

		// 4. Update the template's current_version_id (using transaction)
		if err := s.templateRepo.UpdateCurrentVersionWithTx(tx, draft.SOPTemplateID, draft.ID); err != nil {
			log.Println("Failed to update template current version:", err)
			return fmt.Errorf("failed to update template current version: %w", err)
		}

		// 5. Reload the template with the updated current version (using transaction)
		existingTemplate, err := s.templateRepo.GetByIDWithTx(tx, draft.SOPTemplateID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		template = existingTemplate
		return nil
	})

	if err != nil {
		return nil, err
	}

	return template, nil
}

// GetUserDrafts gets all draft versions created by a user
func (s *SOPService) GetUserDrafts(userID uuid.UUID) ([]models.SOPTemplateVersion, error) {
	return s.versionRepo.GetDraftsByUserID(userID.String())
}

// GetSOPDrafts gets all draft versions for a specific SOP template
func (s *SOPService) GetSOPDrafts(templateID int) ([]models.SOPTemplateVersion, error) {
	return s.versionRepo.GetDraftsByTemplateID(templateID)
}

// GetDraftByTemplateID returns the active draft for a template, if one exists
func (s *SOPService) GetDraftByTemplateID(templateID int) (*models.SOPTemplateVersion, error) {
	return s.versionRepo.GetDraftByTemplateID(templateID)
}

// DeleteDraft deletes a draft version
func (s *SOPService) DeleteDraft(draftID int, userID uuid.UUID) error {
	// Verify the draft exists and belongs to the user
	draft, err := s.versionRepo.GetByID(draftID)
	if err != nil {
		return fmt.Errorf("failed to get draft: %w", err)
	}

	if draft.Status != models.VersionStatusDraft {
		return fmt.Errorf("version is not a draft")
	}

	if draft.CreatedBy.String() != userID.String() {
		return fmt.Errorf("unauthorized to delete this draft")
	}

	return s.versionRepo.Delete(draftID)
}

// GetDraft retrieves a specific draft by ID
func (s *SOPService) GetDraft(draftID int, userID uuid.UUID) (*models.SOPTemplateVersion, error) {
	// Get the draft
	draft, err := s.versionRepo.GetByID(draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	if draft.Status != models.VersionStatusDraft {
		return nil, fmt.Errorf("version is not a draft")
	}

	if draft.CreatedBy.String() != userID.String() {
		return nil, fmt.Errorf("unauthorized to view this draft")
	}

	// Load the steps
	steps, err := s.stepRepo.GetByVersionID(draft.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load steps: %w", err)
	}
	draft.Steps = steps

	return draft, nil
}

// Individual step operations for drafts

// CreateStep creates a single step in a draft version
func (s *SOPService) CreateStep(draftID int, dto *dtos.CreateStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the draft exists and belongs to the user
		draft, err := s.versionRepo.GetByID(draftID)
		if err != nil {
			return fmt.Errorf("failed to get draft: %w", err)
		}

		if draft.Status != models.VersionStatusDraft {
			return fmt.Errorf("version is not a draft")
		}

		if draft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 2. Determine order for the new step
		var order string
		if dto.AfterStepID != nil {
			// Get the step after which we're inserting
			afterStep, err := s.stepRepo.GetByIDAndVersionID(*dto.AfterStepID, draftID)
			if err != nil {
				return fmt.Errorf("failed to get after step: %w", err)
			}

			// Get the last order to find what comes next
			lastOrder, err := s.stepRepo.GetLastOrderByVersionID(draftID)
			if err != nil {
				return fmt.Errorf("failed to get last order: %w", err)
			}

			// Generate order after the specified step
			if afterStep.Order == lastOrder {
				// Inserting at the end
				order = utils.GenerateOrderBetween(afterStep.Order, "")
			} else {
				// Find the next step
				allSteps, err := s.stepRepo.GetByVersionID(draftID)
				if err != nil {
					return fmt.Errorf("failed to get all steps: %w", err)
				}

				var nextOrder string
				foundAfter := false
				for _, s := range allSteps {
					if foundAfter {
						nextOrder = s.Order
						break
					}
					if s.ID == *dto.AfterStepID {
						foundAfter = true
					}
				}

				order = utils.GenerateOrderBetween(afterStep.Order, nextOrder)
			}
		} else {
			// Inserting at the beginning
			allSteps, err := s.stepRepo.GetByVersionID(draftID)
			if err != nil {
				return fmt.Errorf("failed to get all steps: %w", err)
			}

			if len(allSteps) > 0 {
				order = utils.GenerateOrderBetween("", allSteps[0].Order)
			} else {
				order = utils.GenerateOrderBetween("", "")
			}
		}

		// 3. Create the step
		step = &models.SOPStep{
			SOPTemplateVersionID: draftID,
			Order:                order,
			Title:                dto.Title,
			Instructions:         dto.Instructions,
			EstimatedTimeMinutes: dto.EstimatedTimeMinutes,
			ImageURL:             dto.ImageURL,
			VideoURL:             dto.VideoURL,
			RequiresApproval:     dto.RequiresApproval,
		}

		if err := tx.Create(step).Error; err != nil {
			log.Println("Failed to create step:", err)
			return fmt.Errorf("failed to create step: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return step, nil
}

// CreateStepForTemplate creates a step for a template's draft (creates draft if needed)
func (s *SOPService) CreateStepForTemplate(templateID int, dto *dtos.CreateStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the template (we need it to check current version)
		template, err := s.templateRepo.GetByID(templateID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		// 2. Get or create draft for this template
		draft, err := s.versionRepo.GetDraftByTemplateID(templateID)
		if err != nil || draft == nil {
			// No draft exists, create one based on current version

			// Get latest published version to copy from
			latestVersionNumber, err := s.versionRepo.GetLatestPublishedVersionNumber(templateID)
			if err != nil {
				return fmt.Errorf("failed to get latest version number: %w", err)
			}

			// Create draft from current version
			draft = &models.SOPTemplateVersion{
				SOPTemplateID: templateID,
				VersionNumber: latestVersionNumber + 1,
				Status:        models.VersionStatusDraft,
				CreatedBy:     userID,
				IsActive:      true,
			}

			// Copy fields from current version if it exists
			if template.CurrentVersion != nil {
				draft.Description = template.CurrentVersion.Description
				draft.Materials = template.CurrentVersion.Materials
				draft.Equipment = template.CurrentVersion.Equipment
			}

			if err := tx.Create(draft).Error; err != nil {
				return fmt.Errorf("failed to create draft: %w", err)
			}

			// Copy steps from current version if they exist
			if template.CurrentVersion != nil {
				currentSteps, err := s.stepRepo.GetByVersionID(template.CurrentVersion.ID)
				if err == nil && len(currentSteps) > 0 {
					newSteps := make([]models.SOPStep, len(currentSteps))
					for i, oldStep := range currentSteps {
						newSteps[i] = models.SOPStep{
							SOPTemplateVersionID: draft.ID,
							Order:                oldStep.Order,
							Title:                oldStep.Title,
							Instructions:         oldStep.Instructions,
							EstimatedTimeMinutes: oldStep.EstimatedTimeMinutes,
							ImageURL:             oldStep.ImageURL,
							VideoURL:             oldStep.VideoURL,
							RequiresApproval:     oldStep.RequiresApproval,
						}
					}
					if err := s.stepRepo.CreateBatchWithTx(tx, newSteps); err != nil {
						return fmt.Errorf("failed to copy steps: %w", err)
					}
					// Store the newly created steps in the draft so we can reference them later
					// within this transaction without needing to query the database
					draft.Steps = newSteps
				}
			}
		} else {
			// Draft already exists, load its steps from the transaction context
			existingSteps, err := s.stepRepo.GetByVersionIDWithTx(tx, draft.ID)
			if err != nil {
				return fmt.Errorf("failed to get existing draft steps: %w", err)
			}
			draft.Steps = existingSteps
		}

		// Verify the user owns this draft
		if draft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 3. Determine order for the new step
		var order string
		if dto.AfterStepID != nil {
			// The afterStepID might refer to a step in the published version,
			// so we need to find the corresponding step in the draft by looking up
			// its order value first

			// Try to get the step from the draft first
			afterStep, err := s.stepRepo.GetByIDAndVersionID(*dto.AfterStepID, draft.ID)
			if err != nil {
				// If not found in draft, look it up in the current version to get its order
				if template.CurrentVersion != nil {
					publishedStep, err := s.stepRepo.GetByIDAndVersionID(*dto.AfterStepID, template.CurrentVersion.ID)
					if err != nil {
						return fmt.Errorf("failed to get after step: step not found in this version")
					}

					// Now find the step with the same order in the draft
					// Use the in-memory steps we stored earlier to avoid querying uncommitted data
					draftSteps := draft.Steps

					for i := range draftSteps {
						if draftSteps[i].Order == publishedStep.Order {
							afterStep = &draftSteps[i]
							break
						}
					}
					if afterStep == nil {
						return fmt.Errorf("failed to find corresponding step in draft with order='%s'", publishedStep.Order)
					}
				} else {
					return fmt.Errorf("failed to get after step: %w", err)
				}
			}

			// Get the last order to find what comes next
			lastOrder, err := s.stepRepo.GetLastOrderByVersionID(draft.ID)
			if err != nil {
				return fmt.Errorf("failed to get last order: %w", err)
			}

			// Generate order after the specified step
			if afterStep.Order == lastOrder {
				// Inserting at the end
				order = utils.GenerateOrderBetween(afterStep.Order, "")
			} else {
				// Find the next step using in-memory steps
				allSteps := draft.Steps

				var nextOrder string
				foundAfter := false
				for _, s := range allSteps {
					if foundAfter {
						nextOrder = s.Order
						break
					}
					if s.Order == afterStep.Order {
						foundAfter = true
					}
				}

				order = utils.GenerateOrderBetween(afterStep.Order, nextOrder)
			}
		} else {
			// Inserting at the beginning using in-memory steps
			allSteps := draft.Steps

			if len(allSteps) > 0 {
				order = utils.GenerateOrderBetween("", allSteps[0].Order)
			} else {
				order = utils.GenerateOrderBetween("", "")
			}
		}

		// 4. Create the new step
		step = &models.SOPStep{
			SOPTemplateVersionID: draft.ID,
			Order:                order,
			Title:                dto.Title,
			Instructions:         dto.Instructions,
			EstimatedTimeMinutes: dto.EstimatedTimeMinutes,
			ImageURL:             dto.ImageURL,
			VideoURL:             dto.VideoURL,
			RequiresApproval:     dto.RequiresApproval,
		}

		if err := tx.Create(step).Error; err != nil {
			log.Println("Failed to create step:", err)
			return fmt.Errorf("failed to create step: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return step, nil
}

// UpdateStep updates a single step in a draft version
func (s *SOPService) UpdateStep(draftID int, stepID int, dto *dtos.UpdateStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the draft exists and belongs to the user
		draft, err := s.versionRepo.GetByID(draftID)
		if err != nil {
			return fmt.Errorf("failed to get draft: %w", err)
		}

		if draft.Status != models.VersionStatusDraft {
			return fmt.Errorf("version is not a draft")
		}

		if draft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 2. Get the step and verify it belongs to this draft
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, draftID)
		if err != nil {
			return fmt.Errorf("failed to get step: %w", err)
		}

		// 3. Update only provided fields
		if dto.Title != nil {
			step.Title = *dto.Title
		}
		if dto.Instructions != nil {
			step.Instructions = dto.Instructions
		}
		if dto.EstimatedTimeMinutes != nil {
			step.EstimatedTimeMinutes = dto.EstimatedTimeMinutes
		}
		if dto.ImageURL != nil {
			step.ImageURL = dto.ImageURL
		}
		if dto.VideoURL != nil {
			step.VideoURL = dto.VideoURL
		}
		if dto.RequiresApproval != nil {
			step.RequiresApproval = *dto.RequiresApproval
		}

		if err := s.stepRepo.UpdateWithTx(tx, step); err != nil {
			log.Println("Failed to update step:", err)
			return fmt.Errorf("failed to update step: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return step, nil
}

// DeleteStep deletes a single step from a draft version
func (s *SOPService) DeleteStep(draftID int, stepID int, userID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the draft exists and belongs to the user
		draft, err := s.versionRepo.GetByID(draftID)
		if err != nil {
			return fmt.Errorf("failed to get draft: %w", err)
		}

		if draft.Status != models.VersionStatusDraft {
			return fmt.Errorf("version is not a draft")
		}

		if draft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 2. Verify the step belongs to this draft
		_, err = s.stepRepo.GetByIDAndVersionID(stepID, draftID)
		if err != nil {
			return fmt.Errorf("failed to get step: %w", err)
		}

		// 3. Delete the step
		if err := s.stepRepo.DeleteWithTx(tx, stepID); err != nil {
			log.Println("Failed to delete step:", err)
			return fmt.Errorf("failed to delete step: %w", err)
		}

		return nil
	})
}

// UpdateStepForTemplate updates a step in the template's draft (creates draft if needed)
func (s *SOPService) UpdateStepForTemplate(templateID int, stepID int, dto *dtos.UpdateStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the draft for this template (must exist to have steps)
		draft, err := s.versionRepo.GetDraftByTemplateID(templateID)
		if err != nil || draft == nil {
			return fmt.Errorf("no draft found for this template")
		}

		// Verify the user owns this draft
		if draft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 2. Get the step and verify it belongs to this draft
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, draft.ID)
		if err != nil {
			return fmt.Errorf("failed to get step: %w", err)
		}

		// 3. Update only provided fields
		if dto.Title != nil {
			step.Title = *dto.Title
		}
		if dto.Instructions != nil {
			step.Instructions = dto.Instructions
		}
		if dto.EstimatedTimeMinutes != nil {
			step.EstimatedTimeMinutes = dto.EstimatedTimeMinutes
		}
		if dto.ImageURL != nil {
			step.ImageURL = dto.ImageURL
		}
		if dto.VideoURL != nil {
			step.VideoURL = dto.VideoURL
		}
		if dto.RequiresApproval != nil {
			step.RequiresApproval = *dto.RequiresApproval
		}

		if err := s.stepRepo.UpdateWithTx(tx, step); err != nil {
			log.Println("Failed to update step:", err)
			return fmt.Errorf("failed to update step: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return step, nil
}

// DeleteStepForTemplate deletes a step from the template's draft
func (s *SOPService) DeleteStepForTemplate(templateID int, stepID int, userID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the draft for this template
		draft, err := s.versionRepo.GetDraftByTemplateID(templateID)
		if err != nil || draft == nil {
			return fmt.Errorf("no draft found for this template")
		}

		// Verify the user owns this draft
		if draft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 2. Verify the step belongs to this draft
		_, err = s.stepRepo.GetByIDAndVersionID(stepID, draft.ID)
		if err != nil {
			return fmt.Errorf("failed to get step: %w", err)
		}

		// 3. Delete the step
		if err := s.stepRepo.DeleteWithTx(tx, stepID); err != nil {
			log.Println("Failed to delete step:", err)
			return fmt.Errorf("failed to delete step: %w", err)
		}

		return nil
	})
}

// ReorderStep updates the order of a single step by moving it between two other steps
func (s *SOPService) ReorderStep(draftID int, stepID int, dto *dtos.ReorderStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the draft exists and belongs to the user
		draft, err := s.versionRepo.GetByID(draftID)
		if err != nil {
			return fmt.Errorf("failed to get draft: %w", err)
		}

		if draft.Status != models.VersionStatusDraft {
			return fmt.Errorf("version is not a draft")
		}

		if draft.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 2. Get the step to reorder
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, draftID)
		if err != nil {
			return fmt.Errorf("failed to get step: %w", err)
		}

		// 3. Get order values for before and after steps
		beforeOrder, afterOrder, err := s.stepRepo.GetOrderBeforeAndAfter(draftID, dto.BeforeStepID, dto.AfterStepID)
		if err != nil {
			return fmt.Errorf("failed to get order bounds: %w", err)
		}

		// 4. Generate new order value
		newOrder := utils.GenerateOrderBetween(beforeOrder, afterOrder)

		// 5. Update the step's order
		if err := s.stepRepo.UpdateOrderWithTx(tx, stepID, newOrder); err != nil {
			log.Println("Failed to update step order:", err)
			return fmt.Errorf("failed to update step order: %w", err)
		}

		step.Order = newOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	return step, nil
}

// ReorderStepForTemplate updates the order of a step in the template's draft
func (s *SOPService) ReorderStepForTemplate(templateID int, stepID int, dto *dtos.ReorderStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	log.Printf("ReorderStep - templateID: %d, stepID: %d, beforeStepID: %v, afterStepID: %v",
		templateID, stepID, dto.BeforeStepID, dto.AfterStepID)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the draft for this template
		draft, err := s.versionRepo.GetDraftByTemplateID(templateID)
		if err != nil || draft == nil {
			log.Printf("ReorderStep - no draft found for template %d: %v", templateID, err)
			return fmt.Errorf("no draft found for this template")
		}
		log.Printf("ReorderStep - found draft ID: %d", draft.ID)

		// Verify the user owns this draft
		if draft.CreatedBy.String() != userID.String() {
			log.Printf("ReorderStep - unauthorized: user %s doesn't own draft (owner: %s)", userID.String(), draft.CreatedBy.String())
			return fmt.Errorf("unauthorized to modify this draft")
		}

		// 2. Get the step to reorder
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, draft.ID)
		if err != nil {
			log.Printf("ReorderStep - failed to get step %d: %v", stepID, err)
			return fmt.Errorf("failed to get step: %w", err)
		}
		log.Printf("ReorderStep - current step order: %s", step.Order)

		// 3. Get order values for before and after steps
		beforeOrder, afterOrder, err := s.stepRepo.GetOrderBeforeAndAfter(draft.ID, dto.BeforeStepID, dto.AfterStepID)
		if err != nil {
			log.Printf("ReorderStep - failed to get order bounds: %v", err)
			return fmt.Errorf("failed to get order bounds: %w", err)
		}
		log.Printf("ReorderStep - beforeOrder: '%s', afterOrder: '%s'", beforeOrder, afterOrder)

		// 4. Generate new order value
		newOrder := utils.GenerateOrderBetween(beforeOrder, afterOrder)
		log.Printf("ReorderStep - generated new order: %s", newOrder)

		// 5. Check if we need to perform full rebalancing (empty string indicates edge case)
		if newOrder == "" {
			log.Printf("ReorderStep - edge case detected, performing full rebalancing")

			// Get all steps in their current order
			allSteps, err := s.stepRepo.GetByVersionIDWithTx(tx, draft.ID)
			if err != nil {
				log.Printf("ReorderStep - failed to get all steps for rebalancing: %v", err)
				return fmt.Errorf("failed to get steps for rebalancing: %w", err)
			}

			if len(allSteps) == 0 {
				log.Printf("ReorderStep - no steps found for rebalancing")
				return fmt.Errorf("no steps found for rebalancing")
			}

			log.Printf("ReorderStep - rebalancing %d steps", len(allSteps))

			// Build the desired final order of step IDs
			var finalStepOrder []int
			movedStepIncluded := false

			for _, s := range allSteps {
				// Determine where to insert the moved step
				if !movedStepIncluded {
					// Check if we should insert before this step
					if (beforeOrder == "" && s.ID == allSteps[0].ID) || // Insert at beginning
						(dto.AfterStepID != nil && s.ID == *dto.AfterStepID) { // Insert after specified step
						finalStepOrder = append(finalStepOrder, stepID)
						movedStepIncluded = true
					}
				}

				// Skip the moved step in its current position
				if s.ID != stepID {
					finalStepOrder = append(finalStepOrder, s.ID)
				}
			}

			// If we're inserting at the end, add it now
			if !movedStepIncluded {
				finalStepOrder = append(finalStepOrder, stepID)
			}

			log.Printf("ReorderStep - final step order: %v", finalStepOrder)

			// Generate clean order values for all steps
			newOrders := utils.RebalanceOrders(len(finalStepOrder))

			// Update all steps with their new order values
			for i, sid := range finalStepOrder {
				newOrder := newOrders[i]
				if err := s.stepRepo.UpdateOrderWithTx(tx, sid, newOrder); err != nil {
					log.Printf("ReorderStep - failed to update step %d order to %s: %v", sid, newOrder, err)
					return fmt.Errorf("failed to update step order: %w", err)
				}
				log.Printf("ReorderStep - updated step %d to order '%s'", sid, newOrder)

				// Update the step variable if it's the one we're moving
				if sid == stepID {
					step.Order = newOrder
				}
			}

			log.Printf("ReorderStep - full rebalancing complete")
			return nil
		}

		// 6. Update the step's order
		if err := s.stepRepo.UpdateOrderWithTx(tx, stepID, newOrder); err != nil {
			log.Printf("ReorderStep - failed to update step order: %v", err)
			return fmt.Errorf("failed to update step order: %w", err)
		}

		step.Order = newOrder
		log.Printf("ReorderStep - successfully updated step %d to order %s", stepID, newOrder)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return step, nil
}
