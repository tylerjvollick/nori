package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/services"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) RegisterUserRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/user")

	group.Get("/me", append(middlewares, h.Me)...)
}

func (h *UserHandler) Me(c *fiber.Ctx) error {
	authDTO := c.Locals("authDTO").(*dtos.AuthDTO)
	if authDTO == nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "failed to get authDTO",
		})
	}

	user, err := h.userService.GetMe(authDTO)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	return c.Status(http.StatusOK).JSON(user)
}
