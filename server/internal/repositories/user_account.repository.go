package repositories

import (
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type UserAccountRepository struct {
	db *gorm.DB
}

func NewUserAccountRepository(db *gorm.DB) *UserAccountRepository {
	return &UserAccountRepository{db: db}
}

func (r *UserAccountRepository) Create(userID uuid.UUID, accountID uuid.UUID) (*models.UserAccount, error) {
	record := &models.UserAccount{
		UserID:    userID,
		AccountID: accountID,
	}

	// save record
	if err := r.db.Create(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}
