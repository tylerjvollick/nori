package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// UserRepositoryInterface defines methods needed for auth middleware
type UserRepositoryInterface interface {
	GetUserByID(id uuid.UUID) (*models.User, error)
}

// APIKeyRepositoryInterface defines methods needed for auth middleware
type APIKeyRepositoryInterface interface {
	GetByKeyHash(keyHash string) (*models.APIKey, error)
	UpdateLastUsed(id uuid.UUID) error
}

// SpaceMemberRepositoryInterface defines methods needed for space access checks
type SpaceMemberRepositoryInterface interface {
	GetByUserAndSpace(userID, spaceID uuid.UUID) (*models.SpaceMember, error)
	GetByUser(userID uuid.UUID) ([]models.Space, error)
}

// NewAuthMiddleware creates a consolidated auth middleware that:
// - Supports JWT from Authorization header or auth_token cookie
// - Supports API keys with "nori_" prefix
// - Extracts active Space from X-Space-ID header or token claims
// - Stores authDTO in c.Locals("authDTO") for consistent access
func NewAuthMiddleware(
	userRepo UserRepositoryInterface,
	apiKeyRepo APIKeyRepositoryInterface,
	spaceMemberRepo SpaceMemberRepositoryInterface,
	jwtSecret string,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get token from Authorization header or cookie
		token := extractToken(c)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authentication credentials",
			})
		}

		var authDTO *dtos.AuthDTO
		var err error

		// Check if this is an API key (starts with "nori_")
		if strings.HasPrefix(token, "nori_") {
			authDTO, err = authenticateWithAPIKey(token, apiKeyRepo, userRepo, spaceMemberRepo)
		} else {
			authDTO, err = authenticateWithJWT(token, userRepo, jwtSecret)
		}

		if err != nil {
			log.Printf("Authentication error: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired credentials",
			})
		}

		// Extract active SpaceID from X-Space-ID header if present
		// This overrides the space from the JWT token
		spaceIDHeader := c.Get("X-Space-ID")
		if spaceIDHeader != "" {
			if spaceID, err := uuid.Parse(spaceIDHeader); err == nil {
				authDTO.ActiveSpaceID = &spaceID
			}
		}

		// Store authDTO in context using consistent key "authDTO"
		c.Locals("authDTO", authDTO)

		return c.Next()
	}
}

// extractToken extracts the authentication token from either:
// 1. Authorization header (Bearer token)
// 2. auth_token cookie
func extractToken(c *fiber.Ctx) string {
	// Try Authorization header first
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	// Fall back to cookie
	return c.Cookies("nori_token")
}

// authenticateWithJWT validates a JWT token and returns the authDTO
func authenticateWithJWT(tokenString string, userRepo UserRepositoryInterface, jwtSecret string) (*dtos.AuthDTO, error) {
	// Parse and validate JWT
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Extract user ID from token
	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid user id in token claims")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid UUID format in token claims")
	}

	// Fetch user from database
	user, err := userRepo.GetUserByID(userID)
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

	// If no active space in token, use first from RecentSpaces
	if activeSpaceID == nil && len(user.RecentSpaces) > 0 {
		activeSpaceID = &user.RecentSpaces[0]
	}

	return &dtos.AuthDTO{
		User:          *user,
		AccountID:     accountID,
		ActiveSpaceID: activeSpaceID,
	}, nil
}

// hashAPIKey creates a SHA256 hash of the API key for lookup
func hashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// authenticateWithAPIKey validates an API key and returns the authDTO.
// It loads the user who created the key so that role checks (admin, etc.) work.
func authenticateWithAPIKey(
	rawKey string,
	apiKeyRepo APIKeyRepositoryInterface,
	userRepo UserRepositoryInterface,
	spaceMemberRepo SpaceMemberRepositoryInterface,
) (*dtos.AuthDTO, error) {
	// Hash the key for lookup
	keyHash := hashAPIKey(rawKey)

	// Look up the API key by hash
	apiKey, err := apiKeyRepo.GetByKeyHash(keyHash)
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

	// Update LastUsedAt timestamp (don't fail auth if this fails)
	if err := apiKeyRepo.UpdateLastUsed(apiKey.ID); err != nil {
		log.Printf("Failed to update API key last used timestamp: %v", err)
	}

	// Load the user who created the API key so role-based checks (admin, etc.) work.
	user, err := userRepo.GetUserByID(apiKey.CreatedByID)
	if err != nil {
		return nil, errors.New("failed to load API key owner")
	}

	// Resolve active space: first space the user has access to.
	var activeSpaceID *uuid.UUID
	if spaces, err := spaceMemberRepo.GetByUser(user.ID); err == nil && len(spaces) > 0 {
		activeSpaceID = &spaces[0].ID
	}

	return &dtos.AuthDTO{
		User:          *user,
		AccountID:     apiKey.AccountID,
		ActiveSpaceID: activeSpaceID,
	}, nil
}
