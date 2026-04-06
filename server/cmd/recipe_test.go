package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// --- Recipe Create Tests ---

func TestRecipeCreateSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range recipeCmd.Commands() {
		if cmd.Name() == "create" {
			found = true
			break
		}
	}
	assert.True(t, found, "create subcommand should be registered on recipe")
}

func TestRecipeCreateHasFromTOMLFlag(t *testing.T) {
	flag := recipeCreateCmd.Flags().Lookup("from-toml")
	require.NotNil(t, flag, "--from-toml flag should be defined")
}

func TestRecipeCreateHasNameFlag(t *testing.T) {
	flag := recipeCreateCmd.Flags().Lookup("name")
	require.NotNil(t, flag, "--name flag should be defined")
	assert.Equal(t, "", flag.DefValue)
}

func TestRecipeCreateHasJSONFlag(t *testing.T) {
	flag := recipeCreateCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

// writeTOMLFile creates a temp TOML file and returns its path.
func writeTOMLFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "recipe.formula.toml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

const sampleTOML = `formula = "walnut-dining-table"
description = "A walnut dining table recipe"
version = 1
type = "workflow"

[[steps]]
id = "cut"
title = "Cut lumber"
`

func TestRunRecipeCreate_Success(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"
	versionID := 1

	var capturedCreateBody map[string]interface{}
	var capturedVersionBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodPost:
			json.NewDecoder(r.Body).Decode(&capturedCreateBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createRecipeResponse{
				ID:   recipeID,
				Name: "walnut-dining-table",
				Slug: "walnut-dining-table",
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodPost:
			json.NewDecoder(r.Body).Decode(&capturedVersionBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createVersionResponse{
				ID:            versionID,
				VersionNumber: 1,
				Status:        "draft",
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions/%d/publish", recipeID, versionID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(createVersionResponse{
				ID:            versionID,
				VersionNumber: 1,
				Status:        "published",
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tomlPath := writeTOMLFile(t, sampleTOML)
	setupCredentials(t, server.URL)

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.NoError(t, err)

	// Verify recipe name was extracted from TOML.
	require.NotNil(t, capturedCreateBody)
	assert.Equal(t, "walnut-dining-table", capturedCreateBody["name"])

	// Verify TOML content was sent as version content.
	require.NotNil(t, capturedVersionBody)
	assert.Contains(t, capturedVersionBody["content"], "walnut-dining-table")
	assert.Equal(t, "Initial import from TOML file", capturedVersionBody["changeSummary"])

	// Reset flags.
	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_WithNameOverride(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	var capturedCreateBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodPost:
			json.NewDecoder(r.Body).Decode(&capturedCreateBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createRecipeResponse{
				ID:   recipeID,
				Name: "Custom Name",
				Slug: "custom-name",
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createVersionResponse{ID: 1, VersionNumber: 1, Status: "draft"})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions/1/publish", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(createVersionResponse{ID: 1, VersionNumber: 1, Status: "published"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tomlPath := writeTOMLFile(t, sampleTOML)
	setupCredentials(t, server.URL)

	createFromTOMLFlag = tomlPath
	createNameFlag = "Custom Name"
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.NoError(t, err)

	// --name should override the TOML formula field.
	require.NotNil(t, capturedCreateBody)
	assert.Equal(t, "Custom Name", capturedCreateBody["name"])

	createFromTOMLFlag = ""
	createNameFlag = ""
}

func TestRunRecipeCreate_JSONOutput(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createRecipeResponse{
				ID:   recipeID,
				Name: "walnut-dining-table",
				Slug: "walnut-dining-table",
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createVersionResponse{ID: 1, VersionNumber: 1, Status: "draft"})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions/1/publish", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(createVersionResponse{ID: 1, VersionNumber: 1, Status: "published"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tomlPath := writeTOMLFile(t, sampleTOML)
	setupCredentials(t, server.URL)

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = true

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.NoError(t, err)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	var output map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))
	assert.Equal(t, recipeID, output["recipeId"])
	assert.Equal(t, "walnut-dining-table", output["name"])
	assert.Equal(t, "walnut-dining-table", output["slug"])
	assert.Equal(t, float64(1), output["version"])
	assert.Equal(t, "published", output["status"])

	createFromTOMLFlag = ""
	createJSONFlag = false
}

func TestRunRecipeCreate_FileNotFound(t *testing.T) {
	setupCredentials(t, "http://localhost:9999")

	createFromTOMLFlag = "/nonexistent/path/recipe.toml"
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read TOML file")

	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_EmptyFile(t *testing.T) {
	tomlPath := writeTOMLFile(t, "")
	setupCredentials(t, "http://localhost:9999")

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TOML file is empty")

	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_NoRecipeName(t *testing.T) {
	// TOML without a "formula" key and no --name flag.
	tomlPath := writeTOMLFile(t, `version = 1
type = "workflow"
`)

	setupCredentials(t, "http://localhost:9999")

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not determine recipe name")

	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	tomlPath := writeTOMLFile(t, sampleTOML)

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")

	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	tomlPath := writeTOMLFile(t, sampleTOML)
	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")

	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "internal server error"})
	}))
	defer server.Close()

	tomlPath := writeTOMLFile(t, sampleTOML)
	setupCredentials(t, server.URL)

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create recipe failed")

	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_VersionCreationFails(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createRecipeResponse{
				ID:   recipeID,
				Name: "walnut-dining-table",
				Slug: "walnut-dining-table",
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "database error"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tomlPath := writeTOMLFile(t, sampleTOML)
	setupCredentials(t, server.URL)

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version creation failed")

	createFromTOMLFlag = ""
}

func TestRunRecipeCreate_PublishFails(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createRecipeResponse{
				ID:   recipeID,
				Name: "walnut-dining-table",
				Slug: "walnut-dining-table",
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(createVersionResponse{ID: 1, VersionNumber: 1, Status: "draft"})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions/1/publish", recipeID) && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "publish failed"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tomlPath := writeTOMLFile(t, sampleTOML)
	setupCredentials(t, server.URL)

	createFromTOMLFlag = tomlPath
	createNameFlag = ""
	createJSONFlag = false

	err := runRecipeCreate(recipeCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish failed")

	createFromTOMLFlag = ""
}

// --- Recipe Publish Tests ---

func TestRecipePublishSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range recipeCmd.Commands() {
		if cmd.Name() == "publish" {
			found = true
			break
		}
	}
	assert.True(t, found, "publish subcommand should be registered on recipe")
}

func TestRecipePublishHasJSONFlag(t *testing.T) {
	flag := recipePublishCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestRecipePublishRequiresArg(t *testing.T) {
	err := recipePublishCmd.Args(recipePublishCmd, []string{})
	assert.Error(t, err, "should require exactly one argument")
}

func TestRunRecipePublish_Success(t *testing.T) {
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

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionListResponse{
				Items: []versionListItem{
					{ID: 3, VersionNumber: 3, Status: "draft"},
					{ID: 2, VersionNumber: 2, Status: "published"},
					{ID: 1, VersionNumber: 1, Status: "archived"},
				},
				Total: 3,
			})

		case r.URL.Path == "/api/v1/recipe-versions/3/publish" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(publishVersionResponse{
				ID:            3,
				VersionNumber: 3,
				Status:        "published",
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	publishJSONFlag = false

	err := runRecipePublish(recipePublishCmd, []string{"walnut-dining-table"})
	require.NoError(t, err)
}

func TestRunRecipePublish_JSONOutput(t *testing.T) {
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

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionListResponse{
				Items: []versionListItem{
					{ID: 2, VersionNumber: 2, Status: "draft"},
					{ID: 1, VersionNumber: 1, Status: "published"},
				},
				Total: 2,
			})

		case r.URL.Path == "/api/v1/recipe-versions/2/publish" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(publishVersionResponse{
				ID:            2,
				VersionNumber: 2,
				Status:        "published",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	publishJSONFlag = true

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRecipePublish(recipePublishCmd, []string{"walnut-dining-table"})
	require.NoError(t, err)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	var output map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))
	assert.Equal(t, "walnut-dining-table", output["recipeSlug"])
	assert.Equal(t, float64(2), output["versionId"])
	assert.Equal(t, float64(2), output["versionNumber"])
	assert.Equal(t, "published", output["status"])

	publishJSONFlag = false
}

func TestRunRecipePublish_NoDraftVersion(t *testing.T) {
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

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionListResponse{
				Items: []versionListItem{
					{ID: 1, VersionNumber: 1, Status: "published"},
				},
				Total: 1,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	publishJSONFlag = false

	err := runRecipePublish(recipePublishCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no draft version")
}

func TestRunRecipePublish_RecipeNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{},
			Total: 0,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	publishJSONFlag = false

	err := runRecipePublish(recipePublishCmd, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunRecipePublish_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	publishJSONFlag = false

	err := runRecipePublish(recipePublishCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunRecipePublish_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	publishJSONFlag = false

	err := runRecipePublish(recipePublishCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunRecipePublish_PublishServerError(t *testing.T) {
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

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(versionListResponse{
				Items: []versionListItem{
					{ID: 2, VersionNumber: 2, Status: "draft"},
				},
				Total: 1,
			})

		case r.URL.Path == "/api/v1/recipe-versions/2/publish" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errorResponse{Error: "database error"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	publishJSONFlag = false

	err := runRecipePublish(recipePublishCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish failed")
}

// --- Recipe List Tests ---

func TestRecipeListSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range recipeCmd.Commands() {
		if cmd.Name() == "list" {
			found = true
			break
		}
	}
	assert.True(t, found, "list subcommand should be registered on recipe")
}

func TestRecipeListHasJSONFlag(t *testing.T) {
	flag := recipeListCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestRecipeListHasActiveFlag(t *testing.T) {
	flag := recipeListCmd.Flags().Lookup("active")
	require.NotNil(t, flag, "--active flag should be defined")
	assert.Equal(t, "", flag.DefValue)
}

func TestRunRecipeList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/recipes", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{
				{ID: "id-1", Name: "Walnut Dining Table", Slug: "walnut-dining-table", IsActive: true},
				{ID: "id-2", Name: "Cherry Side Table", Slug: "cherry-side-table", IsActive: true},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	listJSONFlag = false
	listActiveFlag = ""

	err := runRecipeList(recipeListCmd, nil)
	require.NoError(t, err)
}

func TestRunRecipeList_WithActiveFilter(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("isActive")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{
				{ID: "id-1", Name: "Walnut Dining Table", Slug: "walnut-dining-table", IsActive: true},
			},
			Total: 1,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	listJSONFlag = false
	listActiveFlag = "true"
	defer func() { listActiveFlag = "" }()

	err := runRecipeList(recipeListCmd, nil)
	require.NoError(t, err)
	assert.Equal(t, "true", receivedQuery)
}

func TestRunRecipeList_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{},
			Total: 0,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	listJSONFlag = false
	listActiveFlag = ""

	err := runRecipeList(recipeListCmd, nil)
	require.NoError(t, err)
}

func TestRunRecipeList_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recipeListResponse{
			Items: []recipeListItem{
				{ID: "id-1", Name: "Walnut Dining Table", Slug: "walnut-dining-table", IsActive: true},
			},
			Total: 1,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	listJSONFlag = true
	listActiveFlag = ""
	defer func() { listJSONFlag = false }()

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRecipeList(recipeListCmd, nil)
	require.NoError(t, err)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	var output []map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))
	require.Len(t, output, 1)
	assert.Equal(t, "walnut-dining-table", output[0]["slug"])
	assert.Equal(t, "Walnut Dining Table", output[0]["name"])
	assert.Equal(t, true, output[0]["isActive"])
}

func TestRunRecipeList_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	listJSONFlag = false
	listActiveFlag = ""

	err := runRecipeList(recipeListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunRecipeList_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	listJSONFlag = false
	listActiveFlag = ""

	err := runRecipeList(recipeListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestRunRecipeList_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	listJSONFlag = false
	listActiveFlag = ""

	err := runRecipeList(recipeListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

// --- Recipe Show Tests ---

func TestRecipeShowSubcommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range recipeCmd.Commands() {
		if cmd.Name() == "show" {
			found = true
			break
		}
	}
	assert.True(t, found, "show subcommand should be registered on recipe")
}

func TestRecipeShowHasJSONFlag(t *testing.T) {
	flag := recipeShowCmd.Flags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestRecipeShowRequiresArg(t *testing.T) {
	err := recipeShowCmd.Args(recipeShowCmd, []string{})
	assert.Error(t, err, "should require exactly one argument")
}

func TestRunRecipeShow_Success(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodGet:
			assert.Equal(t, "walnut-dining-table", r.URL.Query().Get("slug"))
			json.NewEncoder(w).Encode(struct {
				Items []recipeShowDetail `json:"items"`
				Total int64              `json:"total"`
			}{
				Items: []recipeShowDetail{
					{
						ID:       recipeID,
						Name:     "Walnut Dining Table",
						Slug:     "walnut-dining-table",
						IsActive: true,
					},
				},
				Total: 1,
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodGet:
			changeSummary := "Initial import"
			json.NewEncoder(w).Encode(struct {
				Items []recipeShowVersion `json:"items"`
				Total int                 `json:"total"`
			}{
				Items: []recipeShowVersion{
					{
						ID:            1,
						VersionNumber: 1,
						Status:        "published",
						Content: `formula = "walnut-dining-table"

[[steps]]
id = "cut"
title = "Cut lumber"

[[steps]]
id = "sand"
title = "Sand surfaces"
`,
						ChangeSummary: &changeSummary,
					},
				},
				Total: 1,
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	showJSONFlag = false

	err := runRecipeShow(recipeShowCmd, []string{"walnut-dining-table"})
	require.NoError(t, err)
	assert.Equal(t, 2, requestCount, "should make two requests: recipe + versions")
}

func TestRunRecipeShow_JSONOutput(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(struct {
				Items []recipeShowDetail `json:"items"`
				Total int64              `json:"total"`
			}{
				Items: []recipeShowDetail{
					{
						ID:       recipeID,
						Name:     "Walnut Dining Table",
						Slug:     "walnut-dining-table",
						IsActive: true,
					},
				},
				Total: 1,
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(struct {
				Items []recipeShowVersion `json:"items"`
				Total int                 `json:"total"`
			}{
				Items: []recipeShowVersion{
					{
						ID:            1,
						VersionNumber: 1,
						Status:        "published",
						Content: `formula = "walnut-dining-table"

[[steps]]
id = "cut"
title = "Cut lumber"
`,
					},
				},
				Total: 1,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	showJSONFlag = true
	defer func() { showJSONFlag = false }()

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runRecipeShow(recipeShowCmd, []string{"walnut-dining-table"})
	require.NoError(t, err)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	var output map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &output))
	assert.Equal(t, "walnut-dining-table", output["slug"])
	assert.Equal(t, "Walnut Dining Table", output["name"])
	assert.Equal(t, true, output["isActive"])

	// Verify steps were extracted.
	steps, ok := output["steps"].([]interface{})
	require.True(t, ok, "steps should be an array")
	require.Len(t, steps, 1)
	step := steps[0].(map[string]interface{})
	assert.Equal(t, "cut", step["id"])
	assert.Equal(t, "Cut lumber", step["title"])
}

func TestRunRecipeShow_RecipeNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Items []recipeShowDetail `json:"items"`
			Total int64              `json:"total"`
		}{
			Items: []recipeShowDetail{},
			Total: 0,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	showJSONFlag = false

	err := runRecipeShow(recipeShowCmd, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunRecipeShow_NoPublishedVersion(t *testing.T) {
	recipeID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/recipes" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(struct {
				Items []recipeShowDetail `json:"items"`
				Total int64              `json:"total"`
			}{
				Items: []recipeShowDetail{
					{
						ID:       recipeID,
						Name:     "Walnut Dining Table",
						Slug:     "walnut-dining-table",
						IsActive: true,
					},
				},
				Total: 1,
			})

		case r.URL.Path == fmt.Sprintf("/api/v1/recipes/%s/versions", recipeID) && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(struct {
				Items []recipeShowVersion `json:"items"`
				Total int                 `json:"total"`
			}{
				Items: []recipeShowVersion{
					{ID: 1, VersionNumber: 1, Status: "draft", Content: "some content"},
				},
				Total: 1,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	showJSONFlag = false

	// Should succeed — just shows "No published version."
	err := runRecipeShow(recipeShowCmd, []string{"walnut-dining-table"})
	require.NoError(t, err)
}

func TestRunRecipeShow_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	showJSONFlag = false

	err := runRecipeShow(recipeShowCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

func TestRunRecipeShow_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)

	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)

	showJSONFlag = false

	err := runRecipeShow(recipeShowCmd, []string{"walnut-dining-table"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}
