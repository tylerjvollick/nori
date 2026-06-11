package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TaskMaterial links a recipe/job task to a catalog material with the
// quantity consumed per unit produced. Scaled by batch quantity during
// cost computation.
type TaskMaterial struct {
	ID              uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TaskID          string          `gorm:"type:varchar(255);not null;uniqueIndex:uq_task_material_task_id_material_id" json:"taskId"`
	MaterialID      uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:uq_task_material_task_id_material_id" json:"materialId"`
	QuantityPerUnit decimal.Decimal `gorm:"type:numeric(12,4);not null" json:"quantityPerUnit"`
	Notes           *string         `json:"notes,omitempty"`
	CreatedAt       time.Time       `gorm:"default:now()" json:"createdAt"`
	UpdatedAt       time.Time       `gorm:"default:now()" json:"updatedAt"`

	// Relations
	Task     *Task     `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Material *Material `gorm:"foreignKey:MaterialID" json:"material,omitempty"`
}

func (TaskMaterial) TableName() string {
	return "task_material"
}
