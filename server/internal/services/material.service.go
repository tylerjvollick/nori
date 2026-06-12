package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// ErrMaterialNotFound is returned when a material does not exist or does not
// belong to the requested space.
var ErrMaterialNotFound = errors.New("material not found")

// MaterialValidationError indicates invalid input to a material operation.
// Handlers should map it to a 400 response.
type MaterialValidationError struct {
	Message string
}

func (e *MaterialValidationError) Error() string {
	return e.Message
}

func newMaterialValidationError(format string, args ...interface{}) error {
	return &MaterialValidationError{Message: fmt.Sprintf(format, args...)}
}

// MaterialRepositoryInterface defines the repository methods needed by
// MaterialService.
type MaterialRepositoryInterface interface {
	Create(material *models.Material) error
	GetByID(id uuid.UUID) (*models.Material, error)
	List(spaceID uuid.UUID, search string) ([]models.Material, error)
	Update(material *models.Material) error
	Delete(id uuid.UUID) error
}

// MaterialService implements business logic for the material catalog.
type MaterialService struct {
	materialRepo MaterialRepositoryInterface
}

// NewMaterialService creates a new MaterialService.
func NewMaterialService(materialRepo MaterialRepositoryInterface) *MaterialService {
	return &MaterialService{materialRepo: materialRepo}
}

// validMaterialCategories lists the accepted category values.
var validMaterialCategories = map[models.MaterialCategory]bool{
	models.MaterialCategoryLumber:     true,
	models.MaterialCategoryHardware:   true,
	models.MaterialCategoryFinish:     true,
	models.MaterialCategoryConsumable: true,
	models.MaterialCategoryOther:      true,
}

// Create validates and creates a new material in the given space.
func (s *MaterialService) Create(spaceID uuid.UUID, req *dtos.CreateMaterialRequest) (*models.Material, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, newMaterialValidationError("name is required")
	}

	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		return nil, newMaterialValidationError("unit is required")
	}

	if err := validateUnitCost(req.UnitCost); err != nil {
		return nil, err
	}

	category := models.MaterialCategoryOther
	if req.Category != nil && strings.TrimSpace(*req.Category) != "" {
		category = models.MaterialCategory(strings.TrimSpace(*req.Category))
		if !validMaterialCategories[category] {
			return nil, newMaterialValidationError("invalid category %q", *req.Category)
		}
	}

	material := &models.Material{
		ID:       uuid.New(),
		SpaceID:  spaceID,
		Name:     name,
		Category: category,
		Unit:     unit,
		Supplier: req.Supplier,
		SKU:      req.SKU,
		Location: req.Location,
		UnitCost: req.UnitCost,
		IsActive: true,
	}

	if err := s.materialRepo.Create(material); err != nil {
		return nil, fmt.Errorf("failed to create material: %w", err)
	}

	return material, nil
}

// List returns the non-deleted materials in a space, optionally filtered by a
// case-insensitive name search.
func (s *MaterialService) List(spaceID uuid.UUID, search string) ([]models.Material, error) {
	return s.materialRepo.List(spaceID, strings.TrimSpace(search))
}

// GetByID returns a single material, scoped to the given space.
func (s *MaterialService) GetByID(spaceID, id uuid.UUID) (*models.Material, error) {
	material, err := s.materialRepo.GetByID(id)
	if err != nil {
		return nil, ErrMaterialNotFound
	}

	if material.SpaceID != spaceID {
		return nil, ErrMaterialNotFound
	}

	return material, nil
}

// Update applies the provided fields to a material, scoped to the given space.
func (s *MaterialService) Update(spaceID, id uuid.UUID, req *dtos.UpdateMaterialRequest) (*models.Material, error) {
	material, err := s.GetByID(spaceID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, newMaterialValidationError("name is required")
		}
		material.Name = name
	}

	if req.Unit != nil {
		unit := strings.TrimSpace(*req.Unit)
		if unit == "" {
			return nil, newMaterialValidationError("unit is required")
		}
		material.Unit = unit
	}

	if req.Category != nil {
		category := models.MaterialCategory(strings.TrimSpace(*req.Category))
		if !validMaterialCategories[category] {
			return nil, newMaterialValidationError("invalid category %q", *req.Category)
		}
		material.Category = category
	}

	if req.UnitCost != nil {
		if err := validateUnitCost(req.UnitCost); err != nil {
			return nil, err
		}
		material.UnitCost = req.UnitCost
	}

	if req.Supplier != nil {
		material.Supplier = req.Supplier
	}
	if req.SKU != nil {
		material.SKU = req.SKU
	}
	if req.Location != nil {
		material.Location = req.Location
	}
	if req.IsActive != nil {
		material.IsActive = *req.IsActive
	}

	if err := s.materialRepo.Update(material); err != nil {
		return nil, fmt.Errorf("failed to update material: %w", err)
	}

	return material, nil
}

// Delete soft-deletes a material, scoped to the given space.
func (s *MaterialService) Delete(spaceID, id uuid.UUID) error {
	if _, err := s.GetByID(spaceID, id); err != nil {
		return err
	}

	if err := s.materialRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete material: %w", err)
	}

	return nil
}

// validateUnitCost ensures a provided unit cost is non-negative.
func validateUnitCost(cost *decimal.Decimal) error {
	if cost != nil && cost.IsNegative() {
		return newMaterialValidationError("unitCost must be >= 0")
	}
	return nil
}
