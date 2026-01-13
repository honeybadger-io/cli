package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestCheckInCommand(t *testing.T) {
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
		expectedPath   string
		expectedMethod string
		expectedStatus int
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful check-in with ID",
			args:           []string{"--id", "XyZZy"},
			apiKey:         "",
			expectedPath:   "/v1/check_in/XyZZy",
			expectedMethod: "GET",
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "successful check-in with slug",
			args:           []string{"--slug", "daily-backup"},
			apiKey:         "test-api-key",
			expectedPath:   "/v1/check_in/test-api-key/daily-backup",
			expectedMethod: "GET",
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:          "missing api key with slug",
			args:          []string{"--slug", "daily-backup"},
			apiKey:        "",
			expectedError: true,
			errorContains: "API key is required",
		},
		{
			name:          "missing both id and slug",
			args:          []string{},
			apiKey:        "test-api-key",
			expectedError: true,
			errorContains: "either check-in ID (--id) or slug (--slug) is required",
		},
		{
			name:          "both id and slug specified",
			args:          []string{"--id", "XyZZy", "--slug", "daily-backup"},
			apiKey:        "test-api-key",
			expectedError: true,
			errorContains: "cannot specify both check-in ID and slug",
		},
		{
			name:           "server returns error",
			args:           []string{"--id", "invalid-id"},
			apiKey:         "",
			expectedPath:   "/v1/check_in/invalid-id",
			expectedMethod: "GET",
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
			errorContains:  "unexpected status code: 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request
					assert.Equal(t, tt.expectedMethod, r.Method)
					assert.Equal(t, tt.expectedPath, r.URL.Path)

					w.WriteHeader(tt.expectedStatus)
				}),
			)
			defer server.Close()

			// Override the default HTTP client
			http.DefaultClient = server.Client()

			// Reset viper config
			viper.Reset()
			viper.AutomaticEnv()
			viper.SetEnvPrefix("HONEYBADGER")
			if tt.apiKey != "" {
				viper.Set("api_key", tt.apiKey)
			}
			viper.Set("endpoint", server.URL)

			// Create a new command for each test to avoid flag conflicts
			cmd := &cobra.Command{Use: "check-in"}
			cmd.Flags().StringVarP(&checkInCmdID, "id", "i", "", "Check-in ID to report")
			cmd.Flags().StringVarP(&checkInCmdSlug, "slug", "s", "", "Check-in slug to report")
			cmd.RunE = checkInCmd.RunE

			// Execute command
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
