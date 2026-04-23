package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// SpaceMemberRepositoryInterface defines methods needed for space access checks
type SpaceMemberRepositoryInterface interface {
	GetByUserAndSpace(userID, spaceID uuid.UUID) (*models.SpaceMember, error)
}

// RequireAdmin is a middleware that checks if the authenticated user has admin role
// It expects authDTO to be set in c.Locals("authDTO") by the auth middleware
// Returns 403 Forbidden if the user is not an admin
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authDTO, ok := c.Locals("authDTO").(*dtos.AuthDTO)
		if !ok || authDTO == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Check if user has admin role
		if authDTO.User.Role == nil || *authDTO.User.Role != models.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
			})
		}

		return c.Next()
	}
}

// RequirePasswordChanged is a middleware that checks if the user has changed their password
// It expects authDTO to be set in c.Locals("authDTO") by the auth middleware
// If MustChangePassword is true, the user is blocked from all requests
// Returns 403 Forbidden with a specific error code so the frontend knows to redirect to password change
func RequirePasswordChanged() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authDTO, ok := c.Locals("authDTO").(*dtos.AuthDTO)
		if !ok || authDTO == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Check if user must change password
		if authDTO.User.MustChangePassword {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Password change required",
				"code":  "MUST_CHANGE_PASSWORD",
			})
		}

		return c.Next()
	}
}

// RequireSpaceAccess is a middleware that checks if the authenticated user has access to a specific space
// It expects authDTO to be set in c.Locals("authDTO") by the auth middleware
// Admins automatically have access to all spaces
// Regular users must have a SpaceMember record for the space
// The spaceID can come from:
// 1. The route parameter (e.g., /api/spaces/:spaceID/...)
// 2. The active space in authDTO
func RequireSpaceAccess(spaceMemberRepo SpaceMemberRepositoryInterface) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authDTO, ok := c.Locals("authDTO").(*dtos.AuthDTO)
		if !ok || authDTO == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Determine which spaceID to check
		var spaceID uuid.UUID
		var err error

		// Try to get spaceID from route parameter first
		spaceIDParam := c.Params("spaceID")
		if spaceIDParam != "" {
			spaceID, err = uuid.Parse(spaceIDParam)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid space ID format",
				})
			}
		} else if authDTO.ActiveSpaceID != nil {
			// Fall back to active space from authDTO
			spaceID = *authDTO.ActiveSpaceID
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No space ID provided",
			})
		}

		// Admins have automatic access to all spaces
		if authDTO.User.Role != nil && *authDTO.User.Role == models.RoleAdmin {
			return c.Next()
		}

		// For regular users, check if they have a SpaceMember record
		_, err = spaceMemberRepo.GetByUserAndSpace(authDTO.User.ID, spaceID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Access to this space is not permitted",
			})
		}

		return c.Next()
	}
}
