package services

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/utils"
	"gorm.io/gorm"
)

type SOPStepPhotoService struct {
	db              *gorm.DB
	photoRepo       *repositories.SOPStepPhotoRepository
	stepRepo        *repositories.SOPStepRepository
	uploadDir       string
	maxUploadSize   int64
	allowedMimeTypes map[string]bool
}

func NewSOPStepPhotoService(
	db *gorm.DB,
	photoRepo *repositories.SOPStepPhotoRepository,
	stepRepo *repositories.SOPStepRepository,
	uploadDir string,
	maxUploadSize int64,
	allowedMimeTypes []string,
) *SOPStepPhotoService {
	// Convert allowed mime types to map for fast lookup
	mimeMap := make(map[string]bool)
	for _, mimeType := range allowedMimeTypes {
		mimeMap[mimeType] = true
	}

	return &SOPStepPhotoService{
		db:               db,
		photoRepo:        photoRepo,
		stepRepo:         stepRepo,
		uploadDir:        uploadDir,
		maxUploadSize:    maxUploadSize,
		allowedMimeTypes: mimeMap,
	}
}

// UploadPhoto uploads a photo for a step
func (s *SOPStepPhotoService) UploadPhoto(stepID int, file *multipart.FileHeader) (*models.SOPStepPhoto, error) {
	// Verify step exists
	step, err := s.stepRepo.GetByID(stepID)
	if err != nil {
		return nil, fmt.Errorf("step not found: %w", err)
	}

	// Validate file size
	if file.Size > s.maxUploadSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", s.maxUploadSize)
	}

	// Validate mime type
	mimeType := file.Header.Get("Content-Type")
	if !s.allowedMimeTypes[mimeType] {
		return nil, fmt.Errorf("file type %s is not allowed", mimeType)
	}

	// Generate UUID for file
	fileUUID := uuid.New().String()

	// Get file extension
	ext := filepath.Ext(file.Filename)
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
		default:
			ext = ".bin"
		}
	}

	// Build directory path: sops/{sop_template_id}/steps/{step_id}/
	// We need to get the SOP template ID from the step's version
	// For now, let's use version ID as a proxy (we can refactor later to get template ID)
	dirPath := filepath.Join(s.uploadDir, "sops", strconv.Itoa(step.SOPTemplateVersionID), "steps", strconv.Itoa(stepID))
	
	// Create directory if not exists
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		log.Println("Failed to create directory:", err)
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Full file path
	fileName := fileUUID + ext
	fullPath := filepath.Join(dirPath, fileName)

	// Relative path for database
	relativePath := filepath.Join("sops", strconv.Itoa(step.SOPTemplateVersionID), "steps", strconv.Itoa(stepID), fileName)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		log.Println("Failed to open uploaded file:", err)
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		log.Println("Failed to create destination file:", err)
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file contents
	if _, err := io.Copy(dst, src); err != nil {
		log.Println("Failed to copy file:", err)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Get the last order for this step
	lastOrder, err := s.photoRepo.GetLastOrderByStepID(stepID)
	if err != nil {
		// Clean up file if database operation fails
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to get last order: %w", err)
	}

	// Generate new order (append to end)
	newOrder := utils.GenerateOrderBetween(lastOrder, "")

	// Create database record
	photo := &models.SOPStepPhoto{
		SOPStepID: stepID,
		UUID:      fileUUID,
		FilePath:  relativePath,
		FileName:  file.Filename,
		MimeType:  mimeType,
		FileSize:  file.Size,
		Order:     newOrder,
	}

	if err := s.photoRepo.Create(photo); err != nil {
		// Clean up file if database operation fails
		os.Remove(fullPath)
		log.Println("Failed to create photo record:", err)
		return nil, fmt.Errorf("failed to create photo record: %w", err)
	}

	return photo, nil
}

// GetPhotosByStepID gets all photos for a step
func (s *SOPStepPhotoService) GetPhotosByStepID(stepID int) ([]models.SOPStepPhoto, error) {
	return s.photoRepo.GetByStepID(stepID)
}

// GetPhotoByUUID gets a photo by its UUID
func (s *SOPStepPhotoService) GetPhotoByUUID(uuid string) (*models.SOPStepPhoto, error) {
	return s.photoRepo.GetByUUID(uuid)
}

// GetPhotoFilePath gets the full file path for a photo
func (s *SOPStepPhotoService) GetPhotoFilePath(photo *models.SOPStepPhoto) string {
	return filepath.Join(s.uploadDir, photo.FilePath)
}

// DeletePhoto deletes a photo and its file
func (s *SOPStepPhotoService) DeletePhoto(photoID int) error {
	// Get photo record
	photo, err := s.photoRepo.GetByID(photoID)
	if err != nil {
		return err
	}

	// Delete file
	fullPath := s.GetPhotoFilePath(photo)
	if err := os.Remove(fullPath); err != nil {
		// Log error but don't fail - file might already be deleted
		log.Printf("Warning: failed to delete file %s: %v", fullPath, err)
	}

	// Delete database record
	if err := s.photoRepo.Delete(photoID); err != nil {
		return fmt.Errorf("failed to delete photo record: %w", err)
	}

	return nil
}

// ReorderPhoto updates the order of a photo
func (s *SOPStepPhotoService) ReorderPhoto(photoID int, beforePhotoID, afterPhotoID *int) (*models.SOPStepPhoto, error) {
	var photo *models.SOPStepPhoto

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Get the photo
		var err error
		photo, err = s.photoRepo.GetByID(photoID)
		if err != nil {
			return err
		}

		// Get order values for before and after photos
		beforeOrder, afterOrder, err := s.photoRepo.GetOrderBeforeAndAfter(photo.SOPStepID, beforePhotoID, afterPhotoID)
		if err != nil {
			return fmt.Errorf("failed to get order bounds: %w", err)
		}

		// Generate new order value
		newOrder := utils.GenerateOrderBetween(beforeOrder, afterOrder)

		// Update the photo's order
		if err := s.photoRepo.UpdateOrderWithTx(tx, photoID, newOrder); err != nil {
			return fmt.Errorf("failed to update photo order: %w", err)
		}

		photo.Order = newOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	return photo, nil
}

// ParseAllowedMimeTypes parses a comma-separated string of mime types
func ParseAllowedMimeTypes(mimeTypesStr string) []string {
	if mimeTypesStr == "" {
		return []string{"image/jpeg", "image/png", "image/webp"}
	}

	types := strings.Split(mimeTypesStr, ",")
	result := make([]string, 0, len(types))
	for _, t := range types {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
