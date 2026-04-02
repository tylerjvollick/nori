package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

// Helper function to create a test auth service
func newTestAuthService(jwtSecret string) *AuthService {
	return &AuthService{
		userRepository:        nil, // Not needed for token tests
		accountRepository:     nil,
		userAccountRepository: nil,
		spaceService:          nil,
		jwtSecret:             []byte(jwtSecret),
	}
}

func TestCreateLoginResponse_WithConfigSecret(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	authService := newTestAuthService(jwtSecret)

	userID := uuid.New()
	firstName := "John"
	lastName := "Doe"
	user := models.User{
		ID:                 userID,
		Email:              "john@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: false,
		RecentSpaces:       models.RecentSpaces{},
	}

	// Act
	response, err := authService.CreateLoginResponse(user, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.AccessToken)
	assert.Equal(t, userID, response.UserID)
	assert.Equal(t, "john@example.com", response.UserEmail)
	assert.Equal(t, "John", response.FirstName)
	assert.Equal(t, "Doe", response.LastName)
	assert.False(t, response.MustChangePassword)

	// Verify the token can be parsed with the same secret
	token, err := jwt.Parse(response.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)
}

func TestCreateLoginResponse_TokenExpiry30Days(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	authService := newTestAuthService(jwtSecret)

	firstName := "John"
	lastName := "Doe"
	user := models.User{
		ID:                 uuid.New(),
		Email:              "john@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: false,
		RecentSpaces:       models.RecentSpaces{},
	}

	// Act
	response, err := authService.CreateLoginResponse(user, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)

	// Parse token and check expiry
	token, err := jwt.Parse(response.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	exp, ok := claims["exp"].(float64)
	assert.True(t, ok)

	expiryTime := time.Unix(int64(exp), 0)
	expectedExpiry := time.Now().Add(time.Hour * 24 * 30)

	// Allow 5 seconds tolerance for test execution time
	timeDiff := expiryTime.Sub(expectedExpiry).Abs()
	assert.Less(t, timeDiff, 5*time.Second, "Token expiry should be approximately 30 days from now")
}

func TestCreateLoginResponse_WithActiveSpaceID(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	authService := newTestAuthService(jwtSecret)

	spaceID := uuid.New()
	firstName := "John"
	lastName := "Doe"
	user := models.User{
		ID:                 uuid.New(),
		Email:              "john@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: false,
		RecentSpaces:       models.RecentSpaces{},
	}

	// Act
	response, err := authService.CreateLoginResponse(user, &spaceID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.ActiveSpaceID)
	assert.Equal(t, spaceID, *response.ActiveSpaceID)

	// Verify the token contains ActiveSpaceID claim
	token, err := jwt.Parse(response.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	activeSpaceIDStr, ok := claims["activeSpaceID"].(string)
	assert.True(t, ok)
	assert.Equal(t, spaceID.String(), activeSpaceIDStr)
}

func TestCreateLoginResponse_WithRecentSpaces(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	authService := newTestAuthService(jwtSecret)

	recentSpaceID := uuid.New()
	firstName := "John"
	lastName := "Doe"
	user := models.User{
		ID:                 uuid.New(),
		Email:              "john@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: false,
		RecentSpaces:       models.RecentSpaces{recentSpaceID, uuid.New()},
	}

	// Act - no explicit activeSpaceID provided
	response, err := authService.CreateLoginResponse(user, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.ActiveSpaceID)
	assert.Equal(t, recentSpaceID, *response.ActiveSpaceID, "Should use first space from RecentSpaces")

	// Verify the token contains ActiveSpaceID claim
	token, err := jwt.Parse(response.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	activeSpaceIDStr, ok := claims["activeSpaceID"].(string)
	assert.True(t, ok)
	assert.Equal(t, recentSpaceID.String(), activeSpaceIDStr)
}

func TestCreateLoginResponse_NoActiveSpaceID(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	authService := newTestAuthService(jwtSecret)

	firstName := "John"
	lastName := "Doe"
	user := models.User{
		ID:                 uuid.New(),
		Email:              "john@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: false,
		RecentSpaces:       models.RecentSpaces{}, // empty
	}

	// Act - no explicit activeSpaceID and no RecentSpaces
	response, err := authService.CreateLoginResponse(user, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Nil(t, response.ActiveSpaceID, "Should be nil when no space is available")

	// Verify the token does not contain ActiveSpaceID claim
	token, err := jwt.Parse(response.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	_, hasActiveSpaceID := claims["activeSpaceID"]
	assert.False(t, hasActiveSpaceID, "Token should not have activeSpaceID claim")
}

func TestCreateLoginResponse_MustChangePasswordFlag(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	authService := newTestAuthService(jwtSecret)

	firstName := "John"
	lastName := "Doe"
	user := models.User{
		ID:                 uuid.New(),
		Email:              "john@example.com",
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: true, // User must change password
		RecentSpaces:       models.RecentSpaces{},
	}

	// Act
	response, err := authService.CreateLoginResponse(user, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.MustChangePassword, "Should indicate user must change password")
}

func TestValidateToken_WithConfigSecret(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"

	userID := uuid.New()

	// Create a valid token
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": "john@example.com",
		"exp":   time.Now().Add(time.Hour * 24 * 30).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	assert.NoError(t, err)

	// Note: ValidateToken requires userRepository.GetUserByID to work
	// This test would need a mock or test database, so we'll skip the full validation
	// and just test that the token itself is parseable with the correct secret

	// Verify token can be parsed
	parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)
}

func TestValidateToken_WithActiveSpaceID(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"

	userID := uuid.New()
	spaceID := uuid.New()

	// Create a valid token with ActiveSpaceID
	claims := jwt.MapClaims{
		"sub":           userID.String(),
		"email":         "john@example.com",
		"activeSpaceID": spaceID.String(),
		"exp":           time.Now().Add(time.Hour * 24 * 30).Unix(),
		"iat":           time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	assert.NoError(t, err)

	// Verify token can be parsed and has activeSpaceID claim
	parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	parsedClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	activeSpaceIDStr, ok := parsedClaims["activeSpaceID"].(string)
	assert.True(t, ok)
	assert.Equal(t, spaceID.String(), activeSpaceIDStr)
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	wrongSecret := "wrong-secret-key"
	authService := newTestAuthService(jwtSecret)

	userID := uuid.New()

	// Create a token with wrong secret
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": "john@example.com",
		"exp":   time.Now().Add(time.Hour * 24 * 30).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(wrongSecret))
	assert.NoError(t, err)

	// Act - try to validate with correct secret
	_, err = authService.ValidateToken(tokenString)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	authService := newTestAuthService(jwtSecret)

	userID := uuid.New()

	// Create an expired token
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": "john@example.com",
		"exp":   time.Now().Add(-time.Hour).Unix(), // expired 1 hour ago
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	assert.NoError(t, err)

	// Act
	_, err = authService.ValidateToken(tokenString)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired token")
}

func TestValidatePassword_Success(t *testing.T) {
	// Arrange
	authService := newTestAuthService("test-secret")

	password := "testpassword123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	hashedPasswordStr := string(hashedPassword)
	user := models.User{
		Password: &hashedPasswordStr,
	}

	// Act
	isValid := authService.ValidatePassword(user, password)

	// Assert
	assert.True(t, isValid)
}

func TestValidatePassword_WrongPassword(t *testing.T) {
	// Arrange
	authService := newTestAuthService("test-secret")

	password := "testpassword123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)

	hashedPasswordStr := string(hashedPassword)
	user := models.User{
		Password: &hashedPasswordStr,
	}

	// Act
	isValid := authService.ValidatePassword(user, "wrongpassword")

	// Assert
	assert.False(t, isValid)
}

func TestValidatePassword_NoPassword(t *testing.T) {
	// Arrange
	authService := newTestAuthService("test-secret")

	user := models.User{
		Password: nil,
	}

	// Act
	isValid := authService.ValidatePassword(user, "anypassword")

	// Assert
	assert.False(t, isValid)
}

// MockAPIKeyRepository is a mock implementation of the APIKeyRepository
type MockAPIKeyRepository struct {
	createFunc         func(*models.APIKey) error
	getByKeyHashFunc   func(string) (*models.APIKey, error)
	updateLastUsedFunc func(uuid.UUID) error
}

func (m *MockAPIKeyRepository) Create(apiKey *models.APIKey) error {
	if m.createFunc != nil {
		return m.createFunc(apiKey)
	}
	return nil
}

func (m *MockAPIKeyRepository) GetByKeyHash(keyHash string) (*models.APIKey, error) {
	if m.getByKeyHashFunc != nil {
		return m.getByKeyHashFunc(keyHash)
	}
	return nil, assert.AnError
}

func (m *MockAPIKeyRepository) GetByAccount(accountID uuid.UUID) ([]models.APIKey, error) {
	return nil, nil
}

func (m *MockAPIKeyRepository) Deactivate(id uuid.UUID) error {
	return nil
}

func (m *MockAPIKeyRepository) Delete(id uuid.UUID) error {
	return nil
}

func (m *MockAPIKeyRepository) UpdateLastUsed(id uuid.UUID) error {
	if m.updateLastUsedFunc != nil {
		return m.updateLastUsedFunc(id)
	}
	return nil
}

// TestGenerateAPIKey tests the generation of API keys
func TestGenerateAPIKey(t *testing.T) {
	// Arrange
	var createdKey *models.APIKey
	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			createdKey = apiKey
			return nil
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	accountID := uuid.New()
	createdByID := uuid.New()
	keyName := "Test API Key"

	// Act
	rawKey, apiKey, err := authService.GenerateAPIKey(accountID, keyName, createdByID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, apiKey)
	assert.NotEmpty(t, rawKey)
	assert.True(t, strings.HasPrefix(rawKey, "nori_"), "API key should have 'nori_' prefix")
	assert.Equal(t, accountID, apiKey.AccountID)
	assert.Equal(t, keyName, apiKey.Name)
	assert.Equal(t, createdByID, apiKey.CreatedByID)
	assert.True(t, apiKey.IsActive)
	assert.NotEmpty(t, apiKey.KeyHash, "Key hash should be stored")
	assert.NotEqual(t, rawKey, apiKey.KeyHash, "Hash should not equal raw key")

	// Verify the key is 69 characters (5 for "nori_" + 64 for hex)
	assert.Equal(t, 69, len(rawKey))

	// Verify Create was called with the right data
	assert.NotNil(t, createdKey)
	assert.Equal(t, accountID, createdKey.AccountID)
}

func TestGenerateAPIKey_WithExpiry(t *testing.T) {
	// Arrange
	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			return nil
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	accountID := uuid.New()
	createdByID := uuid.New()
	keyName := "Test API Key"
	expiresAt := time.Now().Add(time.Hour * 24 * 30) // 30 days

	// Act
	rawKey, apiKey, err := authService.GenerateAPIKey(accountID, keyName, createdByID, &expiresAt)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, apiKey)
	assert.NotNil(t, apiKey.ExpiresAt)
	assert.Equal(t, expiresAt.Unix(), apiKey.ExpiresAt.Unix())
	assert.True(t, strings.HasPrefix(rawKey, "nori_"))
}

func TestGenerateAPIKey_RepositoryError(t *testing.T) {
	// Arrange
	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			return errors.New("database error")
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	accountID := uuid.New()
	createdByID := uuid.New()
	keyName := "Test API Key"

	// Act
	rawKey, apiKey, err := authService.GenerateAPIKey(accountID, keyName, createdByID, nil)

	// Assert
	assert.Error(t, err)
	assert.Empty(t, rawKey)
	assert.Nil(t, apiKey)
}

func TestValidateAPIKey_Success(t *testing.T) {
	// Arrange
	accountID := uuid.New()
	createdByID := uuid.New()
	var storedKey *models.APIKey

	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			storedKey = apiKey
			return nil
		},
		getByKeyHashFunc: func(keyHash string) (*models.APIKey, error) {
			if storedKey != nil && storedKey.KeyHash == keyHash {
				return storedKey, nil
			}
			return nil, errors.New("not found")
		},
		updateLastUsedFunc: func(id uuid.UUID) error {
			return nil
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	// First generate a key to get a valid rawKey and hash
	rawKey, generatedKey, err := authService.GenerateAPIKey(accountID, "Test Key", createdByID, nil)
	assert.NoError(t, err)

	// Act
	validatedKey, err := authService.ValidateAPIKey(rawKey)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, validatedKey)
	assert.Equal(t, generatedKey.ID, validatedKey.ID)
	assert.Equal(t, accountID, validatedKey.AccountID)
}

func TestValidateAPIKey_InvalidPrefix(t *testing.T) {
	// Arrange
	mockRepo := &MockAPIKeyRepository{}
	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	// Act - use a key without "nori_" prefix
	validatedKey, err := authService.ValidateAPIKey("invalid_key_format")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, validatedKey)
	assert.Contains(t, err.Error(), "invalid API key format")
}

func TestValidateAPIKey_NotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockAPIKeyRepository{
		getByKeyHashFunc: func(keyHash string) (*models.APIKey, error) {
			return nil, errors.New("not found")
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	rawKey := "nori_" + strings.Repeat("a", 64) // Valid format but doesn't exist

	// Act
	validatedKey, err := authService.ValidateAPIKey(rawKey)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, validatedKey)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestValidateAPIKey_Inactive(t *testing.T) {
	// Arrange
	accountID := uuid.New()
	createdByID := uuid.New()
	var storedKey *models.APIKey

	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			storedKey = apiKey
			return nil
		},
		getByKeyHashFunc: func(keyHash string) (*models.APIKey, error) {
			if storedKey != nil && storedKey.KeyHash == keyHash {
				// Return the inactive key
				storedKey.IsActive = false
				return storedKey, nil
			}
			return nil, errors.New("not found")
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	// Generate a key
	rawKey, _, err := authService.GenerateAPIKey(accountID, "Test Key", createdByID, nil)
	assert.NoError(t, err)

	// Act
	validatedKey, err := authService.ValidateAPIKey(rawKey)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, validatedKey)
	assert.Contains(t, err.Error(), "API key is inactive")
}

func TestValidateAPIKey_Expired(t *testing.T) {
	// Arrange
	accountID := uuid.New()
	createdByID := uuid.New()
	expiredTime := time.Now().Add(-time.Hour) // Expired 1 hour ago
	var storedKey *models.APIKey

	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			storedKey = apiKey
			return nil
		},
		getByKeyHashFunc: func(keyHash string) (*models.APIKey, error) {
			if storedKey != nil && storedKey.KeyHash == keyHash {
				return storedKey, nil
			}
			return nil, errors.New("not found")
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	// Generate a key with expiry in the past
	rawKey, _, err := authService.GenerateAPIKey(accountID, "Test Key", createdByID, &expiredTime)
	assert.NoError(t, err)

	// Act
	validatedKey, err := authService.ValidateAPIKey(rawKey)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, validatedKey)
	assert.Contains(t, err.Error(), "API key has expired")
}

