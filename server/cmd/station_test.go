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

func TestStationCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "station" {
			found = true
			break
		}
	}
	assert.True(t, found, "station command should be registered on root")
}

func TestStationSubcommandsRegistered(t *testing.T) {
	subcommands := map[string]bool{"list": false}
	for _, cmd := range stationCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}
	for name, found := range subcommands {
		assert.True(t, found, "%s subcommand should be registered on station", name)
	}
}

func TestStationCommandHasJSONFlag(t *testing.T) {
	flag := stationCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

// --- Station List Tests ---

func TestRunStationList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/stations", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]stationItem{
			{ID: "uuid-1", Name: "Milling", WIPCount: 2, WIPLimit: 3, BufferSize: 1, IsActive: true},
			{ID: "uuid-2", Name: "Assembly", WIPCount: 0, WIPLimit: 2, BufferSize: 0, IsActive: true},
			{ID: "uuid-3", Name: "Finishing", WIPCount: 1, WIPLimit: 1, BufferSize: 2, IsActive: false},
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = false

	err := runStationList(stationListCmd, nil)
	require.NoError(t, err)
}

func TestRunStationList_JSONOutput(t *testing.T) {
	stations := []stationItem{
		{ID: "uuid-1", Name: "Milling", WIPCount: 2, WIPLimit: 3, BufferSize: 1, IsActive: true},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stations)
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = true
	defer func() { stationJSONFlag = false }()

	err := runStationList(stationListCmd, nil)
	require.NoError(t, err)
}

func TestRunStationList_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]stationItem{})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = false

	err := runStationList(stationListCmd, nil)
	require.NoError(t, err)
}

func TestRunStationList_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	stationJSONFlag = false

	err := runStationList(stationListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunStationList_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = false

	err := runStationList(stationListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestRunStationList_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	stationJSONFlag = false

	err := runStationList(stationListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunStationList_ConnectionFailed(t *testing.T) {
	// Start a server and immediately close it to get an unreachable URL
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := server.URL
	server.Close()

	setupCredentials(t, unreachableURL)
	stationJSONFlag = false

	err := runStationList(stationListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to server")
}
