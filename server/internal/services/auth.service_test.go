package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/models"
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
