package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

// SOPVersioningService handles auto-versioning and version comparison for SOPs.
// It creates new versions by deep-copying steps, sub-steps, media, and BOM items,
// and provides diff functionality to compare two versions.
type SOPVersioningService struct {
	db           *gorm.DB
	templateRepo *repositories.SOPTemplateRepository
	versionRepo  *repositories.SOPVersionRepository
	stepRepo     *repositories.SOPStepRepository
	subStepRepo  *repositories.SOPSubStepRepository
	mediaRepo    *repositories.SOPStepMediaRepository
	bomRepo      *repositories.BOMItemRepository
}

func NewSOPVersioningService(
	db *gorm.DB,
	templateRepo *repositories.SOPTemplateRepository,
	versionRepo *repositories.SOPVersionRepository,
	stepRepo *repositories.SOPStepRepository,
	subStepRepo *repositories.SOPSubStepRepository,
	mediaRepo *repositories.SOPStepMediaRepository,
	bomRepo *repositories.BOMItemRepository,
) *SOPVersioningService {
	return &SOPVersioningService{
		db:           db,
		templateRepo: templateRepo,
		versionRepo:  versionRepo,
		stepRepo:     stepRepo,
		subStepRepo:  subStepRepo,
		mediaRepo:    mediaRepo,
		bomRepo:      bomRepo,
	}
}

