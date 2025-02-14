package cmd

import (
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

func TestRunCommand(t *testing.T) {
	// Save original values
	originalClient := http.DefaultClient
	originalEnvAPIKey := os.Getenv("HONEYBADGER_API_KEY")
	defer func() {
		// Restore original values after test
		http.DefaultClient = originalClient
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
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700) // nolint:gosec
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
	err = os.WriteFile(failingScriptPath, []byte(failingScriptContent), 0700) // nolint:gosec
	require.NoError(t, err)

	tests := []struct {
		name           string
		args           []string
		apiKey         string
		expectedPath   string
		expectedStatus int
		expectedError  bool
		validateBody   func(*testing.T, checkInPayload)
	}{
		{
			name:           "successful command execution with ID",
			args:           []string{"--id", "check-123", scriptPath},
			apiKey:         "test-api-key",
			expectedPath:   "/v1/check_in/check-123",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			validateBody: func(t *testing.T, payload checkInPayload) {
				assert.Equal(t, "success", payload.CheckIn.Status)
				assert.Contains(t, payload.CheckIn.Stdout, "Hello, stdout!")
				assert.Contains(t, payload.CheckIn.Stderr, "Error message!")
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, 0)
				assert.Equal(t, 0, payload.CheckIn.ExitCode)
			},
		},
		{
			name:           "successful command execution with slug",
			args:           []string{"--slug", "daily-backup", scriptPath},
			apiKey:         "test-api-key",
			expectedPath:   "/v1/check_in/test-api-key/daily-backup",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			validateBody: func(t *testing.T, payload checkInPayload) {
				assert.Equal(t, "success", payload.CheckIn.Status)
				assert.Contains(t, payload.CheckIn.Stdout, "Hello, stdout!")
				assert.Contains(t, payload.CheckIn.Stderr, "Error message!")
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, 0)
				assert.Equal(t, 0, payload.CheckIn.ExitCode)
			},
		},
		{
			name:          "missing api key",
			args:          []string{"--id", "check-123", "echo", "test"},
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
			name:           "failing command with exit code",
			args:           []string{"--id", "check-123", failingScriptPath},
			apiKey:         "test-api-key",
			expectedPath:   "/v1/check_in/check-123",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			validateBody: func(t *testing.T, payload checkInPayload) {
				assert.Equal(t, "error", payload.CheckIn.Status)
				assert.Contains(t, payload.CheckIn.Stdout, "Hello from failing script!")
				assert.Empty(t, payload.CheckIn.Stderr)
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, 0)
				assert.Equal(t, 42, payload.CheckIn.ExitCode)
			},
		},
		{
			name:           "non-existent command",
			args:           []string{"--id", "check-123", "nonexistent-command"},
			apiKey:         "test-api-key",
			expectedPath:   "/v1/check_in/check-123",
			expectedStatus: http.StatusOK,
			expectedError:  false,
			validateBody: func(t *testing.T, payload checkInPayload) {
				assert.Equal(t, "error", payload.CheckIn.Status)
				assert.Empty(t, payload.CheckIn.Stdout)
				assert.Empty(t, payload.CheckIn.Stderr)
				assert.GreaterOrEqual(t, payload.CheckIn.Duration, 0)
				assert.Equal(t, -1, payload.CheckIn.ExitCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					var payload checkInPayload
					err := json.NewDecoder(r.Body).Decode(&payload)
					assert.NoError(t, err)
					tt.validateBody(t, payload)
				}

				w.WriteHeader(tt.expectedStatus)
			}))
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
			}
		})
	}
}

func TestCheckInPayloadConstruction(t *testing.T) {
	// Create and populate payload
	payload := checkInPayload{}
	payload.CheckIn.Status = "success"
	payload.CheckIn.Duration = 42
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
	assert.Equal(t, 42, decoded.CheckIn.Duration)
	assert.Equal(t, "standard output", decoded.CheckIn.Stdout)
	assert.Equal(t, "error output", decoded.CheckIn.Stderr)
	assert.Equal(t, 0, decoded.CheckIn.ExitCode)
}
