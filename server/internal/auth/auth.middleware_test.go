package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// Mock repositories
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUserByID(id uuid.UUID) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateUser(id uuid.UUID, input interface{}) (*models.User, error) {
	args := m.Called(id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateRecentSpaces(userID uuid.UUID, recentSpaces models.RecentSpaces) error {
	args := m.Called(userID, recentSpaces)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePassword(userID uuid.UUID, hashedPassword string) error {
	args := m.Called(userID, hashedPassword)
	return args.Error(0)
}

func (m *MockUserRepository) ClearMustChangePassword(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

type MockAPIKeyRepository struct {
	mock.Mock
}

func (m *MockAPIKeyRepository) Create(apiKey *models.APIKey) error {
	args := m.Called(apiKey)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) GetByKeyHash(keyHash string) (*models.APIKey, error) {
	args := m.Called(keyHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) GetByAccount(accountID uuid.UUID) ([]models.APIKey, error) {
	args := m.Called(accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.APIKey), args.Error(1)
}

func (m *MockAPIKeyRepository) Deactivate(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockAPIKeyRepository) UpdateLastUsed(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

type MockSpaceMemberRepository struct {
	mock.Mock
}

func (m *MockSpaceMemberRepository) GetByUserAndSpace(userID, spaceID uuid.UUID) (*models.SpaceMember, error) {
	args := m.Called(userID, spaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SpaceMember), args.Error(1)
}

func (m *MockSpaceMemberRepository) GetByUser(userID uuid.UUID) ([]models.Space, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Space), args.Error(1)
}

// Helper functions
func generateTestJWT(userID uuid.UUID, activeSpaceID *uuid.UUID, jwtSecret string) string {
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour * 24 * 30).Unix(),
		"iat":   time.Now().Unix(),
	}
	if activeSpaceID != nil {
		claims["activeSpaceID"] = activeSpaceID.String()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	return tokenString
}

// Tests
func TestAuthMiddleware_JWT_AuthorizationHeader(t *testing.T) {
	app := fiber.New()
	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	jwtSecret := "test-secret"

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup user mock
	firstName := "Test"
	lastName := "User"
	role := models.RoleUser
	user := &models.User{
		ID:                 userID,
		Email:              "test@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		Role:               &role,
		DefaultAccountID:   &accountID,
		RecentSpaces:       models.RecentSpaces{spaceID},
		MustChangePassword: false,
	}
	mockUserRepo.On("GetUserByID", userID).Return(user, nil)

	// Generate test JWT
	token := generateTestJWT(userID, &spaceID, jwtSecret)

	// Setup route
	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
		assert.Equal(t, userID, authDTO.User.ID)
		assert.Equal(t, accountID, authDTO.AccountID)
		assert.NotNil(t, authDTO.ActiveSpaceID)
		assert.Equal(t, spaceID, *authDTO.ActiveSpaceID)
		return c.SendStatus(fiber.StatusOK)
	})

	// Make request with Authorization header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockUserRepo.AssertExpectations(t)
}

func TestAuthMiddleware_JWT_Cookie(t *testing.T) {
	app := fiber.New()
	userID := uuid.New()
	accountID := uuid.New()
	jwtSecret := "test-secret"

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup user mock
	firstName := "Test"
	lastName := "User"
	role := models.RoleUser
	user := &models.User{
		ID:                 userID,
		Email:              "test@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		Role:               &role,
		DefaultAccountID:   &accountID,
		RecentSpaces:       models.RecentSpaces{},
		MustChangePassword: false,
	}
	mockUserRepo.On("GetUserByID", userID).Return(user, nil)

	// Generate test JWT
	token := generateTestJWT(userID, nil, jwtSecret)

	// Setup route
	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
		assert.Equal(t, userID, authDTO.User.ID)
		assert.Equal(t, accountID, authDTO.AccountID)
		return c.SendStatus(fiber.StatusOK)
	})

	// Make request with cookie
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Cookie", "nori_token="+token)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockUserRepo.AssertExpectations(t)
}