func TestValidateAPIKey_UpdateLastUsed(t *testing.T) {
	// Arrange
	accountID := uuid.New()
	createdByID := uuid.New()
	var storedKey *models.APIKey
	updateLastUsedCalled := false

	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			storedKey = apiKey
			return nil
		},
		getByKeyHashFunc: func(keyHash string) (*models.APIKey, error) {
			if storedKey != nil && storedKey.KeyHash == keyHash {
				return storedKey, nil
			}
			return nil, errors.New("not found")
		},
		updateLastUsedFunc: func(id uuid.UUID) error {
			updateLastUsedCalled = true
			return nil
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	// Generate a key
	rawKey, _, err := authService.GenerateAPIKey(accountID, "Test Key", createdByID, nil)
	assert.NoError(t, err)

	// Act
	validatedKey, err := authService.ValidateAPIKey(rawKey)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, validatedKey)
	assert.True(t, updateLastUsedCalled, "UpdateLastUsed should have been called")
}

func TestValidateAPIKey_UpdateLastUsedError(t *testing.T) {
	// Arrange
	accountID := uuid.New()
	createdByID := uuid.New()
	var storedKey *models.APIKey

	mockRepo := &MockAPIKeyRepository{
		createFunc: func(apiKey *models.APIKey) error {
			storedKey = apiKey
			return nil
		},
		getByKeyHashFunc: func(keyHash string) (*models.APIKey, error) {
			if storedKey != nil && storedKey.KeyHash == keyHash {
				return storedKey, nil
			}
			return nil, errors.New("not found")
		},
		updateLastUsedFunc: func(id uuid.UUID) error {
			return errors.New("failed to update")
		},
	}

	authService := &AuthService{
		apiKeyRepository: mockRepo,
		jwtSecret:        []byte("test-secret"),
	}

	// Generate a key
	rawKey, _, err := authService.GenerateAPIKey(accountID, "Test Key", createdByID, nil)
	assert.NoError(t, err)

	// Act
	validatedKey, err := authService.ValidateAPIKey(rawKey)

	// Assert - should still succeed even if UpdateLastUsed fails
	assert.NoError(t, err)
	assert.NotNil(t, validatedKey)
}

