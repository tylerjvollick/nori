package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/tylerjvollick/nori/internal/services"
)

type TusHandler struct {
	tusService *services.TusService
}

func NewTusHandler(tusService *services.TusService) *TusHandler {
	return &TusHandler{tusService: tusService}
}

func (h *TusHandler) RegisterTusRoutes(app *fiber.App, authMiddleware fiber.Handler) {
	// Mount tus handler at /api/tus
	// The tus handler handles all HTTP methods (POST, PATCH, HEAD, OPTIONS, DELETE)
	// We need to strip /api/tus from the path before passing to TUS handler
	app.Use("/api/tus", authMiddleware, func(c *fiber.Ctx) error {
		// Strip /api/tus prefix from path for TUS handler
		c.Request().SetRequestURI(c.Path()[len("/api/tus"):])
		if c.Path() == "/api/tus" {
			c.Request().SetRequestURI("/")
		}
		return adaptor.HTTPHandlerFunc(h.tusService.GetHandler().ServeHTTP)(c)
	})
}
