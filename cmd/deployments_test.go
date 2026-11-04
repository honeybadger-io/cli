package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeploymentsListCommand tests the deployments list command validation
func TestDeploymentsListCommand(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful list",
			projectIDValue: 123,
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody:     `{"results": [], "links": {"self": "/v2/projects/123/deploys"}}`,
			expectedError:  false,
		},
		{
			name:           "missing project ID",
			projectIDValue: 0,
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "project ID is required",
		},
		{
			name:           "missing auth token",
			projectIDValue: 123,
			authToken:      "",
			expectedError:  true,
			errorContains:  "auth token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.projectIDValue != 0 {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(tt.serverStatus)
						_, _ = w.Write([]byte(tt.serverBody))
					}),
				)
				defer server.Close()
				serverURL = server.URL
			} else {
				serverURL = "http://localhost:9999"
			}

			viper.Reset()
			viper.Set("endpoint", serverURL)
			if tt.authToken != "" {
				viper.Set("auth_token", tt.authToken)
			}

			deploymentsProjectID = tt.projectIDValue
			deploymentsOutputFormat = "table"

			err := deploymentsListCmd.RunE(deploymentsListCmd, []string{})

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

// TestDeploymentsViperProjectIDFallback tests that the deployments command uses viper project_id fallback
func TestDeploymentsViperProjectIDFallback(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results": [], "links": {"self": "/v2/projects/123/deploys"}}`))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")
	viper.Set("project_id", 123)

	deploymentsProjectID = 0
	deploymentsOutputFormat = "table"

	err := deploymentsListCmd.RunE(deploymentsListCmd, []string{})
	assert.NoError(t, err)
}

// TestDeploymentsTimestampConversion verifies that human-readable date strings
// are parsed to time.Time and sent as Unix epoch query params to the API.
func TestDeploymentsTimestampConversion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		// 2024-01-01T00:00:00Z = Unix 1704067200
		assert.Equal(t, "1704067200", query.Get("created_after"),
			"created_after should be sent as Unix epoch seconds")
		// 2024-01-02T00:00:00Z = Unix 1704153600
		assert.Equal(t, "1704153600", query.Get("created_before"),
			"created_before should be sent as Unix epoch seconds")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	deploymentsProjectID = 123
	deploymentsCreatedAfter = "2024-01-01"
	deploymentsCreatedBefore = "2024-01-02"
	deploymentsLimit = 25
	deploymentsOutputFormat = "json"

	err := deploymentsListCmd.RunE(deploymentsListCmd, []string{})
	require.NoError(t, err)

	// Reset for other tests
	deploymentsCreatedAfter = ""
	deploymentsCreatedBefore = ""
}

// TestDeploymentsEmptyTimestampsOmitted verifies that empty timestamp flags
// are not sent as query parameters.
func TestDeploymentsEmptyTimestampsOmitted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Empty(t, query.Get("created_after"),
			"empty timestamp should not be sent as a query param")
		assert.Empty(t, query.Get("created_before"),
			"empty timestamp should not be sent as a query param")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	deploymentsProjectID = 123
	deploymentsCreatedAfter = ""
	deploymentsCreatedBefore = ""
	deploymentsOutputFormat = "json"

	err := deploymentsListCmd.RunE(deploymentsListCmd, []string{})
	require.NoError(t, err)
}
