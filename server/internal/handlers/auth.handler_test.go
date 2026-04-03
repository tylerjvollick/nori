package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo implements services.UserRepositoryInterface for handler tests.
type mockUserRepo struct {
	getUserByEmailFunc          func(string) (*models.User, error)
	getUserByIDFunc             func(uuid.UUID) (*models.User, error)
	updatePasswordFunc          func(uuid.UUID, string) error
	clearMustChangePasswordFunc func(uuid.UUID) error
}

func (m *mockUserRepo) GetUserByID(id uuid.UUID) (*models.User, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(id)
	}
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) GetUserByEmail(email string) (*models.User, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(email)
	}
	return nil, errors.New("user not found")
}
func (m *mockUserRepo) CreateUser(*models.User) error {
	return errors.New("not implemented")
}
func (m *mockUserRepo) UpdateUser(uuid.UUID, *repositories.UpdateUserInput) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) UpdateRecentSpaces(uuid.UUID, models.RecentSpaces) error {
	return errors.New("not implemented")
}
func (m *mockUserRepo) UpdatePassword(id uuid.UUID, hashedPassword string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(id, hashedPassword)
	}
	return errors.New("not implemented")
}
func (m *mockUserRepo) ClearMustChangePassword(id uuid.UUID) error {
	if m.clearMustChangePasswordFunc != nil {
		return m.clearMustChangePasswordFunc(id)
	}
	return errors.New("not implemented")
}

const testJWTSecret = "test-secret-key-12345"

// newTestUser creates a models.User with a bcrypt-hashed password.
func newTestUser(email, password string, mustChange bool) *models.User {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	hashedStr := string(hashed)
	first := "Test"
	last := "User"
	return &models.User{
		ID:                 uuid.New(),
		Email:              email,
		Password:           &hashedStr,
		FirstName:          &first,
		LastName:           &last,
		MustChangePassword: mustChange,
		RecentSpaces:       models.RecentSpaces{},
	}
}

// setupLoginApp creates a Fiber app with the auth routes wired to the given mock.
func setupLoginApp(mock *mockUserRepo) *fiber.App {
	app := fiber.New()
	authService := services.NewAuthService(mock, nil, nil, nil, nil, testJWTSecret)
	handler := NewAuthHandler(authService, nil, nil)
	handler.RegisterAuthRoutes(app)
	return app
}

// doLogin sends a POST /auth/login request with the given body and returns the response.
func doLogin(t *testing.T, app *fiber.App, body interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

// parseJSON reads the response body into a map.
func parseJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &result))
	return result
}

// ---------------------------------------------------------------------------
// Route registration tests (preserved from original)
// ---------------------------------------------------------------------------

func TestRegisterEndpoint_ShouldBeDisabled(t *testing.T) {
	app := fiber.New()

	authService := services.NewAuthService(nil, nil, nil, nil, nil, testJWTSecret)
	authHandler := NewAuthHandler(authService, nil, nil)
	authHandler.RegisterAuthRoutes(app)

	requestBody := map[string]string{
		"firstName": "Test",
		"lastName":  "User",
		"email":     "test@example.com",
		"password":  "password123",
	}
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Registration endpoint should not be available (should return 404)")
}

func TestLoginEndpoint_ShouldStillExist(t *testing.T) {
	app := fiber.New()

	authHandler := NewAuthHandler(nil, nil, nil)
	authHandler.RegisterAuthRoutes(app)

	routes := app.GetRoutes()

	var loginRouteExists bool
	var registerRouteExists bool

	for _, route := range routes {
		if route.Path == "/auth/login" && route.Method == "POST" {
			loginRouteExists = true
		}
		if route.Path == "/auth/register" && route.Method == "POST" {
			registerRouteExists = true
		}
	}

	assert.True(t, loginRouteExists, "POST /auth/login route should be registered")
	assert.False(t, registerRouteExists, "POST /auth/register route should NOT be registered")
}

