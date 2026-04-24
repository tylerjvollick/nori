package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/config"
	"github.com/tylerjvollick/nori/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

type mockUserFinder struct {
	getUserByEmailFunc func(string) (*models.User, error)
}

func (m *mockUserFinder) GetUserByEmail(email string) (*models.User, error) {
	return m.getUserByEmailFunc(email)
}

type mockDefaultSpaceCreator struct {
	createDefaultSpaceFunc func(uuid.UUID, uuid.UUID) (*models.Space, error)
}

func (m *mockDefaultSpaceCreator) CreateDefaultSpace(accountID uuid.UUID, creatorUserID uuid.UUID) (*models.Space, error) {
	return m.createDefaultSpaceFunc(accountID, creatorUserID)
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

func newTestSeedService(counter AccountCounter, userCreator UserCreator, userUpdater UserUpdater, accountCreator AccountCreator, userAccountCreator UserAccountCreator, cfg *config.Config) *SeedService {
	return NewSeedService(counter, userCreator, userUpdater, accountCreator, userAccountCreator, nil, nil, cfg)
}

func TestSeedIfNeeded_SkipsWhenAccountExists(t *testing.T) {
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 1, nil },
	}

	svc := newTestSeedService(counter, nil, nil, nil, nil, newTestSeedConfig())

	err := svc.SeedIfNeeded()
	assert.NoError(t, err)
}

func TestSeedIfNeeded_SkipsWhenMultipleAccountsExist(t *testing.T) {
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 5, nil },
	}

	svc := newTestSeedService(counter, nil, nil, nil, nil, newTestSeedConfig())

	err := svc.SeedIfNeeded()
	assert.NoError(t, err)
}

func TestSeedIfNeeded_ErrorCheckingAccountCount(t *testing.T) {
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, errors.New("database connection failed") },
	}

	svc := newTestSeedService(counter, nil, nil, nil, nil, newTestSeedConfig())

	err := svc.SeedIfNeeded()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check account count")
}

func TestSeedIfNeeded_CreatesAdminUserAccountAndRelationship(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()

	assert.NoError(t, err)

	assert.NotNil(t, createdUser)
	assert.Equal(t, cfg.AdminEmail, createdUser.Email)
	assert.NotNil(t, createdUser.Password)
	assert.NotNil(t, createdUser.Role)
	assert.Equal(t, models.RoleAdmin, *createdUser.Role)
	assert.True(t, createdUser.MustChangePassword)
	assert.NotEqual(t, uuid.Nil, createdUser.ID)

	assert.NotEqual(t, cfg.AdminPassword, *createdUser.Password)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(*createdUser.Password), []byte(cfg.AdminPassword)))

	assert.NotNil(t, createdAccount)
	assert.NotNil(t, createdAccount.Name)
	assert.Equal(t, cfg.AccountName, *createdAccount.Name)
	assert.Equal(t, models.Trial, createdAccount.Plan)
	assert.Equal(t, createdUser.ID, createdAccount.CreatedByUserID)
	assert.NotEqual(t, uuid.Nil, createdAccount.ID)

	assert.Equal(t, createdUser.ID, createdUserAccountUserID)
	assert.Equal(t, createdAccount.ID, createdUserAccountAccountID)
	assert.Equal(t, models.RoleAdmin, createdUserAccountRole)
}

func TestSeedIfNeeded_DoesNotCreateSpaces(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()
	assert.NoError(t, err)
}

func TestSeedIfNeeded_ErrorCreatingUser(t *testing.T) {
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 0, nil },
	}

	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			return errors.New("duplicate email")
		},
	}

	svc := newTestSeedService(counter, userCreator, nil, nil, nil, newTestSeedConfig())

	err := svc.SeedIfNeeded()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create admin user")
}

func TestSeedIfNeeded_ErrorCreatingAccount(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, nil, newTestSeedConfig())

	err := svc.SeedIfNeeded()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create account")
}

func TestSeedIfNeeded_ErrorCreatingUserAccountRelationship(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, newTestSeedConfig())

	err := svc.SeedIfNeeded()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user-account relationship")
}

func TestSeedIfNeeded_IsIdempotent(t *testing.T) {
	callCount := 0
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) {
			callCount++
			if callCount == 1 {
				return 0, nil
			}
			return 1, nil
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, newTestSeedConfig())

	err1 := svc.SeedIfNeeded()
	err2 := svc.SeedIfNeeded()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, 1, userCreateCount, "User should only be created once (on first call)")
}

func TestSeedIfNeeded_UserHasNoDefaultAccountID(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Nil(t, createdUser.DefaultAccountID, "User should be created without DefaultAccountID (chicken-and-egg)")
}

func TestSeedIfNeeded_UserHasNoFirstOrLastName(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Nil(t, createdUser.FirstName)
	assert.Nil(t, createdUser.LastName)
}

func TestSeedIfNeeded_AccountReferencesUser(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()

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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.NotNil(t, createdAccount)
	assert.Equal(t, models.Trial, createdAccount.Plan)
}

func TestSeedIfNeeded_SkipPasswordChange(t *testing.T) {
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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.False(t, createdUser.MustChangePassword, "MustChangePassword should be false when SkipPasswordChange is true")
}