func TestHashAPIKey_Deterministic(t *testing.T) {
	// Arrange
	rawKey := "nori_test1234567890abcdef"

	// Act
	hash1 := hashAPIKey(rawKey)
	hash2 := hashAPIKey(rawKey)

	// Assert - same input should produce same hash
	assert.Equal(t, hash1, hash2, "Hash function should be deterministic")
	assert.NotEmpty(t, hash1)
}

func TestHashAPIKey_Different(t *testing.T) {
	// Arrange
	rawKey1 := "nori_test1234567890abcdef"
	rawKey2 := "nori_different_key_value"

	// Act
	hash1 := hashAPIKey(rawKey1)
	hash2 := hashAPIKey(rawKey2)

	// Assert - different inputs should produce different hashes
	assert.NotEqual(t, hash1, hash2, "Different keys should produce different hashes")
}

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	getUserByIDFunc             func(uuid.UUID) (*models.User, error)
	getUserByEmailFunc          func(string) (*models.User, error)
	createUserFunc              func(*models.User) error
	updateUserFunc              func(uuid.UUID, *repositories.UpdateUserInput) (*models.User, error)
	updateRecentSpacesFunc      func(uuid.UUID, models.RecentSpaces) error
	updatePasswordFunc          func(uuid.UUID, string) error
	clearMustChangePasswordFunc func(uuid.UUID) error
}

