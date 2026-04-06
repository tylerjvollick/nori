package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/config"
	"github.com/tylerjvollick/nori/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// Mock implementations for seed service interfaces

type mockAccountCounter struct {
	countFunc func() (int64, error)
}

func (m *mockAccountCounter) Count() (int64, error) {
	return m.countFunc()
}

type mockUserCreator struct {
	createUserFunc func(*models.User) error
}

func (m *mockUserCreator) CreateUser(user *models.User) error {
	return m.createUserFunc(user)
}

type mockAccountCreator struct {
	createAccountFunc func(*models.Account) error
}

func (m *mockAccountCreator) CreateAccount(account *models.Account) error {
	return m.createAccountFunc(account)
}

type mockUserAccountCreator struct {
	createWithRoleFunc func(uuid.UUID, uuid.UUID, models.Role) (*models.UserAccount, error)
}

func (m *mockUserAccountCreator) CreateWithRole(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
	return m.createWithRoleFunc(userID, accountID, role)
}

type mockUserUpdater struct {
	updateDefaultAccountIDFunc  func(uuid.UUID, uuid.UUID) error
	clearMustChangePasswordFunc func(uuid.UUID) error
}

func (m *mockUserUpdater) UpdateDefaultAccountID(userID uuid.UUID, accountID uuid.UUID) error {
	if m.updateDefaultAccountIDFunc != nil {
		return m.updateDefaultAccountIDFunc(userID, accountID)
	}
	return nil
}

func (m *mockUserUpdater) ClearMustChangePassword(userID uuid.UUID) error {
	if m.clearMustChangePasswordFunc != nil {
		return m.clearMustChangePasswordFunc(userID)
	}
	return nil
}

// noopUserUpdater is a convenience mock that does nothing.
func noopUserUpdater() *mockUserUpdater {
	return &mockUserUpdater{}
}

func newTestSeedConfig() *config.Config {
	return &config.Config{
		JWTSecret:     "test-secret",
		AdminEmail:    "admin@example.com",
		AdminPassword: "temppassword123",
		AccountName:   "Test Workshop",
	}
}

func TestSeedIfNeeded_SkipsWhenAccountExists(t *testing.T) {
	// Arrange
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 1, nil },
	}

	svc := NewSeedService(counter, nil, nil, nil, nil, newTestSeedConfig())

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
}

func TestSeedIfNeeded_SkipsWhenMultipleAccountsExist(t *testing.T) {
	// Arrange
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 5, nil },
	}

	svc := NewSeedService(counter, nil, nil, nil, nil, newTestSeedConfig())

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
}

func TestSeedIfNeeded_ErrorCheckingAccountCount(t *testing.T) {
	// Arrange
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, errors.New("database connection failed") },
	}

	svc := NewSeedService(counter, nil, nil, nil, nil, newTestSeedConfig())

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check account count")
}

func TestSeedIfNeeded_CreatesAdminUserAccountAndRelationship(t *testing.T) {
	// Arrange
	cfg := newTestSeedConfig()

	var createdUser *models.User
	var createdAccount *models.Account
	var createdUserAccountRole models.Role
	var createdUserAccountUserID uuid.UUID
	var createdUserAccountAccountID uuid.UUID

	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			createdUser = user
			return nil
		},
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error {
			createdAccount = account
			return nil
		},
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			createdUserAccountUserID = userID
			createdUserAccountAccountID = accountID
			createdUserAccountRole = role
			return &models.UserAccount{
				ID:        uuid.New(),
				UserID:    userID,
				AccountID: accountID,
				Role:      role,
			}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)

	// Verify admin user was created correctly
	assert.NotNil(t, createdUser)
	assert.Equal(t, cfg.AdminEmail, createdUser.Email)
	assert.NotNil(t, createdUser.Password)
	assert.NotNil(t, createdUser.Role)
	assert.Equal(t, models.RoleAdmin, *createdUser.Role)
	assert.True(t, createdUser.MustChangePassword)
	assert.NotEqual(t, uuid.Nil, createdUser.ID)

	// Verify password was hashed (not stored in plain text)
	assert.NotEqual(t, cfg.AdminPassword, *createdUser.Password)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(*createdUser.Password), []byte(cfg.AdminPassword)))

	// Verify account was created correctly
	assert.NotNil(t, createdAccount)
	assert.NotNil(t, createdAccount.Name)
	assert.Equal(t, cfg.AccountName, *createdAccount.Name)
	assert.Equal(t, models.Trial, createdAccount.Plan)
	assert.Equal(t, createdUser.ID, createdAccount.CreatedByUserID)
	assert.NotEqual(t, uuid.Nil, createdAccount.ID)

	// Verify UserAccount was created with admin role
	assert.Equal(t, createdUser.ID, createdUserAccountUserID)
	assert.Equal(t, createdAccount.ID, createdUserAccountAccountID)
	assert.Equal(t, models.RoleAdmin, createdUserAccountRole)
}

