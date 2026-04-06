package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tylerjvollick/nori/internal/cli"
)

// stationInput holds station configuration gathered during interactive prompts.
type stationInput struct {
	Name     string
	WIPLimit int
}

// initSpaceResponse mirrors the relevant fields from POST /api/spaces.
type initSpaceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// initStationResponse mirrors the relevant fields from POST /api/v1/stations.
type initStationResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	WIPLimit int    `json:"wipLimit"`
}

// initAPIKeyResponse mirrors the relevant fields from POST /admin/api-keys.
type initAPIKeyResponse struct {
	RawKey string `json:"rawKey"`
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive first-time setup wizard",
	Long: `Bootstrap a complete Nori installation interactively.

This command walks you through the full setup:
  1. Checks that Docker and Docker Compose are available
  2. Stops any existing Nori containers
  3. Prompts for account, admin, and space configuration
  4. Generates a JWT secret and writes docker/.env
  5. Builds and starts all services via Docker Compose
  6. Waits for the server to become healthy
  7. Creates the admin account, space, and stations via the API
  8. Generates a CLI API key and saves credentials

Must be run from the repository root (docker/docker-compose.dev.yml must exist).`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Nori Setup Wizard ===")
	fmt.Println()

	// Step 1: Check docker / docker compose available
	fmt.Println("Checking prerequisites...")
	if err := checkDocker(); err != nil {
		return err
	}
	fmt.Println("  Docker: OK")
	fmt.Println("  Docker Compose: OK")
	fmt.Println()

	// Step 2: Check for running nori-* containers
	if err := checkExistingContainers(reader); err != nil {
		return err
	}

	// Step 3: Verify docker-compose file exists
	composePath := "docker/docker-compose.dev.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("docker/docker-compose.dev.yml not found — run 'nori init' from the repository root")
	}

	// Step 4: Interactive prompts
	fmt.Println("--- Account Configuration ---")
	accountName, err := cli.PromptStringDefault(reader, "Account name", "My Shop")
	if err != nil {
		return fmt.Errorf("failed to read account name: %w", err)
	}

	fmt.Println()
	fmt.Println("--- Admin User ---")
	adminEmail, err := cli.PromptStringDefault(reader, "Admin email", "admin@nori.dev")
	if err != nil {
		return fmt.Errorf("failed to read admin email: %w", err)
	}

	adminPassword, err := cli.PromptPassword("Admin password")
	if err != nil {
		return fmt.Errorf("failed to read admin password: %w", err)
	}
	if adminPassword == "" {
		return fmt.Errorf("password cannot be empty")
	}

	confirmPassword, err := cli.PromptPassword("Confirm password")
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}
	if adminPassword != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	fmt.Println()
	fmt.Println("--- Space ---")
	spaceName, err := cli.PromptStringDefault(reader, "Space name", "Workshop")
	if err != nil {
		return fmt.Errorf("failed to read space name: %w", err)
	}

	fmt.Println()
	fmt.Println("--- Stations (optional) ---")
	fmt.Println("Add stations for your shop floor. Press Enter with an empty name to finish.")
	stations, err := promptStations(reader)
	if err != nil {
		return fmt.Errorf("failed to read stations: %w", err)
	}

	// Step 5: Generate JWT secret
	jwtSecret, err := generateJWTSecret()
	if err != nil {
		return fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	// Step 6: Generate a random DB password
	dbPassword, err := generateRandomHex(16)
	if err != nil {
		return fmt.Errorf("failed to generate database password: %w", err)
	}

	// Step 7: Write docker/.env
	fmt.Println()
	fmt.Println("Writing docker/.env...")
	if err := writeEnvFile(accountName, adminEmail, adminPassword, jwtSecret, dbPassword); err != nil {
		return fmt.Errorf("failed to write docker/.env: %w", err)
	}
	fmt.Println("  docker/.env written.")

	// Step 8: Build and start containers
	fmt.Println()
	fmt.Println("Building and starting Docker containers...")
	if err := dockerComposeUp(); err != nil {
		return fmt.Errorf("failed to start Docker containers: %w", err)
	}
	fmt.Println("  Containers started.")

	// Step 9: Poll /health until healthy
	fmt.Println()
	fmt.Print("Waiting for server to become healthy...")
	serverURL := "http://localhost:8080"
	if err := waitForHealthy(serverURL, 30*time.Second, 1*time.Second); err != nil {
		return fmt.Errorf("server did not become healthy: %w", err)
	}
	fmt.Println(" ready!")

	// Step 10: POST /auth/login to get JWT
	fmt.Println()
	fmt.Println("Configuring Nori via API...")
	client := cli.NewClientWithURL(serverURL)

	loginResp, err := doLogin(client, adminEmail, adminPassword)
	if err != nil {
		return fmt.Errorf("failed to log in: %w", err)
	}
	client.Token = loginResp.AccessToken

	// Step 11: POST /api/spaces to create space
	spaceResp, err := createSpace(client, spaceName)
	if err != nil {
		return fmt.Errorf("failed to create space: %w", err)
	}
	fmt.Printf("  Space created: %s\n", spaceResp.Name)

	// Set space ID for subsequent requests
	client.SpaceID = spaceResp.ID

	// Step 12: POST /api/v1/stations for each station
	for _, s := range stations {
		stationResp, err := createStation(client, s.Name, s.WIPLimit)
		if err != nil {
			return fmt.Errorf("failed to create station %q: %w", s.Name, err)
		}
		fmt.Printf("  Station created: %s (WIP limit: %d)\n", stationResp.Name, stationResp.WIPLimit)
	}

	// Step 13: POST /admin/api-keys to generate CLI API key
	apiKeyResp, err := createAPIKey(client)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	fmt.Println("  API key generated.")

	// Step 14: Save credentials
	creds := &cli.Credentials{
		ServerURL: serverURL,
		APIKey:    apiKeyResp.RawKey,
		UserID:    loginResp.UserID,
		UserEmail: loginResp.UserEmail,
		SpaceID:   spaceResp.ID,
	}
	if err := cli.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}
	fmt.Println("  Credentials saved to ~/.config/nori/credentials")

	// Step 15: Print success summary
	fmt.Println()
	fmt.Println("=== Setup Complete ===")
	fmt.Println()
	fmt.Printf("  Server:  %s\n", serverURL)
	fmt.Printf("  Account: %s\n", accountName)
	fmt.Printf("  Admin:   %s\n", adminEmail)
	fmt.Printf("  Space:   %s (%s)\n", spaceResp.Name, spaceResp.ID)
	if len(stations) > 0 {
		fmt.Printf("  Stations: %d configured\n", len(stations))
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  nori station list     — view your stations")
	fmt.Println("  nori ready            — see tasks ready for work")
	fmt.Println("  nori task claim <id>  — claim a task")
	fmt.Println()

	return nil
}

