package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/tylerjvollick/nori/internal/services"
)

type SOPStepMediaHandler struct {
	mediaService *services.SOPStepMediaService
}

func NewSOPStepMediaHandler(mediaService *services.SOPStepMediaService) *SOPStepMediaHandler {
	return &SOPStepMediaHandler{mediaService: mediaService}
}

func (h *SOPStepMediaHandler) RegisterMediaRoutes(app *fiber.App) {
	// Media routes under /sops/:id/steps/:stepId/media
	app.Post("/sops/:id/steps/:stepId/media", h.UploadMedia)
	app.Get("/sops/:id/steps/:stepId/media", h.GetStepMedia)
	app.Delete("/media/:mediaId", h.DeleteMedia)
	app.Patch("/media/:mediaId/reorder", h.ReorderMedia)

	// Serve media files
	app.Get("/media/:uuid", h.ServeMedia)

	// Backwards compatibility: keep /photos routes
	app.Post("/sops/:id/steps/:stepId/photos", h.UploadMedia)
	app.Get("/sops/:id/steps/:stepId/photos", h.GetStepMedia)
	app.Delete("/photos/:photoId", h.DeleteMedia)
	app.Patch("/photos/:photoId/reorder", h.ReorderMedia)
	app.Get("/photos/:uuid", h.ServeMedia)
}

// UploadMedia handles media upload for a step (photo or video)
func (h *SOPStepMediaHandler) UploadMedia(c *fiber.Ctx) error {
	stepID, err := strconv.Atoi(c.Params("stepId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid step id",
		})
	}

	// Get uploaded file - try "media" first, then "photo" for backwards compatibility
	file, err := c.FormFile("media")
	if err != nil {
		file, err = c.FormFile("photo")
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "no file uploaded",
			})
		}
	}

	// Upload media
	media, err := h.mediaService.UploadMedia(stepID, file)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":       media.ID,
		"uuid":     media.UUID,
		"fileName": media.FileName,
		"mimeType": media.MimeType,
		"fileSize": media.FileSize,
		"duration": media.Duration,
		"order":    media.Order,
		"url":      "/media/" + media.UUID,
	})
}

// GetStepMedia gets all media for a step
func (h *SOPStepMediaHandler) GetStepMedia(c *fiber.Ctx) error {
	stepID, err := strconv.Atoi(c.Params("stepId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid step id",
		})
	}

	mediaItems, err := h.mediaService.GetMediaByStepID(stepID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Convert to response format
	response := make([]fiber.Map, len(mediaItems))
	for i, media := range mediaItems {
		response[i] = fiber.Map{
			"id":       media.ID,
			"uuid":     media.UUID,
			"fileName": media.FileName,
			"mimeType": media.MimeType,
			"fileSize": media.FileSize,
			"duration": media.Duration,
			"order":    media.Order,
			"url":      "/media/" + media.UUID,
		}
	}

	return c.Status(http.StatusOK).JSON(response)
}

// ServeMedia serves a media file by UUID
func (h *SOPStepMediaHandler) ServeMedia(c *fiber.Ctx) error {
	uuid := c.Params("uuid")

	media, err := h.mediaService.GetMediaByUUID(uuid)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "media not found",
		})
	}

	// Get full file path
	filePath := h.mediaService.GetMediaFilePath(media)

	// Set content type
	c.Set("Content-Type", media.MimeType)
	c.Set("Content-Disposition", "inline; filename=\""+media.FileName+"\"")

	return c.SendFile(filePath)
}

// DeleteMedia deletes media
func (h *SOPStepMediaHandler) DeleteMedia(c *fiber.Ctx) error {
	// Support both mediaId and photoId params for backwards compatibility
	mediaIDStr := c.Params("mediaId")
	if mediaIDStr == "" {
		mediaIDStr = c.Params("photoId")
	}

	mediaID, err := strconv.Atoi(mediaIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid media id",
		})
	}

	if err := h.mediaService.DeleteMedia(mediaID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusNoContent).Send(nil)
}

// ReorderMedia reorders media
func (h *SOPStepMediaHandler) ReorderMedia(c *fiber.Ctx) error {
	// Support both mediaId and photoId params for backwards compatibility
	mediaIDStr := c.Params("mediaId")
	if mediaIDStr == "" {
		mediaIDStr = c.Params("photoId")
	}

	mediaID, err := strconv.Atoi(mediaIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid media id",
		})
	}

	// Parse request body - support both new and old field names
	var body struct {
		BeforeMediaID *int `json:"beforeMediaId"`
		AfterMediaID  *int `json:"afterMediaId"`
		BeforePhotoID *int `json:"beforePhotoId"` // Backwards compatibility
		AfterPhotoID  *int `json:"afterPhotoId"`  // Backwards compatibility
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Use new field names, fallback to old ones
	beforeID := body.BeforeMediaID
	if beforeID == nil {
		beforeID = body.BeforePhotoID
	}
	afterID := body.AfterMediaID
	if afterID == nil {
		afterID = body.AfterPhotoID
	}

	media, err := h.mediaService.ReorderMedia(mediaID, beforeID, afterID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"id":       media.ID,
		"uuid":     media.UUID,
		"fileName": media.FileName,
		"mimeType": media.MimeType,
		"fileSize": media.FileSize,
		"duration": media.Duration,
		"order":    media.Order,
		"url":      "/media/" + media.UUID,
	})
}
