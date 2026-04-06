package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/config"
	"github.com/tylerjvollick/nori/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// AccountCounter checks if any accounts exist in the database
type AccountCounter interface {
	Count() (int64, error)
}

// UserCreator creates users in the database
type UserCreator interface {
	CreateUser(user *models.User) error
}

// AccountCreator creates accounts in the database
type AccountCreator interface {
	CreateAccount(account *models.Account) error
}

// UserUpdater updates user fields
type UserUpdater interface {
	UpdateDefaultAccountID(userID uuid.UUID, accountID uuid.UUID) error
}

// UserAccountCreator creates user-account relationships
type UserAccountCreator interface {
	CreateWithRole(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error)
}

// SeedService handles first-boot database seeding
type SeedService struct {
	accountCounter     AccountCounter
	userCreator        UserCreator
	accountCreator     AccountCreator
	userAccountCreator UserAccountCreator
	cfg                *config.Config
}

// NewSeedService creates a new SeedService
func NewSeedService(
	accountCounter AccountCounter,
	userCreator UserCreator,
	accountCreator AccountCreator,
	userAccountCreator UserAccountCreator,
	cfg *config.Config,
) *SeedService {
	return &SeedService{
		accountCounter:     accountCounter,
		userCreator:        userCreator,
		accountCreator:     accountCreator,
		userAccountCreator: userAccountCreator,
		cfg:                cfg,
	}
}

// SeedIfNeeded checks if the database needs seeding and seeds it if so.
// On first boot (no Account exists):
//  1. Create the admin User with env credentials
//  2. Create the Account with the configured name
//  3. Create the UserAccount join record with admin role
//  4. Do NOT create any Spaces
//
// If an Account already exists, seeding is skipped (idempotent).
func (s *SeedService) SeedIfNeeded() error {
	count, err := s.accountCounter.Count()
	if err != nil {
		return fmt.Errorf("failed to check account count: %w", err)
	}

	if count > 0 {
		log.Println("Seed: account already exists, skipping seed")
		return nil
	}

	log.Println("Seed: no accounts found, seeding database")

	// 1. Create admin user (without DefaultAccountID — chicken-and-egg)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(s.cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)
	adminRole := models.RoleAdmin
	adminUser := &models.User{
		ID:                 uuid.New(),
		Email:              s.cfg.AdminEmail,
		Password:           &hashedPasswordStr,
		Role:               &adminRole,
		MustChangePassword: !s.cfg.SkipPasswordChange,
		RecentSpaces:       models.RecentSpaces{},
	}

	if err := s.userCreator.CreateUser(adminUser); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("Seed: created admin user %s", adminUser.Email)

	// 2. Create the account
	accountName := s.cfg.AccountName
	account := &models.Account{
		ID:              uuid.New(),
		Name:            &accountName,
		Plan:            models.Trial,
		CreatedByUserID: adminUser.ID,
	}

	if err := s.accountCreator.CreateAccount(account); err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	log.Printf("Seed: created account %q", s.cfg.AccountName)

	// 3. Create UserAccount join record with admin role
	_, err = s.userAccountCreator.CreateWithRole(adminUser.ID, account.ID, models.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create user-account relationship: %w", err)
	}

	log.Println("Seed: first-boot seeding complete")

	return nil
}
