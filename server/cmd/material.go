package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// materialItem mirrors the material response from the materials API.
type materialItem struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Unit     string  `json:"unit"`
	Supplier *string `json:"supplier,omitempty"`
	SKU      *string `json:"sku,omitempty"`
	Location *string `json:"location,omitempty"`
	UnitCost *string `json:"unitCost,omitempty"`
	IsActive bool    `json:"isActive"`
}

var materialJSONFlag bool

// Flags shared by material create and material update.
var (
	materialName     string
	materialUnit     string
	materialCost     string
	materialSupplier string
	materialSKU      string
	materialCategory string
	materialLocation string
	materialSearch   string
)

var materialCmd = &cobra.Command{
	Use:   "material",
	Short: "Manage the material catalog",
	Long:  "Material commands: create, list, update, and delete raw materials (lumber, hardware, finish) with unit costs.",
}

var materialCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new material",
	Long:  "Create a new material in the current space, e.g.\n\n  nori material create --name '8/4 Walnut' --unit board_feet --cost 14.50 --supplier 'Goby Walnut'",
	Args:  cobra.NoArgs,
	RunE:  runMaterialCreate,
}

var materialListCmd = &cobra.Command{
	Use:   "list",
	Short: "List materials",
	Long:  "List materials in the current space. Use --search to filter by name.",
	Args:  cobra.NoArgs,
	RunE:  runMaterialList,
}

var materialUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a material",
	Long:  "Update fields on a material, e.g.\n\n  nori material update <id> --cost 15.00",
	Args:  cobra.ExactArgs(1),
	RunE:  runMaterialUpdate,
}

var materialDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a material",
	Long:  "Soft-delete a material. It is removed from listings but retained in the database.",
	Args:  cobra.ExactArgs(1),
	RunE:  runMaterialDelete,
}

func init() {
	materialCmd.PersistentFlags().BoolVar(&materialJSONFlag, "json", false, "Output as JSON")

	materialCreateCmd.Flags().StringVar(&materialName, "name", "", "Material name (required)")
	materialCreateCmd.Flags().StringVar(&materialUnit, "unit", "", "Unit of measure, e.g. board_feet, each, gallons (required)")
	materialCreateCmd.Flags().StringVar(&materialCost, "cost", "", "Unit cost, e.g. 14.50")
	materialCreateCmd.Flags().StringVar(&materialSupplier, "supplier", "", "Supplier name")
	materialCreateCmd.Flags().StringVar(&materialSKU, "sku", "", "Supplier SKU")
	materialCreateCmd.Flags().StringVar(&materialCategory, "category", "", "Category: lumber, hardware, finish, consumable, other")
	materialCreateCmd.Flags().StringVar(&materialLocation, "location", "", "Storage location")
	_ = materialCreateCmd.MarkFlagRequired("name")
	_ = materialCreateCmd.MarkFlagRequired("unit")

	materialListCmd.Flags().StringVar(&materialSearch, "search", "", "Filter materials by name (case-insensitive)")

	materialUpdateCmd.Flags().StringVar(&materialName, "name", "", "Material name")
	materialUpdateCmd.Flags().StringVar(&materialUnit, "unit", "", "Unit of measure")
	materialUpdateCmd.Flags().StringVar(&materialCost, "cost", "", "Unit cost, e.g. 15.00")
	materialUpdateCmd.Flags().StringVar(&materialSupplier, "supplier", "", "Supplier name")
	materialUpdateCmd.Flags().StringVar(&materialSKU, "sku", "", "Supplier SKU")
	materialUpdateCmd.Flags().StringVar(&materialCategory, "category", "", "Category: lumber, hardware, finish, consumable, other")
	materialUpdateCmd.Flags().StringVar(&materialLocation, "location", "", "Storage location")

	materialCmd.AddCommand(materialCreateCmd)
	materialCmd.AddCommand(materialListCmd)
	materialCmd.AddCommand(materialUpdateCmd)
	materialCmd.AddCommand(materialDeleteCmd)
	rootCmd.AddCommand(materialCmd)
}

// materialBasePath builds the space-scoped materials API path from the
// resolved space ID (--space flag > NORI_SPACE env > stored credentials).
func materialBasePath(creds *cli.Credentials) (string, error) {
	spaceID := resolveSpaceID(creds)
	if spaceID == "" {
		return "", fmt.Errorf("no space selected — use --space, set NORI_SPACE, or run 'nori init'")
	}
	return "/api/v1/spaces/" + spaceID + "/materials", nil
}

