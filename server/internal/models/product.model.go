package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Product is what the shop sells. Variants hold the sellable configurations.
type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID     uuid.UUID      `gorm:"type:uuid;not null" json:"spaceId"`
	Name        string         `gorm:"not null" json:"name"`
	Description *string        `json:"description,omitempty"`
	IsActive    bool           `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time      `gorm:"default:now()" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"default:now()" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Space    *Space           `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
}

func (Product) TableName() string {
	return "product"
}

// ProductVariant is a sellable configuration of a product (e.g. wood +
// finish), optionally bound to a recipe with variable bindings and a
// customer-facing price.
type ProductVariant struct {
	ID              uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ProductID       uuid.UUID        `gorm:"type:uuid;not null" json:"productId"`
	Name            string           `gorm:"not null" json:"name"`
	RecipeID        *uuid.UUID       `gorm:"type:uuid" json:"recipeId,omitempty"`
	RecipeVariables JSONB            `gorm:"type:jsonb" json:"recipeVariables,omitempty"`
	Price           *decimal.Decimal `gorm:"type:numeric(12,4)" json:"price,omitempty"`
	IsActive        bool             `gorm:"not null;default:true" json:"isActive"`
	CreatedAt       time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt       time.Time        `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Recipe  *Recipe  `gorm:"foreignKey:RecipeID" json:"recipe,omitempty"`
}

func (ProductVariant) TableName() string {
	return "product_variant"
}
