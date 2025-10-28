package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/services"
	"log"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterAuthRoutes(app *fiber.App) {
	auth := app.Group("/auth")
	auth.Post("/register", h.Register)
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	type request struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
		Password  string `json:"password"`
	}

	var body request
	if err := c.BodyParser(&body); err != nil {
		log.Println("Failed to parse request body:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// call auth service
	user, err := h.authService.CreateUser(body.FirstName, body.LastName, body.Email, body.Password, true)
	if err != nil {
		log.Println("Failed to create user:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// return created user
	return c.Status(http.StatusCreated).JSON(user)
}