func TestSeedIfNeeded_DoesNotCreateSpaces(t *testing.T) {
	// This test verifies the spec requirement:
	// "Do not create any Spaces" during first boot.
	// The seed service has no space dependency, so if it completes
	// without error, no spaces were created.

	cfg := newTestSeedConfig()

	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error { return nil },
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error { return nil },
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert: no error, and the service has no SpaceService dependency at all
	assert.NoError(t, err)
}

func TestSeedIfNeeded_ErrorCreatingUser(t *testing.T) {
	// Arrange
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			return errors.New("duplicate email")
		},
	}

	svc := NewSeedService(counter, userCreator, nil, nil, nil, newTestSeedConfig())

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create admin user")
}

func TestSeedIfNeeded_ErrorCreatingAccount(t *testing.T) {
	// Arrange
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error { return nil },
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error {
			return errors.New("database error")
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, nil, newTestSeedConfig())

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create account")
}

func TestSeedIfNeeded_ErrorCreatingUserAccountRelationship(t *testing.T) {
	// Arrange
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error { return nil },
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error { return nil },
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return nil, errors.New("constraint violation")
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, newTestSeedConfig())

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user-account relationship")
}

func TestSeedIfNeeded_IsIdempotent(t *testing.T) {
	// Arrange: simulate first call seeds, second call sees account exists
	callCount := 0
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) {
			callCount++
			if callCount == 1 {
				return 0, nil // first call: no accounts
			}
			return 1, nil // second call: account exists
		},
	}

	userCreateCount := 0
	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			userCreateCount++
			return nil
		},
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error { return nil },
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, newTestSeedConfig())

	// Act: call twice
	err1 := svc.SeedIfNeeded()
	err2 := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 1, userCreateCount, "User should only be created once (on first call)")
}

func TestSeedIfNeeded_UserHasNoDefaultAccountID(t *testing.T) {
	// Verify that the seed user is created without a DefaultAccountID,
	// since the account doesn't exist yet when the user is created.

	cfg := newTestSeedConfig()

	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	var createdUser *models.User
	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			createdUser = user
			return nil
		},
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error { return nil },
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Nil(t, createdUser.DefaultAccountID, "User should be created without DefaultAccountID (chicken-and-egg)")
}

func TestSeedIfNeeded_UserHasNoFirstOrLastName(t *testing.T) {
	// The seed user is created from env vars which only provide email/password.
	// FirstName and LastName should be nil.

	cfg := newTestSeedConfig()

	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	var createdUser *models.User
	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			createdUser = user
			return nil
		},
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error { return nil },
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Nil(t, createdUser.FirstName)
	assert.Nil(t, createdUser.LastName)
}

func TestSeedIfNeeded_AccountReferencesUser(t *testing.T) {
	// Verify the account's CreatedByUserID matches the created user's ID

	cfg := newTestSeedConfig()

	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	var createdUserID uuid.UUID
	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			createdUserID = user.ID
			return nil
		},
	}

	var createdAccount *models.Account
	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error {
			createdAccount = account
			return nil
		},
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, createdUserID)
	assert.NotNil(t, createdAccount)
	assert.Equal(t, createdUserID, createdAccount.CreatedByUserID)
}

func TestSeedIfNeeded_AccountPlanIsTrial(t *testing.T) {
	cfg := newTestSeedConfig()

	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error { return nil },
	}

	var createdAccount *models.Account
	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error {
			createdAccount = account
			return nil
		},
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, createdAccount)
	assert.Equal(t, models.Trial, createdAccount.Plan)
}

func TestSeedIfNeeded_SkipPasswordChange(t *testing.T) {
	// When SkipPasswordChange is true, user should be created with MustChangePassword: false
	cfg := newTestSeedConfig()
	cfg.SkipPasswordChange = true

	var createdUser *models.User
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			createdUser = user
			return nil
		},
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error { return nil },
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.False(t, createdUser.MustChangePassword, "MustChangePassword should be false when SkipPasswordChange is true")
}

func TestSeedIfNeeded_DefaultRequiresPasswordChange(t *testing.T) {
	// Default config (SkipPasswordChange = false) should create user with MustChangePassword: true
	cfg := newTestSeedConfig()
	// SkipPasswordChange defaults to false

	var createdUser *models.User
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			createdUser = user
			return nil
		},
	}

	accountCreator := &mockAccountCreator{
		createAccountFunc: func(account *models.Account) error { return nil },
	}

	userAccountCreator := &mockUserAccountCreator{
		createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
			return &models.UserAccount{}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	// Act
	err := svc.SeedIfNeeded()

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.True(t, createdUser.MustChangePassword, "MustChangePassword should be true by default")
}
