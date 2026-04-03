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

// CreateWithRole creates a UserAccount record with a specific role
func (r *UserAccountRepository) CreateWithRole(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
	record := &models.UserAccount{
		UserID:    userID,
		AccountID: accountID,
		Role:      role,
	}

	if err := r.db.Create(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}

// GetByAccountID returns all UserAccount records for the specified account
func (r *UserAccountRepository) GetByAccountID(accountID uuid.UUID) ([]models.UserAccount, error) {
	var userAccounts []models.UserAccount
	err := r.db.Where("account_id = ?", accountID).Find(&userAccounts).Error
	if err != nil {
		return nil, err
	}
	return userAccounts, nil
}
