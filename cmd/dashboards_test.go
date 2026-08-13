package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dashboardListBody = `{
	"results": [{"id": "abc123", "title": "Request Health", "widgets": [{"id": "w1", "type": "insights_vis"}], "is_default": true, "shared": true, "project_id": 123}],
	"links": {"self": "/v2/projects/123/dashboards"}
}`

const dashboardGetBody = `{
	"id": "abc123",
	"title": "Request Health",
	"project_id": 123,
	"is_default": false,
	"shared": true,
	"widgets": [
		{"id": "w1", "type": "insights_vis", "presentation": {"title": "Errors Over Time"}},
		{"id": "w2", "type": "errors"}
	]
}`

const dashboardCreatePayload = `{
	"dashboard": {
		"title": "Request Health",
		"default_ts": "P1D",
		"widgets": [
			{"type": "insights_vis", "grid": {"x": 0, "y": 0, "w": 6, "h": 4}}
		]
	}
}`

func TestDashboardsListCommand(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		authToken      string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "successful list",
			projectIDValue: 123,
			authToken:      "test-token",
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
			serverURL := "http://localhost:9999"
			if tt.authToken != "" && tt.projectIDValue != 0 {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "GET", r.Method)

						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(dashboardListBody))
					}),
				)
				defer server.Close()
				serverURL = server.URL
			}

			viper.Reset()
			viper.Set("endpoint", serverURL)
			if tt.authToken != "" {
				viper.Set("auth_token", tt.authToken)
			}

			dashboardsProjectID = tt.projectIDValue
			dashboardsOutputFormat = "table"

			err := dashboardsListCmd.RunE(dashboardsListCmd, []string{})

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

func TestDashboardsGetCommand(t *testing.T) {
	tests := []struct {
		name             string
		projectIDValue   int
		dashboardIDValue string
		authToken        string
		outputFormat     string
		expectedError    bool
		errorContains    string
	}{
		{
			name:             "successful get",
			projectIDValue:   123,
			dashboardIDValue: "abc123",
			authToken:        "test-token",
			outputFormat:     "text",
			expectedError:    false,
		},
		{
			name:             "successful get json",
			projectIDValue:   123,
			dashboardIDValue: "abc123",
			authToken:        "test-token",
			outputFormat:     "json",
			expectedError:    false,
		},
		{
			name:             "missing project ID",
			projectIDValue:   0,
			dashboardIDValue: "abc123",
			authToken:        "test-token",
			outputFormat:     "text",
			expectedError:    true,
			errorContains:    "project ID is required",
		},
		{
			name:             "missing dashboard ID",
			projectIDValue:   123,
			dashboardIDValue: "",
			authToken:        "test-token",
			outputFormat:     "text",
			expectedError:    true,
			errorContains:    "dashboard ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverURL := "http://localhost:9999"
			if tt.authToken != "" && tt.projectIDValue != 0 && tt.dashboardIDValue != "" {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, "GET", r.Method)

						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(dashboardGetBody))
					}),
				)
				defer server.Close()
				serverURL = server.URL
			}

			viper.Reset()
			viper.Set("endpoint", serverURL)
			if tt.authToken != "" {
				viper.Set("auth_token", tt.authToken)
			}

			dashboardsProjectID = tt.projectIDValue
			dashboardID = tt.dashboardIDValue
			dashboardsOutputFormat = tt.outputFormat

			err := dashboardsGetCmd.RunE(dashboardsGetCmd, []string{})

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

// capturedRequest records what the CLI actually put on the wire.
type capturedRequest struct {
	method string
	body   map[string]interface{}
}

// newCapturingServer returns a test server that records the request method and JSON body
// into captured, then replies with status and respBody.
func newCapturingServer(
	t *testing.T,
	status int,
	respBody string,
	captured *capturedRequest,
) *httptest.Server {
	t.Helper()

	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			captured.method = r.Method

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			if len(body) > 0 {
				require.NoError(t, json.Unmarshal(body, &captured.body))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			if respBody != "" {
				_, _ = w.Write([]byte(respBody))
			}
		}),
	)
}

// requireDashboardEnvelope asserts the body was wrapped in a "dashboard" envelope and
// returns the wrapped object.
func requireDashboardEnvelope(t *testing.T, captured capturedRequest) map[string]interface{} {
	t.Helper()

	dashboard, ok := captured.body["dashboard"].(map[string]interface{})
	require.True(t, ok, "request body should be wrapped in a dashboard envelope")

	return dashboard
}

