package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

const testMaterialSpaceID = "11111111-1111-1111-1111-111111111111"

// setupMaterialCredentials stores credentials with a space ID so that
// material commands can build space-scoped URLs.
func setupMaterialCredentials(t *testing.T, serverURL string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("NORI_SPACE", "")
	spaceFlag = ""

	creds := &cli.Credentials{
		ServerURL:   serverURL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
		SpaceID:     testMaterialSpaceID,
	}
	require.NoError(t, cli.SaveCredentials(creds))
}

// resetFlags clears values and Changed state on a command's flags so tests
// don't leak flag state into each other.
func resetFlags(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

func setFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	require.NoError(t, cmd.Flags().Set(name, value))
}

// --- Registration tests ---

func TestMaterialCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "material" {
			found = true
			break
		}
	}
	assert.True(t, found, "material command should be registered on root")
}

func TestMaterialSubcommandsRegistered(t *testing.T) {
	subcommands := map[string]bool{"create": false, "list": false, "update": false, "delete": false}
	for _, cmd := range materialCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}
	for name, found := range subcommands {
		assert.True(t, found, "%s subcommand should be registered on material", name)
	}
}

func TestMaterialCommandHasJSONFlag(t *testing.T) {
	flag := materialCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

// --- Create tests ---

func TestRunMaterialCreate_Success(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/"+testMaterialSpaceID+"/materials", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &receivedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		cost := "14.5"
		supplier := "Goby Walnut"
		json.NewEncoder(w).Encode(materialItem{
			ID:       "mat-1",
			Name:     "8/4 Walnut",
			Category: "lumber",
			Unit:     "board_feet",
			UnitCost: &cost,
			Supplier: &supplier,
			IsActive: true,
		})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)
	resetFlags(t, materialCreateCmd)
	setFlag(t, materialCreateCmd, "name", "8/4 Walnut")
	setFlag(t, materialCreateCmd, "unit", "board_feet")
	setFlag(t, materialCreateCmd, "cost", "14.50")
	setFlag(t, materialCreateCmd, "supplier", "Goby Walnut")
	setFlag(t, materialCreateCmd, "category", "lumber")

	err := runMaterialCreate(materialCreateCmd, nil)
	require.NoError(t, err)

	assert.Equal(t, "8/4 Walnut", receivedBody["name"])
	assert.Equal(t, "board_feet", receivedBody["unit"])
	assert.Equal(t, "14.5", receivedBody["unitCost"])
	assert.Equal(t, "Goby Walnut", receivedBody["supplier"])
	assert.Equal(t, "lumber", receivedBody["category"])
}

func TestRunMaterialCreate_ServerValidationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "unit is required"})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)
	resetFlags(t, materialCreateCmd)
	setFlag(t, materialCreateCmd, "name", "8/4 Walnut")

	err := runMaterialCreate(materialCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unit is required")
}

func TestRunMaterialCreate_InvalidCost(t *testing.T) {
	setupMaterialCredentials(t, "http://localhost:1")
	resetFlags(t, materialCreateCmd)
	setFlag(t, materialCreateCmd, "name", "8/4 Walnut")
	setFlag(t, materialCreateCmd, "unit", "board_feet")
	setFlag(t, materialCreateCmd, "cost", "abc")

	err := runMaterialCreate(materialCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --cost")
}

func TestRunMaterialCreate_NegativeCost(t *testing.T) {
	setupMaterialCredentials(t, "http://localhost:1")
	resetFlags(t, materialCreateCmd)
	setFlag(t, materialCreateCmd, "name", "8/4 Walnut")
	setFlag(t, materialCreateCmd, "unit", "board_feet")
	setFlag(t, materialCreateCmd, "cost", "-1")

	err := runMaterialCreate(materialCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be >= 0")
}

func TestRunMaterialCreate_NoSpaceSelected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)

	// Overwrite credentials without a space ID.
	creds := &cli.Credentials{
		ServerURL:   server.URL,
		AccessToken: "test-token",
		UserID:      "user-123",
		UserEmail:   "test@example.com",
	}
	require.NoError(t, cli.SaveCredentials(creds))

	resetFlags(t, materialCreateCmd)
	setFlag(t, materialCreateCmd, "name", "8/4 Walnut")
	setFlag(t, materialCreateCmd, "unit", "board_feet")

	err := runMaterialCreate(materialCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no space selected")
}

// --- List tests ---

func TestRunMaterialList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/"+testMaterialSpaceID+"/materials", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		cost := "14.5"
		json.NewEncoder(w).Encode([]materialItem{
			{ID: "mat-1", Name: "8/4 Walnut", Category: "lumber", Unit: "board_feet", UnitCost: &cost, IsActive: true},
			{ID: "mat-2", Name: "Titebond III", Category: "other", Unit: "oz", IsActive: true},
		})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)
	resetFlags(t, materialListCmd)
	materialJSONFlag = false

	err := runMaterialList(materialListCmd, nil)
	require.NoError(t, err)
}

func TestRunMaterialList_SearchParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "walnut", r.URL.Query().Get("search"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]materialItem{})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)
	resetFlags(t, materialListCmd)
	setFlag(t, materialListCmd, "search", "walnut")
	materialJSONFlag = false

	err := runMaterialList(materialListCmd, nil)
	require.NoError(t, err)
}

func TestRunMaterialList_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)
	resetFlags(t, materialListCmd)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	err := runMaterialList(materialListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

// --- Update tests ---

func TestRunMaterialUpdate_Success(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/"+testMaterialSpaceID+"/materials/mat-1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &receivedBody))

		w.Header().Set("Content-Type", "application/json")
		cost := "15"
		json.NewEncoder(w).Encode(materialItem{
			ID:       "mat-1",
			Name:     "8/4 Walnut",
			Category: "lumber",
			Unit:     "board_feet",
			UnitCost: &cost,
			IsActive: true,
		})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)
	resetFlags(t, materialUpdateCmd)
	setFlag(t, materialUpdateCmd, "cost", "15.00")

	err := runMaterialUpdate(materialUpdateCmd, []string{"mat-1"})
	require.NoError(t, err)

	// Only the changed flag should be in the body.
	assert.Equal(t, "15", receivedBody["unitCost"])
	assert.Len(t, receivedBody, 1)
}

func TestRunMaterialUpdate_NoFlags(t *testing.T) {
	setupMaterialCredentials(t, "http://localhost:1")
	resetFlags(t, materialUpdateCmd)

	err := runMaterialUpdate(materialUpdateCmd, []string{"mat-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields to update")
}

func TestRunMaterialUpdate_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse{Error: "material not found"})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)
	resetFlags(t, materialUpdateCmd)
	setFlag(t, materialUpdateCmd, "cost", "15.00")

	err := runMaterialUpdate(materialUpdateCmd, []string{"missing-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "material not found")
}

// --- Delete tests ---

func TestRunMaterialDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/"+testMaterialSpaceID+"/materials/mat-1", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)

	err := runMaterialDelete(materialDeleteCmd, []string{"mat-1"})
	require.NoError(t, err)
}

func TestRunMaterialDelete_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse{Error: "material not found"})
	}))
	defer server.Close()

	setupMaterialCredentials(t, server.URL)

	err := runMaterialDelete(materialDeleteCmd, []string{"missing-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "material not found")
}
