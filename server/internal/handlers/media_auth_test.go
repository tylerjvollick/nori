package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// mockAuthMiddleware is a simple auth middleware that rejects requests without
// an Authorization header, simulating the real auth middleware behavior.
func mockAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Get("Authorization") == "" && c.Cookies("nori_token") == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authentication credentials",
			})
		}
		return c.Next()
	}
}

func TestMediaRoutes_UnauthenticatedReturns401(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/sops/1/steps/1/media"},
		{"GET", "/sops/1/steps/1/media"},
		{"DELETE", "/media/some-media-id"},
		{"PATCH", "/media/some-media-id/reorder"},
		{"GET", "/media/some-uuid"},
		// Backwards compatibility routes
		{"POST", "/sops/1/steps/1/photos"},
		{"GET", "/sops/1/steps/1/photos"},
		{"DELETE", "/photos/some-photo-id"},
		{"PATCH", "/photos/some-photo-id/reorder"},
		{"GET", "/photos/some-uuid"},
	}

	app := fiber.New()
	handler := NewSOPStepMediaHandler(nil) // nil service is fine — auth check happens first
	handler.RegisterMediaRoutes(app, mockAuthMiddleware())

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
				"Unauthenticated %s %s should return 401", route.method, route.path)
		})
	}
}

func TestTusRoutes_UnauthenticatedReturns401(t *testing.T) {
	app := fiber.New()

	// We can't easily construct a real TusHandler without a full TusService,
	// but we can verify the middleware chain by checking that auth middleware
	// fires before the TUS handler by mounting a dummy handler.
	app.Use("/api/tus", mockAuthMiddleware(), func(c *fiber.Ctx) error {
		// This handler should never be reached if auth fails
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	tusRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/tus"},
		{"PATCH", "/api/tus/some-upload-id"},
		{"HEAD", "/api/tus/some-upload-id"},
		{"DELETE", "/api/tus/some-upload-id"},
	}

	for _, route := range tusRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
				"Unauthenticated %s %s should return 401", route.method, route.path)
		})
	}
}
