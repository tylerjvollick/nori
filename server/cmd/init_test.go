package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/cli"
)

// fakeExecCommand returns a function that creates *exec.Cmd pointing at the
// test binary's TestHelperProcess with the given exit code and stdout output.
// This is the standard Go pattern for mocking exec.Command in tests.
func fakeExecCommand(exitCode int, stdout string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			fmt.Sprintf("GO_HELPER_EXIT_CODE=%d", exitCode),
			fmt.Sprintf("GO_HELPER_STDOUT=%s", stdout),
		)
		return cmd
	}
}

// fakeExecCommandMulti returns a function that returns different exit codes
// and stdout based on the command invoked. The routingFn receives the command
// name and args and returns (exitCode, stdout).
func fakeExecCommandMulti(routingFn func(name string, args []string) (int, string)) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		exitCode, stdout := routingFn(name, args)
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			fmt.Sprintf("GO_HELPER_EXIT_CODE=%d", exitCode),
			fmt.Sprintf("GO_HELPER_STDOUT=%s", stdout),
		)
		return cmd
	}
}

// TestHelperProcess is invoked by fakeExecCommand. It is not a real test.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Write stdout if requested
	if stdout := os.Getenv("GO_HELPER_STDOUT"); stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}
	// Exit with requested code
	exitCode := 0
	fmt.Sscanf(os.Getenv("GO_HELPER_EXIT_CODE"), "%d", &exitCode)
	os.Exit(exitCode)
}

// restoreExecCommand saves and restores the execCommand package variable.
func restoreExecCommand(t *testing.T) {
	t.Helper()
	original := execCommand
	t.Cleanup(func() { execCommand = original })
}

// ===== Command registration =====

func TestInitCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			found = true
			break
		}
	}
	assert.True(t, found, "init command should be registered on root")
}

// ===== generateJWTSecret / generateRandomHex =====

func TestGenerateJWTSecret(t *testing.T) {
	secret, err := generateJWTSecret()
	require.NoError(t, err)

	// 32 bytes = 64 hex chars
	assert.Len(t, secret, 64)

	// Should be different each time
	secret2, err := generateJWTSecret()
	require.NoError(t, err)
	assert.NotEqual(t, secret, secret2)
}

func TestGenerateRandomHex(t *testing.T) {
	hex16, err := generateRandomHex(16)
	require.NoError(t, err)
	assert.Len(t, hex16, 32)

	hex32, err := generateRandomHex(32)
	require.NoError(t, err)
	assert.Len(t, hex32, 64)
}

// ===== writeEnvFile =====

func TestWriteEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.MkdirAll(tmpDir+"/docker", 0700))
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	err := writeEnvFile("Test Shop", "admin@test.com", "secret123", "jwt-hex-secret", "db-password-hex")
	require.NoError(t, err)

	data, err := os.ReadFile("docker/.env")
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "NORI_ACCOUNT_NAME=Test Shop")
	assert.Contains(t, content, "NORI_ADMIN_EMAIL=admin@test.com")
	assert.Contains(t, content, "NORI_ADMIN_PASSWORD=secret123")
	assert.Contains(t, content, "NORI_JWT_SECRET=jwt-hex-secret")
	assert.Contains(t, content, "POSTGRES_PASSWORD=db-password-hex")
	assert.Contains(t, content, "NORI_SKIP_PASSWORD_CHANGE=true")

	// Verify file permissions
	info, err := os.Stat("docker/.env")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestWriteEnvFile_NoDockerDir(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origDir)

	// No docker/ directory — should fail
	err := writeEnvFile("Shop", "a@b.com", "pw", "jwt", "db")
	require.Error(t, err)
}

// ===== checkDocker =====

func TestCheckDocker_Success(t *testing.T) {
	restoreExecCommand(t)
	execCommand = fakeExecCommand(0, "")

	err := checkDocker()
	require.NoError(t, err)
}

func TestCheckDocker_DockerNotAvailable(t *testing.T) {
	restoreExecCommand(t)
	// All commands fail — docker not installed
	execCommand = fakeExecCommand(1, "")

	err := checkDocker()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker is not installed or not running")
}

