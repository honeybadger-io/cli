package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAlarmsListCommand(t *testing.T) {
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
				"results": [{"id": "alarm-1", "name": "High Error Rate", "state": "ok", "query": "filter event_type::str == \"notice\"", "evaluation_period": "5m"}],
				"links": {"self": "/v2/projects/123/alarms"}
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

			alarmsProjectID = tt.projectIDValue
			alarmsOutputFormat = "table"

			err := alarmsListCmd.RunE(alarmsListCmd, []string{})

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

func TestAlarmsGetCommand(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		alarmIDValue   string
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful get",
			projectIDValue: 123,
			alarmIDValue:   "alarm-1",
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody: `{
				"id": "alarm-1",
				"name": "High Error Rate",
				"state": "ok",
				"query": "filter event_type::str == \"notice\"",
				"evaluation_period": "5m",
				"trigger_config": {"type": "alert_result_count", "config": {"operator": "gt", "value": 10}}
			}`,
			expectedError: false,
		},
		{
			name:           "missing project ID",
			projectIDValue: 0,
			alarmIDValue:   "alarm-1",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "project ID is required",
		},
		{
			name:           "missing alarm ID",
			projectIDValue: 123,
			alarmIDValue:   "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "alarm ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.projectIDValue != 0 && tt.alarmIDValue != "" {
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

			alarmsProjectID = tt.projectIDValue
			alarmID = tt.alarmIDValue
			alarmsOutputFormat = "text"

			err := alarmsGetCmd.RunE(alarmsGetCmd, []string{})

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

func TestAlarmsHistoryCommand(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		alarmIDValue   string
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful history",
			projectIDValue: 123,
			alarmIDValue:   "alarm-1",
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody: `{
				"triggers": [{"id": "trigger-1", "state": "alarm", "result": {"count": 42}, "created_at": "2024-01-01T00:00:00Z"}],
				"links": {"self": "/v2/projects/123/alarms/alarm-1/history"}
			}`,
			expectedError: false,
		},
		{
			name:           "missing alarm ID",
			projectIDValue: 123,
			alarmIDValue:   "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "alarm ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.projectIDValue != 0 && tt.alarmIDValue != "" {
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

			alarmsProjectID = tt.projectIDValue
			alarmID = tt.alarmIDValue
			alarmHistoryPage = 0
			alarmsOutputFormat = "table"

			err := alarmsHistoryCmd.RunE(alarmsHistoryCmd, []string{})

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

func TestAlarmsViperProjectIDFallback(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"results": [{"id": "alarm-1", "name": "High Error Rate", "state": "ok", "query": "filter event_type::str == \"notice\"", "evaluation_period": "5m"}],
				"links": {"self": "/v2/projects/123/alarms"}
			}`))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")
	viper.Set("project_id", 123)

	alarmsProjectID = 0
	alarmsOutputFormat = "table"

	err := alarmsListCmd.RunE(alarmsListCmd, []string{})
	assert.NoError(t, err)
}

func TestAlarmsOutputFormat(t *testing.T) {
	mockResponse := `{
		"results": [{"id": "alarm-1", "name": "High Error Rate", "state": "ok", "query": "filter event_type::str == \"notice\"", "evaluation_period": "5m"}],
		"links": {"self": "/v2/projects/123/alarms"}
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

			alarmsProjectID = 123
			alarmsOutputFormat = tt.format

			err := alarmsListCmd.RunE(alarmsListCmd, []string{})
			assert.NoError(t, err)
		})
	}
}
