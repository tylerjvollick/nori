package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/database"
	"github.com/tylerjvollick/nori/internal/handlers"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
)

type App struct {
	Fiber       *fiber.App
	AuthHandler *handlers.AuthHandler
	// Add other handlers here
}

func New() *App {
	// Connect to database
	database.Connect()

	// Repositories
	userRepo := repositories.NewUserRepository(database.DB)
	accountRepo := repositories.NewAccountRepository(database.DB)
	userAccountRepo := repositories.NewUserAccountRepository(database.DB)
	// Services
	authService := services.NewAuthService(userRepo, accountRepo, userAccountRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)

	// Fiber instance
	app := fiber.New()

	// Register routes
	authHandler.RegisterAuthRoutes(app)

	return &App{
		Fiber:       app,
		AuthHandler: authHandler,
	}
}