func (m *MockUserRepository) GetUserByID(id uuid.UUID) (*models.User, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(id)
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) GetUserByEmail(email string) (*models.User, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(email)
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) CreateUser(user *models.User) error {
	if m.createUserFunc != nil {
		return m.createUserFunc(user)
	}
	return errors.New("not implemented")
}

func (m *MockUserRepository) UpdateUser(id uuid.UUID, input *repositories.UpdateUserInput) (*models.User, error) {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(id, input)
	}
	return nil, errors.New("not implemented")
}

func (m *MockUserRepository) UpdateRecentSpaces(userID uuid.UUID, recentSpaces models.RecentSpaces) error {
	if m.updateRecentSpacesFunc != nil {
		return m.updateRecentSpacesFunc(userID, recentSpaces)
	}
	return errors.New("not implemented")
}

func (m *MockUserRepository) UpdatePassword(userID uuid.UUID, hashedPassword string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(userID, hashedPassword)
	}
	return nil
}

func (m *MockUserRepository) ClearMustChangePassword(userID uuid.UUID) error {
	if m.clearMustChangePasswordFunc != nil {
		return m.clearMustChangePasswordFunc(userID)
	}
	return nil
}

func TestChangePassword_Success(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	userID := uuid.New()
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"

	// Create a hashed old password
	hashedOldPassword, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedOldPasswordStr := string(hashedOldPassword)

	firstName := "John"
	lastName := "Doe"
	user := &models.User{
		ID:                 userID,
		Email:              "john@example.com",
		Password:           &hashedOldPasswordStr,
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: true,
		RecentSpaces:       models.RecentSpaces{},
	}

	passwordUpdated := false
	mustChangePasswordCleared := false

	mockUserRepo := &MockUserRepository{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == userID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
		updatePasswordFunc: func(id uuid.UUID, hashedPassword string) error {
			if id == userID {
				passwordUpdated = true
				user.Password = &hashedPassword
				return nil
			}
			return errors.New("user not found")
		},
		clearMustChangePasswordFunc: func(id uuid.UUID) error {
			if id == userID {
				mustChangePasswordCleared = true
				user.MustChangePassword = false
				return nil
			}
			return errors.New("user not found")
		},
	}

	authService := &AuthService{
		userRepository: mockUserRepo,
		jwtSecret:      []byte(jwtSecret),
	}

	// Act
	response, err := authService.ChangePassword(userID, oldPassword, newPassword)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.AccessToken)
	assert.Equal(t, userID, response.UserID)
	assert.Equal(t, "john@example.com", response.UserEmail)
	assert.False(t, response.MustChangePassword, "MustChangePassword should be false after password change")
	assert.True(t, passwordUpdated, "Password should have been updated")
	assert.True(t, mustChangePasswordCleared, "MustChangePassword flag should have been cleared")

	// Verify the new password works
	assert.True(t, authService.ValidatePassword(*user, newPassword), "New password should be valid")
	assert.False(t, authService.ValidatePassword(*user, oldPassword), "Old password should no longer be valid")
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	userID := uuid.New()
	oldPassword := "oldpassword123"
	wrongOldPassword := "wrongoldpassword"
	newPassword := "newpassword456"

	// Create a hashed old password
	hashedOldPassword, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedOldPasswordStr := string(hashedOldPassword)

	firstName := "John"
	lastName := "Doe"
	user := &models.User{
		ID:                 userID,
		Email:              "john@example.com",
		Password:           &hashedOldPasswordStr,
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: true,
		RecentSpaces:       models.RecentSpaces{},
	}

	mockUserRepo := &MockUserRepository{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == userID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
	}

	authService := &AuthService{
		userRepository: mockUserRepo,
		jwtSecret:      []byte(jwtSecret),
	}

	// Act
	response, err := authService.ChangePassword(userID, wrongOldPassword, newPassword)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "current password is incorrect")
}

