package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPTemplateRepository struct {
	db *gorm.DB
}

func NewSOPTemplateRepository(db *gorm.DB) *SOPTemplateRepository {
	return &SOPTemplateRepository{db: db}
}

func (r *SOPTemplateRepository) Create(template *models.SOPTemplate) error {
	return r.db.Create(template).Error
}

func (r *SOPTemplateRepository) GetByID(id int) (*models.SOPTemplate, error) {
	var template models.SOPTemplate
	err := r.db.Preload("CurrentVersion").Preload("CurrentVersion.Steps").First(&template, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SOP template not found")
		}
		return nil, err
	}
	return &template, nil
}

func (r *SOPTemplateRepository) GetAll() ([]models.SOPTemplate, error) {
	var templates []models.SOPTemplate
	err := r.db.Preload("CurrentVersion").Find(&templates).Error
	return templates, err
}

func (r *SOPTemplateRepository) GetByCreatedBy(userID uuid.UUID) ([]models.SOPTemplate, error) {
	var templates []models.SOPTemplate
	err := r.db.Where("created_by = ?", userID).Preload("CurrentVersion").Find(&templates).Error
	return templates, err
}

func (r *SOPTemplateRepository) Update(template *models.SOPTemplate) error {
	return r.db.Save(template).Error
}

func (r *SOPTemplateRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPTemplate{}, id).Error
}

func (r *SOPTemplateRepository) UpdateCurrentVersion(templateID int, versionID int) error {
	return r.db.Model(&models.SOPTemplate{}).
		Where("id = ?", templateID).
		Update("current_version_id", versionID).Error
}
