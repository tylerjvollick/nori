package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

var variantFlagNames = []string{"name", "recipe", "price", "vars"}

// setVariantFlags resets the shared variant flag state, then applies the
// provided values via the command's flag set so Changed() reflects what a
// real invocation would see. Cleanup restores the pristine state.
func setVariantFlags(t *testing.T, cmd *cobra.Command, values map[string]string) {
	t.Helper()

	reset := func() {
		variantName, variantRecipe, variantPrice, variantVars = "", "", "", ""
		for _, name := range variantFlagNames {
			cmd.Flags().Lookup(name).Changed = false
		}
	}
	reset()
	t.Cleanup(reset)

	for k, v := range values {
		require.NoError(t, cmd.Flags().Set(k, v))
	}
}

// --- Registration tests ---

func TestProductCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "product" {
			found = true
			break
		}
	}
	assert.True(t, found, "product command should be registered on root")
}

func TestProductSubcommandsRegistered(t *testing.T) {
	subcommands := map[string]bool{"create": false, "list": false, "variant": false}
	for _, cmd := range productCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}
	for name, found := range subcommands {
		assert.True(t, found, "%s subcommand should be registered on product", name)
	}
}

func TestProductVariantSubcommandsRegistered(t *testing.T) {
	subcommands := map[string]bool{"create": false, "update": false, "delete": false}
	for _, cmd := range productVariantCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}
	for name, found := range subcommands {
		assert.True(t, found, "%s subcommand should be registered on product variant", name)
	}
}

func TestProductCommandHasJSONFlag(t *testing.T) {
	flag := productCmd.PersistentFlags().Lookup("json")
	require.NotNil(t, flag, "--json flag should be defined")
	assert.Equal(t, "false", flag.DefValue)
}

func TestProductVariantCreateHasFlags(t *testing.T) {
	for _, name := range variantFlagNames {
		flag := productVariantCreateCmd.Flags().Lookup(name)
		require.NotNil(t, flag, "--%s flag should be defined on variant create", name)
	}
}

// --- product create ---

func TestRunProductCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/products", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "Dining Table", body["name"])
		assert.Equal(t, "Solid hardwood", body["description"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(productItem{
			ID:       "uuid-1",
			Name:     "Dining Table",
			IsActive: true,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false
	productCreateName = "Dining Table"
	productCreateDescription = "Solid hardwood"
	defer func() { productCreateName = ""; productCreateDescription = "" }()

	err := runProductCreate(productCreateCmd, nil)
	require.NoError(t, err)
}

func TestRunProductCreate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "name is required"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false
	productCreateName = ""

	err := runProductCreate(productCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestRunProductCreate_NoCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	productJSONFlag = false
	productCreateName = "Dining Table"
	defer func() { productCreateName = "" }()

	err := runProductCreate(productCreateCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
}

// --- product list ---

func TestRunProductList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/products", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]productItem{
			{ID: "uuid-1", Name: "Dining Table", IsActive: true, Variants: []productVariantItem{
				{ID: "v-1", ProductID: "uuid-1", Name: "Walnut / Spray PU", IsActive: true},
			}},
			{ID: "uuid-2", Name: "Dining Chair", IsActive: false},
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false

	err := runProductList(productListCmd, nil)
	require.NoError(t, err)
}

func TestRunProductList_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]productItem{
			{ID: "uuid-1", Name: "Dining Table", IsActive: true},
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = true
	defer func() { productJSONFlag = false }()

	err := runProductList(productListCmd, nil)
	require.NoError(t, err)
}

func TestRunProductList_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]productItem{})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false

	err := runProductList(productListCmd, nil)
	require.NoError(t, err)
}

func TestRunProductList_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	origFunc := cli.SetIsInteractiveFunc(func() bool { return false })
	defer cli.SetIsInteractiveFunc(origFunc)
	productJSONFlag = false

	err := runProductList(productListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication expired or invalid")
}

func TestRunProductList_ConnectionFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := server.URL
	server.Close()

	setupCredentials(t, unreachableURL)
	productJSONFlag = false

	err := runProductList(productListCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to server")
}

// --- product variant create ---

func TestRunProductVariantCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/products/prod-1/variants", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "Walnut / Spray PU", body["name"])
		assert.Equal(t, "recipe-1", body["recipeId"])
		assert.Equal(t, "4200", body["price"])
		vars, ok := body["recipeVariables"].(map[string]interface{})
		require.True(t, ok, "recipeVariables should be a JSON object")
		assert.Equal(t, "Walnut", vars["wood_species"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		price := "4200"
		recipeID := "recipe-1"
		json.NewEncoder(w).Encode(productVariantItem{
			ID:        "v-1",
			ProductID: "prod-1",
			Name:      "Walnut / Spray PU",
			RecipeID:  &recipeID,
			Price:     &price,
			IsActive:  true,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false
	setVariantFlags(t, productVariantCreateCmd, map[string]string{
		"name":   "Walnut / Spray PU",
		"recipe": "recipe-1",
		"price":  "4200",
		"vars":   `{"wood_species":"Walnut"}`,
	})

	err := runProductVariantCreate(productVariantCreateCmd, []string{"prod-1"})
	require.NoError(t, err)
}

func TestRunProductVariantCreate_InvalidPrice(t *testing.T) {
	setupCredentials(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).URL)
	productJSONFlag = false
	setVariantFlags(t, productVariantCreateCmd, map[string]string{
		"name":  "Walnut",
		"price": "not-a-number",
	})

	err := runProductVariantCreate(productVariantCreateCmd, []string{"prod-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --price")
}

func TestRunProductVariantCreate_InvalidVars(t *testing.T) {
	setupCredentials(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).URL)
	productJSONFlag = false
	setVariantFlags(t, productVariantCreateCmd, map[string]string{
		"name": "Walnut",
		"vars": "not-json",
	})

	err := runProductVariantCreate(productVariantCreateCmd, []string{"prod-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --vars")
}

func TestRunProductVariantCreate_RecipeNotInSpace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(errorResponse{Error: "recipe not found in this space"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false
	setVariantFlags(t, productVariantCreateCmd, map[string]string{
		"name":   "Walnut",
		"recipe": "bad-recipe",
	})

	err := runProductVariantCreate(productVariantCreateCmd, []string{"prod-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recipe not found in this space")
}

// --- product variant update ---

func TestRunProductVariantUpdate_PriceOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/products/prod-1/variants/v-1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "4500", body["price"])
		_, hasName := body["name"]
		assert.False(t, hasName, "unchanged flags should not be sent")
		_, hasRecipe := body["recipeId"]
		assert.False(t, hasRecipe, "unchanged flags should not be sent")

		w.Header().Set("Content-Type", "application/json")
		price := "4500"
		json.NewEncoder(w).Encode(productVariantItem{
			ID:        "v-1",
			ProductID: "prod-1",
			Name:      "Walnut / Spray PU",
			Price:     &price,
			IsActive:  true,
		})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false
	setVariantFlags(t, productVariantUpdateCmd, map[string]string{"price": "4500"})

	err := runProductVariantUpdate(productVariantUpdateCmd, []string{"prod-1", "v-1"})
	require.NoError(t, err)
}

func TestRunProductVariantUpdate_NoFlags(t *testing.T) {
	setupCredentials(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).URL)
	productJSONFlag = false
	setVariantFlags(t, productVariantUpdateCmd, nil)

	err := runProductVariantUpdate(productVariantUpdateCmd, []string{"prod-1", "v-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}

func TestRunProductVariantUpdate_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse{Error: "product variant not found"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false
	setVariantFlags(t, productVariantUpdateCmd, map[string]string{"price": "4500"})

	err := runProductVariantUpdate(productVariantUpdateCmd, []string{"prod-1", "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "product variant not found")
}

// --- product variant delete ---

func TestRunProductVariantDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/test-space/products/prod-1/variants/v-1", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false

	err := runProductVariantDelete(productVariantDeleteCmd, []string{"prod-1", "v-1"})
	require.NoError(t, err)
}

func TestRunProductVariantDelete_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse{Error: "product variant not found"})
	}))
	defer server.Close()

	setupCredentials(t, server.URL)
	productJSONFlag = false

	err := runProductVariantDelete(productVariantDeleteCmd, []string{"prod-1", "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "product variant not found")
}
