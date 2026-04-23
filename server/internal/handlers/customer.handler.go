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

// RegisterCustomerRoutes registers customer API routes on the Fiber app.
func (h *CustomerHandler) RegisterCustomerRoutes(app *fiber.App, middlewares ...fiber.Handler) {
	group := app.Group("/api/v1/customers", middlewares...)
	group.Get("", h.ListCustomers)
}

// ListCustomers returns all customers for the active space.
func (h *CustomerHandler) ListCustomers(c *fiber.Ctx) error {
	authDTO, err := requireAuth(c)
	if err != nil {
		return err
	}

	if authDTO.ActiveSpaceID == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "X-Space-ID header is required",
		})
	}

	customers, err := h.customerRepo.GetBySpaceID(*authDTO.ActiveSpaceID)
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
