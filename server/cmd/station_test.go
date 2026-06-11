package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/pflag"
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
	subcommands := map[string]bool{"list": false, "create": false}
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
		assert.Equal(t, "/api/v1/spaces/test-space/stations", r.URL.Path)
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

// --- Station Create Tests ---

func TestRunStationCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/stations", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "Milling", body["name"])
		assert.Equal(t, float64(1), body["wipLimit"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(stationItem{
			ID:       "uuid-1",
			Name:     "Milling",
			WIPLimit: 1,
			IsActive: true,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = false
	stationCreateWIPLimit = 1
	stationCreateDescription = ""

	err := runStationCreate(stationCreateCmd, []string{"Milling"})
	require.NoError(t, err)
}

func TestRunStationCreate_WithFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "Assembly", body["name"])
		assert.Equal(t, float64(5), body["wipLimit"])
		assert.Equal(t, "Main assembly area", body["description"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		desc := "Main assembly area"
		json.NewEncoder(w).Encode(stationItem{
			ID:          "uuid-2",
			Name:        "Assembly",
			Description: &desc,
			WIPLimit:    5,
			IsActive:    true,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = false
	stationCreateWIPLimit = 5
	stationCreateDescription = "Main assembly area"

	err := runStationCreate(stationCreateCmd, []string{"Assembly"})
	require.NoError(t, err)
}

func TestRunStationCreate_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(stationItem{
			ID:       "uuid-1",
			Name:     "Milling",
			WIPLimit: 1,
			IsActive: true,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = true
	stationCreateWIPLimit = 1
	stationCreateDescription = ""
	defer func() { stationJSONFlag = false }()

	err := runStationCreate(stationCreateCmd, []string{"Milling"})
	require.NoError(t, err)
}

func TestRunStationCreate_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	stationJSONFlag = false
	stationCreateWIPLimit = 1
	stationCreateDescription = ""

	err := runStationCreate(stationCreateCmd, []string{"Milling"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunStationCreate_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(errorResponse{Error: "admin access required"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = false
	stationCreateWIPLimit = 1
	stationCreateDescription = ""

	err := runStationCreate(stationCreateCmd, []string{"Milling"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin access required")
}

func TestRunStationCreate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "failed to create station"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	stationJSONFlag = false
	stationCreateWIPLimit = 1
	stationCreateDescription = ""

	err := runStationCreate(stationCreateCmd, []string{"Milling"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create station")
}

func TestRunStationCreate_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	stationJSONFlag = false
	stationCreateWIPLimit = 1
	stationCreateDescription = ""

	err := runStationCreate(stationCreateCmd, []string{"Milling"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunStationCreate_ConnectionFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := server.URL
	server.Close()

	setupCredentials(t, unreachableURL)
	stationJSONFlag = false
	stationCreateWIPLimit = 1
	stationCreateDescription = ""

	err := runStationCreate(stationCreateCmd, []string{"Milling"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to server")
}

func TestStationCreateHasFlags(t *testing.T) {
	wipFlag := stationCreateCmd.Flags().Lookup("wip-limit")
	require.NotNil(t, wipFlag, "--wip-limit flag should be defined")
	assert.Equal(t, "1", wipFlag.DefValue)

	descFlag := stationCreateCmd.Flags().Lookup("description")
	require.NotNil(t, descFlag, "--description flag should be defined")
	assert.Equal(t, "", descFlag.DefValue)
}

// --- Station Update Tests ---

// setStationUpdateFlags sets flags on stationUpdateCmd and registers cleanup
// to reset them so tests do not leak state.
func setStationUpdateFlags(t *testing.T, values map[string]string) {
	t.Helper()
	for name, value := range values {
		require.NoError(t, stationUpdateCmd.Flags().Set(name, value))
	}
	t.Cleanup(func() {
		stationUpdateCostsHour = ""
		stationUpdateClearCostsHour = false
		stationUpdateCmd.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
		})
	})
}

func TestStationUpdateSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range stationCmd.Commands() {
		if cmd.Name() == "update" {
			found = true
			break
		}
	}
	assert.True(t, found, "update subcommand should be registered on station")
}

func TestRunStationUpdate_SetCostsHour(t *testing.T) {
	rate := "45"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/stations/uuid-1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Contains(t, body, "costsHour")
		assert.Equal(t, `"45"`, string(body["costsHour"]))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stationItem{
			ID: "uuid-1", Name: "CNC", WIPLimit: 1, IsActive: true, CostsHour: &rate,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	setStationUpdateFlags(t, map[string]string{"costs-hour": "45.00"})
	stationJSONFlag = false

	err := runStationUpdate(stationUpdateCmd, []string{"uuid-1"})
	require.NoError(t, err)
}

func TestRunStationUpdate_ClearCostsHour(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/stations/uuid-1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Contains(t, body, "costsHour")
		assert.Equal(t, "null", string(body["costsHour"]), "clearing should send explicit null")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stationItem{
			ID: "uuid-1", Name: "CNC", WIPLimit: 1, IsActive: true, CostsHour: nil,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	setStationUpdateFlags(t, map[string]string{"clear-costs-hour": "true"})
	stationJSONFlag = false

	err := runStationUpdate(stationUpdateCmd, []string{"uuid-1"})
	require.NoError(t, err)
}

func TestRunStationUpdate_InvalidCostsHour(t *testing.T) {
	setupCredentials(t, "http://localhost:1") // never reached
	setStationUpdateFlags(t, map[string]string{"costs-hour": "abc"})

	err := runStationUpdate(stationUpdateCmd, []string{"uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --costs-hour")
}

func TestRunStationUpdate_NegativeCostsHour(t *testing.T) {
	setupCredentials(t, "http://localhost:1") // never reached
	setStationUpdateFlags(t, map[string]string{"costs-hour": "-5"})

	err := runStationUpdate(stationUpdateCmd, []string{"uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be negative")
}

func TestRunStationUpdate_NoFlags(t *testing.T) {
	setupCredentials(t, "http://localhost:1") // never reached
	setStationUpdateFlags(t, map[string]string{})

	err := runStationUpdate(stationUpdateCmd, []string{"uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}

func TestRunStationUpdate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "costsHour must not be negative"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	setStationUpdateFlags(t, map[string]string{"costs-hour": "45.00"})

	err := runStationUpdate(stationUpdateCmd, []string{"uuid-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "costsHour must not be negative")
}