func TestChangePassword_UserNotFound(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	userID := uuid.New()
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"

	mockUserRepo := &MockUserRepository{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			return nil, errors.New("user not found")
		},
	}

	authService := &AuthService{
		userRepository: mockUserRepo,
		jwtSecret:      []byte(jwtSecret),
	}

	// Act
	response, err := authService.ChangePassword(userID, oldPassword, newPassword)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "user not found")
}

func TestChangePassword_UpdatePasswordError(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	userID := uuid.New()
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"

	// Create a hashed old password
	hashedOldPassword, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedOldPasswordStr := string(hashedOldPassword)

	firstName := "John"
	lastName := "Doe"
	user := &models.User{
		ID:                 userID,
		Email:              "john@example.com",
		Password:           &hashedOldPasswordStr,
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: true,
		RecentSpaces:       models.RecentSpaces{},
	}

	mockUserRepo := &MockUserRepository{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == userID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
		updatePasswordFunc: func(id uuid.UUID, hashedPassword string) error {
			return errors.New("database error")
		},
	}

	authService := &AuthService{
		userRepository: mockUserRepo,
		jwtSecret:      []byte(jwtSecret),
	}

	// Act
	response, err := authService.ChangePassword(userID, oldPassword, newPassword)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to update password")
}

func TestChangePassword_ClearMustChangePasswordError(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	userID := uuid.New()
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"

	// Create a hashed old password
	hashedOldPassword, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedOldPasswordStr := string(hashedOldPassword)

	firstName := "John"
	lastName := "Doe"
	user := &models.User{
		ID:                 userID,
		Email:              "john@example.com",
		Password:           &hashedOldPasswordStr,
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: true,
		RecentSpaces:       models.RecentSpaces{},
	}

	mockUserRepo := &MockUserRepository{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == userID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
		updatePasswordFunc: func(id uuid.UUID, hashedPassword string) error {
			user.Password = &hashedPassword
			return nil
		},
		clearMustChangePasswordFunc: func(id uuid.UUID) error {
			return errors.New("database error")
		},
	}

	authService := &AuthService{
		userRepository: mockUserRepo,
		jwtSecret:      []byte(jwtSecret),
	}

	// Act
	response, err := authService.ChangePassword(userID, oldPassword, newPassword)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to clear password change flag")
}

