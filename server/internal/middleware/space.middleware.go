package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// RequireSpace extracts :spaceId from the URL path, validates it as a UUID,
// confirms the authenticated user is a member of (or admin of) that space,
// and stores the resolved uuid.UUID in c.Locals("spaceID").
//
// Handlers retrieve the value with spaceIDFromPath(c) instead of reading
// authDTO.ActiveSpaceID directly.
func RequireSpace(spaceMemberRepo SpaceMemberRepositoryInterface) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authDTO, ok := c.Locals("authDTO").(*dtos.AuthDTO)
		if !ok || authDTO == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		spaceIDParam := c.Params("spaceId")
		if spaceIDParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "space ID is required in path",
			})
		}

		spaceID, err := uuid.Parse(spaceIDParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid space ID format",
			})
		}

		// Admins have access to all spaces.
		if authDTO.User.Role != nil && *authDTO.User.Role == models.RoleAdmin {
			c.Locals("spaceID", spaceID)
			return c.Next()
		}

		// Regular users must have a SpaceMember record for this space.
		if _, err := spaceMemberRepo.GetByUserAndSpace(authDTO.User.ID, spaceID); err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "access to this space is not permitted",
			})
		}

		c.Locals("spaceID", spaceID)
		return c.Next()
	}
}