func TestSeedIfNeeded_DefaultRequiresPasswordChange(t *testing.T) {
	cfg := newTestSeedConfig()

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

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)

	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.True(t, createdUser.MustChangePassword, "MustChangePassword should be true by default")
}

// ── E2E Account Seeding Tests ──

func newE2ETestConfig() *config.Config {
	cfg := newTestSeedConfig()
	cfg.E2EAccountEnabled = true
	cfg.E2EAccountEmail = "e2e-test@nori.dev"
	cfg.E2EAccountPassword = "TestPass123!"
	return cfg
}

func fullMocks() (*mockAccountCounter, *mockUserCreator, *mockAccountCreator, *mockUserAccountCreator) {
	return &mockAccountCounter{
			countFunc: func() (int64, error) { return 1, nil }, // admin already exists
		},
		&mockUserCreator{
			createUserFunc: func(user *models.User) error { return nil },
		},
		&mockAccountCreator{
			createAccountFunc: func(account *models.Account) error { return nil },
		},
		&mockUserAccountCreator{
			createWithRoleFunc: func(userID uuid.UUID, accountID uuid.UUID, role models.Role) (*models.UserAccount, error) {
				return &models.UserAccount{}, nil
			},
		}
}

func TestSeedIfNeeded_E2EAccountCreatedWhenEnabled(t *testing.T) {
	cfg := newE2ETestConfig()
	counter, userCreator, accountCreator, userAccountCreator := fullMocks()

	var createdUsers []*models.User
	userCreator.createUserFunc = func(user *models.User) error {
		createdUsers = append(createdUsers, user)
		return nil
	}

	var createdAccounts []*models.Account
	accountCreator.createAccountFunc = func(account *models.Account) error {
		createdAccounts = append(createdAccounts, account)
		return nil
	}

	userFinder := &mockUserFinder{
		getUserByEmailFunc: func(email string) (*models.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	spaceCreated := false
	spaceCreator := &mockDefaultSpaceCreator{
		createDefaultSpaceFunc: func(accountID uuid.UUID, creatorUserID uuid.UUID) (*models.Space, error) {
			spaceCreated = true
			return &models.Space{ID: uuid.New()}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, userFinder, spaceCreator, cfg)
	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.Len(t, createdUsers, 1)
	assert.Equal(t, cfg.E2EAccountEmail, createdUsers[0].Email)
	assert.Equal(t, "E2E", *createdUsers[0].FirstName)
	assert.Equal(t, "Test", *createdUsers[0].LastName)
	assert.False(t, createdUsers[0].MustChangePassword)
	assert.Len(t, createdAccounts, 1)
	assert.Equal(t, "E2E Test Account", *createdAccounts[0].Name)
	assert.True(t, spaceCreated)
}

func TestSeedIfNeeded_E2EAccountSkippedWhenDisabled(t *testing.T) {
	cfg := newTestSeedConfig()
	cfg.E2EAccountEnabled = false
	counter, userCreator, accountCreator, userAccountCreator := fullMocks()

	userCreateCount := 0
	userCreator.createUserFunc = func(user *models.User) error {
		userCreateCount++
		return nil
	}

	svc := newTestSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, cfg)
	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.Equal(t, 0, userCreateCount, "No users should be created when E2E is disabled and admin already exists")
}

func TestSeedIfNeeded_E2EAccountIdempotent(t *testing.T) {
	cfg := newE2ETestConfig()
	counter, userCreator, accountCreator, userAccountCreator := fullMocks()

	userCreateCount := 0
	userCreator.createUserFunc = func(user *models.User) error {
		userCreateCount++
		return nil
	}

	// User already exists
	userFinder := &mockUserFinder{
		getUserByEmailFunc: func(email string) (*models.User, error) {
			return &models.User{ID: uuid.New(), Email: email}, nil
		},
	}

	spaceCreator := &mockDefaultSpaceCreator{
		createDefaultSpaceFunc: func(accountID uuid.UUID, creatorUserID uuid.UUID) (*models.Space, error) {
			t.Fatal("CreateDefaultSpace should not be called when E2E user already exists")
			return nil, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, userFinder, spaceCreator, cfg)
	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.Equal(t, 0, userCreateCount)
}

func TestSeedIfNeeded_E2EAccountCreatedEvenWhenAdminExists(t *testing.T) {
	cfg := newE2ETestConfig()

	// Admin account already exists
	counter := &mockAccountCounter{
		countFunc: func() (int64, error) { return 1, nil },
	}

	e2eUserCreated := false
	userCreator := &mockUserCreator{
		createUserFunc: func(user *models.User) error {
			e2eUserCreated = true
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

	userFinder := &mockUserFinder{
		getUserByEmailFunc: func(email string) (*models.User, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	spaceCreator := &mockDefaultSpaceCreator{
		createDefaultSpaceFunc: func(accountID uuid.UUID, creatorUserID uuid.UUID) (*models.Space, error) {
			return &models.Space{ID: uuid.New()}, nil
		},
	}

	svc := NewSeedService(counter, userCreator, noopUserUpdater(), accountCreator, userAccountCreator, userFinder, spaceCreator, cfg)
	err := svc.SeedIfNeeded()

	assert.NoError(t, err)
	assert.True(t, e2eUserCreated, "E2E user should be created even when admin account already exists")
}
