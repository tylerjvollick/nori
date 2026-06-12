package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/tylerjvollick/nori/internal/models"
)

// Sentinel errors so handlers can map service failures to HTTP statuses.
var (
	// ErrProductNotFound is returned when a product does not exist in the
	// given space (including cross-space access, to avoid leaking existence).
	ErrProductNotFound = errors.New("product not found")
	// ErrProductVariantNotFound is returned when a variant does not exist
	// under the given product.
	ErrProductVariantNotFound = errors.New("product variant not found")
	// ErrVariantRecipeNotFound is returned when a variant references a
	// recipe that does not exist in the same space.
	ErrVariantRecipeNotFound = errors.New("recipe not found in this space")
	// ErrProductNameRequired is returned when a product or variant is
	// created with an empty name.
	ErrProductNameRequired = errors.New("name is required")
)

// ProductRecipeRepo defines the recipe lookup needed to validate variant
// recipe bindings.
type ProductRecipeRepo interface {
	GetByID(id uuid.UUID) (*models.Recipe, error)
}

// ProductService implements product and variant CRUD, scoped to a space.
type ProductService struct {
	db         *gorm.DB
	recipeRepo ProductRecipeRepo
}

// NewProductService creates a new ProductService.
func NewProductService(db *gorm.DB, recipeRepo ProductRecipeRepo) *ProductService {
	return &ProductService{db: db, recipeRepo: recipeRepo}
}

// CreateProductInput holds the fields for creating a product.
type CreateProductInput struct {
	Name        string
	Description *string
}

// UpdateProductInput holds the optional fields for updating a product.
type UpdateProductInput struct {
	Name        *string
	Description *string
	IsActive    *bool
}

// CreateVariantInput holds the fields for creating a product variant.
type CreateVariantInput struct {
	Name            string
	RecipeID        *uuid.UUID
	RecipeVariables models.JSONB
	Price           *decimal.Decimal
}

// UpdateVariantInput holds the optional fields for updating a product variant.
type UpdateVariantInput struct {
	Name            *string
	RecipeID        *uuid.UUID
	RecipeVariables models.JSONB
	Price           *decimal.Decimal
	IsActive        *bool
}

// Create creates a new product in the space.
func (s *ProductService) Create(spaceID uuid.UUID, in CreateProductInput) (*models.Product, error) {
	if in.Name == "" {
		return nil, ErrProductNameRequired
	}

	product := &models.Product{
		ID:          uuid.New(),
		SpaceID:     spaceID,
		Name:        in.Name,
		Description: in.Description,
		IsActive:    true,
	}
	if err := s.db.Create(product).Error; err != nil {
		return nil, fmt.Errorf("creating product: %w", err)
	}
	product.Variants = []models.ProductVariant{}
	return product, nil
}

// List returns all (non-deleted) products in the space with their variants,
// ordered by name.
func (s *ProductService) List(spaceID uuid.UUID) ([]models.Product, error) {
	var products []models.Product
	err := s.db.
		Preload("Variants", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Where("space_id = ?", spaceID).
		Order("name ASC").
		Find(&products).Error
	if err != nil {
		return nil, fmt.Errorf("listing products: %w", err)
	}
	return products, nil
}

// GetByID returns a product in the space by ID, including all its variants.
// Returns ErrProductNotFound when the product does not exist in the space
// (including products that belong to another space).
func (s *ProductService) GetByID(spaceID, id uuid.UUID) (*models.Product, error) {
	var product models.Product
	err := s.db.
		Preload("Variants", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Where("id = ? AND space_id = ?", id, spaceID).
		First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("getting product: %w", err)
	}
	return &product, nil
}

// Update applies the provided fields to a product in the space.
func (s *ProductService) Update(spaceID, id uuid.UUID, in UpdateProductInput) (*models.Product, error) {
	product, err := s.GetByID(spaceID, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		if *in.Name == "" {
			return nil, ErrProductNameRequired
		}
		product.Name = *in.Name
	}
	if in.Description != nil {
		product.Description = in.Description
	}
	if in.IsActive != nil {
		product.IsActive = *in.IsActive
	}

	if err := s.db.Save(product).Error; err != nil {
		return nil, fmt.Errorf("updating product: %w", err)
	}
	return product, nil
}

// Delete soft-deletes a product in the space and deactivates all its
// variants (is_active=false) in a single transaction.
func (s *ProductService) Delete(spaceID, id uuid.UUID) error {
	if _, err := s.GetByID(spaceID, id); err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ProductVariant{}).
			Where("product_id = ?", id).
			Update("is_active", false).Error; err != nil {
			return fmt.Errorf("deactivating variants: %w", err)
		}
		if err := tx.Delete(&models.Product{}, "id = ?", id).Error; err != nil {
			return fmt.Errorf("deleting product: %w", err)
		}
		return nil
	})
}

