package cmd

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const integrationJSON = `{"id":"Ch9wLp","project_id":"Xk9mZp","type":"WebHook","active":true,"events":["occurred","resolved"],"site_ids":null,"check_in_ids":null,"excluded_environments":[],"included_environments":["production"],"filters":[],"alarm_alert_ids":[],"alarm_ok_ids":[],"threshold":100,"rate":"wday","notification_limit":null,"config":{"url":"https://example.com/hook","label":null,"payload_version":null},"created_at":"2026-01-02T03:04:05Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/integrations/Ch9wLp"}}`

const slackIntegrationJSON = `{"id":"Ch8vKo","project_id":"Xk9mZp","type":"Slack","active":false,"events":[],"site_ids":[],"check_in_ids":[],"excluded_environments":[],"included_environments":[],"filters":[],"alarm_alert_ids":[],"alarm_ok_ids":[],"threshold":100,"rate":"wday","notification_limit":5,"connected":false,"config":{"channel":null},"created_at":"2026-01-01T02:03:04Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/integrations/Ch8vKo"}}`

const integrationBody = `{"data":` + integrationJSON + `}`

const integrationsListBody = `{"data":[` + integrationJSON + `,` + slackIntegrationJSON + `],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/integrations","next":null}}`

func TestIntegrationsList(t *testing.T) {
	s := newV3Server(t, ok(integrationsListBody))
	out, err := runCLI(t, v3Env(s), "integrations", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/integrations", req.Path)
	assert.Equal(t, "Bearer hbu_test", req.Auth)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID      TYPE     ACTIVE  EVENTS")
	assert.Contains(t, out, "Ch9wLp  WebHook  Yes     occurred, resolved")
	assert.Contains(t, out, "Ch8vKo  Slack    No      none")
}

func TestIntegrationsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(integrationsListBody))
	out, err := runCLI(t, v3Env(s), "integrations", "list", "--project-id", "Xk9mZp",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Ch9wLp"`)
	assert.Contains(t, out, `"type": "Slack"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestIntegrationsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(integrationsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "integrations", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/integrations", s.only(t).Path)
}

func TestIntegrationsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestIntegrationsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"integrations", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestIntegrationsListReportsAMissingScope(t *testing.T) {
	s := newV3Server(t, status(
		http.StatusForbidden,
		`{"error":{"code":"insufficient_scope","message":"This token can't do that","details":{"required_scope":"integrations:read","token_scopes":["faults:read"]}}}`,
	))
	_, err := runCLI(t, v3Env(s), "integrations", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list integrations")
	assert.Contains(t, err.Error(), "integrations:read")
}

func TestIntegrationsGet(t *testing.T) {
	s := newV3Server(t, ok(integrationBody))
	out, err := runCLI(t, v3Env(s), "integrations", "get", "--project-id", "Xk9mZp",
		"--id", "Ch9wLp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/integrations/Ch9wLp", req.Path)
	assert.Contains(t, out, "ID: Ch9wLp")
	assert.Contains(t, out, "Type: WebHook")
	assert.Contains(t, out, "Active: true")
	assert.Contains(t, out, "Events: occurred, resolved")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
	assert.Contains(t, out, "URL: https://app.honeybadger.io/projects/Xk9mZp/integrations/Ch9wLp")
	assert.NotContains(t, out, "Connected:")
	assert.NotContains(t, out, "Error:")
}

func TestIntegrationsGetShowsWhetherAnOAuthIntegrationIsConnected(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+slackIntegrationJSON+`}`))
	out, err := runCLI(t, v3Env(s), "integrations", "get", "--project-id", "Xk9mZp",
		"--id", "Ch8vKo")
	require.NoError(t, err)
	assert.Contains(t, out, "Connected: false")
	assert.Contains(t, out, "Events: none")
}

func TestIntegrationsGetShowsTheErrorState(t *testing.T) {
	failing := strings.Replace(integrationJSON, `"active":true,`,
		`"active":true,"error":true,"error_reason":"Endpoint answered 500",`, 1)
	s := newV3Server(t, ok(`{"data":`+failing+`}`))
	out, err := runCLI(t, v3Env(s), "integrations", "get", "--project-id", "Xk9mZp",
		"--id", "Ch9wLp")
	require.NoError(t, err)
	assert.Contains(t, out, "Error: Endpoint answered 500")
}

func TestIntegrationsGetJSONPrintsTheIntegration(t *testing.T) {
	s := newV3Server(t, ok(integrationBody))
	out, err := runCLI(t, v3Env(s), "integrations", "get", "--project-id", "Xk9mZp",
		"--id", "Ch9wLp", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Ch9wLp"`)
	assert.Contains(t, out, `"url": "https://example.com/hook"`)
}

func TestIntegrationsGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "get", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestIntegrationsCreateSendsTheFlatBody(t *testing.T) {
	input := `{"type":"WebHook","config":{"url":"https://example.com/hook"},"events":["occurred","resolved"]}`
	s := newV3Server(t, status(http.StatusCreated, integrationBody))
	out, err := runCLI(t, v3Env(s), "integrations", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", input)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/integrations", req.Path)
	assertJSONBody(t, input, req.Body)
	assert.Contains(t, out, "Integration created successfully!")
	assert.Contains(t, out, "ID: Ch9wLp")
	assert.Contains(t, out, "Type: WebHook")
	assert.Contains(t, out, "URL: https://app.honeybadger.io/projects/Xk9mZp/integrations/Ch9wLp")
}

func TestIntegrationsCreateJSONPrintsTheIntegration(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, integrationBody))
	out, err := runCLI(t, v3Env(s), "integrations", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"type":"WebHook"}`, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Ch9wLp"`)
}

func TestIntegrationsCreateRefusesTheWrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"integration":{"type":"WebHook"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(),
		`v3 takes the fields directly: remove the outer "integration" key`)
	assert.Empty(t, s.Requests())
}

func TestIntegrationsCreateRefusesAnUnknownField(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"type":"WebHook","bogus":1}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Empty(t, s.Requests())
}

func TestIntegrationsCreateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "create",
		"--cli-input-json", `{"type":"WebHook"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestIntegrationsUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"turn it off", `{"active":false}`},
		{"change one setting", `{"config":{"url":"https://example.com/new"}}`},
		{"follow every site", `{"site_ids":null}`},
		{"nothing", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(integrationBody))
			out, err := runCLI(t, v3Env(s), "integrations", "update", "--project-id", "Xk9mZp",
				"--id", "Ch9wLp", "--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/integrations/Ch9wLp", req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Integration updated successfully")
		})
	}
}

func TestIntegrationsUpdateRefusesTheWrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "update", "--project-id", "Xk9mZp",
		"--id", "Ch9wLp", "--cli-input-json", `{"integration":{"active":false}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "integration" key`)
	assert.Empty(t, s.Requests())
}

func TestIntegrationsUpdateRefusesTheType(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "update", "--project-id", "Xk9mZp",
		"--id", "Ch9wLp", "--cli-input-json", `{"type":"Slack"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "type"`)
	assert.Empty(t, s.Requests())
}

func TestIntegrationsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "integrations", "delete", "--project-id", "Xk9mZp",
		"--id", "Ch9wLp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/integrations/Ch9wLp", req.Path)
	assert.Contains(t, out, "Integration deleted successfully")
}

func TestIntegrationsDeleteRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "delete", "--id", "Ch9wLp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}
