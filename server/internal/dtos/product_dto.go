package dtos

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/tylerjvollick/nori/internal/models"
)

// CreateProductRequest is the request body for creating a product.
type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// UpdateProductRequest is the request body for updating a product.
// All fields are optional; only provided fields are updated.
type UpdateProductRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsActive    *bool   `json:"isActive,omitempty"`
}

// CreateProductVariantRequest is the request body for creating a product variant.
type CreateProductVariantRequest struct {
	Name            string                 `json:"name"`
	RecipeID        *uuid.UUID             `json:"recipeId,omitempty"`
	RecipeVariables map[string]interface{} `json:"recipeVariables,omitempty"`
	Price           *decimal.Decimal       `json:"price,omitempty"`
}

// UpdateProductVariantRequest is the request body for updating a product
// variant. All fields are optional; only provided fields are updated.
type UpdateProductVariantRequest struct {
	Name            *string                `json:"name,omitempty"`
	RecipeID        *uuid.UUID             `json:"recipeId,omitempty"`
	RecipeVariables map[string]interface{} `json:"recipeVariables,omitempty"`
	Price           *decimal.Decimal       `json:"price,omitempty"`
	IsActive        *bool                  `json:"isActive,omitempty"`
}

// ProductVariantResponse is the API response for a product variant.
type ProductVariantResponse struct {
	ID              string                 `json:"id"`
	ProductID       string                 `json:"productId"`
	Name            string                 `json:"name"`
	RecipeID        *string                `json:"recipeId,omitempty"`
	RecipeVariables map[string]interface{} `json:"recipeVariables,omitempty"`
	Price           *decimal.Decimal       `json:"price,omitempty"`
	IsActive        bool                   `json:"isActive"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// ProductResponse is the API response for a product.
type ProductResponse struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description *string                  `json:"description,omitempty"`
	IsActive    bool                     `json:"isActive"`
	CreatedAt   time.Time                `json:"createdAt"`
	UpdatedAt   time.Time                `json:"updatedAt"`
	Variants    []ProductVariantResponse `json:"variants"`
}

// ProductVariantResponseFromModel converts a ProductVariant model to a response DTO.
func ProductVariantResponseFromModel(v *models.ProductVariant) ProductVariantResponse {
	resp := ProductVariantResponse{
		ID:              v.ID.String(),
		ProductID:       v.ProductID.String(),
		Name:            v.Name,
		RecipeVariables: v.RecipeVariables,
		Price:           v.Price,
		IsActive:        v.IsActive,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
	if v.RecipeID != nil {
		id := v.RecipeID.String()
		resp.RecipeID = &id
	}
	return resp
}

// ProductResponseFromModel converts a Product model to a response DTO,
// including any preloaded variants.
func ProductResponseFromModel(p *models.Product) ProductResponse {
	variants := make([]ProductVariantResponse, len(p.Variants))
	for i := range p.Variants {
		variants[i] = ProductVariantResponseFromModel(&p.Variants[i])
	}
	return ProductResponse{
		ID:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Variants:    variants,
	}
}
