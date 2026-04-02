package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterAuthRoutes(app *fiber.App) {
	auth := app.Group("/auth")
	// Registration endpoint removed - users can only be created by admins
	auth.Post("/login", h.Login)
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

	// Set HTTP-only cookie with 30-day expiry
	cookie := &fiber.Cookie{
		Name:     "nori_token",
		Value:    loginResponse.AccessToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
	}
	c.Cookie(cookie)

	// Also return in JSON body for compatibility
	return c.Status(http.StatusOK).JSON(loginResponse)
}

// Register is disabled. User creation is now restricted to admin users only.
// This endpoint has been removed from the route registration.
// See the admin user management API for creating users.
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	return c.Status(http.StatusNotFound).JSON(fiber.Map{
		"error": "endpoint not available",
	})
}
