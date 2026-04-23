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

func TestJobCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "job" {
			found = true
			break
		}
	}
	assert.True(t, found, "job command should be registered on root")
}

func TestJobSubcommandsRegistered(t *testing.T) {
	subcommands := map[string]bool{"list": false, "show": false}
	for _, cmd := range jobCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}
	for name, found := range subcommands {
		assert.True(t, found, "%s subcommand should be registered on job", name)
	}
}

func TestJobCommandHasJSONFlag(t *testing.T) {
	flag := jobCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestJobListCommandHasStatusFlag(t *testing.T) {
	flag := jobListCmd.Flags().Lookup("status")
	require.NotNil(t, flag, "--status flag should be defined")
	assert.Equal(t, "", flag.DefValue)
}

// --- Job List Tests ---

func TestRunJobList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/tasks", r.URL.Path)
		assert.Equal(t, "job", r.URL.Query().Get("type"))
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobListResponse{
			Items: []jobItem{
				{ID: "shop-a1", Title: "Walnut Dining Table", Status: "active", Priority: 1},
				{ID: "shop-b2", Title: "Cherry Side Table", Status: "open", Priority: 2},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	// Reset status flag
	jobStatusFlag = ""

	err := runJobList(jobListCmd, nil)
	require.NoError(t, err)
}

func TestRunJobList_WithStatusFilter(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("status")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobListResponse{
			Items: []jobItem{
				{ID: "shop-a1", Title: "Walnut Dining Table", Status: "active", Priority: 1},
			},
			Total: 1,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	jobStatusFlag = "active"
	defer func() { jobStatusFlag = "" }()

	err := runJobList(jobListCmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "active", receivedQuery)
}

func TestRunJobList_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobListResponse{
			Items: []jobItem{},
			Total: 0,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	jobStatusFlag = ""

	err := runJobList(jobListCmd, nil)
	require.NoError(t, err)
}

func TestRunJobList_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	jobStatusFlag = ""

	err := runJobList(jobListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunJobList_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	jobStatusFlag = ""

	err := runJobList(jobListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestRunJobList_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	jobStatusFlag = ""

	err := runJobList(jobListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

// --- Job Show Tests ---

func TestRunJobShow_Success(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/spaces/test-space/tasks/shop-a1" {
			// First request: get the job itself
			assert.Equal(t, http.MethodGet, r.Method)
			json.NewEncoder(w).Encode(jobDetail{
				ID:       "shop-a1",
				Title:    "Walnut Dining Table",
				Status:   "active",
				Type:     "job",
				Priority: 1,
			})
			return
		}

		if r.URL.Path == "/api/v1/spaces/test-space/tasks" && r.URL.Query().Get("parentId") == "shop-a1" {
			// Second request: get children
			json.NewEncoder(w).Encode(struct {
				Items []taskChild `json:"items"`
				Total int64       `json:"total"`
			}{
				Items: []taskChild{
					{ID: "shop-a1.1", Title: "Mill lumber", Status: "done", Type: "task", Priority: 1},
					{ID: "shop-a1.2", Title: "Cut mortises", Status: "active", Type: "task", Priority: 2},
					{ID: "shop-a1.3", Title: "Quality check", Status: "open", Type: "gate", Priority: 3},
				},
				Total: 3,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	err := runJobShow(jobShowCmd, []string{"shop-a1"})
	require.NoError(t, err)
	assert.Equal(t, 2, requestCount, "should make two requests: job + children")
}

func TestRunJobShow_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse{Error: "task not found"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	err := runJobShow(jobShowCmd, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestRunJobShow_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	err := runJobShow(jobShowCmd, []string{"shop-a1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunJobShow_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runJobShow(jobShowCmd, []string{"shop-a1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunJobShow_NoChildren(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/spaces/test-space/tasks/shop-a1" {
			json.NewEncoder(w).Encode(jobDetail{
				ID:       "shop-a1",
				Title:    "Walnut Dining Table",
				Status:   "open",
				Type:     "job",
				Priority: 1,
			})
			return
		}

		if r.URL.Path == "/api/v1/spaces/test-space/tasks" {
			json.NewEncoder(w).Encode(struct {
				Items []taskChild `json:"items"`
				Total int64       `json:"total"`
			}{
				Items: []taskChild{},
				Total: 0,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	err := runJobShow(jobShowCmd, []string{"shop-a1"})
	require.NoError(t, err)
}
