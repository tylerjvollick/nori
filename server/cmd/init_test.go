package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

func TestInitCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			found = true
			break
		}
	}
	assert.True(t, found, "init command should be registered on root")
}

func TestGenerateJWTSecret(t *testing.T) {
	secret, err := generateJWTSecret()
	require.NoError(t, err)

	// 32 bytes = 64 hex chars
	assert.Len(t, secret, 64)

	// Should be different each time
	secret2, err := generateJWTSecret()
	require.NoError(t, err)
	assert.NotEqual(t, secret, secret2)
}

func TestGenerateRandomHex(t *testing.T) {
	hex16, err := generateRandomHex(16)
	require.NoError(t, err)
	assert.Len(t, hex16, 32)

	hex32, err := generateRandomHex(32)
	require.NoError(t, err)
	assert.Len(t, hex32, 64)
}

func TestWriteEnvFile(t *testing.T) {
	// Use temp dir as working directory
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.MkdirAll(tmpDir+"/docker", 0700))
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	err := writeEnvFile("Test Shop", "admin@test.com", "secret123", "jwt-hex-secret", "db-password-hex")
	require.NoError(t, err)

	data, err := os.ReadFile("docker/.env")
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "NORI_ACCOUNT_NAME=Test Shop")
	assert.Contains(t, content, "NORI_ADMIN_EMAIL=admin@test.com")
	assert.Contains(t, content, "NORI_ADMIN_PASSWORD=secret123")
	assert.Contains(t, content, "NORI_JWT_SECRET=jwt-hex-secret")
	assert.Contains(t, content, "POSTGRES_PASSWORD=db-password-hex")
	assert.Contains(t, content, "NORI_SKIP_PASSWORD_CHANGE=true")

	// Verify file permissions
	info, err := os.Stat("docker/.env")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestDoLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/login", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "admin@test.com", body["email"])
		assert.Equal(t, "password", body["password"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse{
			AccessToken: "jwt-token",
			UserID:      "user-123",
			UserEmail:   "admin@test.com",
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	resp, err := doLogin(client, "admin@test.com", "password")
	require.NoError(t, err)
	assert.Equal(t, "jwt-token", resp.AccessToken)
	assert.Equal(t, "user-123", resp.UserID)
	assert.Equal(t, "admin@test.com", resp.UserEmail)
}

func TestDoLogin_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	_, err := doLogin(client, "admin@test.com", "wrong")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestCreateSpace_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/spaces", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-jwt", r.Header.Get("Authorization"))

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Workshop", body["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(initSpaceResponse{
			ID:   "space-uuid",
			Name: "Workshop",
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	resp, err := createSpace(client, "Workshop")
	require.NoError(t, err)
	assert.Equal(t, "space-uuid", resp.ID)
	assert.Equal(t, "Workshop", resp.Name)
}

func TestCreateSpace_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(errorResponse{Error: "admin access required"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	_, err := createSpace(client, "Workshop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin access required")
}

func TestCreateStation_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/stations", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "space-123", r.Header.Get("X-Space-ID"))

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Cutting", body["name"])
		assert.Equal(t, float64(2), body["wipLimit"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(initStationResponse{
			ID:       "station-uuid",
			Name:     "Cutting",
			WIPLimit: 2,
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	client.SpaceID = "space-123"
	resp, err := createStation(client, "Cutting", 2)
	require.NoError(t, err)
	assert.Equal(t, "station-uuid", resp.ID)
	assert.Equal(t, "Cutting", resp.Name)
	assert.Equal(t, 2, resp.WIPLimit)
}

func TestCreateAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/admin/api-keys", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "nori-cli", body["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rawKey": "nori_test-api-key-value",
			"apiKey": map[string]interface{}{
				"id":   "key-uuid",
				"name": "nori-cli",
			},
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	resp, err := createAPIKey(client)
	require.NoError(t, err)
	assert.Equal(t, "nori_test-api-key-value", resp.RawKey)
}

func TestCreateAPIKey_EmptyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rawKey": "",
			"apiKey": map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	_, err := createAPIKey(client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty API key")
}

func TestWaitForHealthy_ImmediateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	err := waitForHealthy(server.URL, 5*time.Second, 100*time.Millisecond)
	require.NoError(t, err)
}

func TestWaitForHealthy_Timeout(t *testing.T) {
	// Server that always returns 503
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := waitForHealthy(server.URL, 500*time.Millisecond, 100*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

// TestInitFullAPIFlow tests the complete API interaction sequence used by init,
// without actually running docker or prompts.
func TestInitFullAPIFlow(t *testing.T) {
	callSequence := make([]string, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callSequence = append(callSequence, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/auth/login":
			json.NewEncoder(w).Encode(loginResponse{
				AccessToken: "jwt-token",
				UserID:      "user-123",
				UserEmail:   "admin@test.com",
			})

		case "/api/spaces":
			assert.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(initSpaceResponse{
				ID:   "space-uuid",
				Name: "Workshop",
			})

		case "/api/v1/stations":
			assert.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))
			assert.Equal(t, "space-uuid", r.Header.Get("X-Space-ID"))

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(initStationResponse{
				ID:       "station-uuid",
				Name:     body["name"].(string),
				WIPLimit: int(body["wipLimit"].(float64)),
			})

		case "/admin/api-keys":
			assert.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"rawKey": "nori_test-key-abc123",
				"apiKey": map[string]interface{}{"id": "key-uuid", "name": "nori-cli"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Simulate the API flow that runInit performs after docker is up
	client := cli.NewClientWithURL(server.URL)

	// Login
	loginResp, err := doLogin(client, "admin@test.com", "password")
	require.NoError(t, err)
	client.Token = loginResp.AccessToken

	// Create space
	spaceResp, err := createSpace(client, "Workshop")
	require.NoError(t, err)
	client.SpaceID = spaceResp.ID

	// Create stations
	stations := []stationInput{
		{Name: "Cutting", WIPLimit: 2},
		{Name: "Assembly", WIPLimit: 1},
	}
	for _, s := range stations {
		_, err := createStation(client, s.Name, s.WIPLimit)
		require.NoError(t, err)
	}

	// Create API key
	apiKeyResp, err := createAPIKey(client)
	require.NoError(t, err)

	// Save credentials
	creds := &cli.Credentials{
		ServerURL: server.URL,
		APIKey:    apiKeyResp.RawKey,
		UserID:    loginResp.UserID,
		UserEmail: loginResp.UserEmail,
		SpaceID:   spaceResp.ID,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Verify saved credentials
	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "nori_test-key-abc123", loaded.APIKey)
	assert.Equal(t, "admin@test.com", loaded.UserEmail)
	assert.Equal(t, "space-uuid", loaded.SpaceID)
	assert.Equal(t, "user-123", loaded.UserID)

	// Verify API call sequence
	expected := []string{
		"POST /auth/login",
		"POST /api/spaces",
		"POST /api/v1/stations",
		"POST /api/v1/stations",
		"POST /admin/api-keys",
	}
	assert.Equal(t, expected, callSequence)
}
