package app

import (
	"log"
	"os"
	"strconv"

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
	sopTemplateRepo := repositories.NewSOPTemplateRepository(database.DB)
	sopVersionRepo := repositories.NewSOPVersionRepository(database.DB)
	sopStepRepo := repositories.NewSOPStepRepository(database.DB)
	sopSubStepRepo := repositories.NewSOPSubStepRepository(database.DB)
	_ = sopSubStepRepo // TODO: wire into service when SOPSubStep CRUD endpoints are added
	bomItemRepo := repositories.NewBOMItemRepository(database.DB)
	_ = bomItemRepo // TODO: wire into service when BOMItem CRUD endpoints are added
	sopStepMediaRepo := repositories.NewSOPStepMediaRepository(database.DB)
	spaceRepo := repositories.NewSpaceRepository(database.DB)
	sopCategoryRepo := repositories.NewSOPCategoryRepository(database.DB)
	ticketTypeRepo := repositories.NewTicketTypeRepository(database.DB)
	statusDefRepo := repositories.NewStatusDefinitionRepository(database.DB)

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

	allowedMimeTypesStr := os.Getenv("ALLOWED_MIME_TYPES")
	allowedMimeTypes := services.ParseAllowedMimeTypes(allowedMimeTypesStr)

	// Services
	userService := services.NewUserService(userRepo)
	sopService := services.NewSOPService(database.DB, sopTemplateRepo, sopVersionRepo, sopStepRepo)
	sopMediaService := services.NewSOPStepMediaService(database.DB, sopStepMediaRepo, sopStepRepo, uploadDir, maxUploadSize, allowedMimeTypes)
	spaceService := services.NewSpaceService(spaceRepo, userRepo, services.NewSpaceTemplateService(ticketTypeRepo, statusDefRepo, sopCategoryRepo))
	authService := services.NewAuthService(userRepo, accountRepo, userAccountRepo, spaceService)

	// Initialize tus service for chunked uploads
	tusService, err := services.NewTusService(database.DB, sopStepMediaRepo, sopStepRepo, uploadDir, maxUploadSize, allowedMimeTypes)
	if err != nil {
		log.Fatal("Failed to initialize tus service: " + err.Error())
	}

	// Handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	sopHandler := handlers.NewSOPHandler(sopService)
	sopMediaHandler := handlers.NewSOPStepMediaHandler(sopMediaService)
	tusHandler := handlers.NewTusHandler(tusService)
	spaceHandler := handlers.NewSpaceHandler(spaceService)

	// Fiber instance with CORS and increased body limit for media uploads
	app := fiber.New(fiber.Config{
		BodyLimit: int(maxUploadSize), // Use the configured max upload size
	})

	// Add CORS middleware with TUS-specific headers
	// TUS built-in CORS is disabled, so Fiber handles all CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173,http://localhost:5174",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, Content-Length, " +
			"Tus-Resumable, Upload-Length, Upload-Offset, Upload-Metadata, " +
			"Upload-Defer-Length, Upload-Concat, X-Requested-With",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD",
		ExposeHeaders: "Tus-Resumable, Tus-Version, Tus-Extension, " +
			"Upload-Offset, Upload-Length, Upload-Metadata, Location, Content-Length",
		AllowCredentials: true,
	}))

	// Register routes
	handlers.RegisterHealthRoutes(app, nil)
	authHandler.RegisterAuthRoutes(app)
	userHandler.RegisterUserRoutes(app)
	sopHandler.RegisterSOPRoutes(app)
	sopMediaHandler.RegisterMediaRoutes(app)
	tusHandler.RegisterTusRoutes(app)
	spaceHandler.RegisterSpaceRoutes(app)

	return &App{
		Fiber:       app,
		AuthHandler: authHandler,
	}
}
