package services

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/config"
	"github.com/tylerjvollick/nori/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
	ClearMustChangePassword(userID uuid.UUID) error
}

// UserAccountCreator creates user-account relationships
type UserAccountCreator interface {
	CreateWithRole(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error)
}

// UserFinder looks up users by email (used for E2E idempotency check)
type UserFinder interface {
	GetUserByEmail(email string) (*models.User, error)
}

// DefaultSpaceCreator creates a default space with stations for an account
type DefaultSpaceCreator interface {
	CreateDefaultSpace(accountID uuid.UUID, creatorUserID uuid.UUID) (*models.Space, error)
}

// SeedService handles first-boot database seeding
type SeedService struct {
	accountCounter      AccountCounter
	userCreator         UserCreator
	userUpdater         UserUpdater
	accountCreator      AccountCreator
	userAccountCreator  UserAccountCreator
	userFinder          UserFinder          // optional, for E2E
	defaultSpaceCreator DefaultSpaceCreator // optional, for E2E
	cfg                 *config.Config
}

// NewSeedService creates a new SeedService
func NewSeedService(
	accountCounter AccountCounter,
	userCreator UserCreator,
	userUpdater UserUpdater,
	accountCreator AccountCreator,
	userAccountCreator UserAccountCreator,
	userFinder UserFinder,
	defaultSpaceCreator DefaultSpaceCreator,
	cfg *config.Config,
) *SeedService {
	return &SeedService{
		accountCounter:      accountCounter,
		userCreator:         userCreator,
		userUpdater:         userUpdater,
		accountCreator:      accountCreator,
		userAccountCreator:  userAccountCreator,
		userFinder:          userFinder,
		defaultSpaceCreator: defaultSpaceCreator,
		cfg:                 cfg,
	}
}

// SeedIfNeeded checks if the database needs seeding and seeds it if so.
// On first boot (no Account exists):
//  1. Create the admin User with env credentials
//  2. Create the Account with the configured name
//  3. Create the UserAccount join record with admin role
//  4. Do NOT create any Spaces (user creates via onboarding UI)
//
// When E2E_ACCOUNT_ENABLED is true, also seeds a completely isolated
// test account + user + space + stations (idempotent by email lookup).
func (s *SeedService) SeedIfNeeded() error {
	count, err := s.accountCounter.Count()
	if err != nil {
		return fmt.Errorf("failed to check account count: %w", err)
	}

	if count == 0 {
		if err := s.seedAdmin(); err != nil {
			return err
		}
	} else {
		log.Println("Seed: account already exists, skipping admin seed")
	}

	// E2E account (independent of admin seed, has own idempotency)
	if s.cfg.E2EAccountEnabled {
		if err := s.seedE2EAccount(); err != nil {
			return fmt.Errorf("failed to seed E2E account: %w", err)
		}
	}

	return nil
}

// seedAdmin creates the admin user and account on first boot.
func (s *SeedService) seedAdmin() error {
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

	// GORM skips bool zero-values during Create, so the DB default (true)
	// wins even when we set MustChangePassword=false. Explicitly clear it
	// when SkipPasswordChange is enabled.
	if s.cfg.SkipPasswordChange {
		if err := s.userUpdater.ClearMustChangePassword(adminUser.ID); err != nil {
			return fmt.Errorf("failed to clear must-change-password flag: %w", err)
		}
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

	// 3. Point the admin user's DefaultAccountID to the new account
	if err := s.userUpdater.UpdateDefaultAccountID(adminUser.ID, account.ID); err != nil {
		return fmt.Errorf("failed to set admin default account: %w", err)
	}

	// 4. Create UserAccount join record with admin role
	_, err = s.userAccountCreator.CreateWithRole(adminUser.ID, account.ID, models.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create user-account relationship: %w", err)
	}

	log.Println("Seed: first-boot seeding complete")

	return nil
}

// seedE2EAccount creates a completely isolated test account, user, space,
// and stations. Idempotent: skips if the E2E user already exists.
func (s *SeedService) seedE2EAccount() error {
	if s.userFinder == nil || s.defaultSpaceCreator == nil {
		return fmt.Errorf("E2E seeding requires userFinder and defaultSpaceCreator")
	}

	// Check if E2E user already exists
	_, err := s.userFinder.GetUserByEmail(s.cfg.E2EAccountEmail)
	if err == nil {
		log.Printf("Seed: E2E user %s already exists, skipping", s.cfg.E2EAccountEmail)
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check E2E user: %w", err)
	}

	log.Printf("Seed: creating E2E test account for %s", s.cfg.E2EAccountEmail)

	// 1. Create E2E user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(s.cfg.E2EAccountPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash E2E password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)
	adminRole := models.RoleAdmin
	firstName := "E2E"
	lastName := "Test"
	e2eUser := &models.User{
		ID:                 uuid.New(),
		Email:              s.cfg.E2EAccountEmail,
		Password:           &hashedPasswordStr,
		Role:               &adminRole,
		MustChangePassword: false,
		FirstName:          &firstName,
		LastName:           &lastName,
		RecentSpaces:       models.RecentSpaces{},
	}

	if err := s.userCreator.CreateUser(e2eUser); err != nil {
		return fmt.Errorf("failed to create E2E user: %w", err)
	}

	// Explicitly clear MustChangePassword (GORM zero-value issue)
	if err := s.userUpdater.ClearMustChangePassword(e2eUser.ID); err != nil {
		return fmt.Errorf("failed to clear E2E must-change-password flag: %w", err)
	}

	// 2. Create isolated account
	accountName := "E2E Test Account"
	account := &models.Account{
		ID:              uuid.New(),
		Name:            &accountName,
		Plan:            models.Trial,
		CreatedByUserID: e2eUser.ID,
	}

	if err := s.accountCreator.CreateAccount(account); err != nil {
		return fmt.Errorf("failed to create E2E account: %w", err)
	}

	// 3. Set DefaultAccountID
	if err := s.userUpdater.UpdateDefaultAccountID(e2eUser.ID, account.ID); err != nil {
		return fmt.Errorf("failed to set E2E default account: %w", err)
	}

	// 4. Create UserAccount join
	_, err = s.userAccountCreator.CreateWithRole(e2eUser.ID, account.ID, models.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create E2E user-account relationship: %w", err)
	}

	// 5. Create default space with stations
	_, err = s.defaultSpaceCreator.CreateDefaultSpace(account.ID, e2eUser.ID)
	if err != nil {
		return fmt.Errorf("failed to create E2E default space: %w", err)
	}

	log.Printf("Seed: E2E test account created (%s)", s.cfg.E2EAccountEmail)
	return nil
}
