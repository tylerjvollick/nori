package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Post(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/auth/login", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &Client{
		ServerURL:  server.URL,
		Token:      "test-token",
		httpClient: server.Client(),
	}

	resp, err := client.Post("/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "secret",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, ReadJSON(resp, &result))
	assert.Equal(t, "ok", result["status"])
}

func TestClient_PostWithoutToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClientWithURL(server.URL)

	resp, err := client.Post("/auth/login", map[string]string{
		"email": "test@example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewClient(t *testing.T) {
	creds := &Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "my-token",
		UserID:      "user-1",
		UserEmail:   "test@example.com",
	}

	client := NewClient(creds)
	assert.Equal(t, "http://localhost:8080", client.ServerURL)
	assert.Equal(t, "my-token", client.Token)
	assert.Empty(t, client.SpaceID, "SpaceID should be empty when not set in credentials")
	assert.NotNil(t, client.httpClient)
}

func TestNewClient_PopulatesSpaceID(t *testing.T) {
	creds := &Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "my-token",
		UserID:      "user-1",
		UserEmail:   "test@example.com",
		SpaceID:     "space-uuid-123",
	}

	client := NewClient(creds)
	assert.Equal(t, "space-uuid-123", client.SpaceID)
}

func TestReadJSON_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := server.Client().Get(server.URL)
	require.NoError(t, err)

	var result map[string]string
	err = ReadJSON(resp, &result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestClient_Get(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/status", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("Content-Type")) // GET has no body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &Client{
		ServerURL:  server.URL,
		Token:      "test-token",
		httpClient: server.Client(),
	}

	resp, err := client.Get("/api/status")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]string
	require.NoError(t, ReadJSON(resp, &result))
	assert.Equal(t, "ok", result["status"])
}

func TestNewClient_PrefersAPIKeyOverJWT(t *testing.T) {
	creds := &Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt-token",
		APIKey:      "nori_abc123def456",
		UserID:      "user-1",
		UserEmail:   "test@example.com",
	}

	client := NewClient(creds)
	assert.Equal(t, "nori_abc123def456", client.Token, "should use API key over JWT")
}

func TestNewClient_UsesJWTWhenNoAPIKey(t *testing.T) {
	creds := &Credentials{
		ServerURL:   "http://localhost:8080",
		AccessToken: "jwt-token",
		UserID:      "user-1",
		UserEmail:   "test@example.com",
	}

	client := NewClient(creds)
	assert.Equal(t, "jwt-token", client.Token, "should use JWT when no API key")
}

func TestNewClient_APIKeyOnlyCredentials(t *testing.T) {
	creds := &Credentials{
		ServerURL: "http://localhost:8080",
		APIKey:    "nori_abc123def456",
	}

	client := NewClient(creds)
	assert.Equal(t, "nori_abc123def456", client.Token, "should use API key from API-key-only creds")
}

func TestClient_APIKeySentAsBearerToken(t *testing.T) {
	// Verify the API key is sent as "Bearer nori_..." in the Authorization header
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer nori_abc123def456", authHeader,
			"API key should be sent as Bearer token")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := &Client{
		ServerURL:  server.URL,
		Token:      "nori_abc123def456",
		httpClient: server.Client(),
	}

	resp, err := client.Get("/api/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestIsUnauthorized(t *testing.T) {
	assert.True(t, IsUnauthorized(&http.Response{StatusCode: http.StatusUnauthorized}))
	assert.False(t, IsUnauthorized(&http.Response{StatusCode: http.StatusOK}))
	assert.False(t, IsUnauthorized(&http.Response{StatusCode: http.StatusForbidden}))
}
