package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

func TestStatusCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "status" {
			found = true
			break
		}
	}
	assert.True(t, found, "status command should be registered on root")
}

func TestStatusCommandHasJSONFlag(t *testing.T) {
	flag := statusCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestRunStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(statusHealthResponse{Status: "ok"})
		case "/auth/me":
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			json.NewEncoder(w).Encode(statusMeResponse{
				ID:            "user-123",
				Email:         "admin@example.com",
				ActiveSpaceID: strPtr("space-abc"),
				AccessibleSpaces: []statusMeSpace{
					{ID: "space-abc", Name: "Main Shop"},
				},
			})
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "admin@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := runStatus(statusCmd, nil)
	require.NoError(t, err)
}

func TestRunStatus_SuccessJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(statusHealthResponse{Status: "ok"})
		case "/auth/me":
			json.NewEncoder(w).Encode(statusMeResponse{
				ID:            "user-123",
				Email:         "admin@example.com",
				ActiveSpaceID: strPtr("space-abc"),
				AccessibleSpaces: []statusMeSpace{
					{ID: "space-abc", Name: "Main Shop"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "admin@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Set the JSON flag for this test
	statusJSONFlag = true
	defer func() { statusJSONFlag = false }()

	err := runStatus(statusCmd, nil)
	require.NoError(t, err)
}

func TestRunStatus_SpaceFromFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(statusHealthResponse{Status: "ok"})
		case "/auth/me":
			json.NewEncoder(w).Encode(statusMeResponse{
				ID:    "user-123",
				Email: "admin@example.com",
				AccessibleSpaces: []statusMeSpace{
					{ID: "flag-space-id", Name: "Flag Space"},
					{ID: "creds-space-id", Name: "Creds Space"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "admin@example.com",
		SpaceID:     "creds-space-id",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Set the --space flag to override credentials
	spaceFlag = "flag-space-id"
	defer func() { spaceFlag = "" }()

	err := runStatus(statusCmd, nil)
	require.NoError(t, err)
}

func TestRunStatus_NoSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(statusHealthResponse{Status: "ok"})
		case "/auth/me":
			json.NewEncoder(w).Encode(statusMeResponse{
				ID:               "user-123",
				Email:            "admin@example.com",
				AccessibleSpaces: []statusMeSpace{},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	spaceFlag = ""

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "admin@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := runStatus(statusCmd, nil)
	require.NoError(t, err)
}

func TestRunStatus_HealthFails(t *testing.T) {
	// Start and immediately close a server to get a valid but unreachable URL.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   serverURL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "admin@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := runStatus(statusCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to server")
}

func TestRunStatus_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(statusHealthResponse{Status: "ok"})
		case "/auth/me":
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "bad-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := runStatus(statusCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunStatus_MeServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(statusHealthResponse{Status: "ok"})
		case "/auth/me":
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := runStatus(statusCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestRunStatus_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runStatus(statusCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

// strPtr is a test helper that returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}
