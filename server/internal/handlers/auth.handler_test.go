package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"github.com/tylerjvollick/nori/internal/services"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo implements services.UserRepositoryInterface for handler tests.
type mockUserRepo struct {
	getUserByEmailFunc func(string) (*models.User, error)
}

func (m *mockUserRepo) GetUserByID(uuid.UUID) (*models.User, error) {
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
func (m *mockUserRepo) UpdatePassword(uuid.UUID, string) error {
	return errors.New("not implemented")
}
func (m *mockUserRepo) ClearMustChangePassword(uuid.UUID) error {
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
	handler := NewAuthHandler(authService)
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
	authHandler := NewAuthHandler(authService)
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

	authHandler := NewAuthHandler(nil)
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
