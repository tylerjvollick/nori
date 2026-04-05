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
	versionRepo  *repositories.SOPVersionRepository
	stepRepo     *repositories.SOPStepRepository
}

// parseOptionalUUID converts an optional string to *uuid.UUID.
// Returns nil if the input is nil or empty.
func parseOptionalUUID(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*s)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}
	return &parsed, nil
}

func NewSOPService(
	db *gorm.DB,
	templateRepo *repositories.SOPTemplateRepository,
	versionRepo *repositories.SOPVersionRepository,
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
func (s *SOPService) CreateSOP(dto *dtos.CreateSOPDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPTemplate, error) {
	var template *models.SOPTemplate

	// Use transaction to ensure all creates succeed or all fail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Create the SOP template (without current_version_id yet)
		template = &models.SOPTemplate{
			Name:      dto.Name,
			SpaceID:   &spaceID,
			CreatedBy: userID,
		}

		if err := s.templateRepo.Create(template); err != nil {
			log.Println("Failed to create SOP template:", err)
			return fmt.Errorf("failed to create SOP template: %w", err)
		}

		// 2. Create the first version
		version := &models.SOPVersion{
			SOPTemplateID: template.ID,
			VersionNumber: 1,
			Description:   dto.Description,
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
				SOPVersionID:         version.ID,
				Order:                order,
				Title:                stepDTO.Title,
				Instructions:         stepDTO.Instructions,
				EstimatedTimeMinutes: stepDTO.EstimatedTimeMinutes,
				ImageURL:             stepDTO.ImageURL,
				VideoURL:             stepDTO.VideoURL,
				RequiresApproval:     stepDTO.RequiresApproval,
				LinkedSOPTemplateID:  stepDTO.LinkedSOPTemplateID,
			}
			stationID, err := parseOptionalUUID(stepDTO.StationID)
			if err != nil {
				return fmt.Errorf("invalid station ID on step %d: %w", i, err)
			}
			step.StationID = stationID
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

// UpdateSOP creates a new version of an existing SOP (auto-versioning)
func (s *SOPService) UpdateSOP(templateID int, dto *dtos.UpdateSOPDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPTemplate, error) {
	var template *models.SOPTemplate

	// Use transaction to ensure all creates succeed or all fail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the existing template (scoped to space)
		existingTemplate, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
		if err != nil {
			return fmt.Errorf("failed to get SOP template: %w", err)
		}

		// 2. Get the latest version number
		latestVersionNumber, err := s.versionRepo.GetLatestVersionNumber(templateID)
		if err != nil {
			return fmt.Errorf("failed to get latest version number: %w", err)
		}

		// 3. Create the new version
		newVersion := &models.SOPVersion{
			SOPTemplateID: templateID,
			VersionNumber: latestVersionNumber + 1,
			Description:   dto.Description,
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
				SOPVersionID:         newVersion.ID,
				Order:                order,
				Title:                stepDTO.Title,
				Instructions:         stepDTO.Instructions,
				EstimatedTimeMinutes: stepDTO.EstimatedTimeMinutes,
				ImageURL:             stepDTO.ImageURL,
				VideoURL:             stepDTO.VideoURL,
				RequiresApproval:     stepDTO.RequiresApproval,
				LinkedSOPTemplateID:  stepDTO.LinkedSOPTemplateID,
			}
			stationID, err := parseOptionalUUID(stepDTO.StationID)
			if err != nil {
				return fmt.Errorf("invalid station ID on step %d: %w", i, err)
			}
			step.StationID = stationID
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

// GetSOP gets an SOP template by ID, scoped to a space
func (s *SOPService) GetSOP(templateID int, spaceID uuid.UUID) (*models.SOPTemplate, error) {
	template, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
	if err != nil {
		return nil, err
	}

	// Load steps for the current version if it exists
	if template.CurrentVersion != nil {
		steps, err := s.stepRepo.GetByVersionID(template.CurrentVersion.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load current version steps: %w", err)
		}
		template.CurrentVersion.Steps = steps
	}

	return template, nil
}

// GetAllSOPs gets all SOP templates for a space
func (s *SOPService) GetAllSOPs(spaceID uuid.UUID) ([]models.SOPTemplate, error) {
	return s.templateRepo.GetBySpaceID(spaceID)
}

// GetSOPVersions gets all versions of an SOP template, scoped to a space
func (s *SOPService) GetSOPVersions(templateID int, spaceID uuid.UUID) ([]models.SOPVersion, error) {
	// Verify the template belongs to this space
	_, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
	if err != nil {
		return nil, err
	}
	return s.versionRepo.GetByTemplateID(templateID)
}

// GetSOPVersion gets a specific version of an SOP template
func (s *SOPService) GetSOPVersion(versionID int) (*models.SOPVersion, error) {
	return s.versionRepo.GetByID(versionID)
}

// DeleteSOP deletes an SOP template and all its versions/steps (cascade)
// First verifies the template belongs to the specified space
func (s *SOPService) DeleteSOP(templateID int, spaceID uuid.UUID) error {
	// Verify the template belongs to this space before deleting
	_, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
	if err != nil {
		return err
	}
	return s.templateRepo.Delete(templateID)
}

// Individual step operations

// CreateStep creates a single step in a version
func (s *SOPService) CreateStep(versionID int, dto *dtos.CreateStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the version exists and belongs to the user
		version, err := s.versionRepo.GetByID(versionID)
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}

		if version.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this version")
		}

		// 2. Determine order for the new step
		var order string
		if dto.AfterStepID != nil {
			// Get the step after which we're inserting
			afterStep, err := s.stepRepo.GetByIDAndVersionID(*dto.AfterStepID, versionID)
			if err != nil {
				return fmt.Errorf("failed to get after step: %w", err)
			}

			// Get the last order to find what comes next
			lastOrder, err := s.stepRepo.GetLastOrderByVersionID(versionID)
			if err != nil {
				return fmt.Errorf("failed to get last order: %w", err)
			}

			// Generate order after the specified step
			if afterStep.Order == lastOrder {
				// Inserting at the end
				order = utils.GenerateOrderBetween(afterStep.Order, "")
			} else {
				// Find the next step
				allSteps, err := s.stepRepo.GetByVersionID(versionID)
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
			allSteps, err := s.stepRepo.GetByVersionID(versionID)
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
		stationID, err := parseOptionalUUID(dto.StationID)
		if err != nil {
			return fmt.Errorf("invalid station ID: %w", err)
		}
		step = &models.SOPStep{
			SOPVersionID:         versionID,
			Order:                order,
			Title:                dto.Title,
			Instructions:         dto.Instructions,
			EstimatedTimeMinutes: dto.EstimatedTimeMinutes,
			ImageURL:             dto.ImageURL,
			VideoURL:             dto.VideoURL,
			RequiresApproval:     dto.RequiresApproval,
			StationID:            stationID,
			LinkedSOPTemplateID:  dto.LinkedSOPTemplateID,
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

// CreateStepForTemplate creates a step for a template's current version (creates new version if needed)
func (s *SOPService) CreateStepForTemplate(templateID int, dto *dtos.CreateStepDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the template (scoped to space)
		template, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		// 2. Get or create a new version for this template
		latestVersionNumber, err := s.versionRepo.GetLatestVersionNumber(templateID)
		if err != nil {
			return fmt.Errorf("failed to get latest version number: %w", err)
		}

		// Create a new version based on the current version
		newVersion := &models.SOPVersion{
			SOPTemplateID: templateID,
			VersionNumber: latestVersionNumber + 1,
			CreatedBy:     userID,
			IsActive:      true,
		}

		// Copy fields from current version if it exists
		if template.CurrentVersion != nil {
			newVersion.Description = template.CurrentVersion.Description
		}

		if err := tx.Create(newVersion).Error; err != nil {
			return fmt.Errorf("failed to create version: %w", err)
		}

		// Copy steps from current version if they exist
		var existingSteps []models.SOPStep
		if template.CurrentVersion != nil {
			currentSteps, err := s.stepRepo.GetByVersionID(template.CurrentVersion.ID)
			if err == nil && len(currentSteps) > 0 {
				newSteps := make([]models.SOPStep, len(currentSteps))
				for i, oldStep := range currentSteps {
					newSteps[i] = models.SOPStep{
						SOPVersionID:         newVersion.ID,
						Order:                oldStep.Order,
						Title:                oldStep.Title,
						Instructions:         oldStep.Instructions,
						EstimatedTimeMinutes: oldStep.EstimatedTimeMinutes,
						ImageURL:             oldStep.ImageURL,
						VideoURL:             oldStep.VideoURL,
						RequiresApproval:     oldStep.RequiresApproval,
						StationID:            oldStep.StationID,
						LinkedSOPTemplateID:  oldStep.LinkedSOPTemplateID,
					}
				}
				if err := s.stepRepo.CreateBatchWithTx(tx, newSteps); err != nil {
					return fmt.Errorf("failed to copy steps: %w", err)
				}
				existingSteps = newSteps
			}
		}

		// 3. Determine order for the new step
		var order string
		if dto.AfterStepID != nil {
			// The afterStepID might refer to a step in a previous version,
			// so we need to find the corresponding step in the new version by order value

			// Try to get the step from the new version first
			afterStep, err := s.stepRepo.GetByIDAndVersionID(*dto.AfterStepID, newVersion.ID)
			if err != nil {
				// If not found in new version, look it up in the current version to get its order
				if template.CurrentVersion != nil {
					publishedStep, err := s.stepRepo.GetByIDAndVersionID(*dto.AfterStepID, template.CurrentVersion.ID)
					if err != nil {
						return fmt.Errorf("failed to get after step: step not found in this version")
					}

					// Now find the step with the same order in the new version
					for i := range existingSteps {
						if existingSteps[i].Order == publishedStep.Order {
							afterStep = &existingSteps[i]
							break
						}
					}
					if afterStep == nil {
						return fmt.Errorf("failed to find corresponding step in version with order='%s'", publishedStep.Order)
					}
				} else {
					return fmt.Errorf("failed to get after step: %w", err)
				}
			}

			// Get the last order to find what comes next
			lastOrder, err := s.stepRepo.GetLastOrderByVersionID(newVersion.ID)
			if err != nil {
				return fmt.Errorf("failed to get last order: %w", err)
			}

			// Generate order after the specified step
			if afterStep.Order == lastOrder {
				// Inserting at the end
				order = utils.GenerateOrderBetween(afterStep.Order, "")
			} else {
				// Find the next step using in-memory steps
				var nextOrder string
				foundAfter := false
				for _, s := range existingSteps {
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
			// Inserting at the beginning
			if len(existingSteps) > 0 {
				order = utils.GenerateOrderBetween("", existingSteps[0].Order)
			} else {
				order = utils.GenerateOrderBetween("", "")
			}
		}

		// 4. Create the new step
		stationID, err := parseOptionalUUID(dto.StationID)
		if err != nil {
			return fmt.Errorf("invalid station ID: %w", err)
		}
		step = &models.SOPStep{
			SOPVersionID:         newVersion.ID,
			Order:                order,
			Title:                dto.Title,
			Instructions:         dto.Instructions,
			EstimatedTimeMinutes: dto.EstimatedTimeMinutes,
			ImageURL:             dto.ImageURL,
			VideoURL:             dto.VideoURL,
			RequiresApproval:     dto.RequiresApproval,
			StationID:            stationID,
			LinkedSOPTemplateID:  dto.LinkedSOPTemplateID,
		}

		if err := tx.Create(step).Error; err != nil {
			log.Println("Failed to create step:", err)
			return fmt.Errorf("failed to create step: %w", err)
		}

		// 5. Update the template's current_version_id
		if err := s.templateRepo.UpdateCurrentVersionWithTx(tx, templateID, newVersion.ID); err != nil {
			log.Println("Failed to update template current version:", err)
			return fmt.Errorf("failed to update template current version: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return step, nil
}

// UpdateStep updates a single step in a version
func (s *SOPService) UpdateStep(versionID int, stepID int, dto *dtos.UpdateStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the version exists and belongs to the user
		version, err := s.versionRepo.GetByID(versionID)
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}

		if version.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this version")
		}

		// 2. Get the step and verify it belongs to this version
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, versionID)
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
		if dto.StationID != nil {
			stationID, err := parseOptionalUUID(dto.StationID)
			if err != nil {
				return fmt.Errorf("invalid station ID: %w", err)
			}
			step.StationID = stationID
		}
		if dto.LinkedSOPTemplateID != nil {
			step.LinkedSOPTemplateID = dto.LinkedSOPTemplateID
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

// DeleteStep deletes a single step from a version
func (s *SOPService) DeleteStep(versionID int, stepID int, userID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the version exists and belongs to the user
		version, err := s.versionRepo.GetByID(versionID)
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}

		if version.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this version")
		}

		// 2. Verify the step belongs to this version
		_, err = s.stepRepo.GetByIDAndVersionID(stepID, versionID)
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

// UpdateStepForTemplate updates a step in the template's current version
func (s *SOPService) UpdateStepForTemplate(templateID int, stepID int, dto *dtos.UpdateStepDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the template to find its current version (scoped to space)
		template, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		if template.CurrentVersionID == nil {
			return fmt.Errorf("no current version found for this template")
		}

		versionID := *template.CurrentVersionID

		// 2. Get the step and verify it belongs to the current version
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, versionID)
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
		if dto.StationID != nil {
			stationID, err := parseOptionalUUID(dto.StationID)
			if err != nil {
				return fmt.Errorf("invalid station ID: %w", err)
			}
			step.StationID = stationID
		}
		if dto.LinkedSOPTemplateID != nil {
			step.LinkedSOPTemplateID = dto.LinkedSOPTemplateID
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

// DeleteStepForTemplate deletes a step from the template's current version
func (s *SOPService) DeleteStepForTemplate(templateID int, stepID int, userID uuid.UUID, spaceID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the template to find its current version (scoped to space)
		template, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		if template.CurrentVersionID == nil {
			return fmt.Errorf("no current version found for this template")
		}

		versionID := *template.CurrentVersionID

		// 2. Verify the step belongs to the current version
		_, err = s.stepRepo.GetByIDAndVersionID(stepID, versionID)
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
func (s *SOPService) ReorderStep(versionID int, stepID int, dto *dtos.ReorderStepDTO, userID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Verify the version exists and belongs to the user
		version, err := s.versionRepo.GetByID(versionID)
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}

		if version.CreatedBy.String() != userID.String() {
			return fmt.Errorf("unauthorized to modify this version")
		}

		// 2. Get the step to reorder
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, versionID)
		if err != nil {
			return fmt.Errorf("failed to get step: %w", err)
		}

		// 3. Get order values for before and after steps
		beforeOrder, afterOrder, err := s.stepRepo.GetOrderBeforeAndAfter(versionID, dto.BeforeStepID, dto.AfterStepID)
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

// ReorderStepForTemplate updates the order of a step in the template's current version
func (s *SOPService) ReorderStepForTemplate(templateID int, stepID int, dto *dtos.ReorderStepDTO, userID uuid.UUID, spaceID uuid.UUID) (*models.SOPStep, error) {
	var step *models.SOPStep

	log.Printf("ReorderStep - templateID: %d, stepID: %d, beforeStepID: %v, afterStepID: %v",
		templateID, stepID, dto.BeforeStepID, dto.AfterStepID)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the template to find its current version (scoped to space)
		template, err := s.templateRepo.GetByIDAndSpaceID(templateID, spaceID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		if template.CurrentVersionID == nil {
			return fmt.Errorf("no current version found for this template")
		}

		versionID := *template.CurrentVersionID
		log.Printf("ReorderStep - using version ID: %d", versionID)

		// 2. Get the step to reorder
		step, err = s.stepRepo.GetByIDAndVersionID(stepID, versionID)
		if err != nil {
			log.Printf("ReorderStep - failed to get step %d: %v", stepID, err)
			return fmt.Errorf("failed to get step: %w", err)
		}
		log.Printf("ReorderStep - current step order: %s", step.Order)

		// 3. Get order values for before and after steps
		beforeOrder, afterOrder, err := s.stepRepo.GetOrderBeforeAndAfter(versionID, dto.BeforeStepID, dto.AfterStepID)
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
			allSteps, err := s.stepRepo.GetByVersionIDWithTx(tx, versionID)
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
