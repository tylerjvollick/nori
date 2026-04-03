package repositories

import (
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/utils"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) CreateAccount(account *models.Account) error {
	return r.db.Create(account).Error
}

// Count returns the number of accounts in the database
func (r *AccountRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Account{}).Count(&count).Error
	return count, err
}

func (r *AccountRepository) Create(name string, createdByUserId uuid.UUID, plan models.Plan) (*models.Account, error) {
	// create account object
	account := &models.Account{
		ID:              uuid.New(),
		CreatedByUserID: createdByUserId,
		Name:            utils.PtrString(name),
		Plan:            plan,
	}

	// save record
	if err := r.db.Create(account).Error; err != nil {
		return nil, err
	}

	// everything worked?
	return account, nil
}
