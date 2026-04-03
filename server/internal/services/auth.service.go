package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

// UserRepositoryInterface defines the methods needed from a user repository
type UserRepositoryInterface interface {
	GetUserByID(id uuid.UUID) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) error
	UpdateUser(id uuid.UUID, input *repositories.UpdateUserInput) (*models.User, error)
	UpdateRecentSpaces(userID uuid.UUID, recentSpaces models.RecentSpaces) error
	UpdatePassword(userID uuid.UUID, hashedPassword string) error
	ClearMustChangePassword(userID uuid.UUID) error
}

// APIKeyRepositoryInterface defines the methods needed from an API key repository
type APIKeyRepositoryInterface interface {
	Create(apiKey *models.APIKey) error
	GetByKeyHash(keyHash string) (*models.APIKey, error)
	GetByAccount(accountID uuid.UUID) ([]models.APIKey, error)
	Deactivate(id uuid.UUID) error
	Delete(id uuid.UUID) error
	UpdateLastUsed(id uuid.UUID) error
}

type AuthService struct {
	userRepository        UserRepositoryInterface
	accountRepository     *repositories.AccountRepository
	userAccountRepository *repositories.UserAccountRepository
	apiKeyRepository      APIKeyRepositoryInterface
	spaceService          *SpaceService
	jwtSecret             []byte
}

func NewAuthService(userRepository UserRepositoryInterface, accountRepository *repositories.AccountRepository, userAccountRepository *repositories.UserAccountRepository, apiKeyRepository APIKeyRepositoryInterface, spaceService *SpaceService, jwtSecret string) *AuthService {
	return &AuthService{
		userRepository:        userRepository,
		accountRepository:     accountRepository,
		userAccountRepository: userAccountRepository,
		apiKeyRepository:      apiKeyRepository,
		spaceService:          spaceService,
		jwtSecret:             []byte(jwtSecret),
	}
}

type LoginResponse struct {
	AccessToken        string     `json:"accessToken"`
	UserID             uuid.UUID  `json:"userId"`
	UserEmail          string     `json:"userEmail"`
	FirstName          string     `json:"firstName"`
	LastName           string     `json:"lastName"`
	MustChangePassword bool       `json:"mustChangePassword"`
	ActiveSpaceID      *uuid.UUID `json:"activeSpaceId,omitempty"`
}

func (s *AuthService) ValidatePassword(user models.User, password string) bool {
	if user.Password == nil {
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return false
	}
	return true
}

func (s *AuthService) Authenticate(authHeader string) (*dtos.AuthDTO, error) {
	authDto, err := s.Validate(authHeader)
	if err != nil {
		return nil, err
	}
	return authDto, nil
}

func (s *AuthService) Validate(authHeader string) (*dtos.AuthDTO, error) {
	bearerToken, err := s.GetBearerToken(authHeader)
	if err != nil {
		return nil, errors.New("failed to validate token")
	}
	return s.ValidateToken(bearerToken)
}

func (s *AuthService) ValidateToken(bearerToken string) (*dtos.AuthDTO, error) {
	token, err := jwt.Parse(bearerToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		log.Println("JWT parse error:", err)
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid user id in token claims")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid UUID format in token claims")
	}
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Get the user's default account ID
	accountID := uuid.Nil
	if user.DefaultAccountID != nil {
		accountID = *user.DefaultAccountID
	}

	// Extract ActiveSpaceID from token if present
	var activeSpaceID *uuid.UUID
	if spaceIDStr, ok := claims["activeSpaceID"].(string); ok && spaceIDStr != "" {
		if spaceID, err := uuid.Parse(spaceIDStr); err == nil {
			activeSpaceID = &spaceID
		}
	}

	return &dtos.AuthDTO{
		User:          *user,
		AccountID:     accountID,
		ActiveSpaceID: activeSpaceID,
	}, nil
}

func (s *AuthService) GetBearerToken(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization format")
	}
	tokenString := parts[1]
	return tokenString, nil
}

func (s *AuthService) Login(email, password string) (*LoginResponse, error) {
	// find user by email with password set
	user, err := s.userRepository.GetUserByEmail(email)
	if err != nil {
		// TODO: handle error in same line as failed password.
		return nil, errors.New("invalid email or password")
	}

	// if user is found validate password
	if user != nil {
		isAuthenticated := s.ValidatePassword(*user, password)
		if !isAuthenticated {
			log.Println("failed to validate password")
			return nil, errors.New("invalid Email or password")
		}
	}

	return s.CreateLoginResponse(*user, nil)
}

func (s *AuthService) CreateLoginResponse(user models.User, activeSpaceID *uuid.UUID) (*LoginResponse, error) {
	// Determine ActiveSpaceID: use provided value, or first from RecentSpaces
	var spaceID *uuid.UUID
	if activeSpaceID != nil {
		spaceID = activeSpaceID
	} else if len(user.RecentSpaces) > 0 {
		spaceID = &user.RecentSpaces[0]
	}

	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24 * 30).Unix(), // 30 days
		"iat":   time.Now().Unix(),
	}

	// Add ActiveSpaceID to claims if present
	if spaceID != nil {
		claims["activeSpaceID"] = spaceID.String()
	}

	// create the token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// sign token and return as string
	accessToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		log.Printf("failed to sign access token: %v", err)
		return nil, err
	}
	return &LoginResponse{
		AccessToken:        accessToken,
		UserID:             user.ID,
		UserEmail:          user.Email,
		FirstName:          *user.FirstName,
		LastName:           *user.LastName,
		MustChangePassword: user.MustChangePassword,
		ActiveSpaceID:      spaceID,
	}, nil
}