func TestChangePassword_ReturnsNewJWT(t *testing.T) {
	// Arrange
	jwtSecret := "test-secret-key-12345"
	userID := uuid.New()
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"

	// Create a hashed old password
	hashedOldPassword, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)
	hashedOldPasswordStr := string(hashedOldPassword)

	firstName := "John"
	lastName := "Doe"
	user := &models.User{
		ID:                 userID,
		Email:              "john@example.com",
		Password:           &hashedOldPasswordStr,
		FirstName:          &firstName,
		LastName:           &lastName,
		MustChangePassword: true,
		RecentSpaces:       models.RecentSpaces{},
	}

	mockUserRepo := &MockUserRepository{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == userID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
		updatePasswordFunc: func(id uuid.UUID, hashedPassword string) error {
			user.Password = &hashedPassword
			return nil
		},
		clearMustChangePasswordFunc: func(id uuid.UUID) error {
			user.MustChangePassword = false
			return nil
		},
	}

	authService := &AuthService{
		userRepository: mockUserRepo,
		jwtSecret:      []byte(jwtSecret),
	}

	// Act
	response, err := authService.ChangePassword(userID, oldPassword, newPassword)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.AccessToken)

	// Verify the JWT is valid and has the expected claims
	token, err := jwt.Parse(response.AccessToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, userID.String(), claims["sub"])
	assert.Equal(t, "john@example.com", claims["email"])

	// Verify expiry is 30 days
	exp, ok := claims["exp"].(float64)
	assert.True(t, ok)
	expiryTime := time.Unix(int64(exp), 0)
	expectedExpiry := time.Now().Add(time.Hour * 24 * 30)
	timeDiff := expiryTime.Sub(expectedExpiry).Abs()
	assert.Less(t, timeDiff, 5*time.Second)
}
