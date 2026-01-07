package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAccountsListCommand(t *testing.T) {
	tests := []struct {
		name          string
		authToken     string
		serverStatus  int
		serverBody    string
		expectedError bool
		errorContains string
	}{
		{
			name:         "successful list",
			authToken:    "test-token",
			serverStatus: http.StatusOK,
			serverBody: `{
				"results": [
					{"id": "abc123", "email": "test@example.com", "name": "Test Account"}
				]
			}`,
			expectedError: false,
		},
		{
			name:          "missing auth token",
			authToken:     "",
			expectedError: true,
			errorContains: "auth token is required",
		},
		{
			name:          "unauthorized",
			authToken:     "invalid-token",
			serverStatus:  http.StatusUnauthorized,
			serverBody:    `{"error": "Unauthorized"}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server only if we expect to make requests
			var serverURL string
			if tt.authToken != "" {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "GET", r.Method)
						assert.Equal(t, "/v2/accounts", r.URL.Path)

						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(tt.serverStatus)
						_, _ = w.Write([]byte(tt.serverBody))
					}),
				)
				defer server.Close()
				serverURL = server.URL
			} else {
				serverURL = "http://localhost:9999" // Won't be called
			}

			// Reset viper and set config
			viper.Reset()
			viper.Set("endpoint", serverURL)
			if tt.authToken != "" {
				viper.Set("auth_token", tt.authToken)
			}

			// Reset command state
			accountsOutputFormat = "table"

			// Execute command
			err := accountsListCmd.RunE(accountsListCmd, []string{})

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

func TestAccountsGetCommand(t *testing.T) {
	tests := []struct {
		name           string
		accountIDValue string
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful get",
			accountIDValue: "abc123",
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody: `{
				"id": "abc123",
				"email": "test@example.com",
				"name": "Test Account"
			}`,
			expectedError: false,
		},
		{
			name:           "missing account ID",
			accountIDValue: "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "account ID is required",
		},
		{
			name:           "missing auth token",
			accountIDValue: "abc123",
			authToken:      "",
			expectedError:  true,
			errorContains:  "auth token is required",
		},
		{
			name:           "not found",
			accountIDValue: "notfound",
			authToken:      "test-token",
			serverStatus:   http.StatusNotFound,
			serverBody:     `{"error": "Not found"}`,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.accountIDValue != "" {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "GET", r.Method)
						assert.Contains(t, r.URL.Path, "/v2/accounts/")

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

			// Set global state that the command reads
			accountID = tt.accountIDValue
			accountsOutputFormat = "text"

			err := accountsGetCmd.RunE(accountsGetCmd, []string{})

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

func TestAccountsOutputFormat(t *testing.T) {
	mockResponse := `{
		"results": [
			{"id": "abc123", "email": "test@example.com", "name": "Test Account"}
		]
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

			accountsOutputFormat = tt.format

			err := accountsListCmd.RunE(accountsListCmd, []string{})
			assert.NoError(t, err)
		})
	}
}

func TestAccountsUsersListCommand(t *testing.T) {
	tests := []struct {
		name           string
		accountIDValue string
		authToken      string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful list users",
			accountIDValue: "abc123",
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody: `{"results": [
				{"id": 1, "name": "User 1", "email": "user1@example.com", "role": "Owner"}
			]}`,
			expectedError: false,
		},
		{
			name:           "missing account ID",
			accountIDValue: "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "account ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.accountIDValue != "" {
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

			accountID = tt.accountIDValue
			accountsOutputFormat = "table"

			err := accountsUsersListCmd.RunE(accountsUsersListCmd, []string{})

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

func TestAccountsInvitationsCreateCommand(t *testing.T) {
	tests := []struct {
		name           string
		accountIDValue string
		authToken      string
		cliInputJSON   string
		serverStatus   int
		serverBody     string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful create",
			accountIDValue: "abc123",
			authToken:      "test-token",
			cliInputJSON:   `{"invitation": {"email": "new@example.com", "role": "Member"}}`,
			serverStatus:   http.StatusCreated,
			serverBody: `{
				"id": 1,
				"email": "new@example.com",
				"role": "Member",
				"token": "invite-token",
				"created_at": "2024-01-01T00:00:00Z"
			}`,
			expectedError: false,
		},
		{
			name:           "missing account ID",
			accountIDValue: "",
			authToken:      "test-token",
			cliInputJSON:   `{"invitation": {"email": "new@example.com"}}`,
			expectedError:  true,
			errorContains:  "account ID is required",
		},
		{
			name:           "missing JSON payload",
			accountIDValue: "abc123",
			authToken:      "test-token",
			cliInputJSON:   "",
			expectedError:  true,
			errorContains:  "JSON payload is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.accountIDValue != "" && tt.cliInputJSON != "" {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "POST", r.Method)

						// Verify JSON payload
						var payload map[string]interface{}
						err := json.NewDecoder(r.Body).Decode(&payload)
						assert.NoError(t, err)

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

			accountID = tt.accountIDValue
			accountCLIInputJSON = tt.cliInputJSON
			accountsOutputFormat = "text"

			err := accountsInvitationsCreateCmd.RunE(accountsInvitationsCreateCmd, []string{})

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
