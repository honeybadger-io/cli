package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectsCreateCommand(t *testing.T) {
	tests := []struct {
		name           string
		accountIDValue string
		cliInputJSON   string
		authToken      string
		expectedRawQ   string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "create with account ID",
			accountIDValue: "K7xmQq",
			cliInputJSON:   `{"project": {"name": "My Project"}}`,
			authToken:      "test-token",
			expectedRawQ:   "account_id=K7xmQq",
			expectedError:  false,
		},
		{
			name:           "create without account ID omits query param",
			accountIDValue: "",
			cliInputJSON:   `{"project": {"name": "My Project"}}`,
			authToken:      "test-token",
			expectedRawQ:   "",
			expectedError:  false,
		},
		{
			name:           "missing JSON payload",
			accountIDValue: "K7xmQq",
			cliInputJSON:   "",
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "JSON payload is required",
		},
		{
			name:           "missing auth token",
			accountIDValue: "K7xmQq",
			cliInputJSON:   `{"project": {"name": "My Project"}}`,
			authToken:      "",
			expectedError:  true,
			errorContains:  "auth token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string
			if !tt.expectedError {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "POST", r.Method)
						assert.Equal(t, "/v2/projects", r.URL.Path)
						assert.Equal(t, tt.expectedRawQ, r.URL.RawQuery)

						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						assert.JSONEq(t, `{"project": {"name": "My Project"}}`, string(body))

						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusCreated)
						_, _ = w.Write([]byte(`{"id": 123, "name": "My Project"}`))
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

			projectAccountID = tt.accountIDValue
			projectCLIInputJSON = tt.cliInputJSON
			projectOutputFormat = "text"

			err := projectsCreateCmd.RunE(projectsCreateCmd, []string{})

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
