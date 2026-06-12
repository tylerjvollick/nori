package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// CustomerRepoInterface defines the repository methods needed by CustomerHandler.
type CustomerRepoInterface interface {
	GetBySpaceID(spaceID uuid.UUID) ([]models.Customer, error)
}

// CustomerHandler handles HTTP requests for customers.
type CustomerHandler struct {
	customerRepo CustomerRepoInterface
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(customerRepo CustomerRepoInterface) *CustomerHandler {
	return &CustomerHandler{customerRepo: customerRepo}
}

// RegisterCustomerRoutes registers customer API routes under a space-scoped router.
// Expects the router to already have :spaceId in its path and RequireSpace middleware applied.
func (h *CustomerHandler) RegisterCustomerRoutes(router fiber.Router, middlewares ...fiber.Handler) {
	group := router.Group("/customers", middlewares...)
	group.Get("", h.ListCustomers)
}

// ListCustomers returns all customers for the space.
func (h *CustomerHandler) ListCustomers(c *fiber.Ctx) error {
	if _, err := requireAuth(c); err != nil {
		return err
	}

	spaceID, err := spaceIDFromPath(c)
	if err != nil {
		return err
	}

	customers, err := h.customerRepo.GetBySpaceID(spaceID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list customers",
		})
	}

	resp := make([]dtos.CustomerResponse, len(customers))
	for i, cu := range customers {
		resp[i] = dtos.CustomerResponseFromModel(&cu)
	}

	return c.JSON(resp)
}