// checkDocker verifies that docker and docker compose are available.
func checkDocker() error {
	if err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Run(); err != nil {
		return fmt.Errorf("docker is not installed or not running — install Docker and try again")
	}
	if err := exec.Command("docker", "compose", "version").Run(); err != nil {
		return fmt.Errorf("docker compose is not available — install Docker Compose V2 and try again")
	}
	return nil
}

// checkExistingContainers looks for running nori-* containers and offers to stop them.
func checkExistingContainers(reader *bufio.Reader) error {
	out, err := exec.Command("docker", "ps", "--filter", "name=nori-", "--format", "{{.Names}}").Output()
	if err != nil {
		return nil // Can't check, proceed anyway
	}

	names := strings.TrimSpace(string(out))
	if names == "" {
		return nil
	}

	fmt.Println("Found running Nori containers:")
	for _, name := range strings.Split(names, "\n") {
		fmt.Printf("  - %s\n", name)
	}

	confirm, err := cli.PromptConfirm(reader, "Stop existing containers and continue?")
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	if !confirm {
		return fmt.Errorf("aborted — stop existing containers manually and try again")
	}

	fmt.Println("Stopping existing containers...")
	cmd := exec.Command("docker", "compose", "-f", "docker/docker-compose.dev.yml", "--env-file", "docker/.env", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Ignore error from compose down — containers may not be from this compose file.
	// Fall back to stopping individual containers.
	if err := cmd.Run(); err != nil {
		for _, name := range strings.Split(names, "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				exec.Command("docker", "stop", name).Run()
			}
		}
	}
	fmt.Println()

	return nil
}

