package models

import (
	"time"

	"github.com/google/uuid"
)

// Tag is a reusable label within a space. Tags can be applied to Tasks
// and Recipes via join tables.
type Tag struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	SpaceID   uuid.UUID `gorm:"type:uuid;not null" json:"spaceId"`
	Name      string    `gorm:"not null" json:"name"`
	Color     *string   `json:"color,omitempty"`
	CreatedAt time.Time `gorm:"default:now()" json:"createdAt"`

	// Relations
	Space   *Space   `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	Tasks   []Task   `gorm:"many2many:task_tag;joinForeignKey:TagID;joinReferences:TaskID" json:"tasks,omitempty"`
	Recipes []Recipe `gorm:"many2many:recipe_tag;joinForeignKey:TagID;joinReferences:RecipeID" json:"recipes,omitempty"`
}

func (Tag) TableName() string {
	return "tag"
}

// TaskTag is the join table between Task and Tag.
type TaskTag struct {
	TaskID string    `gorm:"type:varchar(255);primaryKey" json:"taskId"`
	TagID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"tagId"`
}

func (TaskTag) TableName() string {
	return "task_tag"
}

// RecipeTag is the join table between Recipe and Tag.
type RecipeTag struct {
	RecipeID uuid.UUID `gorm:"type:uuid;primaryKey" json:"recipeId"`
	TagID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"tagId"`
}

func (RecipeTag) TableName() string {
	return "recipe_tag"
}
