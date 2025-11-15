package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/services"
)

type SOPStepPhotoHandler struct {
	photoService *services.SOPStepPhotoService
}

func NewSOPStepPhotoHandler(photoService *services.SOPStepPhotoService) *SOPStepPhotoHandler {
	return &SOPStepPhotoHandler{photoService: photoService}
}

func (h *SOPStepPhotoHandler) RegisterPhotoRoutes(app *fiber.App) {
	// Photo routes under /sops/:id/steps/:stepId/photos
	app.Post("/sops/:id/steps/:stepId/photos", h.UploadPhoto)
	app.Get("/sops/:id/steps/:stepId/photos", h.GetStepPhotos)
	app.Delete("/photos/:photoId", h.DeletePhoto)
	app.Patch("/photos/:photoId/reorder", h.ReorderPhoto)
	
	// Serve photo files
	app.Get("/photos/:uuid", h.ServePhoto)
}

// UploadPhoto handles photo upload for a step
func (h *SOPStepPhotoHandler) UploadPhoto(c *fiber.Ctx) error {
	stepID, err := strconv.Atoi(c.Params("stepId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid step id",
		})
	}

	// Get uploaded file
	file, err := c.FormFile("photo")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "no file uploaded",
		})
	}

	// Upload photo
	photo, err := h.photoService.UploadPhoto(stepID, file)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":       photo.ID,
		"uuid":     photo.UUID,
		"fileName": photo.FileName,
		"mimeType": photo.MimeType,
		"fileSize": photo.FileSize,
		"order":    photo.Order,
		"url":      "/photos/" + photo.UUID,
	})
}

// GetStepPhotos gets all photos for a step
func (h *SOPStepPhotoHandler) GetStepPhotos(c *fiber.Ctx) error {
	stepID, err := strconv.Atoi(c.Params("stepId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid step id",
		})
	}

	photos, err := h.photoService.GetPhotosByStepID(stepID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response format
	response := make([]fiber.Map, len(photos))
	for i, photo := range photos {
		response[i] = fiber.Map{
			"id":       photo.ID,
			"uuid":     photo.UUID,
			"fileName": photo.FileName,
			"mimeType": photo.MimeType,
			"fileSize": photo.FileSize,
			"order":    photo.Order,
			"url":      "/photos/" + photo.UUID,
		}
	}

	return c.Status(http.StatusOK).JSON(response)
}

// ServePhoto serves a photo file by UUID
func (h *SOPStepPhotoHandler) ServePhoto(c *fiber.Ctx) error {
	uuid := c.Params("uuid")

	photo, err := h.photoService.GetPhotoByUUID(uuid)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "photo not found",
		})
	}

	// Get full file path
	filePath := h.photoService.GetPhotoFilePath(photo)

	// Set content type
	c.Set("Content-Type", photo.MimeType)
	c.Set("Content-Disposition", "inline; filename=\""+photo.FileName+"\"")

	return c.SendFile(filePath)
}

// DeletePhoto deletes a photo
func (h *SOPStepPhotoHandler) DeletePhoto(c *fiber.Ctx) error {
	photoID, err := strconv.Atoi(c.Params("photoId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid photo id",
		})
	}

	if err := h.photoService.DeletePhoto(photoID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// ReorderPhoto reorders a photo
func (h *SOPStepPhotoHandler) ReorderPhoto(c *fiber.Ctx) error {
	photoID, err := strconv.Atoi(c.Params("photoId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid photo id",
		})
	}

	// Parse request body
	var body struct {
		BeforePhotoID *int `json:"beforePhotoId"`
		AfterPhotoID  *int `json:"afterPhotoId"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	photo, err := h.photoService.ReorderPhoto(photoID, body.BeforePhotoID, body.AfterPhotoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"id":       photo.ID,
		"uuid":     photo.UUID,
		"fileName": photo.FileName,
		"mimeType": photo.MimeType,
		"fileSize": photo.FileSize,
		"order":    photo.Order,
		"url":      "/photos/" + photo.UUID,
	})
}
