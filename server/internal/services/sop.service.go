package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
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
		for _, stepDTO := range dto.Steps {
			step := models.SOPStep{
				SOPTemplateVersionID: version.ID,
				StepNumber:           stepDTO.StepNumber,
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
		for _, stepDTO := range dto.Steps {
			step := models.SOPStep{
				SOPTemplateVersionID: newVersion.ID,
				StepNumber:           stepDTO.StepNumber,
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

// GetSOP gets an SOP template by ID with its current version
func (s *SOPService) GetSOP(templateID int) (*models.SOPTemplate, error) {
	return s.templateRepo.GetByID(templateID)
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
		for _, stepDTO := range dto.Steps {
			step := models.SOPStep{
				SOPTemplateVersionID: draft.ID,
				StepNumber:           stepDTO.StepNumber,
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

		// 6. Create maps for efficient lookup
		existingStepMap := make(map[int]*models.SOPStep)
		for i := range existingSteps {
			existingStepMap[existingSteps[i].StepNumber] = &existingSteps[i]
		}

		dtoStepMap := make(map[int]bool)
		var updatedSteps []models.SOPStep

		// 7. Update or insert steps from DTO
		for _, stepDTO := range dto.Steps {
			dtoStepMap[stepDTO.StepNumber] = true

			if existingStep, exists := existingStepMap[stepDTO.StepNumber]; exists {
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
					StepNumber:           stepDTO.StepNumber,
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
		for stepNumber, existingStep := range existingStepMap {
			if !dtoStepMap[stepNumber] {
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

		// 3. Update the draft to published
		draft.Status = models.VersionStatusPublished
		summary := changeSummary
		draft.ChangeSummary = &summary

		if err := s.versionRepo.Update(draft); err != nil {
			log.Println("Failed to publish draft:", err)
			return fmt.Errorf("failed to publish draft: %w", err)
		}

		// 4. Update the template's current_version_id
		existingTemplate, err := s.templateRepo.GetByID(draft.SOPTemplateID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		existingTemplate.CurrentVersionID = &draft.ID
		if err := s.templateRepo.Update(existingTemplate); err != nil {
			log.Println("Failed to update template current version:", err)
			return fmt.Errorf("failed to update template current version: %w", err)
		}

		existingTemplate.CurrentVersion = draft
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