func TestAuthMiddleware_APIKey(t *testing.T) {
	app := fiber.New()
	accountID := uuid.New()
	userID := uuid.New()
	spaceID := uuid.New()
	apiKeyID := uuid.New()
	rawKey := "nori_1234567890abcdef"
	keyHash := hashAPIKey(rawKey)
	jwtSecret := "test-secret"

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup API key mock
	apiKey := &models.APIKey{
		ID:          apiKeyID,
		AccountID:   accountID,
		Name:        "Test Key",
		KeyHash:     keyHash,
		IsActive:    true,
		ExpiresAt:   nil,
		CreatedByID: userID,
	}
	mockAPIKeyRepo.On("GetByKeyHash", keyHash).Return(apiKey, nil)
	mockAPIKeyRepo.On("UpdateLastUsed", apiKeyID).Return(nil)

	// Setup user mock for API key owner lookup
	adminRole := models.RoleAdmin
	user := &models.User{
		ID:               userID,
		Email:            "admin@test.com",
		Role:             &adminRole,
		DefaultAccountID: &accountID,
	}
	mockUserRepo.On("GetUserByID", userID).Return(user, nil)

	// Setup space member mock
	mockSpaceMemberRepo.On("GetByUser", userID).Return([]models.Space{
		{ID: spaceID, Name: "Test Space"},
	}, nil)

	// Setup route
	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
		assert.Equal(t, accountID, authDTO.AccountID)
		assert.Equal(t, userID, authDTO.User.ID)
		assert.NotNil(t, authDTO.User.Role)
		assert.Equal(t, models.RoleAdmin, *authDTO.User.Role)
		assert.NotNil(t, authDTO.ActiveSpaceID)
		assert.Equal(t, spaceID, *authDTO.ActiveSpaceID)
		return c.SendStatus(fiber.StatusOK)
	})

	// Make request with API key
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockAPIKeyRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockSpaceMemberRepo.AssertExpectations(t)
}

func TestAuthMiddleware_SpaceIDHeaderIgnored(t *testing.T) {
	// X-Space-ID header is no longer processed by the auth middleware.
	// Space context is now resolved from the URL path by RequireSpace middleware.
	// Verify that sending the header has no effect on authDTO.ActiveSpaceID.
	app := fiber.New()
	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	jwtSecret := "test-secret"

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	firstName := "Test"
	lastName := "User"
	role := models.RoleUser
	user := &models.User{
		ID:                 userID,
		Email:              "test@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		Role:               &role,
		DefaultAccountID:   &accountID,
		RecentSpaces:       models.RecentSpaces{},
		MustChangePassword: false,
	}
	mockUserRepo.On("GetUserByID", userID).Return(user, nil)

	token := generateTestJWT(userID, nil, jwtSecret)

	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
		// Header must NOT set ActiveSpaceID anymore.
		assert.Nil(t, authDTO.ActiveSpaceID)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Space-ID", spaceID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockUserRepo.AssertExpectations(t)
}

func TestAuthMiddleware_MissingAuth(t *testing.T) {
	app := fiber.New()
	jwtSecret := "test-secret"

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup route
	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Make request without auth
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidJWT(t *testing.T) {
	app := fiber.New()
	jwtSecret := "test-secret"

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup route
	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Make request with invalid token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_ExpiredAPIKey(t *testing.T) {
	app := fiber.New()
	accountID := uuid.New()
	rawKey := "nori_1234567890abcdef"
	keyHash := hashAPIKey(rawKey)
	jwtSecret := "test-secret"
	pastTime := time.Now().Add(-24 * time.Hour)

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup expired API key mock
	apiKey := &models.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Name:      "Expired Key",
		KeyHash:   keyHash,
		IsActive:  true,
		ExpiresAt: &pastTime,
	}
	mockAPIKeyRepo.On("GetByKeyHash", keyHash).Return(apiKey, nil)

	// Setup route
	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Make request with expired API key
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	mockAPIKeyRepo.AssertExpectations(t)
}

func TestAuthMiddleware_InactiveAPIKey(t *testing.T) {
	app := fiber.New()
	accountID := uuid.New()
	rawKey := "nori_1234567890abcdef"
	keyHash := hashAPIKey(rawKey)
	jwtSecret := "test-secret"

	mockUserRepo := new(MockUserRepository)
	mockAPIKeyRepo := new(MockAPIKeyRepository)
	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup inactive API key mock
	apiKey := &models.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Name:      "Inactive Key",
		KeyHash:   keyHash,
		IsActive:  false,
		ExpiresAt: nil,
	}
	mockAPIKeyRepo.On("GetByKeyHash", keyHash).Return(apiKey, nil)

	// Setup route
	app.Use(NewAuthMiddleware(mockUserRepo, mockAPIKeyRepo, mockSpaceMemberRepo, jwtSecret))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Make request with inactive API key
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	mockAPIKeyRepo.AssertExpectations(t)
}
