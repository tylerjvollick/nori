package repositories

import (
	"fmt"

	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPTemplateVersionRepository struct {
	db *gorm.DB
}

func NewSOPTemplateVersionRepository(db *gorm.DB) *SOPTemplateVersionRepository {
	return &SOPTemplateVersionRepository{db: db}
}

func (r *SOPTemplateVersionRepository) Create(version *models.SOPTemplateVersion) error {
	return r.db.Create(version).Error
}

func (r *SOPTemplateVersionRepository) GetByID(id int) (*models.SOPTemplateVersion, error) {
	var version models.SOPTemplateVersion
	err := r.db.Preload("Steps").First(&version, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SOP template version not found")
		}
		return nil, err
	}
	return &version, nil
}

func (r *SOPTemplateVersionRepository) GetByTemplateID(templateID int) ([]models.SOPTemplateVersion, error) {
	var versions []models.SOPTemplateVersion
	err := r.db.Where("sop_template_id = ?", templateID).
		Order("version_number DESC").
		Preload("Steps").
		Find(&versions).Error
	return versions, err
}

func (r *SOPTemplateVersionRepository) GetLatestVersionNumber(templateID int) (int, error) {
	var version models.SOPTemplateVersion
	err := r.db.Where("sop_template_id = ?", templateID).
		Order("version_number DESC").
		First(&version).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return version.VersionNumber, nil
}

func (r *SOPTemplateVersionRepository) Update(version *models.SOPTemplateVersion) error {
	return r.db.Save(version).Error
}

func (r *SOPTemplateVersionRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPTemplateVersion{}, id).Error
}