func TestCheckDocker_ComposeNotAvailable(t *testing.T) {
	restoreExecCommand(t)
	// docker version succeeds, docker compose version fails
	execCommand = fakeExecCommandMulti(func(name string, args []string) (int, string) {
		if len(args) > 0 && args[0] == "compose" {
			return 1, ""
		}
		return 0, ""
	})

	err := checkDocker()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker compose is not available")
}

// ===== checkExistingContainers =====

func TestCheckExistingContainers_NoContainers(t *testing.T) {
	restoreExecCommand(t)
	// docker ps returns empty output
	execCommand = fakeExecCommand(0, "")

	reader := bufio.NewReader(strings.NewReader(""))
	err := checkExistingContainers(reader)
	require.NoError(t, err)
}

func TestCheckExistingContainers_Found_UserConfirms(t *testing.T) {
	restoreExecCommand(t)
	// Route: docker ps returns container names, compose down succeeds
	execCommand = fakeExecCommandMulti(func(name string, args []string) (int, string) {
		for _, a := range args {
			if a == "--filter" {
				return 0, "nori-server\nnori-db\n"
			}
		}
		return 0, ""
	})

	// User confirms stopping containers
	reader := bufio.NewReader(strings.NewReader("y\n"))
	err := checkExistingContainers(reader)
	require.NoError(t, err)
}

