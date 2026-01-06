package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDeployCommand(t *testing.T) {
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

	tests := []struct {
		name           string
		args           []string
		apiKey         string
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful deploy",
			args: []string{
				"--environment",
				"production",
				"--repository",
				"github.com/org/repo",
				"--revision",
				"abc123",
				"--user",
				"testuser",
			},
			apiKey:         "test-api-key",
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name:          "missing api key",
			args:          []string{"--environment", "production"},
			apiKey:        "",
			expectedError: true,
		},
		{
			name:          "missing required environment",
			args:          []string{"--repository", "github.com/org/repo"},
			apiKey:        "test-api-key",
			expectedError: true,
		},
		{
			name:           "unauthorized",
			args:           []string{"--environment", "production"},
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, "/v1/deploys", r.URL.Path)
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
					assert.Equal(t, tt.apiKey, r.Header.Get("X-API-Key"))

					if tt.apiKey == "invalid-key" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}

					// Verify payload
					var payload deployPayload
					err := json.NewDecoder(r.Body).Decode(&payload)
					assert.NoError(t, err)

					// Verify timestamp format
					_, err = time.Parse(time.RFC3339, payload.Deploy.Timestamp)
					assert.NoError(t, err)

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
			cmd := &cobra.Command{Use: "deploy"}
			cmd.Flags().
				StringVarP(&environment, "environment", "e", "", "Environment being deployed to")
			cmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository being deployed")
			cmd.Flags().StringVarP(&revision, "revision", "v", "", "Revision being deployed")
			cmd.Flags().StringVarP(&localUser, "user", "u", "", "Local username")
			cmd.RunE = deployCmd.RunE

			// Reset command state
			environment = ""
			repository = ""
			revision = ""
			localUser = ""

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

func TestDeployPayloadConstruction(t *testing.T) {
	// Set up test environment
	environment := "production"
	repository := "github.com/org/repo"
	revision := "abc123"
	localUser := "testuser"

	// Create and populate payload
	payload := deployPayload{}
	payload.Deploy.Environment = environment
	payload.Deploy.Repository = repository
	payload.Deploy.Revision = revision
	payload.Deploy.LocalUser = localUser
	payload.Deploy.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	assert.NoError(t, err)

	// Unmarshal back to verify structure
	var decoded deployPayload
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)

	// Verify fields
	assert.Equal(t, environment, decoded.Deploy.Environment)
	assert.Equal(t, repository, decoded.Deploy.Repository)
	assert.Equal(t, revision, decoded.Deploy.Revision)
	assert.Equal(t, localUser, decoded.Deploy.LocalUser)

	// Verify timestamp format
	_, err = time.Parse(time.RFC3339, decoded.Deploy.Timestamp)
	assert.NoError(t, err)
}