// CreateVariant creates a variant under a product in the space. If a recipe
// is referenced, it must exist in the same space.
func (s *ProductService) CreateVariant(spaceID, productID uuid.UUID, in CreateVariantInput) (*models.ProductVariant, error) {
	if in.Name == "" {
		return nil, ErrProductNameRequired
	}
	if _, err := s.GetByID(spaceID, productID); err != nil {
		return nil, err
	}
	if err := s.validateRecipeInSpace(spaceID, in.RecipeID); err != nil {
		return nil, err
	}

	variant := &models.ProductVariant{
		ID:              uuid.New(),
		ProductID:       productID,
		Name:            in.Name,
		RecipeID:        in.RecipeID,
		RecipeVariables: in.RecipeVariables,
		Price:           in.Price,
		IsActive:        true,
	}
	if err := s.db.Create(variant).Error; err != nil {
		return nil, fmt.Errorf("creating product variant: %w", err)
	}
	return variant, nil
}

// UpdateVariant applies the provided fields to a variant under a product in
// the space. A changed recipe reference must exist in the same space.
func (s *ProductService) UpdateVariant(spaceID, productID, variantID uuid.UUID, in UpdateVariantInput) (*models.ProductVariant, error) {
	variant, err := s.getVariant(spaceID, productID, variantID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		if *in.Name == "" {
			return nil, ErrProductNameRequired
		}
		variant.Name = *in.Name
	}
	if in.RecipeID != nil {
		if err := s.validateRecipeInSpace(spaceID, in.RecipeID); err != nil {
			return nil, err
		}
		variant.RecipeID = in.RecipeID
	}
	if in.RecipeVariables != nil {
		variant.RecipeVariables = in.RecipeVariables
	}
	if in.Price != nil {
		variant.Price = in.Price
	}
	if in.IsActive != nil {
		variant.IsActive = *in.IsActive
	}

	if err := s.db.Save(variant).Error; err != nil {
		return nil, fmt.Errorf("updating product variant: %w", err)
	}
	return variant, nil
}

// DeleteVariant hard-deletes a variant under a product in the space.
func (s *ProductService) DeleteVariant(spaceID, productID, variantID uuid.UUID) error {
	variant, err := s.getVariant(spaceID, productID, variantID)
	if err != nil {
		return err
	}
	if err := s.db.Delete(variant).Error; err != nil {
		return fmt.Errorf("deleting product variant: %w", err)
	}
	return nil
}

// getVariant loads a variant, verifying both that the product belongs to the
// space and that the variant belongs to the product.
func (s *ProductService) getVariant(spaceID, productID, variantID uuid.UUID) (*models.ProductVariant, error) {
	if _, err := s.GetByID(spaceID, productID); err != nil {
		return nil, err
	}

	var variant models.ProductVariant
	err := s.db.
		Where("id = ? AND product_id = ?", variantID, productID).
		First(&variant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductVariantNotFound
		}
		return nil, fmt.Errorf("getting product variant: %w", err)
	}
	return &variant, nil
}

// validateRecipeInSpace verifies that a referenced recipe exists in the same
// space. A nil recipeID is allowed (variants may be recipe-less).
func (s *ProductService) validateRecipeInSpace(spaceID uuid.UUID, recipeID *uuid.UUID) error {
	if recipeID == nil {
		return nil
	}
	recipe, err := s.recipeRepo.GetByID(*recipeID)
	if err != nil || recipe.SpaceID != spaceID {
		return ErrVariantRecipeNotFound
	}
	return nil
}