func TestCheckExistingContainers_Found_UserDeclines(t *testing.T) {
	restoreExecCommand(t)
	execCommand = fakeExecCommandMulti(func(name string, args []string) (int, string) {
		for _, a := range args {
			if a == "--filter" {
				return 0, "nori-server\n"
			}
		}
		return 0, ""
	})

	// User declines
	reader := bufio.NewReader(strings.NewReader("n\n"))
	err := checkExistingContainers(reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
}

func TestCheckExistingContainers_DockerPsFails(t *testing.T) {
	restoreExecCommand(t)
	// docker ps fails — should proceed silently
	execCommand = fakeExecCommand(1, "")

	reader := bufio.NewReader(strings.NewReader(""))
	err := checkExistingContainers(reader)
	require.NoError(t, err, "should proceed silently when docker ps fails")
}

func TestCheckExistingContainers_ComposeDownFails_FallsBackToStop(t *testing.T) {
	restoreExecCommand(t)
	var stopCalled atomic.Int32

	execCommand = fakeExecCommandMulti(func(name string, args []string) (int, string) {
		// docker ps --filter returns containers
		for _, a := range args {
			if a == "--filter" {
				return 0, "nori-server\nnori-db\n"
			}
		}
		// docker compose ... down fails
		for _, a := range args {
			if a == "down" {
				return 1, ""
			}
		}
		// docker stop succeeds (fallback)
		for _, a := range args {
			if a == "stop" {
				stopCalled.Add(1)
				return 0, ""
			}
		}
		return 0, ""
	})

	reader := bufio.NewReader(strings.NewReader("y\n"))
	err := checkExistingContainers(reader)
	require.NoError(t, err)
	// The fallback should have attempted to stop individual containers
	assert.GreaterOrEqual(t, stopCalled.Load(), int32(1), "should fall back to docker stop")
}

// ===== dockerComposeUp =====

func TestDockerComposeUp_Success(t *testing.T) {
	restoreExecCommand(t)
	execCommand = fakeExecCommand(0, "")

	err := dockerComposeUp()
	require.NoError(t, err)
}

func TestDockerComposeUp_Failure(t *testing.T) {
	restoreExecCommand(t)
	execCommand = fakeExecCommand(1, "")

	err := dockerComposeUp()
	require.Error(t, err)
}

// ===== promptStations =====

func TestPromptStations_TwoStations(t *testing.T) {
	input := "Cutting\n2\nAssembly\n1\n\n"
	reader := bufio.NewReader(strings.NewReader(input))

	stations, err := promptStations(reader)
	require.NoError(t, err)
	require.Len(t, stations, 2)

	assert.Equal(t, "Cutting", stations[0].Name)
	assert.Equal(t, 2, stations[0].WIPLimit)
	assert.Equal(t, "Assembly", stations[1].Name)
	assert.Equal(t, 1, stations[1].WIPLimit)
}

func TestPromptStations_NoStations(t *testing.T) {
	// Immediately blank line = no stations
	input := "\n"
	reader := bufio.NewReader(strings.NewReader(input))

	stations, err := promptStations(reader)
	require.NoError(t, err)
	assert.Empty(t, stations)
}

func TestPromptStations_DefaultWIPLimit(t *testing.T) {
	// Station name, then blank for WIP limit (default 1), then blank to finish
	input := "Sanding\n\n\n"
	reader := bufio.NewReader(strings.NewReader(input))

	stations, err := promptStations(reader)
	require.NoError(t, err)
	require.Len(t, stations, 1)
	assert.Equal(t, "Sanding", stations[0].Name)
	assert.Equal(t, 1, stations[0].WIPLimit)
}

func TestPromptStations_InvalidWIPLimit(t *testing.T) {
	input := "Cutting\nabc\n"
	reader := bufio.NewReader(strings.NewReader(input))

	_, err := promptStations(reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid number")
}

// ===== doLogin =====

func TestDoLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/login", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "admin@test.com", body["email"])
		assert.Equal(t, "password", body["password"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse{
			AccessToken: "jwt-token",
			UserID:      "user-123",
			UserEmail:   "admin@test.com",
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	resp, err := doLogin(client, "admin@test.com", "password")
	require.NoError(t, err)
	assert.Equal(t, "jwt-token", resp.AccessToken)
	assert.Equal(t, "user-123", resp.UserID)
	assert.Equal(t, "admin@test.com", resp.UserEmail)
}

func TestDoLogin_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(errorResponse{Error: "invalid credentials"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	_, err := doLogin(client, "admin@test.com", "wrong")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestDoLogin_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "internal error"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	_, err := doLogin(client, "admin@test.com", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
}

func TestDoLogin_ConnectionError(t *testing.T) {
	// Point at a server that doesn't exist
	client := cli.NewClientWithURL("http://127.0.0.1:1")
	_, err := doLogin(client, "admin@test.com", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

// ===== createSpace =====

func TestCreateSpace_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/spaces", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-jwt", r.Header.Get("Authorization"))

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Workshop", body["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(initSpaceResponse{
			ID:   "space-uuid",
			Name: "Workshop",
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	resp, err := createSpace(client, "Workshop")
	require.NoError(t, err)
	assert.Equal(t, "space-uuid", resp.ID)
	assert.Equal(t, "Workshop", resp.Name)
}

func TestCreateSpace_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(errorResponse{Error: "admin access required"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	_, err := createSpace(client, "Workshop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin access required")
}

func TestCreateSpace_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "database error"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	_, err := createSpace(client, "Workshop")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// ===== createStation =====

func TestCreateStation_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/spaces/space-123/stations", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Cutting", body["name"])
		assert.Equal(t, float64(2), body["wipLimit"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(initStationResponse{
			ID:       "station-uuid",
			Name:     "Cutting",
			WIPLimit: 2,
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	client.SpaceID = "space-123"
	resp, err := createStation(client, "Cutting", 2)
	require.NoError(t, err)
	assert.Equal(t, "station-uuid", resp.ID)
	assert.Equal(t, "Cutting", resp.Name)
	assert.Equal(t, 2, resp.WIPLimit)
}

func TestCreateStation_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "station name is required"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	client.SpaceID = "space-123"
	_, err := createStation(client, "", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "station name is required")
}

func TestCreateStation_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: "internal server error"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	client.SpaceID = "space-123"
	_, err := createStation(client, "Cutting", 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error")
}

// ===== createAPIKey =====

func TestCreateAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/admin/api-keys", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "nori-cli", body["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rawKey": "nori_test-api-key-value",
			"apiKey": map[string]interface{}{
				"id":   "key-uuid",
				"name": "nori-cli",
			},
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	resp, err := createAPIKey(client)
	require.NoError(t, err)
	assert.Equal(t, "nori_test-api-key-value", resp.RawKey)
}

func TestCreateAPIKey_EmptyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rawKey": "",
			"apiKey": map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	_, err := createAPIKey(client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty API key")
}

func TestCreateAPIKey_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(errorResponse{Error: "admin access required"})
	}))
	defer server.Close()

	client := cli.NewClientWithURL(server.URL)
	client.Token = "test-jwt"
	_, err := createAPIKey(client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin access required")
}

// ===== waitForHealthy =====

func TestWaitForHealthy_ImmediateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	err := waitForHealthy(server.URL, 5*time.Second, 100*time.Millisecond)
	require.NoError(t, err)
}

func TestWaitForHealthy_Timeout(t *testing.T) {
	// Server that always returns 503
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := waitForHealthy(server.URL, 500*time.Millisecond, 100*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestWaitForHealthy_EventuallySucceeds(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	err := waitForHealthy(server.URL, 5*time.Second, 50*time.Millisecond)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, callCount.Load(), int32(3))
}

func TestWaitForHealthy_ServerNotReachable(t *testing.T) {
	// Point at a server that doesn't exist
	err := waitForHealthy("http://127.0.0.1:1", 300*time.Millisecond, 50*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

// ===== TestInitFullAPIFlow =====

// TestInitFullAPIFlow tests the complete API interaction sequence used by init,
// without actually running docker or prompts.
func TestInitFullAPIFlow(t *testing.T) {
	callSequence := make([]string, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callSequence = append(callSequence, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/auth/login":
			json.NewEncoder(w).Encode(loginResponse{
				AccessToken: "jwt-token",
				UserID:      "user-123",
				UserEmail:   "admin@test.com",
			})

		case "/api/spaces":
			assert.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(initSpaceResponse{
				ID:   "space-uuid",
				Name: "Workshop",
			})

		case "/api/v1/spaces/space-uuid/stations":
			assert.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(initStationResponse{
				ID:       "station-uuid",
				Name:     body["name"].(string),
				WIPLimit: int(body["wipLimit"].(float64)),
			})

		case "/admin/api-keys":
			assert.Equal(t, "Bearer jwt-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"rawKey": "nori_test-key-abc123",
				"apiKey": map[string]interface{}{"id": "key-uuid", "name": "nori-cli"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Simulate the API flow that runInit performs after docker is up
	client := cli.NewClientWithURL(server.URL)

	// Login
	loginResp, err := doLogin(client, "admin@test.com", "password")
	require.NoError(t, err)
	client.Token = loginResp.AccessToken

	// Create space
	spaceResp, err := createSpace(client, "Workshop")
	require.NoError(t, err)
	client.SpaceID = spaceResp.ID

	// Create stations
	stations := []stationInput{
		{Name: "Cutting", WIPLimit: 2},
		{Name: "Assembly", WIPLimit: 1},
	}
	for _, s := range stations {
		_, err := createStation(client, s.Name, s.WIPLimit)
		require.NoError(t, err)
	}

	// Create API key
	apiKeyResp, err := createAPIKey(client)
	require.NoError(t, err)

	// Save credentials
	creds := &cli.Credentials{
		ServerURL: server.URL,
		APIKey:    apiKeyResp.RawKey,
		UserID:    loginResp.UserID,
		UserEmail: loginResp.UserEmail,
		SpaceID:   spaceResp.ID,
	}
	require.NoError(t, cli.SaveCredentials(creds))

	// Verify saved credentials
	loaded, err := cli.LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "nori_test-key-abc123", loaded.APIKey)
	assert.Equal(t, "admin@test.com", loaded.UserEmail)
	assert.Equal(t, "space-uuid", loaded.SpaceID)
	assert.Equal(t, "user-123", loaded.UserID)

	// Verify API call sequence
	expected := []string{
		"POST /auth/login",
		"POST /api/spaces",
		"POST /api/v1/spaces/space-uuid/stations",
		"POST /api/v1/spaces/space-uuid/stations",
		"POST /admin/api-keys",
	}
	assert.Equal(t, expected, callSequence)
}
