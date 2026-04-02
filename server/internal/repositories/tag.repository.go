package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(tag *models.Tag) error {
	return r.db.Create(tag).Error
}

func (r *TagRepository) GetByID(id uuid.UUID) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.First(&tag, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

func (r *TagRepository) GetBySpaceID(spaceID uuid.UUID) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Where("space_id = ?", spaceID).
		Order("name ASC").
		Find(&tags).Error
	return tags, err
}

func (r *TagRepository) GetByName(spaceID uuid.UUID, name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Where("space_id = ? AND name = ?", spaceID, name).First(&tag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

func (r *TagRepository) Update(tag *models.Tag) error {
	return r.db.Save(tag).Error
}

func (r *TagRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Tag{}, "id = ?", id).Error
}

// AddTagToSOPTemplate creates an association between an SOPTemplate and a Tag.
func (r *TagRepository) AddTagToSOPTemplate(sopTemplateID int, tagID uuid.UUID) error {
	join := models.SOPTemplateTag{
		SOPTemplateID: sopTemplateID,
		TagID:         tagID,
	}
	return r.db.Create(&join).Error
}

// RemoveTagFromSOPTemplate removes the association between an SOPTemplate and a Tag.
func (r *TagRepository) RemoveTagFromSOPTemplate(sopTemplateID int, tagID uuid.UUID) error {
	return r.db.Where("sop_template_id = ? AND tag_id = ?", sopTemplateID, tagID).
		Delete(&models.SOPTemplateTag{}).Error
}

// GetTagsBySOPTemplateID returns all tags associated with a given SOPTemplate.
func (r *TagRepository) GetTagsBySOPTemplateID(sopTemplateID int) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Joins("JOIN sop_template_tag ON sop_template_tag.tag_id = tag.id").
		Where("sop_template_tag.sop_template_id = ?", sopTemplateID).
		Order("tag.name ASC").
		Find(&tags).Error
	return tags, err
}

// GetSOPTemplateIDsByTagID returns all SOPTemplate IDs associated with a given Tag.
func (r *TagRepository) GetSOPTemplateIDsByTagID(tagID uuid.UUID) ([]int, error) {
	var ids []int
	err := r.db.Model(&models.SOPTemplateTag{}).
		Where("tag_id = ?", tagID).
		Pluck("sop_template_id", &ids).Error
	return ids, err
}

// AddTagToTicket creates an association between a Ticket and a Tag.
func (r *TagRepository) AddTagToTicket(ticketID uuid.UUID, tagID uuid.UUID) error {
	join := models.TicketTag{
		TicketID: ticketID,
		TagID:    tagID,
	}
	return r.db.Create(&join).Error
}

// RemoveTagFromTicket removes the association between a Ticket and a Tag.
func (r *TagRepository) RemoveTagFromTicket(ticketID uuid.UUID, tagID uuid.UUID) error {
	return r.db.Where("ticket_id = ? AND tag_id = ?", ticketID, tagID).
		Delete(&models.TicketTag{}).Error
}

// GetTagsByTicketID returns all tags associated with a given Ticket.
func (r *TagRepository) GetTagsByTicketID(ticketID uuid.UUID) ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Joins("JOIN ticket_tag ON ticket_tag.tag_id = tag.id").
		Where("ticket_tag.ticket_id = ?", ticketID).
		Order("tag.name ASC").
		Find(&tags).Error
	return tags, err
}