func (s *AuthService) CreateUser(firstName, lastName, email, password string, createDefaultAccount bool) (*models.User, error) {
	// check to see if use already exists
	existing, err := s.userRepository.GetUserByEmail(email)
	if err == nil && existing != nil {
		return nil, errors.New("email already in use")
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// build the user object
	user := &models.User{
		ID:        uuid.New(),
		FirstName: &firstName,
		LastName:  &lastName,
		Email:     email,
		Password:  ptrString(string(hashedPassword)),
	}

	// save user
	if err := s.userRepository.CreateUser(user); err != nil {
		return nil, err
	}

	if createDefaultAccount {
		defaultAccount, err := s.CreateAccount(email, user.ID, models.Trial)
		if err != nil {
			return nil, err
		}

		// update user with default account id.
		// user.DefaultAccountID = defaultAccount.
		user, err = s.userRepository.UpdateUser(user.ID, &repositories.UpdateUserInput{DefaultAccountID: &defaultAccount.ID})
		log.Println("Updating user default account")
		if err != nil {
			return nil, err
		}

		// add link able
		userAccountRepository, err := s.userAccountRepository.Create(user.ID, defaultAccount.ID)
		if err != nil {
			return nil, err
		}
		if userAccountRepository == nil {
			return nil, nil
		}

		// Create default space for the account
		_, err = s.spaceService.CreateDefaultSpace(defaultAccount.ID)
		if err != nil {
			log.Printf("Failed to create default space for account %s: %v", defaultAccount.ID, err)
			return nil, err
		}
	}

	// do not return password
	user.Password = nil
	return user, nil
}

func (s *AuthService) CreateAccount(name string, createdByUserId uuid.UUID, plan models.Plan) (*models.Account, error) {
	// save account
	account, err := s.accountRepository.Create(name, createdByUserId, plan)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func ptrString(s string) *string {
	return &s
}

// hashAPIKey creates a SHA256 hash of the API key for storage and lookup
func hashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// GenerateAPIKey creates a new API key with the "nori_" prefix and returns the raw key
// The raw key is returned only once; the hash is stored in the database
func (s *AuthService) GenerateAPIKey(accountID uuid.UUID, name string, createdByID uuid.UUID, expiresAt *time.Time) (rawKey string, apiKey *models.APIKey, err error) {
	// Generate random bytes for the key
	randomBytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(randomBytes); err != nil {
		return "", nil, errors.New("failed to generate random key")
	}

	// Create raw key with "nori_" prefix
	rawKey = "nori_" + hex.EncodeToString(randomBytes)

	// Hash the raw key with SHA256 for storage
	keyHash := hashAPIKey(rawKey)

	// Create APIKey model
	apiKey = &models.APIKey{
		ID:          uuid.New(),
		AccountID:   accountID,
		Name:        name,
		KeyHash:     keyHash,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		CreatedByID: createdByID,
	}

	// Save to database
	if err := s.apiKeyRepository.Create(apiKey); err != nil {
		return "", nil, err
	}

	return rawKey, apiKey, nil
}

// ValidateAPIKey checks if an API key is valid, active, and not expired
// Returns the APIKey model and updates LastUsedAt
func (s *AuthService) ValidateAPIKey(rawKey string) (*models.APIKey, error) {
	// Check for "nori_" prefix
	if !strings.HasPrefix(rawKey, "nori_") {
		return nil, errors.New("invalid API key format")
	}

	// Hash the provided key for lookup
	keyHash := hashAPIKey(rawKey)

	// Look up the API key by hash
	apiKey, err := s.apiKeyRepository.GetByKeyHash(keyHash)
	if err != nil {
		return nil, errors.New("invalid API key")
	}

	// Check if the key is active
	if !apiKey.IsActive {
		return nil, errors.New("API key is inactive")
	}

	// Check if the key has expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("API key has expired")
	}

	// Update LastUsedAt timestamp
	if err := s.apiKeyRepository.UpdateLastUsed(apiKey.ID); err != nil {
		log.Printf("Failed to update API key last used timestamp: %v", err)
		// Don't fail authentication if we can't update the timestamp
	}

	return apiKey, nil
}

// ChangePassword changes a user's password after verifying their current password
// It hashes the new password, clears the MustChangePassword flag, and returns a new JWT
func (s *AuthService) ChangePassword(userID uuid.UUID, oldPassword, newPassword string) (*LoginResponse, error) {
	// Get the user
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Verify the old password
	if !s.ValidatePassword(*user, oldPassword) {
		return nil, errors.New("current password is incorrect")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash new password")
	}

	// Update the password
	if err := s.userRepository.UpdatePassword(userID, string(hashedPassword)); err != nil {
		return nil, errors.New("failed to update password")
	}

	// Clear the MustChangePassword flag
	if err := s.userRepository.ClearMustChangePassword(userID); err != nil {
		return nil, errors.New("failed to clear password change flag")
	}

	// Update the user object to reflect changes for JWT generation
	user.MustChangePassword = false

	// Generate and return a new JWT
	return s.CreateLoginResponse(*user, nil)
}
