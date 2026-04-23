package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
)

// requireAuth safely extracts the AuthDTO from Fiber context locals.
// If the middleware didn't set the value (or set a wrong type), it returns
// a 401 JSON response instead of panicking on a nil type assertion.
//
// Usage:
//
//	authDTO, err := requireAuth(c)
//	if err != nil {
//	    return err
//	}
func requireAuth(c *fiber.Ctx) (*dtos.AuthDTO, error) {
	authDTO, ok := c.Locals("authDTO").(*dtos.AuthDTO)
	if !ok || authDTO == nil {
		return nil, c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	return authDTO, nil
}

// spaceIDFromPath returns the space UUID stored by the RequireSpace middleware.
// Call this instead of reading authDTO.ActiveSpaceID in handlers that are
// mounted under a /:spaceId path segment.
func spaceIDFromPath(c *fiber.Ctx) (uuid.UUID, error) {
	id, ok := c.Locals("spaceID").(uuid.UUID)
	if !ok {
		return uuid.Nil, c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "space ID not resolved — RequireSpace middleware missing",
		})
	}
	return id, nil
}
