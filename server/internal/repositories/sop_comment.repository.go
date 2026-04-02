package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type SOPCommentRepository struct {
	db *gorm.DB
}

func NewSOPCommentRepository(db *gorm.DB) *SOPCommentRepository {
	return &SOPCommentRepository{db: db}
}

func (r *SOPCommentRepository) Create(comment *models.SOPComment) error {
	return r.db.Create(comment).Error
}

func (r *SOPCommentRepository) GetByID(id uuid.UUID) (*models.SOPComment, error) {
	var comment models.SOPComment
	err := r.db.First(&comment, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("sop comment not found")
		}
		return nil, err
	}
	return &comment, nil
}

func (r *SOPCommentRepository) GetBySOPID(sopTemplateID int) ([]models.SOPComment, error) {
	var comments []models.SOPComment
	err := r.db.Where("sop_template_id = ?", sopTemplateID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *SOPCommentRepository) GetByStepID(stepID int) ([]models.SOPComment, error) {
	var comments []models.SOPComment
	err := r.db.Where("sop_step_id = ?", stepID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *SOPCommentRepository) GetBySubStepID(subStepID int) ([]models.SOPComment, error) {
	var comments []models.SOPComment
	err := r.db.Where("sop_sub_step_id = ?", subStepID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *SOPCommentRepository) GetByUserID(userID uuid.UUID) ([]models.SOPComment, error) {
	var comments []models.SOPComment
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&comments).Error
	return comments, err
}

func (r *SOPCommentRepository) Update(comment *models.SOPComment) error {
	return r.db.Save(comment).Error
}

func (r *SOPCommentRepository) Resolve(id uuid.UUID) error {
	return r.db.Model(&models.SOPComment{}).
		Where("id = ?", id).
		Update("is_resolved", true).Error
}

func (r *SOPCommentRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.SOPComment{}, "id = ?", id).Error
}
