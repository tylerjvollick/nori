package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecipeVersionStatus represents the lifecycle state of a recipe version.
type RecipeVersionStatus string

const (
	RecipeVersionStatusDraft     RecipeVersionStatus = "draft"
	RecipeVersionStatusPublished RecipeVersionStatus = "published"
	RecipeVersionStatusArchived  RecipeVersionStatus = "archived"
)

// Recipe is a process template identity. Recipes are versioned TOML documents
// that can be "poured" into task graphs via the formula engine.
type Recipe struct {
	ID               uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID          uuid.UUID      `gorm:"type:uuid;not null" json:"spaceId"`
	Name             string         `gorm:"not null" json:"name"`
	Slug             string         `gorm:"type:varchar(255);not null" json:"slug"`
	Description      *string        `json:"description,omitempty"`
	CategoryID       *uuid.UUID     `gorm:"type:uuid" json:"categoryId,omitempty"`
	CurrentVersionID *int           `json:"currentVersionId,omitempty"`
	ExtendsRecipeID  *uuid.UUID     `gorm:"type:uuid" json:"extendsRecipeId,omitempty"`
	CreatedByID      uuid.UUID      `gorm:"type:uuid;not null" json:"createdById"`
	IsActive         bool           `gorm:"not null;default:true" json:"isActive"`
	CreatedAt        time.Time      `gorm:"default:now()" json:"createdAt"`
	UpdatedAt        time.Time      `gorm:"default:now()" json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	// Relations
	Space          *Space          `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	CurrentVersion *RecipeVersion  `gorm:"foreignKey:CurrentVersionID" json:"currentVersion,omitempty"`
	ExtendsRecipe  *Recipe         `gorm:"foreignKey:ExtendsRecipeID" json:"extendsRecipe,omitempty"`
	CreatedBy      *User           `gorm:"foreignKey:CreatedByID" json:"createdBy,omitempty"`
	Versions       []RecipeVersion `gorm:"foreignKey:RecipeID" json:"versions,omitempty"`
	Tags           []Tag           `gorm:"many2many:recipe_tag;joinForeignKey:RecipeID;joinReferences:TagID" json:"tags,omitempty"`
}

func (Recipe) TableName() string {
	return "recipe"
}

// RecipeVersion is a versioned, immutable snapshot of a recipe's TOML content.
type RecipeVersion struct {
	ID            int                 `gorm:"primaryKey;autoIncrement" json:"id"`
	RecipeID      uuid.UUID           `gorm:"type:uuid;not null" json:"recipeId"`
	VersionNumber int                 `gorm:"not null" json:"versionNumber"`
	Status        RecipeVersionStatus `gorm:"type:varchar(50);not null;default:'draft'" json:"status"`
	Content       *string             `gorm:"type:text" json:"content,omitempty"`
	RootTaskID    *string             `gorm:"type:varchar(255)" json:"rootTaskId,omitempty"`
	ChangeSummary *string             `json:"changeSummary,omitempty"`
	AuthorID      uuid.UUID           `gorm:"type:uuid;not null" json:"authorId"`
	PublishedAt   *time.Time          `json:"publishedAt,omitempty"`
	CreatedAt     time.Time           `gorm:"default:now()" json:"createdAt"`
	UpdatedAt     time.Time           `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Recipe   *Recipe `gorm:"foreignKey:RecipeID" json:"recipe,omitempty"`
	Author   *User   `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	RootTask *Task   `gorm:"foreignKey:RootTaskID" json:"rootTask,omitempty"`
}

func (RecipeVersion) TableName() string {
	return "recipe_version"
}
