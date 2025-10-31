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
