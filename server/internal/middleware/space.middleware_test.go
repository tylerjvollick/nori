package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
)

func TestRequireSpace_Success_Admin(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	role := models.RoleAdmin

	mockRepo := new(MockSpaceMemberRepository)

	app.Get("/spaces/:spaceId/data",
		func(c *fiber.Ctx) error {
			authDTO := &dtos.AuthDTO{
				User:      models.User{ID: userID, Role: &role},
				AccountID: accountID,
			}
			c.Locals("authDTO", authDTO)
			return c.Next()
		},
		RequireSpace(mockRepo),
		func(c *fiber.Ctx) error {
			id, ok := c.Locals("spaceID").(uuid.UUID)
			assert.True(t, ok)
			assert.Equal(t, spaceID, id)
			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID.String()+"/data", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockRepo.AssertNotCalled(t, "GetByUserAndSpace")
}

func TestRequireSpace_Success_Member(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	role := models.RoleUser

	mockRepo := new(MockSpaceMemberRepository)
	mockRepo.On("GetByUserAndSpace", userID, spaceID).Return(&models.SpaceMember{
		ID: uuid.New(), UserID: userID, SpaceID: spaceID,
	}, nil)

	app.Get("/spaces/:spaceId/data",
		func(c *fiber.Ctx) error {
			authDTO := &dtos.AuthDTO{
				User:      models.User{ID: userID, Role: &role},
				AccountID: accountID,
			}
			c.Locals("authDTO", authDTO)
			return c.Next()
		},
		RequireSpace(mockRepo),
		func(c *fiber.Ctx) error {
			id, ok := c.Locals("spaceID").(uuid.UUID)
			assert.True(t, ok)
			assert.Equal(t, spaceID, id)
			return c.SendStatus(fiber.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID.String()+"/data", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockRepo.AssertExpectations(t)
}

func TestRequireSpace_Forbidden_NotMember(t *testing.T) {
	app := fiber.New()

	userID := uuid.New()
	accountID := uuid.New()
	spaceID := uuid.New()
	role := models.RoleUser

	mockRepo := new(MockSpaceMemberRepository)
	mockRepo.On("GetByUserAndSpace", userID, spaceID).Return(nil, errors.New("not found"))

	app.Get("/spaces/:spaceId/data",
		func(c *fiber.Ctx) error {
			authDTO := &dtos.AuthDTO{
				User:      models.User{ID: userID, Role: &role},
				AccountID: accountID,
			}
			c.Locals("authDTO", authDTO)
			return c.Next()
		},
		RequireSpace(mockRepo),
		func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) },
	)

	req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID.String()+"/data", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	mockRepo.AssertExpectations(t)
}

func TestRequireSpace_BadRequest_InvalidUUID(t *testing.T) {
	app := fiber.New()

	role := models.RoleUser
	mockRepo := new(MockSpaceMemberRepository)

	app.Get("/spaces/:spaceId/data",
		func(c *fiber.Ctx) error {
			authDTO := &dtos.AuthDTO{
				User:      models.User{ID: uuid.New(), Role: &role},
				AccountID: uuid.New(),
			}
			c.Locals("authDTO", authDTO)
			return c.Next()
		},
		RequireSpace(mockRepo),
		func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) },
	)

	req := httptest.NewRequest(http.MethodGet, "/spaces/not-a-uuid/data", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRequireSpace_Unauthorized_MissingAuthDTO(t *testing.T) {
	app := fiber.New()

	mockRepo := new(MockSpaceMemberRepository)
	spaceID := uuid.New()

	app.Get("/spaces/:spaceId/data",
		RequireSpace(mockRepo),
		func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) },
	)

	req := httptest.NewRequest(http.MethodGet, "/spaces/"+spaceID.String()+"/data", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
