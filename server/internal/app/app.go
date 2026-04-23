package app

import (
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/tylerjvollick/nori/internal/auth"
	"github.com/tylerjvollick/nori/internal/config"
	"github.com/tylerjvollick/nori/internal/database"
	"github.com/tylerjvollick/nori/internal/handlers"
	"github.com/tylerjvollick/nori/internal/middleware"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
)

type App struct {
	Fiber       *fiber.App
	AuthHandler *handlers.AuthHandler
	Config      *config.Config
	// Add other handlers here
}

func New(cfg *config.Config) *App {
	// Connect to database
	database.Connect()

	// Repositories
	userRepo := repositories.NewUserRepository(database.DB)
	accountRepo := repositories.NewAccountRepository(database.DB)
	userAccountRepo := repositories.NewUserAccountRepository(database.DB)
	apiKeyRepo := repositories.NewAPIKeyRepository(database.DB)
	spaceRepo := repositories.NewSpaceRepository(database.DB)
	spaceMemberRepo := repositories.NewSpaceMemberRepository(database.DB)
	taskRepo := repositories.NewTaskRepository(database.DB)
	taskDepRepo := repositories.NewTaskDepRepository(database.DB)
	recipeRepo := repositories.NewRecipeRepository(database.DB)
	stationRepo := repositories.NewStationRepository(database.DB)

	// Get photo upload configuration from environment
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	maxUploadSizeStr := os.Getenv("MAX_UPLOAD_SIZE")
	maxUploadSize := int64(10485760) // Default 10MB
	if maxUploadSizeStr != "" {
		if size, err := strconv.ParseInt(maxUploadSizeStr, 10, 64); err == nil {
			maxUploadSize = size
		}
	}

	// First-boot seed
	seedService := services.NewSeedService(accountRepo, userRepo, userRepo, accountRepo, userAccountRepo, cfg)
	if err := seedService.SeedIfNeeded(); err != nil {
		log.Fatal("Failed to seed database: " + err.Error())
	}

	// Services
	adminUserService := services.NewAdminUserService(userRepo, userAccountRepo, spaceRepo, spaceMemberRepo)
	spaceService := services.NewSpaceService(spaceRepo, userRepo, spaceMemberRepo, stationRepo)
	authService := services.NewAuthService(userRepo, accountRepo, userAccountRepo, apiKeyRepo, spaceService, cfg.JWTSecret)
	timeEventRepo := repositories.NewTimeEventRepository(database.DB)
	taskService := services.NewTaskService(taskRepo, taskDepRepo, timeEventRepo)
	readyWorkService := services.NewReadyWorkService(database.DB)
	recipeService := services.NewRecipeService(database.DB, recipeRepo, taskRepo, taskDepRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, spaceMemberRepo, spaceRepo)
	spaceHandler := handlers.NewSpaceHandler(spaceService, spaceMemberRepo)
	taskHandler := handlers.NewTaskHandler(taskService, readyWorkService)
	adminUserHandler := handlers.NewAdminUserHandler(adminUserService)
	adminAPIKeyHandler := handlers.NewAdminAPIKeyHandler(authService, apiKeyRepo)
	adminSpaceMemberHandler := handlers.NewAdminSpaceMemberHandler(spaceMemberRepo, spaceRepo)
	recipeHandler := handlers.NewRecipeHandler(recipeRepo, recipeService)
	stationHandler := handlers.NewStationHandler(stationRepo)
	taskDepHandler := handlers.NewTaskDepHandler(taskDepRepo, taskService)

	// Fiber instance with CORS and increased body limit for media uploads
	app := fiber.New(fiber.Config{
		BodyLimit: int(maxUploadSize), // Use the configured max upload size
	})

	// Add CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173,http://localhost:5174",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Content-Length, X-Requested-With, X-Space-ID",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD",
		AllowCredentials: true,
	}))

	// Middleware
	authMiddleware := auth.NewAuthMiddleware(userRepo, apiKeyRepo, spaceMemberRepo, cfg.JWTSecret)
	requirePasswordChanged := middleware.RequirePasswordChanged()

	// ── Public routes (no auth) ────────────────────────────────────────
	handlers.RegisterHealthRoutes(app)
	authHandler.RegisterAuthRoutes(app)

	// ── Auth-only routes (no RequirePasswordChanged) ───────────────────
	// /auth/change-password, /auth/logout, /auth/me need auth but must
	// remain accessible to users who still need to change their password.
	authHandler.RegisterProtectedAuthRoutes(app, authMiddleware)

	// ── Fully guarded routes (auth + password changed) ─────────────────
	spaceHandler.RegisterSpaceRoutes(app, authMiddleware, requirePasswordChanged)
	taskHandler.RegisterTaskRoutes(app, authMiddleware, requirePasswordChanged)
	recipeHandler.RegisterRecipeRoutes(app, authMiddleware, requirePasswordChanged)
	recipeHandler.RegisterRecipeVersionRoutes(app, authMiddleware, requirePasswordChanged)
	stationHandler.RegisterStationRoutes(app, authMiddleware, requirePasswordChanged)
	taskDepHandler.RegisterTaskDepRoutes(app, authMiddleware, requirePasswordChanged)

	// ── Admin routes (auth + password changed + admin role) ────────────
	admin := app.Group("/admin", authMiddleware, requirePasswordChanged, middleware.RequireAdmin())
	adminUserHandler.RegisterAdminUserRoutes(admin)
	adminAPIKeyHandler.RegisterAdminAPIKeyRoutes(admin)
	adminSpaceMemberHandler.RegisterAdminSpaceMemberRoutes(admin)

	return &App{
		Fiber:       app,
		AuthHandler: authHandler,
		Config:      cfg,
	}
}
