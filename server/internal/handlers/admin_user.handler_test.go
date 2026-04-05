package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/middleware"
	"github.com/tylerjvollick/nori/internal/models"
)

func TestAdminUserRoutes_AllRegistered(t *testing.T) {
	// This test verifies that all admin user routes are registered

	app := fiber.New()
	admin := app.Group("/admin")
	handler := NewAdminUserHandler(nil) // nil is okay for route registration test
	handler.RegisterAdminUserRoutes(admin)

	routes := app.GetRoutes()

	expectedRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/admin/users"},
		{"GET", "/admin/users"},
		{"PUT", "/admin/users/:id"},
		{"DELETE", "/admin/users/:id"},
	}

	for _, expected := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Method == expected.method && route.Path == expected.path {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected route %s %s to be registered", expected.method, expected.path)
	}
}

func TestCreateUser_ValidatesRequiredFields(t *testing.T) {
	testCases := []struct {
		name        string
		requestBody map[string]interface{}
		expectedMsg string
	}{
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"firstName":    "John",
				"lastName":     "Doe",
				"tempPassword": "TempPass123!",
				"role":         "user",
			},
			expectedMsg: "email is required",
		},
		{
			name: "missing firstName",
			requestBody: map[string]interface{}{
				"email":        "test@example.com",
				"lastName":     "Doe",
				"tempPassword": "TempPass123!",
				"role":         "user",
			},
			expectedMsg: "firstName is required",
		},
		{
			name: "missing lastName",
			requestBody: map[string]interface{}{
				"email":        "test@example.com",
				"firstName":    "John",
				"tempPassword": "TempPass123!",
				"role":         "user",
			},
			expectedMsg: "lastName is required",
		},
		{
			name: "missing tempPassword",
			requestBody: map[string]interface{}{
				"email":     "test@example.com",
				"firstName": "John",
				"lastName":  "Doe",
				"role":      "user",
			},
			expectedMsg: "tempPassword is required",
		},
		{
			name: "invalid email format",
			requestBody: map[string]interface{}{
				"email":        "notanemail",
				"firstName":    "John",
				"lastName":     "Doe",
				"tempPassword": "TempPass123!",
				"role":         "user",
			},
			expectedMsg: "invalid email format",
		},
		{
			name: "invalid role",
			requestBody: map[string]interface{}{
				"email":        "test@example.com",
				"firstName":    "John",
				"lastName":     "Doe",
				"tempPassword": "TempPass123!",
				"role":         "superuser",
			},
			expectedMsg: "invalid role, must be 'admin' or 'user'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			app := fiber.New()
			handler := NewAdminUserHandler(nil)

			// Set up a route without the middleware to test validation logic
			app.Post("/test/users", func(c *fiber.Ctx) error {
				// Mock authDTO
				adminRole := models.RoleAdmin
				accountID := uuid.New()
				authDTO := &dtos.AuthDTO{
					User: models.User{
						ID:   uuid.New(),
						Role: &adminRole,
					},
					AccountID: accountID,
				}
				c.Locals("authDTO", authDTO)
				return handler.CreateUser(c)
			})

			body, err := json.Marshal(tc.requestBody)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/test/users", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Act
			resp, err := app.Test(req, -1)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Check response body contains expected error message
			var responseBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedMsg, responseBody["error"])
		})
	}
}

func TestUpdateUser_ValidatesUserID(t *testing.T) {
	// Arrange
	app := fiber.New()
	handler := NewAdminUserHandler(nil)

	// Set up a route without the middleware to test validation logic
	app.Put("/test/users/:id", func(c *fiber.Ctx) error {
		// Mock authDTO
		adminRole := models.RoleAdmin
		accountID := uuid.New()
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   uuid.New(),
				Role: &adminRole,
			},
			AccountID: accountID,
		}
		c.Locals("authDTO", authDTO)
		return handler.UpdateUser(c)
	})

	requestBody := map[string]interface{}{
		"firstName": "Jane",
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/test/users/invalid-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := app.Test(req, -1)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "invalid user ID", responseBody["error"])
}

func TestUpdateUser_ValidatesRole(t *testing.T) {
	// Arrange
	app := fiber.New()
	handler := NewAdminUserHandler(nil)

	// Set up a route without the middleware to test validation logic
	app.Put("/test/users/:id", func(c *fiber.Ctx) error {
		// Mock authDTO
		adminRole := models.RoleAdmin
		accountID := uuid.New()
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   uuid.New(),
				Role: &adminRole,
			},
			AccountID: accountID,
		}
		c.Locals("authDTO", authDTO)
		return handler.UpdateUser(c)
	})

	invalidRole := "superadmin"
	requestBody := map[string]interface{}{
		"role": invalidRole,
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/test/users/"+userID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := app.Test(req, -1)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "invalid role, must be 'admin' or 'user'", responseBody["error"])
}

func TestDeleteUser_ValidatesUserID(t *testing.T) {
	// Arrange
	app := fiber.New()
	handler := NewAdminUserHandler(nil)

	// Set up a route without the middleware to test validation logic
	app.Delete("/test/users/:id", func(c *fiber.Ctx) error {
		// Mock authDTO
		adminRole := models.RoleAdmin
		accountID := uuid.New()
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   uuid.New(),
				Role: &adminRole,
			},
			AccountID: accountID,
		}
		c.Locals("authDTO", authDTO)
		return handler.DeleteUser(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/test/users/invalid-uuid", nil)

	// Act
	resp, err := app.Test(req, -1)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "invalid user ID", responseBody["error"])
}

func TestAdminUserRoutes_RequireAdminMiddleware(t *testing.T) {
	// This test verifies that admin routes require admin role

	handler := NewAdminUserHandler(nil)

	// Create a regular user (non-admin)
	userRole := models.RoleUser
	accountID := uuid.New()
	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:   uuid.New(),
			Role: &userRole,
		},
		AccountID: accountID,
	}

	// Set up app with auth mock and RequireAdmin middleware
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		return c.Next()
	})

	// Apply RequireAdmin middleware to the admin group
	admin := app.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	handler.RegisterAdminUserRoutes(admin)

	requestBody := map[string]interface{}{
		"email":        "test@example.com",
		"firstName":    "John",
		"lastName":     "Doe",
		"tempPassword": "TempPass123!",
		"role":         "user",
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := app.Test(req, -1)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "Non-admin users should be forbidden from accessing admin routes")

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "Admin access required", responseBody["error"])
}