// CreateNewVersion creates a new SOPVersion by deep-copying all content from the
// current version (steps, sub-steps, media references, BOM items) and updates
// the template's CurrentVersionID to point to it.
//
// Parameters:
//   - templateID: the SOPTemplate to version
//   - userID: the user creating the version
//   - changeSummary: optional description of what changed
//   - description: optional version-level description (nil keeps previous)
func (s *SOPVersioningService) CreateNewVersion(
	templateID int,
	userID uuid.UUID,
	changeSummary *string,
	description *string,
) (*models.SOPVersion, error) {
	var newVersion *models.SOPVersion

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Get the template with its current version
		template, err := s.templateRepo.GetByIDWithTx(tx, templateID)
		if err != nil {
			return fmt.Errorf("failed to get SOP template: %w", err)
		}

		// 2. Get the latest version number for incrementing
		latestVersionNumber, err := s.versionRepo.GetLatestVersionNumber(templateID)
		if err != nil {
			return fmt.Errorf("failed to get latest version number: %w", err)
		}

		// 3. Determine the description for the new version
		versionDescription := description
		if versionDescription == nil && template.CurrentVersion != nil {
			versionDescription = template.CurrentVersion.Description
		}

		// 4. Create the new version record
		newVersion = &models.SOPVersion{
			SOPTemplateID: templateID,
			VersionNumber: latestVersionNumber + 1,
			Description:   versionDescription,
			CreatedBy:     userID,
			ChangeSummary: changeSummary,
			IsActive:      true,
		}

		if err := tx.Create(newVersion).Error; err != nil {
			log.Println("Failed to create new SOP version:", err)
			return fmt.Errorf("failed to create new SOP version: %w", err)
		}

		// 5. Deep-copy from the current version if it exists
		if template.CurrentVersion != nil {
			if err := s.deepCopyVersion(tx, template.CurrentVersion.ID, newVersion.ID); err != nil {
				return fmt.Errorf("failed to deep-copy version: %w", err)
			}
		}

		// 6. Update the template's current_version_id
		if err := s.templateRepo.UpdateCurrentVersionWithTx(tx, templateID, newVersion.ID); err != nil {
			log.Println("Failed to update template current version:", err)
			return fmt.Errorf("failed to update template current version: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Reload the version with its steps for the response
	loaded, err := s.versionRepo.GetByID(newVersion.ID)
	if err != nil {
		return newVersion, nil // Return the version even if reload fails
	}
	return loaded, nil
}

// deepCopyVersion copies all steps, sub-steps, media references, and BOM items
// from one version to another within a transaction.
// It builds mapping tables (old step ID → new step ID, old sub-step ID → new sub-step ID)
// so that media and BOM item references are correctly remapped.
func (s *SOPVersioningService) deepCopyVersion(tx *gorm.DB, fromVersionID, toVersionID int) error {
	// 1. Copy steps
	oldSteps, err := s.stepRepo.GetByVersionIDWithTx(tx, fromVersionID)
	if err != nil {
		return fmt.Errorf("failed to get steps for copying: %w", err)
	}

	// Maps from old IDs to new IDs for remapping child references
	stepIDMap := make(map[int]int)    // old step ID → new step ID
	subStepIDMap := make(map[int]int) // old sub-step ID → new sub-step ID

	for _, oldStep := range oldSteps {
		newStep := models.SOPStep{
			SOPVersionID:         toVersionID,
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

		if err := tx.Create(&newStep).Error; err != nil {
			return fmt.Errorf("failed to copy step: %w", err)
		}
		stepIDMap[oldStep.ID] = newStep.ID

		// 2. Copy sub-steps for this step
		oldSubSteps, err := s.subStepRepo.GetByStepIDWithTx(tx, oldStep.ID)
		if err != nil {
			return fmt.Errorf("failed to get sub-steps for copying: %w", err)
		}

		for _, oldSubStep := range oldSubSteps {
			newSubStep := models.SOPSubStep{
				SOPStepID:    newStep.ID,
				DisplayOrder: oldSubStep.DisplayOrder,
				Title:        oldSubStep.Title,
				Instructions: oldSubStep.Instructions,
			}

			if err := tx.Create(&newSubStep).Error; err != nil {
				return fmt.Errorf("failed to copy sub-step: %w", err)
			}
			subStepIDMap[oldSubStep.ID] = newSubStep.ID

			// 3. Copy media for this sub-step
			oldMedia, err := s.mediaRepo.GetBySubStepIDWithTx(tx, oldSubStep.ID)
			if err != nil {
				return fmt.Errorf("failed to get sub-step media for copying: %w", err)
			}

			for _, m := range oldMedia {
				newSubStepID := newSubStep.ID
				newMedia := models.SOPStepMedia{
					SOPSubStepID: &newSubStepID,
					UUID:         m.UUID, // Same file reference
					FilePath:     m.FilePath,
					FileName:     m.FileName,
					MimeType:     m.MimeType,
					FileSize:     m.FileSize,
					Duration:     m.Duration,
					Order:        m.Order,
				}

				if err := tx.Create(&newMedia).Error; err != nil {
					return fmt.Errorf("failed to copy sub-step media: %w", err)
				}
			}
		}

		// 4. Copy media for this step (step-level, not sub-step-level)
		oldStepMedia, err := s.mediaRepo.GetByStepIDWithTx(tx, oldStep.ID)
		if err != nil {
			return fmt.Errorf("failed to get step media for copying: %w", err)
		}

		for _, m := range oldStepMedia {
			newStepID := newStep.ID
			newMedia := models.SOPStepMedia{
				SOPStepID: &newStepID,
				UUID:      m.UUID,
				FilePath:  m.FilePath,
				FileName:  m.FileName,
				MimeType:  m.MimeType,
				FileSize:  m.FileSize,
				Duration:  m.Duration,
				Order:     m.Order,
			}

			if err := tx.Create(&newMedia).Error; err != nil {
				return fmt.Errorf("failed to copy step media: %w", err)
			}
		}
	}

	// 5. Copy BOM items, remapping step references
	oldBOMItems, err := s.bomRepo.GetByVersionIDWithTx(tx, fromVersionID)
	if err != nil {
		return fmt.Errorf("failed to get BOM items for copying: %w", err)
	}

	for _, oldItem := range oldBOMItems {
		newItem := models.BOMItem{
			SOPVersionID: toVersionID,
			MaterialID:   oldItem.MaterialID,
			Name:         oldItem.Name,
			Quantity:     oldItem.Quantity,
			Unit:         oldItem.Unit,
			UnitCost:     oldItem.UnitCost,
			Notes:        oldItem.Notes,
		}

		// Remap step reference if present
		if oldItem.SOPStepID != nil {
			if newStepID, ok := stepIDMap[*oldItem.SOPStepID]; ok {
				newItem.SOPStepID = &newStepID
			}
		}

		if err := tx.Create(&newItem).Error; err != nil {
			return fmt.Errorf("failed to copy BOM item: %w", err)
		}
	}

	return nil
}

// VersionDiff represents the differences between two SOP versions.
type VersionDiff struct {
	FromVersionID     int           `json:"fromVersionId"`
	ToVersionID       int           `json:"toVersionId"`
	FromVersionNumber int           `json:"fromVersionNumber"`
	ToVersionNumber   int           `json:"toVersionNumber"`
	StepsAdded        []StepSummary `json:"stepsAdded"`
	StepsRemoved      []StepSummary `json:"stepsRemoved"`
	StepsModified     []StepDiff    `json:"stepsModified"`
	BOMItemsAdded     []BOMSummary  `json:"bomItemsAdded"`
	BOMItemsRemoved   []BOMSummary  `json:"bomItemsRemoved"`
	DescriptionChange *StringChange `json:"descriptionChange,omitempty"`
}

// StepSummary is a lightweight representation of a step for diff output.
type StepSummary struct {
	ID    int    `json:"id"`
	Order string `json:"order"`
	Title string `json:"title"`
}

// StepDiff describes what changed in a step between two versions.
// Steps are matched by display order position since IDs differ across versions.
type StepDiff struct {
	Order    string        `json:"order"`
	OldTitle string        `json:"oldTitle"`
	NewTitle string        `json:"newTitle"`
	Changes  []FieldChange `json:"changes"`
}

// FieldChange describes a single field that changed.
type FieldChange struct {
	Field    string `json:"field"`
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}

// StringChange describes a change to a string field (e.g., description).
type StringChange struct {
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}

// BOMSummary is a lightweight representation of a BOM item for diff output.
type BOMSummary struct {
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
	Unit     string `json:"unit"`
}

// CompareVersions produces a diff between two versions of the same SOP template.
// Both version IDs must belong to the same template.
func (s *SOPVersioningService) CompareVersions(fromVersionID, toVersionID int) (*VersionDiff, error) {
	// 1. Load both versions with their steps
	fromVersion, err := s.versionRepo.GetByID(fromVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from-version: %w", err)
	}

	toVersion, err := s.versionRepo.GetByID(toVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get to-version: %w", err)
	}

	// Validate both versions belong to the same template
	if fromVersion.SOPTemplateID != toVersion.SOPTemplateID {
		return nil, fmt.Errorf("versions belong to different templates (%d vs %d)",
			fromVersion.SOPTemplateID, toVersion.SOPTemplateID)
	}

	diff := &VersionDiff{
		FromVersionID:     fromVersionID,
		ToVersionID:       toVersionID,
		FromVersionNumber: fromVersion.VersionNumber,
		ToVersionNumber:   toVersion.VersionNumber,
	}

	// 2. Compare descriptions
	oldDesc := ptrToString(fromVersion.Description)
	newDesc := ptrToString(toVersion.Description)
	if oldDesc != newDesc {
		diff.DescriptionChange = &StringChange{
			OldValue: oldDesc,
			NewValue: newDesc,
		}
	}

	// 3. Compare steps by matching on order position
	fromSteps, err := s.stepRepo.GetByVersionID(fromVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from-version steps: %w", err)
	}

	toSteps, err := s.stepRepo.GetByVersionID(toVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get to-version steps: %w", err)
	}

	// Build maps keyed by order for matching
	fromStepMap := make(map[string]*models.SOPStep)
	for i := range fromSteps {
		fromStepMap[fromSteps[i].Order] = &fromSteps[i]
	}

	toStepMap := make(map[string]*models.SOPStep)
	for i := range toSteps {
		toStepMap[toSteps[i].Order] = &toSteps[i]
	}

	// Find added and modified steps
	for _, toStep := range toSteps {
		fromStep, exists := fromStepMap[toStep.Order]
		if !exists {
			diff.StepsAdded = append(diff.StepsAdded, StepSummary{
				ID:    toStep.ID,
				Order: toStep.Order,
				Title: toStep.Title,
			})
			continue
		}

		// Check for modifications
		changes := compareSteps(fromStep, &toStep)
		if len(changes) > 0 {
			diff.StepsModified = append(diff.StepsModified, StepDiff{
				Order:    toStep.Order,
				OldTitle: fromStep.Title,
				NewTitle: toStep.Title,
				Changes:  changes,
			})
		}
	}

	// Find removed steps
	for _, fromStep := range fromSteps {
		if _, exists := toStepMap[fromStep.Order]; !exists {
			diff.StepsRemoved = append(diff.StepsRemoved, StepSummary{
				ID:    fromStep.ID,
				Order: fromStep.Order,
				Title: fromStep.Title,
			})
		}
	}

	// 4. Compare BOM items
	fromBOM, err := s.bomRepo.GetByVersionID(fromVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from-version BOM items: %w", err)
	}

	toBOM, err := s.bomRepo.GetByVersionID(toVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get to-version BOM items: %w", err)
	}

	// Build maps keyed by name+unit for matching
	fromBOMMap := make(map[string]*models.BOMItem)
	for i := range fromBOM {
		key := fromBOM[i].Name + "|" + fromBOM[i].Unit
		fromBOMMap[key] = &fromBOM[i]
	}

	toBOMMap := make(map[string]*models.BOMItem)
	for i := range toBOM {
		key := toBOM[i].Name + "|" + toBOM[i].Unit
		toBOMMap[key] = &toBOM[i]
	}

	for _, item := range toBOM {
		key := item.Name + "|" + item.Unit
		if _, exists := fromBOMMap[key]; !exists {
			diff.BOMItemsAdded = append(diff.BOMItemsAdded, BOMSummary{
				Name:     item.Name,
				Quantity: item.Quantity.String(),
				Unit:     item.Unit,
			})
		}
	}

	for _, item := range fromBOM {
		key := item.Name + "|" + item.Unit
		if _, exists := toBOMMap[key]; !exists {
			diff.BOMItemsRemoved = append(diff.BOMItemsRemoved, BOMSummary{
				Name:     item.Name,
				Quantity: item.Quantity.String(),
				Unit:     item.Unit,
			})
		}
	}

	return diff, nil
}

// compareSteps returns field-level changes between two steps.
func compareSteps(from, to *models.SOPStep) []FieldChange {
	var changes []FieldChange

	if from.Title != to.Title {
		changes = append(changes, FieldChange{
			Field:    "title",
			OldValue: from.Title,
			NewValue: to.Title,
		})
	}

	if ptrToString(from.Instructions) != ptrToString(to.Instructions) {
		changes = append(changes, FieldChange{
			Field:    "instructions",
			OldValue: ptrToString(from.Instructions),
			NewValue: ptrToString(to.Instructions),
		})
	}

	if ptrToInt(from.EstimatedTimeMinutes) != ptrToInt(to.EstimatedTimeMinutes) {
		changes = append(changes, FieldChange{
			Field:    "estimatedTimeMinutes",
			OldValue: fmt.Sprintf("%d", ptrToInt(from.EstimatedTimeMinutes)),
			NewValue: fmt.Sprintf("%d", ptrToInt(to.EstimatedTimeMinutes)),
		})
	}

	if from.RequiresApproval != to.RequiresApproval {
		changes = append(changes, FieldChange{
			Field:    "requiresApproval",
			OldValue: fmt.Sprintf("%t", from.RequiresApproval),
			NewValue: fmt.Sprintf("%t", to.RequiresApproval),
		})
	}

	fromStationID := ptrUUIDToString(from.StationID)
	toStationID := ptrUUIDToString(to.StationID)
	if fromStationID != toStationID {
		changes = append(changes, FieldChange{
			Field:    "stationId",
			OldValue: fromStationID,
			NewValue: toStationID,
		})
	}

	if ptrToInt2(from.LinkedSOPTemplateID) != ptrToInt2(to.LinkedSOPTemplateID) {
		changes = append(changes, FieldChange{
			Field:    "linkedSopTemplateId",
			OldValue: fmt.Sprintf("%d", ptrToInt2(from.LinkedSOPTemplateID)),
			NewValue: fmt.Sprintf("%d", ptrToInt2(to.LinkedSOPTemplateID)),
		})
	}

	return changes
}

// Helper functions for nil-safe comparisons

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrToInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func ptrToInt2(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func ptrUUIDToString(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}
