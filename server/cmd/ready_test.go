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

func TestReadyCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "ready" {
			found = true
			break
		}
	}
	assert.True(t, found, "ready command should be registered on root")
}

func TestReadyCommandHasJSONFlag(t *testing.T) {
	flag := readyCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestRunReady_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/tasks/ready", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(readyResponse{
			Items: []readyTask{
				{ID: "shop-a1.1", Title: "Cut mortises", Priority: 1},
				{ID: "shop-a1.2", Title: "Cut tenons", Priority: 2},
			},
			Total: 2,
		})
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

	// Run the command (output goes to stdout)
	err := runReady(readyCmd, nil)
	require.NoError(t, err)
}

func TestRunReady_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(readyResponse{
			Items: []readyTask{},
			Total: 0,
		})
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

	err := runReady(readyCmd, nil)
	require.NoError(t, err)
}

func TestRunReady_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Override isInteractiveFunc so Handle401 uses non-interactive path
	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "bad-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := runReady(readyCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunReady_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
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

	err := runReady(readyCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestRunReady_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runReady(readyCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunReady_SpaceIDHeader(t *testing.T) {
	var receivedSpaceID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSpaceID = r.Header.Get("X-Space-ID")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(readyResponse{Items: []readyTask{}, Total: 0})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
		SpaceID:     "test-space-id",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// NewClient now populates SpaceID from credentials automatically
	client := cli.NewClient(creds)
	resp, err := client.Get("/api/v1/tasks/ready")
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "test-space-id", receivedSpaceID)
}

func TestRunReady_SpaceIDFromCredentials(t *testing.T) {
	var receivedSpaceID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSpaceID = r.Header.Get("X-Space-ID")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(readyResponse{Items: []readyTask{}, Total: 0})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Clear any --space flag state from other tests
	spaceFlag = ""

	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
		SpaceID:     "creds-space-id",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	err := runReady(readyCmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "creds-space-id", receivedSpaceID)
}

func TestResolveSpaceID_FlagOverridesAll(t *testing.T) {
	creds := &cli.Credentials{
		SpaceID: "creds-space",
	}

	// Set flag
	spaceFlag = "flag-space"
	t.Setenv("NORI_SPACE", "env-space")
	defer func() { spaceFlag = "" }()

	result := resolveSpaceID(creds)
	assert.Equal(t, "flag-space", result)
}

func TestResolveSpaceID_EnvOverridesCreds(t *testing.T) {
	creds := &cli.Credentials{
		SpaceID: "creds-space",
	}

	spaceFlag = ""
	t.Setenv("NORI_SPACE", "env-space")

	result := resolveSpaceID(creds)
	assert.Equal(t, "env-space", result)
}

func TestResolveSpaceID_FallsThroughToCreds(t *testing.T) {
	creds := &cli.Credentials{
		SpaceID: "creds-space",
	}

	spaceFlag = ""
	t.Setenv("NORI_SPACE", "")

	result := resolveSpaceID(creds)
	assert.Equal(t, "creds-space", result)
}