// promptStations interactively collects station names and WIP limits.
func promptStations(reader *bufio.Reader) ([]stationInput, error) {
	var stations []stationInput
	for {
		name, err := cli.PromptString(reader, "Station name (blank to finish)")
		if err != nil {
			return nil, err
		}
		if name == "" {
			break
		}

		wipLimit, err := cli.PromptInt(reader, "WIP limit", 1)
		if err != nil {
			return nil, err
		}

		stations = append(stations, stationInput{Name: name, WIPLimit: wipLimit})
	}
	return stations, nil
}

// generateJWTSecret generates a 32-byte hex-encoded random secret.
func generateJWTSecret() (string, error) {
	return generateRandomHex(32)
}

// generateRandomHex generates n random bytes and returns them hex-encoded.
func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// writeEnvFile writes the docker/.env file with all configuration values.
func writeEnvFile(accountName, adminEmail, adminPassword, jwtSecret, dbPassword string) error {
	content := fmt.Sprintf(`# Generated by 'nori init' — do not commit this file.

# --- PostgreSQL ---
POSTGRES_USER=postgres
POSTGRES_PASSWORD=%s
POSTGRES_DB=nori

# --- Nori Server ---
MAX_UPLOAD_SIZE=1073741824
ALLOWED_MIME_TYPES=image/jpeg,image/png,image/webp,image/gif,video/mp4,video/quicktime,video/webm
NORI_JWT_SECRET=%s
NORI_ADMIN_EMAIL=%s
NORI_ADMIN_PASSWORD=%s
NORI_ACCOUNT_NAME=%s
NORI_SKIP_PASSWORD_CHANGE=true
`, dbPassword, jwtSecret, adminEmail, adminPassword, accountName)

	return os.WriteFile("docker/.env", []byte(content), 0600)
}

// dockerComposeUp runs docker compose up -d --build.
func dockerComposeUp() error {
	cmd := exec.Command("docker", "compose",
		"-f", "docker/docker-compose.dev.yml",
		"--env-file", "docker/.env",
		"up", "-d", "--build",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// waitForHealthy polls the server's /health endpoint until it returns OK or the timeout is reached.
func waitForHealthy(serverURL string, timeout, interval time.Duration) error {
	httpClient := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(serverURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		fmt.Print(".")
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout after %s", timeout)
}

// doLogin authenticates with the server and returns the login response.
func doLogin(client *cli.Client, email, password string) (*loginResponse, error) {
	resp, err := client.Post("/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("login failed: %s", errResp.Error)
		}
		return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp loginResponse
	if err := cli.ReadJSON(resp, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	return &loginResp, nil
}

// createSpace creates a space via the API.
func createSpace(client *cli.Client, name string) (*initSpaceResponse, error) {
	resp, err := client.Post("/api/spaces", map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var spaceResp initSpaceResponse
	if err := cli.ReadJSON(resp, &spaceResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &spaceResp, nil
}

// createStation creates a station via the API.
func createStation(client *cli.Client, name string, wipLimit int) (*initStationResponse, error) {
	resp, err := client.Post("/api/v1/stations", map[string]interface{}{
		"name":     name,
		"wipLimit": wipLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var stationResp initStationResponse
	if err := cli.ReadJSON(resp, &stationResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &stationResp, nil
}

// createAPIKey generates a CLI API key via the admin endpoint.
func createAPIKey(client *cli.Client) (*initAPIKeyResponse, error) {
	resp, err := client.Post("/admin/api-keys", map[string]interface{}{
		"name": "nori-cli",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp errorResponse
		if err := cli.ReadJSON(resp, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// The response has { rawKey: "...", apiKey: { ... } }
	var raw json.RawMessage
	if err := cli.ReadJSON(resp, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var apiKeyResp initAPIKeyResponse
	if err := json.Unmarshal(raw, &apiKeyResp); err != nil {
		return nil, fmt.Errorf("failed to parse API key response: %w", err)
	}

	if apiKeyResp.RawKey == "" {
		return nil, fmt.Errorf("server returned empty API key")
	}

	return &apiKeyResp, nil
}
