package handlers

import (
	"net/http"
	"net/mail"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/middleware"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/services"
)

type AdminUserHandler struct {
	adminUserService *services.AdminUserService
}

func NewAdminUserHandler(adminUserService *services.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{
		adminUserService: adminUserService,
	}
}

func (h *AdminUserHandler) RegisterAdminUserRoutes(app *fiber.App) {
	admin := app.Group("/admin")

	// Apply RequireAdmin middleware to all admin routes
	admin.Use(middleware.RequireAdmin())

	// User management endpoints
	admin.Post("/users", h.CreateUser)
	admin.Get("/users", h.ListUsers)
	admin.Put("/users/:id", h.UpdateUser)
	admin.Delete("/users/:id", h.DeleteUser)
}

// CreateUserRequest defines the request body for creating a user
type CreateUserRequest struct {
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	TempPassword string `json:"tempPassword"`
	Role         string `json:"role"` // "admin" or "user"
}

// UpdateUserRequest defines the request body for updating a user
type UpdateUserRequest struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Role      *string `json:"role,omitempty"` // "admin" or "user"
}

// CreateUser handles POST /admin/users
func (h *AdminUserHandler) CreateUser(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.Email == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "email is required",
		})
	}
	if req.FirstName == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "firstName is required",
		})
	}
	if req.LastName == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "lastName is required",
		})
	}
	if req.TempPassword == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "tempPassword is required",
		})
	}

	// Validate email format
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid email format",
		})
	}

	// Validate and parse role
	var role models.Role
	if req.Role == "" {
		role = models.RoleUser // Default to user role
	} else if req.Role == "admin" {
		role = models.RoleAdmin
	} else if req.Role == "user" {
		role = models.RoleUser
	} else {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid role, must be 'admin' or 'user'",
		})
	}

	// Create user
	user, err := h.adminUserService.CreateUser(
		authDTO.AccountID,
		req.Email,
		req.FirstName,
		req.LastName,
		req.TempPassword,
		role,
	)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Don't return the password hash
	user.Password = nil

	return c.Status(http.StatusCreated).JSON(user)
}

// ListUsers handles GET /admin/users
func (h *AdminUserHandler) ListUsers(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	users, err := h.adminUserService.ListUsers(authDTO.AccountID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Don't return password hashes
	for i := range users {
		users[i].Password = nil
	}

	return c.Status(http.StatusOK).JSON(users)
}

// UpdateUser handles PUT /admin/users/:id
func (h *AdminUserHandler) UpdateUser(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Parse user ID from path parameter
	userIDStr := c.Params("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user ID",
		})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Parse role if provided
	var role *models.Role
	if req.Role != nil {
		if *req.Role == "admin" {
			r := models.RoleAdmin
			role = &r
		} else if *req.Role == "user" {
			r := models.RoleUser
			role = &r
		} else {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid role, must be 'admin' or 'user'",
			})
		}
	}

	// Update user
	user, err := h.adminUserService.UpdateUser(userID, req.FirstName, req.LastName, role)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Don't return the password hash
	user.Password = nil

	return c.Status(http.StatusOK).JSON(user)
}

// DeleteUser handles DELETE /admin/users/:id
func (h *AdminUserHandler) DeleteUser(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Parse user ID from path parameter
	userIDStr := c.Params("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user ID",
		})
	}

	// Delete user
	err = h.adminUserService.DeleteUser(userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}
