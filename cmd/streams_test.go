package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestStreamsListCommand(t *testing.T) {
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
			serverBody: `{
				"results": [{"id": "abc123", "name": "Rails Logs", "slug": "rails-logs", "internal": false, "project_id": 123, "created_at": "2024-01-01T00:00:00Z"}]
			}`,
			expectedError: false,
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
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "GET", r.Method)
						assert.Equal(t, "/v2/projects/123/streams", r.URL.Path)

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

			streamsProjectID = tt.projectIDValue
			streamsOutputFormat = "table"

			err := streamsListCmd.RunE(streamsListCmd, []string{})

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

func TestStreamsViperProjectIDFallback(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"results": []}`))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")
	viper.Set("project_id", 123)

	streamsProjectID = 0
	streamsOutputFormat = "table"

	err := streamsListCmd.RunE(streamsListCmd, []string{})
	assert.NoError(t, err)
}

func TestStreamsOutputFormat(t *testing.T) {
	mockResponse := `{
		"results": [{"id": "abc123", "name": "Rails Logs", "slug": "rails-logs", "internal": true, "project_id": null, "created_at": "2024-01-01T00:00:00Z"}]
	}`

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockResponse))
		}),
	)
	defer server.Close()

	tests := []struct {
		name   string
		format string
	}{
		{name: "table format", format: "table"},
		{name: "json format", format: "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("endpoint", server.URL)
			viper.Set("auth_token", "test-token")

			streamsProjectID = 123
			streamsOutputFormat = tt.format

			err := streamsListCmd.RunE(streamsListCmd, []string{})
			assert.NoError(t, err)
		})
	}
}