// parseCostFlag validates the --cost flag value.
func parseCostFlag(value string) (string, error) {
	cost, err := decimal.NewFromString(value)
	if err != nil {
		return "", fmt.Errorf("invalid --cost value %q: must be a decimal number", value)
	}
	if cost.IsNegative() {
		return "", fmt.Errorf("invalid --cost value %q: must be >= 0", value)
	}
	return cost.String(), nil
}

// materialAPIError extracts a server error message from a non-success response.
func materialAPIError(resp *http.Response) error {
	var errResp errorResponse
	if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("server error: %s", errResp.Error)
	}
	return fmt.Errorf("server returned status %d", resp.StatusCode)
}

func printMaterialJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func formatOptional(s *string) string {
	if s == nil || *s == "" {
		return "-"
	}
	return *s
}

// runMaterialCreate implements `nori material create`.
func runMaterialCreate(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	basePath, err := materialBasePath(creds)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"name": materialName,
		"unit": materialUnit,
	}
	if cmd.Flags().Changed("cost") {
		cost, err := parseCostFlag(materialCost)
		if err != nil {
			return err
		}
		body["unitCost"] = cost
	}
	if materialSupplier != "" {
		body["supplier"] = materialSupplier
	}
	if materialSKU != "" {
		body["sku"] = materialSKU
	}
	if materialCategory != "" {
		body["category"] = materialCategory
	}
	if materialLocation != "" {
		body["location"] = materialLocation
	}

	client := newClientWithSpace(creds)
	resp, err := client.Post(basePath, body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return materialAPIError(resp)
	}

	var material materialItem
	if err := cli.ReadJSON(resp, &material); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if materialJSONFlag {
		return printMaterialJSON(material)
	}

	fmt.Printf("Created material: %s (%s)\n", material.Name, material.ID)
	return nil
}

// runMaterialList implements `nori material list`.
func runMaterialList(cmd *cobra.Command, args []string) error {
	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	basePath, err := materialBasePath(creds)
	if err != nil {
		return err
	}

	path := basePath
	if materialSearch != "" {
		path += "?search=" + url.QueryEscape(materialSearch)
	}

	client := newClientWithSpace(creds)
	resp, err := client.Get(path)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return materialAPIError(resp)
	}

	var materials []materialItem
	if err := cli.ReadJSON(resp, &materials); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if materialJSONFlag {
		return printMaterialJSON(materials)
	}

	if len(materials) == 0 {
		fmt.Println("No materials found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tUNIT\tCOST\tSUPPLIER\tSKU")
	for _, m := range materials {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			m.ID, m.Name, m.Category, m.Unit,
			formatOptional(m.UnitCost), formatOptional(m.Supplier), formatOptional(m.SKU))
	}
	return w.Flush()
}

// runMaterialUpdate implements `nori material update <id>`.
func runMaterialUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	basePath, err := materialBasePath(creds)
	if err != nil {
		return err
	}

	body := map[string]interface{}{}
	if cmd.Flags().Changed("name") {
		body["name"] = materialName
	}
	if cmd.Flags().Changed("unit") {
		body["unit"] = materialUnit
	}
	if cmd.Flags().Changed("cost") {
		cost, err := parseCostFlag(materialCost)
		if err != nil {
			return err
		}
		body["unitCost"] = cost
	}
	if cmd.Flags().Changed("supplier") {
		body["supplier"] = materialSupplier
	}
	if cmd.Flags().Changed("sku") {
		body["sku"] = materialSKU
	}
	if cmd.Flags().Changed("category") {
		body["category"] = materialCategory
	}
	if cmd.Flags().Changed("location") {
		body["location"] = materialLocation
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update — provide at least one flag, e.g. --cost 15.00")
	}

	client := newClientWithSpace(creds)
	resp, err := client.Put(basePath+"/"+id, body)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return materialAPIError(resp)
	}

	var material materialItem
	if err := cli.ReadJSON(resp, &material); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if materialJSONFlag {
		return printMaterialJSON(material)
	}

	fmt.Printf("Updated material: %s\n", material.Name)
	return nil
}

// runMaterialDelete implements `nori material delete <id>`.
func runMaterialDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	creds, err := cli.LoadCredentials()
	if err != nil {
		return err
	}

	basePath, err := materialBasePath(creds)
	if err != nil {
		return err
	}

	client := newClientWithSpace(creds)
	resp, err := client.Delete(basePath + "/" + id)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := cli.Handle401(resp); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return materialAPIError(resp)
	}

	fmt.Printf("Deleted material %s\n", id)
	return nil
}