// TestDashboardsCreateCommandSendsEnvelopedPayload asserts the payload is unwrapped from the
// "dashboard" envelope on input and re-wrapped on the wire, rather than double- or un-wrapped.
func TestDashboardsCreateCommandSendsEnvelopedPayload(t *testing.T) {
	var captured capturedRequest

	server := newCapturingServer(t, http.StatusCreated, dashboardGetBody, &captured)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	dashboardsProjectID = 123
	dashboardsOutputFormat = "text"
	dashboardCLIInputJSON = dashboardCreatePayload

	err := dashboardsCreateCmd.RunE(dashboardsCreateCmd, []string{})
	require.NoError(t, err)

	assert.Equal(t, "POST", captured.method)

	dashboard := requireDashboardEnvelope(t, captured)
	assert.Equal(t, "Request Health", dashboard["title"])
	assert.Equal(t, "P1D", dashboard["default_ts"])

	widgets, ok := dashboard["widgets"].([]interface{})
	require.True(t, ok, "widgets should survive the round trip")
	assert.Len(t, widgets, 1)
}

func TestDashboardsCreateCommandValidation(t *testing.T) {
	tests := []struct {
		name           string
		projectIDValue int
		authToken      string
		inputJSON      string
		errorContains  string
	}{
		{
			name:           "missing project ID",
			projectIDValue: 0,
			authToken:      "test-token",
			inputJSON:      dashboardCreatePayload,
			errorContains:  "project ID is required",
		},
		{
			name:           "missing payload",
			projectIDValue: 123,
			authToken:      "test-token",
			inputJSON:      "",
			errorContains:  "JSON payload is required",
		},
		{
			name:           "missing auth token",
			projectIDValue: 123,
			authToken:      "",
			inputJSON:      dashboardCreatePayload,
			errorContains:  "auth token is required",
		},
		{
			name:           "malformed payload",
			projectIDValue: 123,
			authToken:      "test-token",
			inputJSON:      "{not json",
			errorContains:  "failed to parse JSON payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("endpoint", "http://localhost:9999")
			if tt.authToken != "" {
				viper.Set("auth_token", tt.authToken)
			}

			dashboardsProjectID = tt.projectIDValue
			dashboardsOutputFormat = "text"
			dashboardCLIInputJSON = tt.inputJSON

			err := dashboardsCreateCmd.RunE(dashboardsCreateCmd, []string{})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

// TestDashboardsCreateCommandReadsFileInput covers the file:// form of --cli-input-json, which
// is how a real dashboard payload is supplied.
func TestDashboardsCreateCommandReadsFileInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard.json")
	require.NoError(t, os.WriteFile(path, []byte(dashboardCreatePayload), 0o600))

	var captured capturedRequest

	server := newCapturingServer(t, http.StatusCreated, dashboardGetBody, &captured)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	dashboardsProjectID = 123
	dashboardsOutputFormat = "text"
	dashboardCLIInputJSON = "file://" + path

	err := dashboardsCreateCmd.RunE(dashboardsCreateCmd, []string{})
	require.NoError(t, err)

	dashboard := requireDashboardEnvelope(t, captured)
	assert.Equal(t, "Request Health", dashboard["title"])
}

func TestDashboardsUpdateCommand(t *testing.T) {
	var captured capturedRequest

	server := newCapturingServer(t, http.StatusOK, "", &captured)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	dashboardsProjectID = 123
	dashboardID = "abc123"
	dashboardCLIInputJSON = dashboardCreatePayload

	err := dashboardsUpdateCmd.RunE(dashboardsUpdateCmd, []string{})
	require.NoError(t, err)

	assert.Equal(t, "PUT", captured.method)

	dashboard := requireDashboardEnvelope(t, captured)
	assert.Equal(t, "Request Health", dashboard["title"])
}

func TestDashboardsUpdateCommandValidation(t *testing.T) {
	tests := []struct {
		name             string
		projectIDValue   int
		dashboardIDValue string
		inputJSON        string
		errorContains    string
	}{
		{
			name:             "missing dashboard ID",
			projectIDValue:   123,
			dashboardIDValue: "",
			inputJSON:        dashboardCreatePayload,
			errorContains:    "dashboard ID is required",
		},
		{
			name:             "missing payload",
			projectIDValue:   123,
			dashboardIDValue: "abc123",
			inputJSON:        "",
			errorContains:    "JSON payload is required",
		},
		{
			name:             "missing project ID",
			projectIDValue:   0,
			dashboardIDValue: "abc123",
			inputJSON:        dashboardCreatePayload,
			errorContains:    "project ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			viper.Set("endpoint", "http://localhost:9999")
			viper.Set("auth_token", "test-token")

			dashboardsProjectID = tt.projectIDValue
			dashboardID = tt.dashboardIDValue
			dashboardCLIInputJSON = tt.inputJSON

			err := dashboardsUpdateCmd.RunE(dashboardsUpdateCmd, []string{})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestDashboardsDeleteCommand(t *testing.T) {
	var captured capturedRequest

	server := newCapturingServer(t, http.StatusOK, "", &captured)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	dashboardsProjectID = 123
	dashboardID = "abc123"

	err := dashboardsDeleteCmd.RunE(dashboardsDeleteCmd, []string{})
	require.NoError(t, err)
	assert.Equal(t, "DELETE", captured.method)
}

func TestDashboardsDeleteCommandRequiresID(t *testing.T) {
	viper.Reset()
	viper.Set("endpoint", "http://localhost:9999")
	viper.Set("auth_token", "test-token")

	dashboardsProjectID = 123
	dashboardID = ""

	err := dashboardsDeleteCmd.RunE(dashboardsDeleteCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dashboard ID is required")
}

func TestDashboardsViperProjectIDFallback(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(dashboardListBody))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")
	viper.Set("project_id", 123)

	dashboardsProjectID = 0
	dashboardsOutputFormat = "table"

	err := dashboardsListCmd.RunE(dashboardsListCmd, []string{})
	assert.NoError(t, err)
}

func TestDashboardsOutputFormat(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(dashboardListBody))
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

			dashboardsProjectID = 123
			dashboardsOutputFormat = tt.format

			err := dashboardsListCmd.RunE(dashboardsListCmd, []string{})
			assert.NoError(t, err)
		})
	}
}

// TestParseDashboardRequestAcceptsGetOutput is the regression guard for the get-to-update
// workflow the update help text documents. `get --output json` emits a BARE dashboard object
// carrying read-only fields; parsing it must retain the title and widgets. Before both shapes
// were accepted this produced a zero-value request, so update sent an empty widget list and
// still printed "successfully updated" -- a silent wipe.
func TestParseDashboardRequestAcceptsGetOutput(t *testing.T) {
	request, err := parseDashboardRequest(dashboardGetBody)
	require.NoError(t, err)

	assert.Equal(t, "Request Health", request.Title)
	assert.Len(t, request.Widgets, 2, "widgets must survive a get -> update round trip")
}

func TestParseDashboardRequestShapes(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTitle string
		expectedCount int
		errorContains string
	}{
		{
			name:          "enveloped payload",
			input:         dashboardCreatePayload,
			expectedTitle: "Request Health",
			expectedCount: 1,
		},
		{
			name:          "bare dashboard object",
			input:         `{"title": "Bare", "widgets": [{"type": "insights_vis"}]}`,
			expectedTitle: "Bare",
			expectedCount: 1,
		},
		{
			name:          "bare object with explicitly empty widgets is allowed",
			input:         `{"title": "Cleared", "widgets": []}`,
			expectedTitle: "Cleared",
			expectedCount: 0,
		},
		{
			name:          "empty object is refused rather than wiping the dashboard",
			input:         `{}`,
			errorContains: "contains no dashboard fields",
		},
		{
			name:          "unrelated object is refused",
			input:         `{"something_else": true}`,
			errorContains: "contains no dashboard fields",
		},
		{
			name:          "null envelope is refused",
			input:         `{"dashboard": null}`,
			errorContains: "contains no dashboard fields",
		},
		{
			name:          "malformed json",
			input:         `{not json`,
			errorContains: "failed to parse JSON payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := parseDashboardRequest(tt.input)

			if tt.errorContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedTitle, request.Title)
			assert.Len(t, request.Widgets, tt.expectedCount)
		})
	}
}

