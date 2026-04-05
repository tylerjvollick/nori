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
	assert.NotNil(t, client.httpClient)
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
