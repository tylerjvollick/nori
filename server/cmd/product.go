package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// productVariantItem mirrors the product variant API response.
type productVariantItem struct {
	ID              string                 `json:"id"`
	ProductID       string                 `json:"productId"`
	Name            string                 `json:"name"`
	RecipeID        *string                `json:"recipeId,omitempty"`
	RecipeVariables map[string]interface{} `json:"recipeVariables,omitempty"`
	Price           *string                `json:"price,omitempty"`
	IsActive        bool                   `json:"isActive"`
}

// productItem mirrors the product API response.
type productItem struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	IsActive    bool                 `json:"isActive"`
	Variants    []productVariantItem `json:"variants"`
}

var productJSONFlag bool

// Flags for product create.
var (
	productCreateName        string
	productCreateDescription string
)

// Flags for variant create/update.
var (
	variantName   string
	variantRecipe string
	variantPrice  string
	variantVars   string
)

var productCmd = &cobra.Command{
	Use:   "product",
	Short: "Manage products",
	Long:  "Product commands: manage what the shop sells and its sellable variants.",
}

var productCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new product",
	Long:  "Create a new product in the current space.",
	RunE:  runProductCreate,
}

var productListCmd = &cobra.Command{
	Use:   "list",
	Short: "List products",
	Long:  "List all products in the current space with their variants.",
	RunE:  runProductList,
}

var productVariantCmd = &cobra.Command{
	Use:   "variant",
	Short: "Manage product variants",
	Long:  "Variant commands: sellable configurations of a product, optionally bound to a recipe with variable bindings and a price.",
}

var productVariantCreateCmd = &cobra.Command{
	Use:   "create <productId>",
	Short: "Create a product variant",
	Long:  "Create a variant under a product. Optionally bind it to a recipe (--recipe) with variable bindings (--vars) and a customer-facing price (--price).",
	Args:  cobra.ExactArgs(1),
	RunE:  runProductVariantCreate,
}

var productVariantUpdateCmd = &cobra.Command{
	Use:   "update <productId> <variantId>",
	Short: "Update a product variant",
	Long:  "Update a variant's name, recipe binding, variables, or price. Only the provided flags are changed.",
	Args:  cobra.ExactArgs(2),
	RunE:  runProductVariantUpdate,
}

var productVariantDeleteCmd = &cobra.Command{
	Use:   "delete <productId> <variantId>",
	Short: "Delete a product variant",
	Long:  "Permanently delete a variant from a product.",
	Args:  cobra.ExactArgs(2),
	RunE:  runProductVariantDelete,
}

func init() {
	productCmd.PersistentFlags().BoolVar(&productJSONFlag, "json", false, "Output as JSON")

	productCreateCmd.Flags().StringVar(&productCreateName, "name", "", "Product name (required)")
	productCreateCmd.Flags().StringVar(&productCreateDescription, "description", "", "Product description")
	productCreateCmd.MarkFlagRequired("name") //nolint:errcheck

	for _, c := range []*cobra.Command{productVariantCreateCmd, productVariantUpdateCmd} {
		c.Flags().StringVar(&variantName, "name", "", "Variant name")
		c.Flags().StringVar(&variantRecipe, "recipe", "", "Recipe ID to bind the variant to")
		c.Flags().StringVar(&variantPrice, "price", "", "Customer-facing price (e.g. 4200 or 4200.50)")
		c.Flags().StringVar(&variantVars, "vars", "", `Recipe variable bindings as JSON (e.g. '{"wood_species":"Walnut"}')`)
	}
	productVariantCreateCmd.MarkFlagRequired("name") //nolint:errcheck

	productVariantCmd.AddCommand(productVariantCreateCmd)
	productVariantCmd.AddCommand(productVariantUpdateCmd)
	productVariantCmd.AddCommand(productVariantDeleteCmd)

	productCmd.AddCommand(productCreateCmd)
	productCmd.AddCommand(productListCmd)
	productCmd.AddCommand(productVariantCmd)
	rootCmd.AddCommand(productCmd)
}

// checkProductResponse validates an API response, returning a descriptive
// error for non-success statuses.
func checkProductResponse(resp *http.Response, wantStatus int) error {
	if err := cli.Handle401(resp); err != nil {
		return err
	}
	if resp.StatusCode != wantStatus {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("server error: %s", errResp.Error)
		}
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}

