package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

func RegisterAuthRoutes(app *fiber.App, conn *pgx.Conn) {
	app.Post("/auth/register", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
		 
	})
} 
