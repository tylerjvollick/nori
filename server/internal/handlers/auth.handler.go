package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

type AuthHandler struct {
	authService     *services.AuthService
	spaceMemberRepo SpaceMemberRepositoryInterface
	spaceRepo       SpaceRepositoryInterface
}

func NewAuthHandler(authService *services.AuthService, spaceMemberRepo SpaceMemberRepositoryInterface, spaceRepo SpaceRepositoryInterface) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		spaceMemberRepo: spaceMemberRepo,
		spaceRepo:       spaceRepo,
	}
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
	auth.Post("/logout", h.Logout)
	auth.Get("/me", h.Me)
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

// Me handles GET /auth/me.
// Returns the current user's info, role, active space, and list of accessible spaces.
// Admin users see all spaces in the account; regular users see only their memberships.
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	authDTO, ok := c.Locals("authDTO").(*dtos.AuthDTO)
	if !ok || authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	user := authDTO.User

	// Determine accessible spaces based on role
	var spaces []models.Space
	var err error

	if user.Role != nil && *user.Role == models.RoleAdmin {
		// Admins have implicit access to all spaces in the account
		spaces, err = h.spaceRepo.FindByAccountID(authDTO.AccountID)
	} else {
		// Regular users only see spaces they're members of
		spaces, err = h.spaceMemberRepo.GetByUser(user.ID)
	}

	if err != nil {
		log.Println("Failed to fetch accessible spaces:", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch accessible spaces",
		})
	}

	// Convert spaces to response DTOs
	spacesDTOs := make([]dtos.SpaceResponseDTO, len(spaces))
	for i, s := range spaces {
		spacesDTOs[i] = dtos.SpaceResponseDTO{
			ID:        s.ID,
			Name:      s.Name,
			AccountID: s.AccountID,
			IsDefault: s.IsDefault,
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return c.Status(http.StatusOK).JSON(dtos.MeResponseDTO{
		ID:                 user.ID,
		Email:              user.Email,
		FirstName:          user.FirstName,
		LastName:           user.LastName,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
		ActiveSpaceID:      authDTO.ActiveSpaceID,
		AccessibleSpaces:   spacesDTOs,
	})
}

// Logout handles POST /auth/logout.
// Clears the HTTP-only nori_token cookie, effectively ending the browser session.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	clearAuthCookie(c)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "logged out",
	})
}

// clearAuthCookie expires the nori_token cookie immediately, clearing the session.
func clearAuthCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "nori_token",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
		Expires:  time.Now().Add(-1 * time.Hour), // expired
	})
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
