package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

func TestRecipeCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "recipe" {
			found = true
			break
		}
	}
	assert.True(t, found, "recipe command should be registered on root")
}

func TestRecipePourSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range recipeCmd.Commands() {
		if cmd.Name() == "pour" {
			found = true
			break
		}
	}
	assert.True(t, found, "pour subcommand should be registered on recipe")
}

func TestRecipePourHasVarFlag(t *testing.T) {
	flag := recipePourCmd.Flags().Lookup("var")
	require.NotNil(t, flag, "--var flag should be defined")
}

func TestRecipePourHasOrderFlag(t *testing.T) {
	flag := recipePourCmd.Flags().Lookup("order")
	require.NotNil(t, flag, "--order flag should be defined")
	assert.Equal(t, "", flag.DefValue)
}

func TestRecipePourHasJSONFlag(t *testing.T) {
	flag := recipePourCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestRunRecipePour_Success(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodGet:
			assert.Equal(t, "walnut-dining-table", r.URL.Query().Get("slug"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(recipeListResponse{
				Items: []recipeListItem{
					{ID: recipeID, Name: "Walnut Dining Table", Slug: "walnut-dining-table"},
				},
				Total: 1,
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/pour", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(pourResponse{
				ID:    "nori-abcd1234",
				Title: "Walnut Dining Table",
				Type:  "job",
			})

		case r.URL.Path == "/api/v1/tasks" && r.Method == http.MethodGet:
			assert.Equal(t, "nori-abcd1234", r.URL.Query().Get("parentId"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(taskListResponse{Total: 5})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	// Reset flags for this test.
	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"walnut-dining-table"})
	require.NoError(t, err)
}

func TestRunRecipePour_WithVarsAndOrder(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"
	orderID := "660e8400-e29b-41d4-a716-446655440001"

	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(recipeListResponse{
				Items: []recipeListItem{
					{ID: recipeID, Name: "Walnut Dining Table", Slug: "walnut-dining-table"},
				},
				Total: 1,
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/pour", recipeID) && r.Method == http.MethodPost:
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(pourResponse{
				ID:    "nori-abcd1234",
				Title: "Cherry Dining Table",
				Type:  "job",
			})

		case r.URL.Path == "/api/v1/tasks" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(taskListResponse{Total: 8})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	// Set flags for this test.
	pourVarFlags = []string{"wood_species=Cherry", "table_length=60"}
	pourOrderFlag = orderID
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"walnut-dining-table"})
	require.NoError(t, err)

	// Verify the vars were sent.
	require.NotNil(t, capturedBody)
	vars, ok := capturedBody["vars"].(map[string]interface{})
	require.True(t, ok, "vars should be a map")
	assert.Equal(t, "Cherry", vars["wood_species"])
	assert.Equal(t, "60", vars["table_length"])

	// Verify the order ID was sent.
	assert.Equal(t, orderID, capturedBody["orderId"])

	// Reset flags.
	pourVarFlags = nil
	pourOrderFlag = ""
}

func TestRunRecipePour_RecipeNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{},
			Total: 0,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunRecipePour_PourError(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(recipeListResponse{
				Items: []recipeListItem{
					{ID: recipeID, Name: "Walnut Dining Table", Slug: "walnut-dining-table"},
				},
				Total: 1,
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/pour", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(errorResponse{Error: "recipe has no published version"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no published version")
}

func TestRunRecipePour_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunRecipePour_InvalidVarFormat(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{
				{ID: recipeID, Name: "Walnut Dining Table", Slug: "walnut-dining-table"},
			},
			Total: 1,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	pourVarFlags = []string{"bad-format-no-equals"}
	pourOrderFlag = ""
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --var format")

	pourVarFlags = nil
}

func TestRunRecipePour_InvalidOrderUUID(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{
				{ID: recipeID, Name: "Walnut Dining Table", Slug: "walnut-dining-table"},
			},
			Total: 1,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	pourVarFlags = nil
	pourOrderFlag = "not-a-uuid"
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --order UUID")

	pourOrderFlag = ""
}

func TestRunRecipePour_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	pourVarFlags = nil
	pourOrderFlag = ""
	pourJSONFlag = false

	err := runRecipePour(recipePourCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}
