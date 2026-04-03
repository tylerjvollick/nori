package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterAuthRoutes registers public auth routes (no auth middleware required).
func (h *AuthHandler) RegisterAuthRoutes(app *fiber.App) {
	auth := app.Group("/auth")
	// Registration endpoint removed - users can only be created by admins
	auth.Post("/login", h.Login)
}

// RegisterProtectedAuthRoutes registers auth routes that require authentication
// but NOT the RequirePasswordChanged guard. This allows users who must change
// their password to access the change-password endpoint.
func (h *AuthHandler) RegisterProtectedAuthRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	auth := app.Group("/auth", authMiddleware)
	auth.Post("/change-password", h.ChangePassword)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var body request
	if err := c.BodyParser(&body); err != nil {
		log.Println("Failed to parse request body:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	loginResponse, err := h.authService.Login(body.Email, body.Password)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid email or password",
		})
	}

	setAuthCookie(c, loginResponse.AccessToken)

	// Also return in JSON body for compatibility
	return c.Status(http.StatusOK).JSON(loginResponse)
}

// ChangePassword handles POST /auth/change-password.
// Requires authentication (JWT or API key). The user provides their current
// password and a new password. On success, the MustChangePassword flag is
// cleared and a fresh JWT is returned (both as a cookie and in the response body).
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	type request struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}

	var body request
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "currentPassword and newPassword are required",
		})
	}

	// Get authenticated user from middleware context
	authDTO, ok := c.Locals("authDTO").(*dtos.AuthDTO)
	if !ok || authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	loginResponse, err := h.authService.ChangePassword(authDTO.User.ID, body.CurrentPassword, body.NewPassword)
	if err != nil {
		// The service returns "current password is incorrect" for wrong password
		if err.Error() == "current password is incorrect" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		// Other errors are internal
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to change password",
		})
	}

	setAuthCookie(c, loginResponse.AccessToken)

	return c.Status(http.StatusOK).JSON(loginResponse)
}

// setAuthCookie sets the nori_token HTTP-only cookie with 30-day expiry.
func setAuthCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "nori_token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
	})
}

// Register is disabled. User creation is now restricted to admin users only.
// This endpoint has been removed from the route registration.
// See the admin user management API for creating users.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	return c.Status(http.StatusNotFound).JSON(fiber.Map{
		"error": "endpoint not available",
	})
}
