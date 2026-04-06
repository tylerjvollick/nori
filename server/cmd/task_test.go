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

func TestTaskCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "task" {
			found = true
			break
		}
	}
	assert.True(t, found, "task command should be registered on root")
}

func TestTaskSubcommandsRegistered(t *testing.T) {
	subcommands := map[string]bool{"claim": false, "complete": false, "pause": false}
	for _, cmd := range taskCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}
	for name, found := range subcommands {
		assert.True(t, found, "%s subcommand should be registered on task", name)
	}
}

func TestTaskCommandHasJSONFlag(t *testing.T) {
	flag := taskCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func setupTestServer(t *testing.T, expectedPath string, expectedMethod string, responseBody interface{}, statusCode int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedPath, r.URL.Path)
		assert.Equal(t, expectedMethod, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
		}
		json.NewEncoder(w).Encode(responseBody)
	}))
	t.Cleanup(server.Close)
	return server
}

func setupCredentials(t *testing.T, serverURL string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := &cli.Credentials{
		ServerURL:   serverURL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))
}

// --- Claim Tests ---

func TestRunTaskClaim_Success(t *testing.T) {
	resp := taskActionResponse{
		ID:     "shop-a1.1",
		Title:  "Cut mortises",
		Status: "active",
	}
	server := setupTestServer(t, "/api/v1/tasks/shop-a1.1/claim", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskClaim(taskClaimCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskClaim_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/tasks/shop-a1.1/claim",
		http.MethodPost,
		errorResponse{Error: `task "shop-a1.1" cannot be claimed: status is "active", must be "open"`},
		http.StatusConflict,
	)
	setupCredentials(t, server.URL)

	err := runTaskClaim(taskClaimCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be claimed")
}

func TestRunTaskClaim_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	err := runTaskClaim(taskClaimCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

// --- Complete Tests ---

func TestRunTaskComplete_Success(t *testing.T) {
	resp := taskActionResponse{
		ID:     "shop-a1.1",
		Title:  "Cut mortises",
		Status: "done",
	}
	server := setupTestServer(t, "/api/v1/tasks/shop-a1.1/complete", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskComplete(taskCompleteCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskComplete_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/tasks/shop-a1.1/complete",
		http.MethodPost,
		errorResponse{Error: `task "shop-a1.1" cannot be completed: status is "open", must be "active"`},
		http.StatusConflict,
	)
	setupCredentials(t, server.URL)

	err := runTaskComplete(taskCompleteCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be completed")
}

// --- Pause Tests ---

func TestRunTaskPause_Success(t *testing.T) {
	resp := taskActionResponse{
		ID:     "shop-a1.1",
		Title:  "Cut mortises",
		Status: "paused",
	}
	server := setupTestServer(t, "/api/v1/tasks/shop-a1.1/pause", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskPause(taskPauseCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskPause_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/tasks/shop-a1.1/pause",
		http.MethodPost,
		errorResponse{Error: `task "shop-a1.1" cannot be paused: status is "open", must be "active"`},
		http.StatusConflict,
	)
	setupCredentials(t, server.URL)

	err := runTaskPause(taskPauseCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be paused")
}

// --- No Credentials Test ---

func TestRunTaskAction_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runTaskClaim(taskClaimCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

// --- Server Error Test ---

func TestRunTaskAction_ServerError(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/tasks/shop-a1.1/claim",
		http.MethodPost,
		errorResponse{Error: "database error"},
		http.StatusInternalServerError,
	)
	setupCredentials(t, server.URL)

	err := runTaskClaim(taskClaimCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}
