package repositories

import (
	"fmt"

	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPVersionRepository struct {
	db *gorm.DB
}

func NewSOPVersionRepository(db *gorm.DB) *SOPVersionRepository {
	return &SOPVersionRepository{db: db}
}

func (r *SOPVersionRepository) Create(version *models.SOPVersion) error {
	return r.db.Create(version).Error
}

func (r *SOPVersionRepository) GetByID(id int) (*models.SOPVersion, error) {
	var version models.SOPVersion
	err := r.db.Preload("Steps").First(&version, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("SOP version not found")
		}
		return nil, err
	}
	return &version, nil
}

func (r *SOPVersionRepository) GetByTemplateID(templateID int) ([]models.SOPVersion, error) {
	var versions []models.SOPVersion
	err := r.db.Where("sop_template_id = ?", templateID).
		Order("version_number DESC").
		Preload("Steps").
		Find(&versions).Error
	return versions, err
}

func (r *SOPVersionRepository) GetLatestVersionNumber(templateID int) (int, error) {
	var version models.SOPVersion
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

func (r *SOPVersionRepository) Update(version *models.SOPVersion) error {
	return r.db.Save(version).Error
}

func (r *SOPVersionRepository) UpdateWithTx(tx *gorm.DB, version *models.SOPVersion) error {
	return tx.Save(version).Error
}

func (r *SOPVersionRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPVersion{}, id).Error
}
