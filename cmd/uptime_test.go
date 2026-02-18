package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUptimeSitesListCommand(t *testing.T) {
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
				"results": [{"id": "site1", "name": "Site 1", "url": "https://example.com", "state": "up", "active": true, "frequency": 5}],
				"links": {"self": "/v2/projects/123/sites"}
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

			uptimeProjectID = tt.projectIDValue
			uptimeOutputFormat = "table"

			err := uptimeSitesListCmd.RunE(uptimeSitesListCmd, []string{})

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

func TestUptimeSitesGetCommand(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		siteIDValue    string
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful get",
			projectIDValue: 123,
			siteIDValue:    "site1",
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody: `{
				"id": "site1",
				"name": "Site 1",
				"url": "https://example.com",
				"state": "up",
				"active": true,
				"frequency": 5,
				"match_type": "contains"
			}`,
			expectedError: false,
		},
		{
			name:           "missing project ID",
			projectIDValue: 0,
			siteIDValue:    "site1",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "project ID is required",
		},
		{
			name:           "missing site ID",
			projectIDValue: 123,
			siteIDValue:    "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "site ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.projectIDValue != 0 && tt.siteIDValue != "" {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "GET", r.Method)

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

			uptimeProjectID = tt.projectIDValue
			uptimeSiteID = tt.siteIDValue
			uptimeOutputFormat = "text"

			err := uptimeSitesGetCmd.RunE(uptimeSitesGetCmd, []string{})

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

func TestUptimeViperProjectIDFallback(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"id": "site1", "name": "Site 1", "url": "https://example.com", "state": "up", "active": true, "frequency": 5}],
				"links": {"self": "/v2/projects/123/sites"}
			}`))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")
	viper.Set("project_id", 123)

	uptimeProjectID = 0
	uptimeOutputFormat = "table"

	err := uptimeSitesListCmd.RunE(uptimeSitesListCmd, []string{})
	assert.NoError(t, err)
}

func TestUptimeOutputFormat(t *testing.T) {
	mockResponse := `{
		"results": [{"id": "site1", "name": "Site 1", "url": "https://example.com", "state": "up", "active": true, "frequency": 5}],
		"links": {"self": "/v2/projects/123/sites"}
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

			uptimeProjectID = 123
			uptimeOutputFormat = tt.format

			err := uptimeSitesListCmd.RunE(uptimeSitesListCmd, []string{})
			assert.NoError(t, err)
		})
	}
}

// TestOutagesTimestampConversion verifies that human-readable date strings
// are parsed to time.Time and sent as Unix epoch query params to the API.
func TestOutagesTimestampConversion(t *testing.T) {
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

	uptimeProjectID = 123
	uptimeSiteID = "abc123"
	uptimeCreatedAfter = "2024-01-01"
	uptimeCreatedBefore = "2024-01-02"
	uptimeLimit = 25
	uptimeOutputFormat = "json"

	err := uptimeOutagesCmd.RunE(uptimeOutagesCmd, []string{})
	require.NoError(t, err)

	// Reset for other tests
	uptimeCreatedAfter = ""
	uptimeCreatedBefore = ""
}

// TestOutagesEmptyTimestampsOmitted verifies that empty timestamp flags
// are not sent as query parameters.
func TestOutagesEmptyTimestampsOmitted(t *testing.T) {
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

	uptimeProjectID = 123
	uptimeSiteID = "abc123"
	uptimeCreatedAfter = ""
	uptimeCreatedBefore = ""
	uptimeOutputFormat = "json"

	err := uptimeOutagesCmd.RunE(uptimeOutagesCmd, []string{})
	require.NoError(t, err)
}

// TestUptimeChecksTimestampConversion verifies the same date-string-to-epoch
// conversion works for the uptime checks command.
func TestUptimeChecksTimestampConversion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "1704067200", query.Get("created_after"))
		assert.Equal(t, "1704153600", query.Get("created_before"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	uptimeProjectID = 123
	uptimeSiteID = "abc123"
	uptimeCreatedAfter = "2024-01-01T00:00:00Z"
	uptimeCreatedBefore = "2024-01-02T00:00:00Z"
	uptimeLimit = 25
	uptimeOutputFormat = "json"

	err := uptimeChecksCmd.RunE(uptimeChecksCmd, []string{})
	require.NoError(t, err)

	uptimeCreatedAfter = ""
	uptimeCreatedBefore = ""
}
