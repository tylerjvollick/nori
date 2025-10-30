package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
	userService := services.NewUserService(userRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)

	// Fiber instance with CORS
	app := fiber.New()

	// Add CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173,http://localhost:5174",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Register routes
	handlers.RegisterHealthRoutes(app, nil)
	authHandler.RegisterAuthRoutes(app)
	userHandler.RegisterUserRoutes(app)

	return &App{
		Fiber:       app,
		AuthHandler: authHandler,
	}
}
