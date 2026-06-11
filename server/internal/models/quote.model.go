package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// QuoteStatus is the lifecycle state of a quote.
type QuoteStatus string

const (
	QuoteStatusDraft     QuoteStatus = "draft"
	QuoteStatusSent      QuoteStatus = "sent"
	QuoteStatusAccepted  QuoteStatus = "accepted"
	QuoteStatusDeclined  QuoteStatus = "declined"
	QuoteStatusCancelled QuoteStatus = "cancelled"
)

// Quote is a lightweight record of priced work for a customer. No task tree
// is created until the quote is accepted and rolled into a job.
type Quote struct {
	ID            uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID       uuid.UUID        `gorm:"type:uuid;not null" json:"spaceId"`
	CustomerID    *uuid.UUID       `gorm:"type:uuid" json:"customerId,omitempty"`
	Status        QuoteStatus      `gorm:"type:varchar(20);not null;default:'draft'" json:"status"`
	Notes         *string          `json:"notes,omitempty"`
	Markup        *decimal.Decimal `gorm:"type:numeric(12,4)" json:"markup,omitempty"`
	OverrideTotal *decimal.Decimal `gorm:"type:numeric(12,4)" json:"overrideTotal,omitempty"`
	AcceptedAt    *time.Time       `json:"acceptedAt,omitempty"`
	CreatedByID   uuid.UUID        `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt     time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt     time.Time        `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Space     *Space      `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	Customer  *Customer   `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	CreatedBy *User       `gorm:"foreignKey:CreatedByID" json:"createdBy,omitempty"`
	Lines     []QuoteLine `gorm:"foreignKey:QuoteID" json:"lines,omitempty"`
}

func (Quote) TableName() string {
	return "quote"
}

// QuoteLine is one priced item on a quote: a product variant (or a bare
// recipe for non-product quotes) and quantity, with the dry-run cost
// snapshot frozen at quote time.
type QuoteLine struct {
	ID               uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	QuoteID          uuid.UUID        `gorm:"type:uuid;not null" json:"quoteId"`
	ProductVariantID *uuid.UUID       `gorm:"type:uuid" json:"productVariantId,omitempty"`
	RecipeID         *uuid.UUID       `gorm:"type:uuid" json:"recipeId,omitempty"`
	Quantity         int              `gorm:"not null" json:"quantity"`
	UnitPrice        *decimal.Decimal `gorm:"type:numeric(12,4)" json:"unitPrice,omitempty"`
	CostSnapshot     JSONB            `gorm:"type:jsonb" json:"costSnapshot,omitempty"`
	RecipeVariables  JSONB            `gorm:"type:jsonb" json:"recipeVariables,omitempty"`
	Notes            *string          `json:"notes,omitempty"`
	CreatedAt        time.Time        `gorm:"default:now()" json:"createdAt"`
	UpdatedAt        time.Time        `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Quote          *Quote          `gorm:"foreignKey:QuoteID" json:"quote,omitempty"`
	ProductVariant *ProductVariant `gorm:"foreignKey:ProductVariantID" json:"productVariant,omitempty"`
	Recipe         *Recipe         `gorm:"foreignKey:RecipeID" json:"recipe,omitempty"`
}

func (QuoteLine) TableName() string {
	return "quote_line"
}
