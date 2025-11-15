package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/tus/tusd/v2/pkg/filestore"
	"github.com/tus/tusd/v2/pkg/handler"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

type TusService struct {
	db               *gorm.DB
	mediaRepo        *repositories.SOPStepMediaRepository
	stepRepo         *repositories.SOPStepRepository
	handler          *handler.Handler
	uploadDir        string
	tusUploadDir     string
	maxUploadSize    int64
	allowedMimeTypes map[string]bool
}

func NewTusService(
	db *gorm.DB,
	mediaRepo *repositories.SOPStepMediaRepository,
	stepRepo *repositories.SOPStepRepository,
	uploadDir string,
	maxUploadSize int64,
	allowedMimeTypes []string,
) (*TusService, error) {
	// Convert allowed mime types to map for fast lookup
	mimeMap := make(map[string]bool)
	for _, mimeType := range allowedMimeTypes {
		mimeMap[mimeType] = true
	}

	// Create tus upload directory (separate from final storage)
	tusUploadDir := filepath.Join(uploadDir, "tus-uploads")
	if err := os.MkdirAll(tusUploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create tus upload directory: %w", err)
	}

	// Create file store for tus
	store := filestore.New(tusUploadDir)

	// Create tus handler
	composer := handler.NewStoreComposer()
	store.UseIn(composer)

	tusHandler, err := handler.NewHandler(handler.Config{
		BasePath:                "/api/tus/",
		StoreComposer:           composer,
		MaxSize:                 maxUploadSize,
		NotifyCompleteUploads:   true,
		RespectForwardedHeaders: true,
		Cors: &handler.CorsConfig{
			Disable: true, // Disable TUS built-in CORS, let Fiber handle it
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create tus handler: %w", err)
	}

	service := &TusService{
		db:               db,
		mediaRepo:        mediaRepo,
		stepRepo:         stepRepo,
		handler:          tusHandler,
		uploadDir:        uploadDir,
		tusUploadDir:     tusUploadDir,
		maxUploadSize:    maxUploadSize,
		allowedMimeTypes: mimeMap,
	}

	// Setup completion handler
	go service.handleUploadCompletions()

	return service, nil
}

// GetHandler returns the tus handler for use in routing
func (s *TusService) GetHandler() *handler.Handler {
	return s.handler
}

// handleUploadCompletions processes completed uploads
func (s *TusService) handleUploadCompletions() {
	for {
		event := <-s.handler.CompleteUploads
		go s.processCompletedUpload(event)
	}
}

// processCompletedUpload moves the uploaded file to permanent storage and creates DB record
func (s *TusService) processCompletedUpload(event handler.HookEvent) {
	info := event.Upload

	// Extract metadata
	metadata := info.MetaData
	stepIDStr, ok := metadata["stepId"]
	if !ok {
		log.Printf("Upload %s missing stepId metadata", info.ID)
		return
	}

	stepID, err := strconv.Atoi(stepIDStr)
	if err != nil {
		log.Printf("Invalid stepId %s: %v", stepIDStr, err)
		return
	}

	fileName, ok := metadata["filename"]
	if !ok {
		fileName = "upload"
	}

	mimeType, ok := metadata["filetype"]
	if !ok {
		mimeType = "application/octet-stream"
	}

	// Validate mime type
	if !s.allowedMimeTypes[mimeType] {
		log.Printf("Upload %s has disallowed mime type: %s", info.ID, mimeType)
		return
	}

	// Verify step exists
	step, err := s.stepRepo.GetByID(stepID)
	if err != nil {
		log.Printf("Step %d not found: %v", stepID, err)
		return
	}

	// Generate UUID for permanent storage
	fileUUID := uuid.New().String()

	// Get file extension
	ext := filepath.Ext(fileName)
	if ext == "" {
		// Infer from mime type
		switch mimeType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		case "video/mp4":
			ext = ".mp4"
		case "video/quicktime":
			ext = ".mov"
		case "video/webm":
			ext = ".webm"
		default:
			ext = ".bin"
		}
	}

	// Build permanent storage path
	dirPath := filepath.Join(s.uploadDir, "sops", strconv.Itoa(step.SOPTemplateVersionID), "steps", strconv.Itoa(stepID))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Printf("Failed to create directory %s: %v", dirPath, err)
		return
	}

	permanentFileName := fileUUID + ext
	permanentPath := filepath.Join(dirPath, permanentFileName)
	relativePath := filepath.Join("sops", strconv.Itoa(step.SOPTemplateVersionID), "steps", strconv.Itoa(stepID), permanentFileName)

	// Get the temporary tus file
	tusFilePath := filepath.Join(s.tusUploadDir, info.ID)
	infoFilePath := tusFilePath + ".info"

	// Move file to permanent storage
	if err := os.Rename(tusFilePath, permanentPath); err != nil {
		// If rename fails (different filesystem), try copy
		if err := copyFile(tusFilePath, permanentPath); err != nil {
			log.Printf("Failed to move file from %s to %s: %v", tusFilePath, permanentPath, err)
			return
		}
		os.Remove(tusFilePath)
	}

	// Clean up tus metadata file
	os.Remove(infoFilePath)

	// Get last order for this step
	lastOrder, err := s.mediaRepo.GetLastOrderByStepID(stepID)
	if err != nil {
		log.Printf("Failed to get last order: %v", err)
		os.Remove(permanentPath)
		return
	}

	// Import utils for ordering
	newOrder := generateOrderBetween(lastOrder, "")

	// Create database record
	media := &models.SOPStepMedia{
		SOPStepID: stepID,
		UUID:      fileUUID,
		FilePath:  relativePath,
		FileName:  fileName,
		MimeType:  mimeType,
		FileSize:  info.Size,
		Order:     newOrder,
	}

	if err := s.mediaRepo.Create(media); err != nil {
		log.Printf("Failed to create media record: %v", err)
		os.Remove(permanentPath)
		return
	}

	log.Printf("Successfully processed upload %s for step %d", info.ID, stepID)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// generateOrderBetween generates a lexicographically ordered string between two bounds
// This is duplicated from utils package to avoid import cycle
func generateOrderBetween(before, after string) string {
	// Simple implementation - in production use the utils version
	if before == "" && after == "" {
		return "m"
	}
	if before == "" {
		if len(after) == 0 {
			return "m"
		}
		firstChar := after[0]
		if firstChar > 'a' {
			return string(firstChar - 1)
		}
		return "a" + after
	}
	if after == "" {
		if len(before) == 0 {
			return "m"
		}
		lastChar := before[len(before)-1]
		if lastChar < 'z' {
			return before[:len(before)-1] + string(lastChar+1)
		}
		return before + "m"
	}
	
	// Find midpoint
	minLen := len(before)
	if len(after) < minLen {
		minLen = len(after)
	}
	
	for i := 0; i < minLen; i++ {
		if before[i] != after[i] {
			mid := (int(before[i]) + int(after[i])) / 2
			if mid > int(before[i]) {
				return before[:i] + string(rune(mid))
			}
			return before[:i+1] + "m"
		}
	}
	
	return before + "m"
}
