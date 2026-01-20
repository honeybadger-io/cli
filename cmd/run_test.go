package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCheckInPayload mirrors checkInPayload but with int64 Duration for test assertions
type testCheckInPayload struct {
	CheckIn struct {
		Status   string `json:"status"`
		Duration int64  `json:"duration,omitempty"`
		Stdout   string `json:"stdout,omitempty"`
		Stderr   string `json:"stderr,omitempty"`
		ExitCode int    `json:"exit_code"`
	} `json:"check_in"`
}

func TestRunCommand(t *testing.T) {
	// Save original values
	originalClient := http.DefaultClient
	originalEnvAPIKey := os.Getenv("HONEYBADGER_API_KEY")
	originalExitFunc := exitFunc
	defer func() {
		// Restore original values after test
		http.DefaultClient = originalClient
		exitFunc = originalExitFunc
		if err := os.Setenv("HONEYBADGER_API_KEY", originalEnvAPIKey); err != nil {
			t.Errorf("error restoring environment variable: %v", err)
		}
	}()

	// Unset environment variable for tests
	if err := os.Unsetenv("HONEYBADGER_API_KEY"); err != nil {
		t.Errorf("error unsetting environment variable: %v", err)
	}

	// Create a test script
	var scriptExt string
	var scriptContent string
	if runtime.GOOS == "windows" {
		scriptExt = ".bat"
		scriptContent = `@echo off
echo Hello, stdout!
echo Error message! 1>&2
exit 0
`
	} else {
		scriptExt = ".sh"
		scriptContent = `#!/bin/sh
echo "Hello, stdout!"
echo "Error message!" >&2
exit 0
`
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test"+scriptExt)
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0o700) // nolint:gosec
	require.NoError(t, err)

	// Create a failing script
	failingScriptPath := filepath.Join(tmpDir, "failing-test"+scriptExt)
	var failingScriptContent string
	if runtime.GOOS == "windows" {
		failingScriptContent = `@echo off
echo Hello from failing script!
exit 42
`
	} else {
		failingScriptContent = `#!/bin/sh
echo "Hello from failing script!"
exit 42
`
	}
	err = os.WriteFile(failingScriptPath, []byte(failingScriptContent), 0o700) // nolint:gosec
	require.NoError(t, err)

	tests := []struct {
		name             string
		args             []string
		apiKey           string
		expectedPath     string
		expectedStatus   int
		expectedError    bool
		expectedExitCode int
		validateBody     func(*testing.T, testCheckInPayload)
	}{
		{
			name:             "successful command execution with ID",
			args:             []string{"--id", "check-123", scriptPath},
			apiKey:           "", // API key not required when using --id
			expectedPath:     "/v1/check_in/check-123",
			expectedStatus:   http.StatusOK,
			expectedError:    false,
			expectedExitCode: 0,
			validateBody: func(t *testing.T, payload testCheckInPayload) {
				assert.Equal(t, "success", payload.CheckIn.Status)
				assert.Contains(t, payload.CheckIn.Stdout, "Hello, stdout!")
				assert.Contains(t, payload.CheckIn.Stderr, "Error message!")
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, int64(0))
				assert.Equal(t, 0, payload.CheckIn.ExitCode)
			},
		},
		{
			name:             "successful command execution with slug",
			args:             []string{"--slug", "daily-backup", scriptPath},
			apiKey:           "test-api-key",
			expectedPath:     "/v1/check_in/test-api-key/daily-backup",
			expectedStatus:   http.StatusOK,
			expectedError:    false,
			expectedExitCode: 0,
			validateBody: func(t *testing.T, payload testCheckInPayload) {
				assert.Equal(t, "success", payload.CheckIn.Status)
				assert.Contains(t, payload.CheckIn.Stdout, "Hello, stdout!")
				assert.Contains(t, payload.CheckIn.Stderr, "Error message!")
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, int64(0))
				assert.Equal(t, 0, payload.CheckIn.ExitCode)
			},
		},
		{
			name:          "missing api key with slug",
			args:          []string{"--slug", "daily-backup", "echo", "test"},
			apiKey:        "",
			expectedError: true,
		},
		{
			name:          "missing both id and slug",
			args:          []string{"echo", "test"},
			apiKey:        "test-api-key",
			expectedError: true,
		},
		{
			name:          "both id and slug specified",
			args:          []string{"--id", "check-123", "--slug", "daily-backup", "echo", "test"},
			apiKey:        "test-api-key",
			expectedError: true,
		},
		{
			name:             "failing command with exit code",
			args:             []string{"--id", "check-123", failingScriptPath},
			apiKey:           "",
			expectedPath:     "/v1/check_in/check-123",
			expectedStatus:   http.StatusOK,
			expectedError:    false,
			expectedExitCode: 42,
			validateBody: func(t *testing.T, payload testCheckInPayload) {
				assert.Equal(t, "error", payload.CheckIn.Status)
				assert.Contains(t, payload.CheckIn.Stdout, "Hello from failing script!")
				assert.Empty(t, payload.CheckIn.Stderr)
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, int64(0))
				assert.Equal(t, 42, payload.CheckIn.ExitCode)
			},
		},
		{
			name:             "non-existent command",
			args:             []string{"--id", "check-123", "nonexistent-command"},
			apiKey:           "",
			expectedPath:     "/v1/check_in/check-123",
			expectedStatus:   http.StatusOK,
			expectedError:    false,
			expectedExitCode: -1,
			validateBody: func(t *testing.T, payload testCheckInPayload) {
				assert.Equal(t, "error", payload.CheckIn.Status)
				assert.Empty(t, payload.CheckIn.Stdout)
				assert.Empty(t, payload.CheckIn.Stderr)
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, int64(0))
				assert.Equal(t, -1, payload.CheckIn.ExitCode)
			},
		},
		{
			name:             "reporting failure still exits with command code",
			args:             []string{"--id", "check-123", scriptPath},
			apiKey:           "",
			expectedPath:     "/v1/check_in/check-123",
			expectedStatus:   http.StatusInternalServerError,
			expectedError:    false,
			expectedExitCode: 0,
			validateBody: func(t *testing.T, payload testCheckInPayload) {
				assert.Equal(t, "success", payload.CheckIn.Status)
				assert.Contains(t, payload.CheckIn.Stdout, "Hello, stdout!")
				assert.Contains(t, payload.CheckIn.Stderr, "Error message!")
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, int64(0))
				assert.Equal(t, 0, payload.CheckIn.ExitCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables to avoid test pollution
			checkInID = ""
			slug = ""
			runExitCode = 0

			// Track exit code
			var capturedExitCode int
			exitFunc = func(code int) {
				capturedExitCode = code
			}

			// Create a test server
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, tt.expectedPath, r.URL.Path)
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

					if tt.apiKey == "invalid-key" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}

					// Verify payload
					if tt.validateBody != nil {
						var payload testCheckInPayload
						err := json.NewDecoder(r.Body).Decode(&payload)
						assert.NoError(t, err)
						tt.validateBody(t, payload)
					}

					w.WriteHeader(tt.expectedStatus)
				}),
			)
			defer server.Close()

			// Override the default HTTP client
			http.DefaultClient = server.Client()

			// Reset viper config
			viper.Reset()
			// Disable environment variable loading
			viper.AutomaticEnv()
			viper.SetEnvPrefix("HONEYBADGER")
			if tt.apiKey != "" {
				viper.Set("api_key", tt.apiKey)
			}
			viper.Set("endpoint", server.URL)

			// Create a new command for each test to avoid flag conflicts
			cmd := &cobra.Command{Use: "run"}
			cmd.Flags().StringVarP(&checkInID, "id", "i", "", "Check-in ID to report")
			cmd.Flags().StringVarP(&slug, "slug", "s", "", "Check-in slug to report")
			cmd.RunE = runCmd.RunE

			// Execute command
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify exit code propagation
				assert.Equal(t, tt.expectedExitCode, capturedExitCode)
			}
		})
	}
}

