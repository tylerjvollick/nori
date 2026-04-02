package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/services"
)

func TestRegisterEndpoint_ShouldBeDisabled(t *testing.T) {
	// Arrange
	app := fiber.New()

	// Create auth handler - registration should be removed from routes
	authService := &services.AuthService{}
	authHandler := NewAuthHandler(authService)
	authHandler.RegisterAuthRoutes(app)

	requestBody := map[string]string{
		"firstName": "Test",
		"lastName":  "User",
		"email":     "test@example.com",
		"password":  "password123",
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp, err := app.Test(req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Registration endpoint should not be available (should return 404)")
}

func TestLoginEndpoint_ShouldStillExist(t *testing.T) {
	// Arrange
	app := fiber.New()

	authHandler := NewAuthHandler(nil)
	authHandler.RegisterAuthRoutes(app)

	// Act - Get all registered routes
	routes := app.GetRoutes()

	// Find login route
	var loginRouteExists bool
	var registerRouteExists bool

	for _, route := range routes {
		if route.Path == "/auth/login" && route.Method == "POST" {
			loginRouteExists = true
		}
		if route.Path == "/auth/register" && route.Method == "POST" {
			registerRouteExists = true
		}
	}

	// Assert
	assert.True(t, loginRouteExists, "POST /auth/login route should be registered")
	assert.False(t, registerRouteExists, "POST /auth/register route should NOT be registered")
}
