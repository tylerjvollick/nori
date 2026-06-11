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
	customerRepo := repositories.NewCustomerRepository(database.DB)
	subTaskRepo := repositories.NewSubTaskRepository(database.DB)

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

	// Services (SpaceService created early so seed can use it for E2E)
	spaceService := services.NewSpaceService(spaceRepo, userRepo, spaceMemberRepo, stationRepo)

	// First-boot seed
	seedService := services.NewSeedService(accountRepo, userRepo, userRepo, accountRepo, userAccountRepo, userRepo, spaceService, cfg)
	if err := seedService.SeedIfNeeded(); err != nil {
		log.Fatal("Failed to seed database: " + err.Error())
	}

	// Services
	adminUserService := services.NewAdminUserService(userRepo, userAccountRepo, spaceRepo, spaceMemberRepo)
	authService := services.NewAuthService(userRepo, accountRepo, userAccountRepo, apiKeyRepo, spaceService, cfg.JWTSecret)
	timeEventRepo := repositories.NewTimeEventRepository(database.DB)
	timeEntryRepo := repositories.NewTimeEntryRepository(database.DB)
	costEntryRepo := repositories.NewCostEntryRepository(database.DB)
	taskService := services.NewTaskService(taskRepo, taskDepRepo, timeEventRepo)
	readyWorkService := services.NewReadyWorkService(database.DB)
	recipeService := services.NewRecipeService(database.DB, recipeRepo, taskRepo, taskDepRepo)
	costService := services.NewCostService(costEntryRepo, timeEventRepo, taskRepo, spaceRepo, stationRepo)
	timeEntryService := services.NewTimeEntryService(timeEntryRepo, taskRepo)
	taskService.SetTimeEntryStopper(timeEntryService)
	productService := services.NewProductService(database.DB, recipeRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, spaceMemberRepo, spaceRepo)
	spaceHandler := handlers.NewSpaceHandler(spaceService, spaceMemberRepo)
	taskHandler := handlers.NewTaskHandler(taskService, readyWorkService)
	adminUserHandler := handlers.NewAdminUserHandler(adminUserService)
	adminAPIKeyHandler := handlers.NewAdminAPIKeyHandler(authService, apiKeyRepo)
	adminSpaceMemberHandler := handlers.NewAdminSpaceMemberHandler(spaceMemberRepo, spaceRepo)
	recipeHandler := handlers.NewRecipeHandler(recipeRepo, recipeService, recipeService, recipeService)
	stationHandler := handlers.NewStationHandler(stationRepo)
	costHandler := handlers.NewCostHandler(costService)
	jobHandler := handlers.NewJobHandler(taskService, costService, recipeService)
	taskDepHandler := handlers.NewTaskDepHandler(taskDepRepo, taskService)
	customerHandler := handlers.NewCustomerHandler(customerRepo)
	subTaskHandler := handlers.NewSubTaskHandler(subTaskRepo, taskService)
	timeEntryHandler := handlers.NewTimeEntryHandler(timeEntryService)
	productHandler := handlers.NewProductHandler(productService)

	// Fiber instance with CORS and increased body limit for media uploads
	app := fiber.New(fiber.Config{
		BodyLimit: int(maxUploadSize), // Use the configured max upload size
	})

	// Add CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173,http://localhost:5174",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Content-Length, X-Requested-With",
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

	// ── Space-scoped routes (/api/v1/spaces/:spaceId/...) ─────────────
	spaceScoped := app.Group("/api/v1/spaces/:spaceId", authMiddleware, requirePasswordChanged, middleware.RequireSpace(spaceMemberRepo))
	stationHandler.RegisterStationRoutes(spaceScoped)
	costHandler.RegisterCostRoutes(spaceScoped)
	customerHandler.RegisterCustomerRoutes(spaceScoped)
	recipeHandler.RegisterRecipeRoutes(spaceScoped)
	recipeHandler.RegisterRecipeVersionRoutes(spaceScoped)
	jobHandler.RegisterJobRoutes(spaceScoped)
	taskHandler.RegisterTaskRoutes(spaceScoped)
	taskDepHandler.RegisterTaskDepRoutes(spaceScoped)
	subTaskHandler.RegisterSubTaskRoutes(spaceScoped)
	timeEntryHandler.RegisterTimeEntryRoutes(spaceScoped)
	productHandler.RegisterProductRoutes(spaceScoped)

	// ── Admin routes (auth + password changed + admin role) ────────────
	admin := app.Group("/admin", authMiddleware, requirePasswordChanged, middleware.RequireAdmin())
	adminUserHandler.RegisterAdminUserRoutes(admin)
	adminAPIKeyHandler.RegisterAdminAPIKeyRoutes(admin)
	adminSpaceMemberHandler.RegisterAdminSpaceMemberRoutes(admin)

	// ── Dev-only test routes (only registered when NORI_ENV=development) ──
	if os.Getenv("NORI_ENV") == "development" {
		testHandler := handlers.NewTestHandler(database.DB)
		testHandler.RegisterTestRoutes(app, authMiddleware, requirePasswordChanged)
		log.Println("Test routes registered (NORI_ENV=development)")
	}

	return &App{
		Fiber:       app,
		AuthHandler: authHandler,
		Config:      cfg,
	}
}
