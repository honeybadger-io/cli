package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestTeamsListCommand(t *testing.T) {
	tests := []struct {
		name           string
		accountIDValue int
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful list",
			accountIDValue: 123,
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody: `[
				{"id": 1, "name": "Team 1", "account_id": 123, "created_at": "2024-01-01T00:00:00Z"}
			]`,
			expectedError: false,
		},
		{
			name:           "missing account ID",
			accountIDValue: 0,
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "account ID is required",
		},
		{
			name:           "missing auth token",
			accountIDValue: 123,
			authToken:      "",
			expectedError:  true,
			errorContains:  "auth token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.accountIDValue != 0 {
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

			teamsAccountID = tt.accountIDValue
			teamsOutputFormat = "table"

			err := teamsListCmd.RunE(teamsListCmd, []string{})

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

func TestTeamsGetCommand(t *testing.T) {
	tests := []struct {
		name          string
		teamIDValue   int
		authToken     string
		serverStatus  int
		serverBody    string
		expectedError bool
		errorContains string
	}{
		{
			name:         "successful get",
			teamIDValue:  1,
			authToken:    "test-token",
			serverStatus: http.StatusOK,
			serverBody: `{
				"id": 1,
				"name": "Team 1",
				"account_id": 123,
				"created_at": "2024-01-01T00:00:00Z"
			}`,
			expectedError: false,
		},
		{
			name:          "missing team ID",
			teamIDValue:   0,
			authToken:     "test-token",
			expectedError: true,
			errorContains: "team ID is required",
		},
		{
			name:          "missing auth token",
			teamIDValue:   1,
			authToken:     "",
			expectedError: true,
			errorContains: "auth token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.teamIDValue != 0 {
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

			teamID = tt.teamIDValue
			teamsOutputFormat = "text"

			err := teamsGetCmd.RunE(teamsGetCmd, []string{})

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

func TestTeamsCreateCommand(t *testing.T) {
	tests := []struct {
		name           string
		accountIDValue int
		teamNameValue  string
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful create",
			accountIDValue: 123,
			teamNameValue:  "New Team",
			authToken:      "test-token",
			serverStatus:   http.StatusCreated,
			serverBody: `{
				"id": 1,
				"name": "New Team",
				"account_id": 123,
				"created_at": "2024-01-01T00:00:00Z"
			}`,
			expectedError: false,
		},
		{
			name:           "missing account ID",
			accountIDValue: 0,
			teamNameValue:  "New Team",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "account ID is required",
		},
		{
			name:           "missing team name",
			accountIDValue: 123,
			teamNameValue:  "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "team name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.accountIDValue != 0 && tt.teamNameValue != "" {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "POST", r.Method)

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

			teamsAccountID = tt.accountIDValue
			teamName = tt.teamNameValue
			teamsOutputFormat = "text"

			err := teamsCreateCmd.RunE(teamsCreateCmd, []string{})

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

func TestTeamsMembersListCommand(t *testing.T) {
	tests := []struct {
		name          string
		teamIDValue   int
		authToken     string
		serverStatus  int
		serverBody    string
		expectedError bool
		errorContains string
	}{
		{
			name:         "successful list members",
			teamIDValue:  1,
			authToken:    "test-token",
			serverStatus: http.StatusOK,
			serverBody: `[
				{"id": 1, "name": "Member 1", "email": "member1@example.com", "admin": true}
			]`,
			expectedError: false,
		},
		{
			name:          "missing team ID",
			teamIDValue:   0,
			authToken:     "test-token",
			expectedError: true,
			errorContains: "team ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.teamIDValue != 0 {
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

			teamID = tt.teamIDValue
			teamsOutputFormat = "table"

			err := teamsMembersListCmd.RunE(teamsMembersListCmd, []string{})

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

func TestTeamsOutputFormat(t *testing.T) {
	mockResponse := `[
		{"id": 1, "name": "Team 1", "account_id": 123, "created_at": "2024-01-01T00:00:00Z"}
	]`

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

			teamsAccountID = 123
			teamsOutputFormat = tt.format

			err := teamsListCmd.RunE(teamsListCmd, []string{})
			assert.NoError(t, err)
		})
	}
}
