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

func (r *SOPTemplateVersionRepository) UpdateWithTx(tx *gorm.DB, version *models.SOPTemplateVersion) error {
	return tx.Save(version).Error
}

func (r *SOPTemplateVersionRepository) Delete(id int) error {
	return r.db.Delete(&models.SOPTemplateVersion{}, id).Error
}

// GetDraftsByUserID gets all draft versions created by a specific user
func (r *SOPTemplateVersionRepository) GetDraftsByUserID(userID string) ([]models.SOPTemplateVersion, error) {
	var versions []models.SOPTemplateVersion
	err := r.db.Where("created_by = ? AND status = ?", userID, models.VersionStatusDraft).
		Preload("SOPTemplate").
		Preload("Steps").
		Order("updated_at DESC").
		Find(&versions).Error
	return versions, err
}

// GetDraftsByTemplateID gets all draft versions for a specific SOP template
func (r *SOPTemplateVersionRepository) GetDraftsByTemplateID(templateID int) ([]models.SOPTemplateVersion, error) {
	var versions []models.SOPTemplateVersion
	err := r.db.Where("sop_template_id = ? AND status = ?", templateID, models.VersionStatusDraft).
		Preload("Steps").
		Order("updated_at DESC").
		Find(&versions).Error
	return versions, err
}

// GetDraftByTemplateID returns the active draft for a template, if one exists
func (r *SOPTemplateVersionRepository) GetDraftByTemplateID(templateID int) (*models.SOPTemplateVersion, error) {
	var draft models.SOPTemplateVersion
	err := r.db.Where("sop_template_id = ? AND status = ?", templateID, models.VersionStatusDraft).
		Preload("Steps").
		First(&draft).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil // No draft exists - this is OK
	}
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

// GetPublishedVersions gets all published versions for a specific SOP template
func (r *SOPTemplateVersionRepository) GetPublishedVersions(templateID int) ([]models.SOPTemplateVersion, error) {
	var versions []models.SOPTemplateVersion
	err := r.db.Where("sop_template_id = ? AND status = ?", templateID, models.VersionStatusPublished).
		Order("version_number DESC").
		Preload("Steps").
		Find(&versions).Error
	return versions, err
}

// GetLatestPublishedVersionNumber gets the latest published version number for a template
func (r *SOPTemplateVersionRepository) GetLatestPublishedVersionNumber(templateID int) (int, error) {
	var version models.SOPTemplateVersion
	err := r.db.Where("sop_template_id = ? AND status = ?", templateID, models.VersionStatusPublished).
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
