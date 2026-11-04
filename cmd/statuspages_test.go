package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestStatuspagesListCommand tests the statuspages list command validation
func TestStatuspagesListCommand(t *testing.T) {
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
			name:           "successful list",
			accountIDValue: "123",
			authToken:      "test-token",
			serverStatus:   http.StatusOK,
			serverBody:     `{"results": []}`,
			expectedError:  false,
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
			accountIDValue: "123",
			authToken:      "",
			expectedError:  true,
			errorContains:  "auth token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if tt.authToken != "" && tt.accountIDValue != "" {
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

			statuspagesAccountID = tt.accountIDValue
			statuspagesOutputFormat = "table"

			err := statuspagesListCmd.RunE(statuspagesListCmd, []string{})

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
