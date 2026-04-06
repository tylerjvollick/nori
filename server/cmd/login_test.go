package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://localhost:8080", "http://localhost:8080"},
		{"http://localhost:8080/", "http://localhost:8080"},
		{"https://nori.example.com", "https://nori.example.com"},
		{"https://nori.example.com/", "https://nori.example.com"},
		{"localhost:8080", "http://localhost:8080"},
		{"nori.example.com", "http://nori.example.com"},
		{"  http://localhost:8080  ", "http://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeURL(tt.input))
		})
	}
}

func TestLoginCommandRegistered(t *testing.T) {
	// Verify the login command is registered on the root command
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "login" {
			found = true
			break
		}
	}
	assert.True(t, found, "login command should be registered on root")
}

func TestServeCommandRegistered(t *testing.T) {
	// Verify the serve command is registered on the root command
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			found = true
			break
		}
	}
	assert.True(t, found, "serve command should be registered on root")
}

// TestLoginFlow_Success tests the core login flow by simulating a server and
// verifying credentials are stored. Note: we can't test the interactive prompts
// easily, so this tests the underlying client + credentials integration.
func TestLoginFlow_Success(t *testing.T) {
	// Set up a fake server that returns a successful login response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/login", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "test@example.com", body["email"])
		assert.Equal(t, "password123", body["password"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse{
			AccessToken:        "jwt-token-abc",
			UserID:             "user-uuid-123",
			UserEmail:          "test@example.com",
			FirstName:          "Test",
			LastName:           "User",
			MustChangePassword: false,
		})
	}))
	defer server.Close()

	// Use temp dir for credentials
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Simulate the login flow (without interactive prompts)
	client := cli.NewClientWithURL(server.URL)
	resp, err := client.Post("/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp loginResponse
	require.NoError(t, cli.ReadJSON(resp, &loginResp))
	assert.Equal(t, "jwt-token-abc", loginResp.AccessToken)
	assert.False(t, loginResp.MustChangePassword)

	// Save credentials
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: loginResp.AccessToken,
		UserID:      loginResp.UserID,
		UserEmail:   loginResp.UserEmail,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Verify credentials were saved
	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "jwt-token-abc", loaded.AccessToken)
	assert.Equal(t, "test@example.com", loaded.UserEmail)

	// Verify file permissions
	path := filepath.Join(tmpDir, ".config", "nori", "credentials")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// TestLoginFlow_MustChangePassword tests the login flow when the server
// returns mustChangePassword=true.
func TestLoginFlow_MustChangePassword(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/auth/login":
			json.NewEncoder(w).Encode(loginResponse{
				AccessToken:        "initial-token",
				UserID:             "user-uuid-123",
				UserEmail:          "test@example.com",
				MustChangePassword: true,
			})
		case "/auth/change-password":
			// Verify the authorization header uses the initial token
			assert.Equal(t, "Bearer initial-token", r.Header.Get("Authorization"))

			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "old-pass", body["currentPassword"])
			assert.Equal(t, "new-pass", body["newPassword"])

			json.NewEncoder(w).Encode(loginResponse{
				AccessToken:        "new-token-after-change",
				UserID:             "user-uuid-123",
				UserEmail:          "test@example.com",
				MustChangePassword: false,
			})
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Step 1: Login
	client := cli.NewClientWithURL(server.URL)
	resp, err := client.Post("/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "old-pass",
	})
	require.NoError(t, err)

	var loginResp loginResponse
	require.NoError(t, cli.ReadJSON(resp, &loginResp))
	assert.True(t, loginResp.MustChangePassword)

	// Step 2: Change password
	client.Token = loginResp.AccessToken
	resp, err = client.Post("/auth/change-password", map[string]string{
		"currentPassword": "old-pass",
		"newPassword":     "new-pass",
	})
	require.NoError(t, err)

	var changeResp loginResponse
	require.NoError(t, cli.ReadJSON(resp, &changeResp))
	assert.Equal(t, "new-token-after-change", changeResp.AccessToken)

	// Step 3: Save with new token
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: changeResp.AccessToken,
		UserID:      changeResp.UserID,
		UserEmail:   changeResp.UserEmail,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "new-token-after-change", loaded.AccessToken)
}

// TestLoginFlow_InvalidCredentials tests that invalid credentials return an error.
func TestLoginFlow_InvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{
			Error: "invalid email or password",
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	resp, err := client.Post("/auth/login", map[string]string{
		"email":    "wrong@example.com",
		"password": "wrong",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

// TestCredentialsUsedBySubsequentCommands verifies that stored credentials
// can be loaded and used to set the Authorization header.
func TestCredentialsUsedBySubsequentCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the authorization header is set
		assert.Equal(t, "Bearer stored-jwt-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Store credentials
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "stored-jwt-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Load and use
	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)

	client := cli.NewClient(loaded)
	resp, err := client.Post("/some/endpoint", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// TestLoginFlow_SavesActiveSpaceID verifies that the ActiveSpaceID from the
// login response is saved to credentials.
func TestLoginFlow_SavesActiveSpaceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		spaceID := "550e8400-e29b-41d4-a716-446655440000"
		json.NewEncoder(w).Encode(loginResponse{
			AccessToken:        "jwt-token-abc",
			UserID:             "user-uuid-123",
			UserEmail:          "test@example.com",
			MustChangePassword: false,
			ActiveSpaceID:      &spaceID,
		})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	client := cli.NewClientWithURL(server.URL)
	resp, err := client.Post("/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	require.NoError(t, err)

	var loginResp loginResponse
	require.NoError(t, cli.ReadJSON(resp, &loginResp))
	require.NotNil(t, loginResp.ActiveSpaceID)

	// Simulate what runLogin does
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: loginResp.AccessToken,
		UserID:      loginResp.UserID,
		UserEmail:   loginResp.UserEmail,
	}
	if loginResp.ActiveSpaceID != nil {
		creds.SpaceID = *loginResp.ActiveSpaceID
	}
	require.NoError(t, cli.SaveCredentials(creds))

	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", loaded.SpaceID)
}