// TestDashboardsUpdateRefusesEmptyPayload proves the guard reaches the command, not just the
// parser: an empty payload must never produce a request.
func TestDashboardsUpdateRefusesEmptyPayload(t *testing.T) {
	var captured capturedRequest

	server := newCapturingServer(t, http.StatusOK, "", &captured)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	dashboardsProjectID = 123
	dashboardID = "abc123"
	dashboardCLIInputJSON = `{}`

	err := dashboardsUpdateCmd.RunE(dashboardsUpdateCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains no dashboard fields")
	assert.Empty(t, captured.method, "no request should reach the API")
}

func TestWidgetHelpers(t *testing.T) {
	widget := map[string]interface{}{
		"id":           "w1",
		"type":         "insights_vis",
		"presentation": map[string]interface{}{"title": "Errors Over Time"},
	}

	assert.Equal(t, "w1", widgetString(widget, "id"))
	assert.Equal(t, "insights_vis", widgetString(widget, "type"))
	assert.Equal(t, "Errors Over Time", widgetTitle(widget))

	// Missing and wrong-typed fields must degrade to empty strings, not panic.
	assert.Equal(t, "", widgetString(widget, "missing"))
	assert.Equal(t, "", widgetString(map[string]interface{}{"id": 42}, "id"))
	assert.Equal(t, "", widgetTitle(map[string]interface{}{}))
	assert.Equal(t, "", widgetTitle(map[string]interface{}{"presentation": "not-an-object"}))
	assert.Equal(
		t,
		"",
		widgetTitle(map[string]interface{}{"presentation": map[string]interface{}{}}),
	)

	assert.Equal(t, "✓", checkmark(true))
	assert.Equal(t, " ", checkmark(false))
}