// variantRequestBody builds the request body for variant create/update from
// the variant flags. When onlyChanged is set, only flags explicitly provided
// on the command line are included (for partial updates).
func variantRequestBody(cmd *cobra.Command, onlyChanged bool) (map[string]interface{}, error) {
	body := map[string]interface{}{}

	if !onlyChanged || cmd.Flags().Changed("name") {
		if variantName != "" {
			body["name"] = variantName
		}
	}
	if cmd.Flags().Changed("recipe") {
		body["recipeId"] = variantRecipe
	}
	if cmd.Flags().Changed("price") {
		if _, err := decimal.NewFromString(variantPrice); err != nil {
			return nil, fmt.Errorf("invalid --price %q: must be a decimal number", variantPrice)
		}
		body["price"] = variantPrice
	}
	if cmd.Flags().Changed("vars") {
		var vars map[string]interface{}
		if err := json.Unmarshal([]byte(variantVars), &vars); err != nil {
			return nil, fmt.Errorf("invalid --vars: must be a JSON object: %w", err)
		}
		body["recipeVariables"] = vars
	}

	return body, nil
}

// printJSON writes v to stdout as indented JSON.
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// runProductCreate implements `nori product create --name <name>`.
func runProductCreate(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	body := map[string]interface{}{"name": productCreateName}
	if productCreateDescription != "" {
		body["description"] = productCreateDescription
	}

	resp, err := client.Post(client.SpacePath("/products"), body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	if err := checkProductResponse(resp, http.StatusCreated); err != nil {
		return err
	}

	var product productItem
	if err := cli.ReadJSON(resp, &product); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if productJSONFlag {
		return printJSON(product)
	}
	fmt.Printf("Created product: %s (%s)\n", product.Name, product.ID)
	return nil
}

// runProductList implements `nori product list`.
func runProductList(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	resp, err := client.Get(client.SpacePath("/products"))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	if err := checkProductResponse(resp, http.StatusOK); err != nil {
		return err
	}

	var products []productItem
	if err := cli.ReadJSON(resp, &products); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if productJSONFlag {
		return printJSON(products)
	}

	if len(products) == 0 {
		fmt.Println("No products found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tVARIANTS\tACTIVE")
	for _, p := range products {
		active := "yes"
		if !p.IsActive {
			active = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", p.ID, p.Name, len(p.Variants), active)
	}
	return w.Flush()
}

// runProductVariantCreate implements `nori product variant create <productId>`.
func runProductVariantCreate(cmd *cobra.Command, args []string) error {
	productID := args[0]

	body, err := variantRequestBody(cmd, false)
	if err != nil {
		return err
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	resp, err := client.Post(client.SpacePath("/products/"+productID+"/variants"), body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	if err := checkProductResponse(resp, http.StatusCreated); err != nil {
		return err
	}

	var variant productVariantItem
	if err := cli.ReadJSON(resp, &variant); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if productJSONFlag {
		return printJSON(variant)
	}
	price := "-"
	if variant.Price != nil {
		price = *variant.Price
	}
	fmt.Printf("Created variant: %s (%s, price: %s)\n", variant.Name, variant.ID, price)
	return nil
}

// runProductVariantUpdate implements `nori product variant update <productId> <variantId>`.
func runProductVariantUpdate(cmd *cobra.Command, args []string) error {
	productID, variantID := args[0], args[1]

	body, err := variantRequestBody(cmd, true)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return fmt.Errorf("nothing to update: provide at least one of --name, --recipe, --price, --vars")
	}

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	resp, err := client.Put(client.SpacePath("/products/"+productID+"/variants/"+variantID), body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	if err := checkProductResponse(resp, http.StatusOK); err != nil {
		return err
	}

	var variant productVariantItem
	if err := cli.ReadJSON(resp, &variant); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if productJSONFlag {
		return printJSON(variant)
	}
	fmt.Printf("Updated variant: %s (%s)\n", variant.Name, variant.ID)
	return nil
}

// runProductVariantDelete implements `nori product variant delete <productId> <variantId>`.
func runProductVariantDelete(cmd *cobra.Command, args []string) error {
	productID, variantID := args[0], args[1]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}
	client := newClientWithSpace(creds)

	resp, err := client.Delete(client.SpacePath("/products/" + productID + "/variants/" + variantID))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	if err := checkProductResponse(resp, http.StatusNoContent); err != nil {
		return err
	}
	resp.Body.Close()

	if !productJSONFlag {
		fmt.Printf("Deleted variant %s\n", variantID)
	}
	return nil
}
