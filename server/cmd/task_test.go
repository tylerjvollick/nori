package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	subcommands := map[string]bool{"start": false, "complete": false, "pause": false, "resume": false, "skip": false, "add": false, "note": false}
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
		// Time entry side-effect calls (start/pause) are non-blocking best-effort.
		// Return 200 empty JSON for them without asserting.
		if strings.Contains(r.URL.Path, "/time-entries/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{})
			return
		}

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
		SpaceID:     "test-space",
	}
	require.NoError(t, cli.SaveCredentials(creds))
}

// --- Start Tests ---

func TestRunTaskStart_Success(t *testing.T) {
	resp := taskActionResponse{
		ID:     "shop-a1.1",
		Title:  "Cut mortises",
		Status: "active",
	}
	server := setupTestServer(t, "/api/v1/spaces/test-space/tasks/shop-a1.1/start", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskStart(taskStartCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskStart_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/spaces/test-space/tasks/shop-a1.1/start",
		http.MethodPost,
		errorResponse{Error: `task "shop-a1.1" cannot be started: status is "active", must be "open"`},
		http.StatusConflict,
	)
	setupCredentials(t, server.URL)

	err := runTaskStart(taskStartCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be started")
}

// --- Complete Tests ---

func TestRunTaskComplete_Success(t *testing.T) {
	resp := taskActionResponse{
		ID:     "shop-a1.1",
		Title:  "Cut mortises",
		Status: "done",
	}
	server := setupTestServer(t, "/api/v1/spaces/test-space/tasks/shop-a1.1/complete", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskComplete(taskCompleteCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskComplete_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/spaces/test-space/tasks/shop-a1.1/complete",
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
	server := setupTestServer(t, "/api/v1/spaces/test-space/tasks/shop-a1.1/pause", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskPause(taskPauseCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskPause_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/spaces/test-space/tasks/shop-a1.1/pause",
		http.MethodPost,
		errorResponse{Error: `task "shop-a1.1" cannot be paused: status is "open", must be "active"`},
		http.StatusConflict,
	)
	setupCredentials(t, server.URL)

	err := runTaskPause(taskPauseCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be paused")
}

// --- Resume Tests ---

func TestRunTaskResume_Success(t *testing.T) {
	resp := taskActionResponse{
		ID:     "shop-a1.1",
		Title:  "Cut mortises",
		Status: "active",
	}
	server := setupTestServer(t, "/api/v1/spaces/test-space/tasks/shop-a1.1/resume", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskResume(taskResumeCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskResume_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/spaces/test-space/tasks/shop-a1.1/resume",
		http.MethodPost,
		errorResponse{Error: `task "shop-a1.1" cannot be resumed: status is "active", must be "paused"`},
		http.StatusConflict,
	)
	setupCredentials(t, server.URL)

	err := runTaskResume(taskResumeCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be resumed")
}

// --- Skip Tests ---

func TestRunTaskSkip_Success(t *testing.T) {
	resp := taskActionResponse{
		ID:     "shop-a1.1",
		Title:  "Cut mortises",
		Status: "skipped",
	}
	server := setupTestServer(t, "/api/v1/spaces/test-space/tasks/shop-a1.1/skip", http.MethodPost, resp, http.StatusOK)
	setupCredentials(t, server.URL)

	err := runTaskSkip(taskSkipCmd, []string{"shop-a1.1"})
	require.NoError(t, err)
}

func TestRunTaskSkip_Conflict(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/spaces/test-space/tasks/shop-a1.1/skip",
		http.MethodPost,
		errorResponse{Error: `task "shop-a1.1" cannot be skipped: status is "done" (already terminal)`},
		http.StatusConflict,
	)
	setupCredentials(t, server.URL)

	err := runTaskSkip(taskSkipCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be skipped")
}

// --- No Credentials Test ---

func TestRunTaskAction_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runTaskComplete(taskCompleteCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

// --- Server Error Test ---

func TestRunTaskAction_ServerError(t *testing.T) {
	server := setupTestServer(t,
		"/api/v1/spaces/test-space/tasks/shop-a1.1/complete",
		http.MethodPost,
		errorResponse{Error: "database error"},
		http.StatusInternalServerError,
	)
	setupCredentials(t, server.URL)

	err := runTaskComplete(taskCompleteCmd, []string{"shop-a1.1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// --- Task Add Tests ---

func TestTaskAddCmd_FlagsDefined(t *testing.T) {
	for _, name := range []string{"station", "after", "type", "parent"} {
		flag := taskAddCmd.Flags().Lookup(name)
		require.NotNil(t, flag, "--%s flag should be defined", name)
	}
	// --type defaults to "task"
	typeFlag := taskAddCmd.Flags().Lookup("type")
	assert.Equal(t, "task", typeFlag.DefValue)
}

func TestRunTaskAdd_Success_WithParent(t *testing.T) {
	// When --parent is specified, no active task lookup needed.
	parentID := "job-abc"
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/spaces/test-space/tasks/"+parentID+"/children" && r.Method == http.MethodPost {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "Sand surfaces", body["title"])
			assert.Equal(t, "task", body["type"])

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(taskAddResponse{
				ID:       "job-abc.3",
				Title:    "Sand surfaces",
				Status:   "open",
				Type:     "task",
				ParentID: &parentID,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	// Set the --parent flag
	oldParent := taskAddParentID
	taskAddParentID = parentID
	defer func() { taskAddParentID = oldParent }()

	err := runTaskAdd(taskAddCmd, []string{"Sand surfaces"})
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "should only call the children endpoint, not active task lookup")
}

func TestRunTaskAdd_Success_InfersParentFromActiveTask(t *testing.T) {
	// When no --parent, infers job from current active task's parentId.
	parentID := "job-xyz"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Active task lookup
		if r.URL.Path == "/api/v1/spaces/test-space/tasks" && r.Method == http.MethodGet {
			assert.Contains(t, r.URL.RawQuery, "status=active")
			assert.Contains(t, r.URL.RawQuery, "assigneeId=user-123")
			json.NewEncoder(w).Encode(activeTaskListResponse{
				Items: []activeTaskResponse{
					{ID: "job-xyz.2", Title: "Cut tenons", Status: "active", ParentID: &parentID},
				},
				Total: 1,
			})
			return
		}

		// Create child task
		if r.URL.Path == "/api/v1/spaces/test-space/tasks/"+parentID+"/children" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(taskAddResponse{
				ID:       "job-xyz.3",
				Title:    "Extra sanding",
				Status:   "open",
				Type:     "task",
				ParentID: &parentID,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	// Clear --parent flag
	oldParent := taskAddParentID
	taskAddParentID = ""
	defer func() { taskAddParentID = oldParent }()

	err := runTaskAdd(taskAddCmd, []string{"Extra sanding"})
	require.NoError(t, err)
}

func TestRunTaskAdd_Success_WithStation(t *testing.T) {
	parentID := "job-abc"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Station list lookup
		if r.URL.Path == "/api/v1/spaces/test-space/stations" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]stationItem{
				{ID: "station-uuid-1", Name: "Table Saw"},
				{ID: "station-uuid-2", Name: "Workbench"},
			})
			return
		}

		// Create child task
		if r.URL.Path == "/api/v1/spaces/test-space/tasks/"+parentID+"/children" && r.Method == http.MethodPost {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "station-uuid-1", body["stationId"])

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(taskAddResponse{
				ID:       "job-abc.1",
				Title:    "Rip boards",
				Status:   "open",
				Type:     "task",
				ParentID: &parentID,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	oldParent := taskAddParentID
	oldStation := taskAddStation
	taskAddParentID = parentID
	taskAddStation = "Table Saw"
	defer func() {
		taskAddParentID = oldParent
		taskAddStation = oldStation
	}()

	err := runTaskAdd(taskAddCmd, []string{"Rip boards"})
	require.NoError(t, err)
}

func TestRunTaskAdd_Success_WithAfter(t *testing.T) {
	parentID := "job-abc"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/spaces/test-space/tasks/"+parentID+"/children" && r.Method == http.MethodPost {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "job-abc.1", body["afterId"])

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(taskAddResponse{
				ID:       "job-abc.2",
				Title:    "Sand after cutting",
				Status:   "open",
				Type:     "task",
				ParentID: &parentID,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	oldParent := taskAddParentID
	oldAfter := taskAddAfter
	taskAddParentID = parentID
	taskAddAfter = "job-abc.1"
	defer func() {
		taskAddParentID = oldParent
		taskAddAfter = oldAfter
	}()

	err := runTaskAdd(taskAddCmd, []string{"Sand after cutting"})
	require.NoError(t, err)
}

func TestRunTaskAdd_Success_JSONOutput(t *testing.T) {
	parentID := "job-abc"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(taskAddResponse{
			ID:       "job-abc.1",
			Title:    "New step",
			Status:   "open",
			Type:     "task",
			ParentID: &parentID,
		})
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	oldParent := taskAddParentID
	oldJSON := taskJSONFlag
	taskAddParentID = parentID
	taskJSONFlag = true
	defer func() {
		taskAddParentID = oldParent
		taskJSONFlag = oldJSON
	}()

	err := runTaskAdd(taskAddCmd, []string{"New step"})
	require.NoError(t, err)
}

func TestRunTaskAdd_StationNotFound(t *testing.T) {
	parentID := "job-abc"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/spaces/test-space/stations" {
			json.NewEncoder(w).Encode([]stationItem{
				{ID: "station-uuid-1", Name: "Table Saw"},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	oldParent := taskAddParentID
	oldStation := taskAddStation
	taskAddParentID = parentID
	taskAddStation = "Nonexistent Station"
	defer func() {
		taskAddParentID = oldParent
		taskAddStation = oldStation
	}()

	err := runTaskAdd(taskAddCmd, []string{"Some task"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunTaskAdd_NoActiveTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(activeTaskListResponse{
			Items: []activeTaskResponse{},
			Total: 0,
		})
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	oldParent := taskAddParentID
	taskAddParentID = ""
	defer func() { taskAddParentID = oldParent }()

	err := runTaskAdd(taskAddCmd, []string{"Some task"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active task found")
}

func TestRunTaskAdd_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	oldParent := taskAddParentID
	taskAddParentID = "job-abc"
	defer func() { taskAddParentID = oldParent }()

	err := runTaskAdd(taskAddCmd, []string{"Some task"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunTaskAdd_ServerError(t *testing.T) {
	parentID := "job-abc"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	oldParent := taskAddParentID
	taskAddParentID = parentID
	defer func() { taskAddParentID = oldParent }()

	err := runTaskAdd(taskAddCmd, []string{"Some task"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// --- Task Note Tests ---

func TestRunTaskNote_Success(t *testing.T) {
	activeTaskID := "job-abc.2"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Active task lookup
		if r.URL.Path == "/api/v1/spaces/test-space/tasks" && r.Method == http.MethodGet {
			parentID := "job-abc"
			json.NewEncoder(w).Encode(activeTaskListResponse{
				Items: []activeTaskResponse{
					{ID: activeTaskID, Title: "Cut tenons", Status: "active", ParentID: &parentID},
				},
				Total: 1,
			})
			return
		}

		// Add note
		if r.URL.Path == "/api/v1/spaces/test-space/tasks/"+activeTaskID+"/notes" && r.Method == http.MethodPost {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "Used deeper mortise", body["text"])

			note := "Used deeper mortise"
			json.NewEncoder(w).Encode(taskNoteResponse{
				ID:             activeTaskID,
				Title:          "Cut tenons",
				DeviationNotes: &note,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	err := runTaskNote(taskNoteCmd, []string{"Used deeper mortise"})
	require.NoError(t, err)
}

func TestRunTaskNote_Success_JSONOutput(t *testing.T) {
	activeTaskID := "job-abc.2"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/spaces/test-space/tasks" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(activeTaskListResponse{
				Items: []activeTaskResponse{
					{ID: activeTaskID, Title: "Cut tenons", Status: "active"},
				},
				Total: 1,
			})
			return
		}

		if r.URL.Path == "/api/v1/spaces/test-space/tasks/"+activeTaskID+"/notes" {
			note := "Some note"
			json.NewEncoder(w).Encode(taskNoteResponse{
				ID:             activeTaskID,
				Title:          "Cut tenons",
				DeviationNotes: &note,
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	oldJSON := taskJSONFlag
	taskJSONFlag = true
	defer func() { taskJSONFlag = oldJSON }()

	err := runTaskNote(taskNoteCmd, []string{"Some note"})
	require.NoError(t, err)
}

func TestRunTaskNote_NoActiveTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(activeTaskListResponse{
			Items: []activeTaskResponse{},
			Total: 0,
		})
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	err := runTaskNote(taskNoteCmd, []string{"Some note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active task found")
}

func TestRunTaskNote_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := runTaskNote(taskNoteCmd, []string{"Some note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunTaskNote_ServerError(t *testing.T) {
	activeTaskID := "job-abc.2"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/api/v1/spaces/test-space/tasks" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(activeTaskListResponse{
				Items: []activeTaskResponse{
					{ID: activeTaskID, Title: "Cut tenons", Status: "active"},
				},
				Total: 1,
			})
			return
		}

		if r.URL.Path == "/api/v1/spaces/test-space/tasks/"+activeTaskID+"/notes" {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	err := runTaskNote(taskNoteCmd, []string{"Some note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestRunTaskNote_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	t.Cleanup(server.Close)
	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	err := runTaskNote(taskNoteCmd, []string{"Some note"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}
