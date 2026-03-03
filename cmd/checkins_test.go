package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestCheckinsListCommand(t *testing.T) {
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
				"results": [{"id": "abc123", "name": "Daily Backup", "slug": "daily-backup", "schedule_type": "simple", "report_period": "1 day"}],
				"links": {"self": "/v2/projects/123/check_ins"}
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

			checkinsProjectID = tt.projectIDValue
			checkinsOutputFormat = "table"

			err := checkinsListCmd.RunE(checkinsListCmd, []string{})

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

func TestCheckinsGetCommand(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		checkinIDValue string
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful get",
			projectIDValue: 123,
			checkinIDValue: "abc123",
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody: `{
				"id": "abc123",
				"name": "Daily Backup",
				"slug": "daily-backup",
				"schedule_type": "simple",
				"report_period": "1 day"
			}`,
			expectedError: false,
		},
		{
			name:           "missing project ID",
			projectIDValue: 0,
			checkinIDValue: "abc123",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "project ID is required",
		},
		{
			name:           "missing checkin ID",
			projectIDValue: 123,
			checkinIDValue: "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "check-in ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.projectIDValue != 0 && tt.checkinIDValue != "" {
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

			checkinsProjectID = tt.projectIDValue
			checkinID = tt.checkinIDValue
			checkinsOutputFormat = "text"

			err := checkinsGetCmd.RunE(checkinsGetCmd, []string{})

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

func TestCheckinsViperProjectIDFallback(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"id": "abc123", "name": "Daily Backup", "slug": "daily-backup", "schedule_type": "simple", "report_period": "1 day"}],
				"links": {"self": "/v2/projects/123/check_ins"}
			}`))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")
	viper.Set("project_id", 123)

	checkinsProjectID = 0
	checkinsOutputFormat = "table"

	err := checkinsListCmd.RunE(checkinsListCmd, []string{})
	assert.NoError(t, err)
}

func TestCheckinsOutputFormat(t *testing.T) {
	mockResponse := `{
		"results": [{"id": "abc123", "name": "Daily Backup", "slug": "daily-backup", "schedule_type": "simple", "report_period": "1 day"}],
		"links": {"self": "/v2/projects/123/check_ins"}
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

			checkinsProjectID = 123
			checkinsOutputFormat = tt.format

			err := checkinsListCmd.RunE(checkinsListCmd, []string{})
			assert.NoError(t, err)
		})
	}
}
