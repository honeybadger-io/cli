package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetFaultUpdateFlags clears flag values and Changed state between test runs
func resetFaultUpdateFlags(t *testing.T) {
	t.Helper()

	faultResolved = false
	faultIgnored = false
	faultResolveOnDeploy = false
	faultAssigneeID = 0
	faultUnassign = false

	for _, name := range []string{"resolved", "ignored", "resolve-on-deploy"} {
		flag := faultsUpdateCmd.Flags().Lookup(name)
		require.NotNil(t, flag)
		flag.Changed = false
	}
}

func TestFaultsUpdateCommand(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		faultIDValue   int
		authToken      string
		setFlags       map[string]string
		assigneeID     int
		unassign       bool
		expectedBody   string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "resolve fault",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			setFlags:       map[string]string{"resolved": "true"},
			expectedBody:   `{"fault":{"resolved":true}}`,
			expectedError:  false,
		},
		{
			name:           "un-resolve fault",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			setFlags:       map[string]string{"resolved": "false"},
			expectedBody:   `{"fault":{"resolved":false}}`,
			expectedError:  false,
		},
		{
			name:           "ignore fault",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			setFlags:       map[string]string{"ignored": "true"},
			expectedBody:   `{"fault":{"ignored":true}}`,
			expectedError:  false,
		},
		{
			name:           "resolve on deploy",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			setFlags:       map[string]string{"resolve-on-deploy": "true"},
			expectedBody:   `{"fault":{"resolve_on_deploy":true}}`,
			expectedError:  false,
		},
		{
			name:           "assign fault",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			assigneeID:     42,
			expectedBody:   `{"fault":{"assignee_id":42}}`,
			expectedError:  false,
		},
		{
			name:           "unassign fault",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			unassign:       true,
			expectedBody:   `{"fault":{"assignee_id":null}}`,
			expectedError:  false,
		},
		{
			name:           "missing project ID",
			projectIDValue: 0,
			faultIDValue:   456,
			authToken:      "test-token",
			setFlags:       map[string]string{"resolved": "true"},
			expectedError:  true,
			errorContains:  "project ID is required",
		},
		{
			name:           "missing fault ID",
			projectIDValue: 123,
			faultIDValue:   0,
			authToken:      "test-token",
			setFlags:       map[string]string{"resolved": "true"},
			expectedError:  true,
			errorContains:  "fault ID is required",
		},
		{
			name:           "nothing to update",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			expectedError:  true,
			errorContains:  "nothing to update",
		},
		{
			name:           "assignee-id and unassign are mutually exclusive",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "test-token",
			assigneeID:     42,
			unassign:       true,
			expectedError:  true,
			errorContains:  "mutually exclusive",
		},
		{
			name:           "missing auth token",
			projectIDValue: 123,
			faultIDValue:   456,
			authToken:      "",
			setFlags:       map[string]string{"resolved": "true"},
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
						assert.Equal(t, "PUT", r.Method)
						assert.Equal(t, "/v2/projects/123/faults/456", r.URL.Path)

						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						assert.JSONEq(t, tt.expectedBody, string(body))

						w.WriteHeader(http.StatusNoContent)
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

			resetFaultUpdateFlags(t)
			faultsProjectID = tt.projectIDValue
			faultID = tt.faultIDValue
			faultAssigneeID = tt.assigneeID
			faultUnassign = tt.unassign
			for name, value := range tt.setFlags {
				require.NoError(t, faultsUpdateCmd.Flags().Set(name, value))
			}

			err := faultsUpdateCmd.RunE(faultsUpdateCmd, []string{})

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

func TestInsightsQueryStreamIDs(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var request map[string]interface{}
			require.NoError(t, json.Unmarshal(body, &request))
			assert.Equal(t, []interface{}{"abc123", "def456"}, request["stream_ids"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"meta": {"query": "fields @ts", "fields": [], "rows": 0, "total_rows": 0},
				"results": []
			}`))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	insightsProjectID = 123
	insightsQuery = "fields @ts"
	insightsTimestamp = ""
	insightsTimezone = ""
	insightsStreamIDs = []string{"abc123", "def456"}
	insightsOutputFormat = "table"
	defer func() { insightsStreamIDs = nil }()

	err := insightsQueryCmd.RunE(insightsQueryCmd, []string{})
	assert.NoError(t, err)
}
