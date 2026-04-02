package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

// MockSpaceMemberRepository for testing
type MockSpaceMemberRepository struct {
	mock.Mock
}

func (m *MockSpaceMemberRepository) GetByUserAndSpace(userID, spaceID uuid.UUID) (*models.SpaceMember, error) {
	args := m.Called(userID, spaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SpaceMember), args.Error(1)
}

// RequireAdmin Tests

func TestRequireAdmin_Success(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	role := models.RoleAdmin

	// Setup route with RequireAdmin middleware
	app.Use(func(c *fiber.Ctx) error {
		// Simulate auth middleware setting authDTO
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: &role,
			},
			AccountID: accountID,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRequireAdmin_Forbidden_UserRole(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	role := models.RoleUser

	// Setup route with RequireAdmin middleware
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: &role,
			},
			AccountID: accountID,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRequireAdmin_Forbidden_NilRole(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()

	// Setup route with RequireAdmin middleware
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: nil, // No role assigned
			},
			AccountID: accountID,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRequireAdmin_Unauthorized_MissingAuthDTO(t *testing.T) {
	app := fiber.New()

	// Setup route without setting authDTO
	app.Use(RequireAdmin())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRequireAdmin_Unauthorized_NilAuthDTO(t *testing.T) {
	app := fiber.New()

	// Setup route with nil authDTO
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", nil)
		return c.Next()
	})
	app.Use(RequireAdmin())
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// RequireSpaceAccess Tests

func TestRequireSpaceAccess_Success_AdminUser(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	role := models.RoleAdmin

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)
	// Admin users should not trigger repository call

	// Setup route with RequireSpaceAccess middleware
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: &role,
			},
			AccountID:     accountID,
			ActiveSpaceID: &spaceID,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/space", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/space", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify that GetByUserAndSpace was NOT called for admin
	mockSpaceMemberRepo.AssertNotCalled(t, "GetByUserAndSpace")
}

func TestRequireSpaceAccess_Success_RegularUserWithAccess(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	role := models.RoleUser

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)
	spaceMember := &models.SpaceMember{
		ID:      uuid.New(),
		UserID:  userID,
		SpaceID: spaceID,
	}
	mockSpaceMemberRepo.On("GetByUserAndSpace", userID, spaceID).Return(spaceMember, nil)

	// Setup route with RequireSpaceAccess middleware
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: &role,
			},
			AccountID:     accountID,
			ActiveSpaceID: &spaceID,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/space", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/space", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockSpaceMemberRepo.AssertExpectations(t)
}

func TestRequireSpaceAccess_Success_SpaceIDFromRouteParam(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	role := models.RoleUser

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)
	spaceMember := &models.SpaceMember{
		ID:      uuid.New(),
		UserID:  userID,
		SpaceID: spaceID,
	}
	mockSpaceMemberRepo.On("GetByUserAndSpace", userID, spaceID).Return(spaceMember, nil)

	// Setup route with spaceID route parameter
	// Apply middleware on the specific route
	app.Get("/spaces/:spaceID/data",
		func(c *fiber.Ctx) error {
			authDTO := &dtos.AuthDTO{
				User: models.User{
					ID:   userID,
					Role: &role,
				},
				AccountID:     accountID,
				ActiveSpaceID: nil, // No active space in authDTO
			}
			c.Locals("authDTO", authDTO)
			return c.Next()
		},
		RequireSpaceAccess(mockSpaceMemberRepo),
		func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

	req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID.String()+"/data", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockSpaceMemberRepo.AssertExpectations(t)
}

func TestRequireSpaceAccess_Forbidden_UserNoAccess(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	role := models.RoleUser

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)
	// Return error to simulate no access
	mockSpaceMemberRepo.On("GetByUserAndSpace", userID, spaceID).Return(nil, errors.New("not found"))

	// Setup route with RequireSpaceAccess middleware
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: &role,
			},
			AccountID:     accountID,
			ActiveSpaceID: &spaceID,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/space", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/space", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockSpaceMemberRepo.AssertExpectations(t)
}

func TestRequireSpaceAccess_Forbidden_NilRole(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)
	// User with nil role should check space membership
	mockSpaceMemberRepo.On("GetByUserAndSpace", userID, spaceID).Return(nil, errors.New("not found"))

	// Setup route with RequireSpaceAccess middleware
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: nil, // No role assigned
			},
			AccountID:     accountID,
			ActiveSpaceID: &spaceID,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/space", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/space", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	mockSpaceMemberRepo.AssertExpectations(t)
}

func TestRequireSpaceAccess_BadRequest_NoSpaceID(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	role := models.RoleUser

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup route without spaceID in authDTO or route param
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: &role,
			},
			AccountID:     accountID,
			ActiveSpaceID: nil, // No active space
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/space", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/space", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRequireSpaceAccess_BadRequest_InvalidSpaceIDFormat(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	role := models.RoleUser

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup route with invalid spaceID format
	app.Use(func(c *fiber.Ctx) error {
		authDTO := &dtos.AuthDTO{
			User: models.User{
				ID:   userID,
				Role: &role,
			},
			AccountID:     accountID,
			ActiveSpaceID: nil,
		}
		c.Locals("authDTO", authDTO)
		return c.Next()
	})
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/spaces/:spaceID/data", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/spaces/invalid-uuid/data", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRequireSpaceAccess_Unauthorized_MissingAuthDTO(t *testing.T) {
	app := fiber.New()

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup route without setting authDTO
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/space", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/space", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRequireSpaceAccess_Unauthorized_NilAuthDTO(t *testing.T) {
	app := fiber.New()

	mockSpaceMemberRepo := new(MockSpaceMemberRepository)

	// Setup route with nil authDTO
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", nil)
		return c.Next()
	})
	app.Use(RequireSpaceAccess(mockSpaceMemberRepo))
	app.Get("/space", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/space", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