func TestCheckInPayloadConstruction(t *testing.T) {
	// Create and populate payload
	payload := checkInPayload{}
	payload.CheckIn.Status = "success"
	payload.CheckIn.Duration = 42000 // milliseconds
	payload.CheckIn.Stdout = "standard output"
	payload.CheckIn.Stderr = "error output"
	payload.CheckIn.ExitCode = 0

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	assert.NoError(t, err)

	// Unmarshal back to verify structure
	var decoded checkInPayload
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)

	// Verify fields
	assert.Equal(t, "success", decoded.CheckIn.Status)
	assert.Equal(t, int64(42000), decoded.CheckIn.Duration)
	assert.Equal(t, "standard output", decoded.CheckIn.Stdout)
	assert.Equal(t, "error output", decoded.CheckIn.Stderr)
	assert.Equal(t, 0, decoded.CheckIn.ExitCode)
}

func TestLimitedBuffer(t *testing.T) {
	limiter := &sharedLimiter{remaining: maxOutputSize}
	stdout := &limitedBuffer{limiter: limiter}
	stderr := &limitedBuffer{limiter: limiter}

	half := maxOutputSize / 2
	_, err := stdout.Write(bytes.Repeat([]byte("a"), half))
	assert.NoError(t, err)

	_, err = stderr.Write(bytes.Repeat([]byte("b"), half+100))
	assert.NoError(t, err)

	assert.Equal(t, maxOutputSize, len(stdout.String())+len(stderr.String()))
	assert.Contains(t, stderr.String(), "[output truncated]")
}