// ---------------------------------------------------------------------------
// Login behaviour tests
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	user := newTestUser("alice@example.com", "correct-password", false)

	mock := &mockUserRepo{
		getUserByEmailFunc: func(email string) (*models.User, error) {
			if email == user.Email {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
	}
	app := setupLoginApp(mock)

	resp := doLogin(t, app, map[string]string{
		"email":    "alice@example.com",
		"password": "correct-password",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)

	// JSON body should include expected fields
	assert.NotEmpty(t, body["accessToken"], "response should include accessToken")
	assert.Equal(t, user.ID.String(), body["userId"])
	assert.Equal(t, "alice@example.com", body["userEmail"])
	assert.Equal(t, "Test", body["firstName"])
	assert.Equal(t, "User", body["lastName"])
	assert.Equal(t, false, body["mustChangePassword"])

	// HTTP-only cookie should be set
	cookies := resp.Cookies()
	var noriCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "nori_token" {
			noriCookie = c
			break
		}
	}
	require.NotNil(t, noriCookie, "nori_token cookie should be set")
	assert.True(t, noriCookie.HttpOnly, "cookie should be HTTP-only")
	assert.Equal(t, "/", noriCookie.Path)
	assert.NotEmpty(t, noriCookie.Value, "cookie value should be the JWT")
	// Cookie value should match the JSON accessToken
	assert.Equal(t, body["accessToken"], noriCookie.Value)
}

func TestLogin_MustChangePassword(t *testing.T) {
	user := newTestUser("bob@example.com", "temp-password", true)

	mock := &mockUserRepo{
		getUserByEmailFunc: func(email string) (*models.User, error) {
			if email == user.Email {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
	}
	app := setupLoginApp(mock)

	resp := doLogin(t, app, map[string]string{
		"email":    "bob@example.com",
		"password": "temp-password",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, true, body["mustChangePassword"],
		"response should signal the client to redirect to password change")
}

func TestLogin_InvalidPassword(t *testing.T) {
	user := newTestUser("carol@example.com", "real-password", false)

	mock := &mockUserRepo{
		getUserByEmailFunc: func(email string) (*models.User, error) {
			if email == user.Email {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
	}
	app := setupLoginApp(mock)

	resp := doLogin(t, app, map[string]string{
		"email":    "carol@example.com",
		"password": "wrong-password",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "invalid email or password")
}

func TestLogin_UserNotFound(t *testing.T) {
	mock := &mockUserRepo{} // default returns "user not found"
	app := setupLoginApp(mock)

	resp := doLogin(t, app, map[string]string{
		"email":    "nobody@example.com",
		"password": "whatever",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "invalid email or password")
}

func TestLogin_InvalidRequestBody(t *testing.T) {
	mock := &mockUserRepo{}
	app := setupLoginApp(mock)

	// Send malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "invalid request body")
}

func TestLogin_EmptyBody(t *testing.T) {
	mock := &mockUserRepo{}
	app := setupLoginApp(mock)

	// Send empty body — BodyParser returns an error for empty JSON input
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "invalid request body")
}

func TestLogin_CookieNotSetOnFailure(t *testing.T) {
	mock := &mockUserRepo{} // no users
	app := setupLoginApp(mock)

	resp := doLogin(t, app, map[string]string{
		"email":    "nobody@example.com",
		"password": "whatever",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Should NOT set the nori_token cookie on failed login
	cookies := resp.Cookies()
	for _, c := range cookies {
		assert.NotEqual(t, "nori_token", c.Name, "nori_token cookie should not be set on failed login")
	}
}

// ---------------------------------------------------------------------------
// Change password helpers
// ---------------------------------------------------------------------------

// fakeAuthMiddleware injects the given authDTO into c.Locals("authDTO"),
// simulating what the real auth middleware does after validating a token.
func fakeAuthMiddleware(authDTO *dtos.AuthDTO) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if authDTO != nil {
			c.Locals("authDTO", authDTO)
		}
		return c.Next()
	}
}

// setupChangePasswordApp creates a Fiber app with the change-password route
// wired to the given mock user repo and fake auth middleware.
func setupChangePasswordApp(mock *mockUserRepo, authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	authService := services.NewAuthService(mock, nil, nil, nil, nil, testJWTSecret)
	handler := NewAuthHandler(authService, nil, nil)
	handler.RegisterProtectedAuthRoutes(app, fakeAuthMiddleware(authDTO))
	return app
}

// doChangePassword sends a POST /auth/change-password request and returns the response.
func doChangePassword(t *testing.T, app *fiber.App, body interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

// ---------------------------------------------------------------------------
// Change password route registration
// ---------------------------------------------------------------------------

func TestChangePasswordRoute_ShouldBeRegistered(t *testing.T) {
	app := fiber.New()
	handler := NewAuthHandler(nil, nil, nil)
	noopMiddleware := func(c *fiber.Ctx) error { return c.Next() }
	handler.RegisterProtectedAuthRoutes(app, noopMiddleware)

	routes := app.GetRoutes()
	var found bool
	for _, route := range routes {
		if route.Path == "/auth/change-password" && route.Method == "POST" {
			found = true
			break
		}
	}
	assert.True(t, found, "POST /auth/change-password route should be registered")
}

// ---------------------------------------------------------------------------
// Change password behaviour tests
// ---------------------------------------------------------------------------

func TestChangePassword_Success(t *testing.T) {
	user := newTestUser("alice@example.com", "old-password", true)

	mock := &mockUserRepo{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
		updatePasswordFunc: func(id uuid.UUID, hashedPassword string) error {
			return nil
		},
		clearMustChangePasswordFunc: func(id uuid.UUID) error {
			return nil
		},
	}

	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	resp := doChangePassword(t, app, map[string]string{
		"currentPassword": "old-password",
		"newPassword":     "new-secure-password",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.NotEmpty(t, body["accessToken"], "response should include a new accessToken")
	assert.Equal(t, user.ID.String(), body["userId"])
	assert.Equal(t, "alice@example.com", body["userEmail"])
	assert.Equal(t, false, body["mustChangePassword"],
		"mustChangePassword should be false after password change")
}

func TestChangePassword_SetsCookie(t *testing.T) {
	user := newTestUser("alice@example.com", "old-password", true)

	mock := &mockUserRepo{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
		updatePasswordFunc:          func(uuid.UUID, string) error { return nil },
		clearMustChangePasswordFunc: func(uuid.UUID) error { return nil },
	}

	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	resp := doChangePassword(t, app, map[string]string{
		"currentPassword": "old-password",
		"newPassword":     "new-secure-password",
	})

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify the nori_token cookie is set with the new JWT
	body := parseJSON(t, resp)

	// Re-do the request since parseJSON consumed the body; check cookies from resp
	resp2 := doChangePassword(t, app, map[string]string{
		"currentPassword": "old-password",
		"newPassword":     "new-secure-password",
	})
	_ = body

	cookies := resp2.Cookies()
	var noriCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "nori_token" {
			noriCookie = c
			break
		}
	}
	require.NotNil(t, noriCookie, "nori_token cookie should be set after password change")
	assert.True(t, noriCookie.HttpOnly, "cookie should be HTTP-only")
	assert.Equal(t, "/", noriCookie.Path)
	assert.NotEmpty(t, noriCookie.Value, "cookie value should be the new JWT")
}

func TestChangePassword_WrongCurrentPassword(t *testing.T) {
	user := newTestUser("alice@example.com", "real-password", true)

	mock := &mockUserRepo{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
	}

	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	resp := doChangePassword(t, app, map[string]string{
		"currentPassword": "wrong-password",
		"newPassword":     "new-password",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, "current password is incorrect", body["error"])
}

func TestChangePassword_MissingCurrentPassword(t *testing.T) {
	user := newTestUser("alice@example.com", "password", false)

	mock := &mockUserRepo{}
	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	resp := doChangePassword(t, app, map[string]string{
		"newPassword": "new-password",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "currentPassword and newPassword are required")
}

func TestChangePassword_MissingNewPassword(t *testing.T) {
	user := newTestUser("alice@example.com", "password", false)

	mock := &mockUserRepo{}
	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	resp := doChangePassword(t, app, map[string]string{
		"currentPassword": "password",
	})

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "currentPassword and newPassword are required")
}

func TestChangePassword_EmptyBody(t *testing.T) {
	user := newTestUser("alice@example.com", "password", false)

	mock := &mockUserRepo{}
	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	// Send empty JSON body
	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "currentPassword and newPassword are required")
}

func TestChangePassword_InvalidJSON(t *testing.T) {
	user := newTestUser("alice@example.com", "password", false)

	mock := &mockUserRepo{}
	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "invalid request body")
}

func TestChangePassword_NoAuthDTO(t *testing.T) {
	mock := &mockUserRepo{}
	// Pass nil authDTO — simulates unauthenticated request reaching the handler
	app := setupChangePasswordApp(mock, nil)

	resp := doChangePassword(t, app, map[string]string{
		"currentPassword": "old",
		"newPassword":     "new",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "authentication required")
}

func TestChangePassword_CookieNotSetOnFailure(t *testing.T) {
	user := newTestUser("alice@example.com", "real-password", true)

	mock := &mockUserRepo{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
	}

	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	resp := doChangePassword(t, app, map[string]string{
		"currentPassword": "wrong-password",
		"newPassword":     "new-password",
	})

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	cookies := resp.Cookies()
	for _, c := range cookies {
		assert.NotEqual(t, "nori_token", c.Name,
			"nori_token cookie should not be set on failed password change")
	}
}

func TestChangePassword_InternalError(t *testing.T) {
	user := newTestUser("alice@example.com", "old-password", true)

	mock := &mockUserRepo{
		getUserByIDFunc: func(id uuid.UUID) (*models.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, errors.New("user not found")
		},
		updatePasswordFunc: func(uuid.UUID, string) error {
			return errors.New("database connection lost")
		},
	}

	authDTO := &dtos.AuthDTO{User: *user}
	app := setupChangePasswordApp(mock, authDTO)

	resp := doChangePassword(t, app, map[string]string{
		"currentPassword": "old-password",
		"newPassword":     "new-password",
	})

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, "failed to change password", body["error"],
		"should not leak internal error details")
}

// ---------------------------------------------------------------------------
// Logout helpers
// ---------------------------------------------------------------------------

// setupLogoutApp creates a Fiber app with the logout route wired to a fake
// auth middleware that injects the given authDTO.
func setupLogoutApp(authDTO *dtos.AuthDTO) *fiber.App {
	app := fiber.New()
	handler := NewAuthHandler(nil, nil, nil)
	handler.RegisterProtectedAuthRoutes(app, fakeAuthMiddleware(authDTO))
	return app
}

// doLogout sends a POST /auth/logout request and returns the response.
func doLogout(t *testing.T, app *fiber.App) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

// ---------------------------------------------------------------------------
// Logout route registration
// ---------------------------------------------------------------------------

func TestLogoutRoute_ShouldBeRegistered(t *testing.T) {
	app := fiber.New()
	handler := NewAuthHandler(nil, nil, nil)
	noopMiddleware := func(c *fiber.Ctx) error { return c.Next() }
	handler.RegisterProtectedAuthRoutes(app, noopMiddleware)

	routes := app.GetRoutes()
	var found bool
	for _, route := range routes {
		if route.Path == "/auth/logout" && route.Method == "POST" {
			found = true
			break
		}
	}
	assert.True(t, found, "POST /auth/logout route should be registered")
}

// ---------------------------------------------------------------------------
// Logout behaviour tests
// ---------------------------------------------------------------------------

func TestLogout_Success(t *testing.T) {
	user := newTestUser("alice@example.com", "password", false)
	authDTO := &dtos.AuthDTO{User: *user}
	app := setupLogoutApp(authDTO)

	resp := doLogout(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, "logged out", body["message"])
}

func TestLogout_ClearsCookie(t *testing.T) {
	user := newTestUser("alice@example.com", "password", false)
	authDTO := &dtos.AuthDTO{User: *user}
	app := setupLogoutApp(authDTO)

	resp := doLogout(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// The nori_token cookie should be set with an empty value and past expiry
	cookies := resp.Cookies()
	var noriCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "nori_token" {
			noriCookie = c
			break
		}
	}
	require.NotNil(t, noriCookie, "nori_token cookie should be present in response")
	assert.Empty(t, noriCookie.Value, "cookie value should be empty")
	assert.True(t, noriCookie.HttpOnly, "cookie should remain HTTP-only")
	assert.Equal(t, "/", noriCookie.Path)
	assert.True(t, noriCookie.Expires.Before(time.Now()),
		"cookie expiry should be in the past to clear it from the browser")
}

func TestLogout_NoAuthDTO(t *testing.T) {
	// When no authDTO is injected (nil), the middleware still passes through
	// because our fake middleware doesn't block. The handler itself doesn't
	// need to inspect the user — it just clears the cookie.
	app := setupLogoutApp(nil)

	resp := doLogout(t, app)

	// Even without auth context, the handler clears the cookie and returns OK.
	// In production, the real auth middleware would reject unauthenticated
	// requests before they reach this handler.
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, "logged out", body["message"])
}

// ---------------------------------------------------------------------------
// Me endpoint mock repos
// ---------------------------------------------------------------------------

// mockMeSpaceMemberRepo implements SpaceMemberRepositoryInterface for /auth/me tests.
type mockMeSpaceMemberRepo struct {
	getByUserFunc func(userID uuid.UUID) ([]models.Space, error)
}

func (m *mockMeSpaceMemberRepo) Create(*models.SpaceMember) error {
	return errors.New("not implemented")
}
func (m *mockMeSpaceMemberRepo) Delete(uuid.UUID, uuid.UUID) error {
	return errors.New("not implemented")
}
func (m *mockMeSpaceMemberRepo) GetByUserAndSpace(uuid.UUID, uuid.UUID) (*models.SpaceMember, error) {
	return nil, errors.New("not implemented")
}
func (m *mockMeSpaceMemberRepo) GetByUser(userID uuid.UUID) ([]models.Space, error) {
	if m.getByUserFunc != nil {
		return m.getByUserFunc(userID)
	}
	return nil, nil
}
func (m *mockMeSpaceMemberRepo) GetBySpace(uuid.UUID) ([]models.SpaceMember, error) {
	return nil, errors.New("not implemented")
}

// mockMeSpaceRepo implements SpaceRepositoryInterface for /auth/me tests.
type mockMeSpaceRepo struct {
	findByAccountIDFunc func(accountID uuid.UUID) ([]models.Space, error)
}

func (m *mockMeSpaceRepo) FindByID(uuid.UUID) (*models.Space, error) {
	return nil, errors.New("not implemented")
}
func (m *mockMeSpaceRepo) FindByAccountID(accountID uuid.UUID) ([]models.Space, error) {
	if m.findByAccountIDFunc != nil {
		return m.findByAccountIDFunc(accountID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Me endpoint helpers
// ---------------------------------------------------------------------------

// setupMeApp creates a Fiber app with the /auth/me route wired to mock repos
// and a fake auth middleware that injects the given authDTO.
func setupMeApp(
	authDTO *dtos.AuthDTO,
	spaceMemberRepo SpaceMemberRepositoryInterface,
	spaceRepo SpaceRepositoryInterface,
) *fiber.App {
	app := fiber.New()
	handler := NewAuthHandler(nil, spaceMemberRepo, spaceRepo)
	handler.RegisterProtectedAuthRoutes(app, fakeAuthMiddleware(authDTO))
	return app
}

// doMe sends a GET /auth/me request and returns the response.
func doMe(t *testing.T, app *fiber.App) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

// ---------------------------------------------------------------------------
// Me route registration
// ---------------------------------------------------------------------------

func TestMeRoute_ShouldBeRegistered(t *testing.T) {
	app := fiber.New()
	handler := NewAuthHandler(nil, nil, nil)
	noopMiddleware := func(c *fiber.Ctx) error { return c.Next() }
	handler.RegisterProtectedAuthRoutes(app, noopMiddleware)

	routes := app.GetRoutes()
	var found bool
	for _, route := range routes {
		if route.Path == "/auth/me" && route.Method == "GET" {
			found = true
			break
		}
	}
	assert.True(t, found, "GET /auth/me route should be registered")
}

// ---------------------------------------------------------------------------
// Me behaviour tests
// ---------------------------------------------------------------------------

func TestMe_AdminSuccess(t *testing.T) {
	adminRole := models.RoleAdmin
	accountID := uuid.New()
	activeSpaceID := uuid.New()
	first := "Admin"
	last := "User"

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:        uuid.New(),
			Email:     "admin@example.com",
			FirstName: &first,
			LastName:  &last,
			Role:      &adminRole,
		},
		AccountID:     accountID,
		ActiveSpaceID: &activeSpaceID,
	}

	space1 := models.Space{
		ID:        activeSpaceID,
		Name:      "Production",
		AccountID: accountID,
		IsDefault: true,
	}
	space2 := models.Space{
		ID:        uuid.New(),
		Name:      "Sales",
		AccountID: accountID,
		IsDefault: false,
	}

	spaceRepo := &mockMeSpaceRepo{
		findByAccountIDFunc: func(id uuid.UUID) ([]models.Space, error) {
			assert.Equal(t, accountID, id)
			return []models.Space{space1, space2}, nil
		},
	}

	app := setupMeApp(authDTO, &mockMeSpaceMemberRepo{}, spaceRepo)
	resp := doMe(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)

	assert.Equal(t, authDTO.User.ID.String(), body["id"])
	assert.Equal(t, "admin@example.com", body["email"])
	assert.Equal(t, "Admin", body["firstName"])
	assert.Equal(t, "User", body["lastName"])
	assert.Equal(t, "admin", body["role"])
	assert.Equal(t, activeSpaceID.String(), body["activeSpaceId"])

	spaces, ok := body["accessibleSpaces"].([]interface{})
	require.True(t, ok, "accessibleSpaces should be an array")
	assert.Len(t, spaces, 2)

	// Verify first space
	s1 := spaces[0].(map[string]interface{})
	assert.Equal(t, space1.ID.String(), s1["id"])
	assert.Equal(t, "Production", s1["name"])
	assert.Equal(t, accountID.String(), s1["accountId"])
	assert.Equal(t, true, s1["isDefault"])
}

func TestMe_RegularUserSuccess(t *testing.T) {
	userRole := models.RoleUser
	accountID := uuid.New()
	activeSpaceID := uuid.New()
	first := "Regular"
	last := "User"

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:        uuid.New(),
			Email:     "user@example.com",
			FirstName: &first,
			LastName:  &last,
			Role:      &userRole,
		},
		AccountID:     accountID,
		ActiveSpaceID: &activeSpaceID,
	}

	memberSpace := models.Space{
		ID:        activeSpaceID,
		Name:      "Workshop",
		AccountID: accountID,
		IsDefault: false,
	}

	spaceMemberRepo := &mockMeSpaceMemberRepo{
		getByUserFunc: func(userID uuid.UUID) ([]models.Space, error) {
			assert.Equal(t, authDTO.User.ID, userID)
			return []models.Space{memberSpace}, nil
		},
	}

	// spaceRepo should NOT be called for regular users
	spaceRepo := &mockMeSpaceRepo{
		findByAccountIDFunc: func(uuid.UUID) ([]models.Space, error) {
			t.Error("FindByAccountID should not be called for regular users")
			return nil, nil
		},
	}

	app := setupMeApp(authDTO, spaceMemberRepo, spaceRepo)
	resp := doMe(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)

	assert.Equal(t, authDTO.User.ID.String(), body["id"])
	assert.Equal(t, "user@example.com", body["email"])
	assert.Equal(t, "Regular", body["firstName"])
	assert.Equal(t, "User", body["lastName"])
	assert.Equal(t, "user", body["role"])
	assert.Equal(t, activeSpaceID.String(), body["activeSpaceId"])

	spaces, ok := body["accessibleSpaces"].([]interface{})
	require.True(t, ok, "accessibleSpaces should be an array")
	assert.Len(t, spaces, 1)
	s1 := spaces[0].(map[string]interface{})
	assert.Equal(t, memberSpace.ID.String(), s1["id"])
	assert.Equal(t, "Workshop", s1["name"])
}

func TestMe_NoActiveSpace(t *testing.T) {
	userRole := models.RoleUser
	first := "New"
	last := "User"

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:        uuid.New(),
			Email:     "new@example.com",
			FirstName: &first,
			LastName:  &last,
			Role:      &userRole,
		},
		AccountID:     uuid.New(),
		ActiveSpaceID: nil, // no active space
	}

	app := setupMeApp(authDTO, &mockMeSpaceMemberRepo{}, &mockMeSpaceRepo{})
	resp := doMe(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)

	assert.Equal(t, authDTO.User.ID.String(), body["id"])
	// activeSpaceId should be absent (omitempty)
	_, hasActiveSpace := body["activeSpaceId"]
	assert.False(t, hasActiveSpace, "activeSpaceId should be omitted when nil")

	spaces, ok := body["accessibleSpaces"].([]interface{})
	require.True(t, ok, "accessibleSpaces should be an array")
	assert.Len(t, spaces, 0, "should be empty when user has no space memberships")
}

func TestMe_NoAuthDTO(t *testing.T) {
	app := setupMeApp(nil, &mockMeSpaceMemberRepo{}, &mockMeSpaceRepo{})
	resp := doMe(t, app)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Contains(t, body["error"], "authentication required")
}

func TestMe_NilRole(t *testing.T) {
	// A user with no role set — should behave like a regular user
	first := "No"
	last := "Role"

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:        uuid.New(),
			Email:     "norole@example.com",
			FirstName: &first,
			LastName:  &last,
			Role:      nil,
		},
		AccountID:     uuid.New(),
		ActiveSpaceID: nil,
	}

	calledGetByUser := false
	spaceMemberRepo := &mockMeSpaceMemberRepo{
		getByUserFunc: func(userID uuid.UUID) ([]models.Space, error) {
			calledGetByUser = true
			return nil, nil
		},
	}

	app := setupMeApp(authDTO, spaceMemberRepo, &mockMeSpaceRepo{})
	resp := doMe(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, calledGetByUser, "should use GetByUser for nil role (not FindByAccountID)")

	body := parseJSON(t, resp)
	assert.Nil(t, body["role"], "role should be null when not set")
}

func TestMe_SpaceQueryError(t *testing.T) {
	adminRole := models.RoleAdmin

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:    uuid.New(),
			Email: "admin@example.com",
			Role:  &adminRole,
		},
		AccountID: uuid.New(),
	}

	spaceRepo := &mockMeSpaceRepo{
		findByAccountIDFunc: func(uuid.UUID) ([]models.Space, error) {
			return nil, errors.New("database connection lost")
		},
	}

	app := setupMeApp(authDTO, &mockMeSpaceMemberRepo{}, spaceRepo)
	resp := doMe(t, app)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, "failed to fetch accessible spaces", body["error"],
		"should not leak internal error details")
}

func TestMe_RegularUserSpaceQueryError(t *testing.T) {
	userRole := models.RoleUser

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:    uuid.New(),
			Email: "user@example.com",
			Role:  &userRole,
		},
		AccountID: uuid.New(),
	}

	spaceMemberRepo := &mockMeSpaceMemberRepo{
		getByUserFunc: func(uuid.UUID) ([]models.Space, error) {
			return nil, errors.New("database connection lost")
		},
	}

	app := setupMeApp(authDTO, spaceMemberRepo, &mockMeSpaceRepo{})
	resp := doMe(t, app)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, "failed to fetch accessible spaces", body["error"],
		"should not leak internal error details")
}

func TestMe_EmptyAccessibleSpaces(t *testing.T) {
	adminRole := models.RoleAdmin

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:    uuid.New(),
			Email: "admin@example.com",
			Role:  &adminRole,
		},
		AccountID: uuid.New(),
	}

	spaceRepo := &mockMeSpaceRepo{
		findByAccountIDFunc: func(uuid.UUID) ([]models.Space, error) {
			return []models.Space{}, nil // no spaces exist
		},
	}

	app := setupMeApp(authDTO, &mockMeSpaceMemberRepo{}, spaceRepo)
	resp := doMe(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)
	spaces, ok := body["accessibleSpaces"].([]interface{})
	require.True(t, ok, "accessibleSpaces should be an array")
	assert.Len(t, spaces, 0, "accessibleSpaces should be an empty array, not null")
}

func TestMe_OptionalFieldsOmitted(t *testing.T) {
	userRole := models.RoleUser

	authDTO := &dtos.AuthDTO{
		User: models.User{
			ID:        uuid.New(),
			Email:     "minimal@example.com",
			Role:      &userRole,
			FirstName: nil, // not set
			LastName:  nil, // not set
		},
		AccountID:     uuid.New(),
		ActiveSpaceID: nil, // no active space
	}

	app := setupMeApp(authDTO, &mockMeSpaceMemberRepo{}, &mockMeSpaceRepo{})
	resp := doMe(t, app)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := parseJSON(t, resp)
	assert.Equal(t, authDTO.User.ID.String(), body["id"])
	assert.Equal(t, "minimal@example.com", body["email"])

	// firstName, lastName, activeSpaceId should be omitted
	_, hasFirst := body["firstName"]
	_, hasLast := body["lastName"]
	_, hasActive := body["activeSpaceId"]
	assert.False(t, hasFirst, "firstName should be omitted when nil")
	assert.False(t, hasLast, "lastName should be omitted when nil")
	assert.False(t, hasActive, "activeSpaceId should be omitted when nil")
}
