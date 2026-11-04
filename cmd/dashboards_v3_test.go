package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dashboardJSON = `{"id":"Db4rKn","project_id":"Xk9mZp","title":"Request Health","default_ts":"P1D","widgets":[{"id":"w1","type":"insights_vis","grid":{"x":0,"y":0,"w":6,"h":4},"presentation":{"title":"Errors Over Time"},"config":{"streams":["default"],"query":"stats count() by bin(1h)"}},{"id":"w2","type":"errors"}],"is_default":true,"shared":false,"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-03T04:05:06Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/insights/dashboards/Db4rKn"}}`

const emptyDashboardJSON = `{"id":"Db5sLo","project_id":"Xk9mZp","title":"Empty","default_ts":null,"widgets":[],"is_default":false,"shared":true,"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/insights/dashboards/Db5sLo"}}`

const dashboardBody = `{"data":` + dashboardJSON + `}`

const dashboardsListBody = `{"data":[` + dashboardJSON + `,` + emptyDashboardJSON + `],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/dashboards","next":null}}`

func TestDashboardsList(t *testing.T) {
	s := newV3Server(t, ok(dashboardsListBody))
	out, err := runCLI(t, v3Env(s), "dashboards", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/dashboards", req.Path)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID      TITLE           WIDGETS  DEFAULT  SHARED  UPDATED")
	assert.Contains(t, out, "Db4rKn  Request Health  2        ✓                2026-01-03 04:05")
	assert.Contains(t, out, "Db5sLo  Empty           0                 ✓       2026-01-02 03:04")
}

func TestDashboardsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(dashboardsListBody))
	out, err := runCLI(t, v3Env(s), "dashboards", "list", "--project-id", "Xk9mZp",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Db4rKn"`)
	assert.Contains(t, out, `"title": "Request Health"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestDashboardsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(dashboardsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "dashboards", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/dashboards", s.only(t).Path)
}

func TestDashboardsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestDashboardsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"dashboards", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestDashboardsGet(t *testing.T) {
	s := newV3Server(t, ok(dashboardBody))
	out, err := runCLI(t, v3Env(s), "dashboards", "get", "--project-id", "Xk9mZp", "--id", "Db4rKn")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/dashboards/Db4rKn", req.Path)
	assert.Contains(t, out, "ID: Db4rKn")
	assert.Contains(t, out, "Title: Request Health")
	assert.Contains(t, out, "Project ID: Xk9mZp")
	assert.Contains(t, out, "Default: true")
	assert.Contains(t, out, "Shared: false")
	assert.Contains(t, out, "Widgets: 2")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
	assert.Contains(t, out, "Updated: 2026-01-03 04:05:06")
	assert.Contains(t, out, "WIDGET ID  TYPE          TITLE")
	assert.Contains(t, out, "w1         insights_vis  Errors Over Time")
	assert.Contains(t, out, "w2         errors")
}

func TestDashboardsGetJSONPrintsTheDashboard(t *testing.T) {
	s := newV3Server(t, ok(dashboardBody))
	out, err := runCLI(t, v3Env(s), "dashboards", "get", "--project-id", "Xk9mZp", "--id", "Db4rKn",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Db4rKn"`)
	assert.Contains(t, out, `"presentation": {`)
}

func TestDashboardsGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "get", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestDashboardsCreateSendsTheFlatBody(t *testing.T) {
	input := `{"title":"Request Health","default_ts":"P1D","widgets":[{"type":"insights_vis","grid":{"x":0,"y":0,"w":6,"h":4},"presentation":{"title":"Errors Over Time"},"config":{"streams":["default"],"query":"stats count() by bin(1h)"}}]}`
	s := newV3Server(t, status(http.StatusCreated, dashboardBody))
	out, err := runCLI(t, v3Env(s), "dashboards", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", input)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/dashboards", req.Path)
	assertJSONBody(t, input, req.Body)
	assert.Contains(t, out, "Dashboard created successfully!")
	assert.Contains(t, out, "ID: Db4rKn")
	assert.Contains(t, out, "Title: Request Health")
	assert.Contains(t, out, "Widgets: 2")
}

func TestDashboardsCreateTitleOnly(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, `{"data":`+emptyDashboardJSON+`}`))
	_, err := runCLI(t, v3Env(s), "dashboards", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"title":"Empty"}`)
	require.NoError(t, err)
	assertJSONBody(t, `{"title":"Empty"}`, s.only(t).Body)
}

func TestDashboardsCreateJSONPrintsTheDashboard(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, dashboardBody))
	out, err := runCLI(t, v3Env(s), "dashboards", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"title":"Request Health"}`, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Db4rKn"`)
}

func TestDashboardsCreateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"dashboard":{"title":"x"}}`)
	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		`v3 takes the fields directly: remove the outer "dashboard" key`,
	)
	assert.Empty(t, s.Requests())
}

func TestDashboardsCreateReadsAFile(t *testing.T) {
	path := t.TempDir() + "/dashboard.json"
	require.NoError(t, writeFile(path, `{"title":"From File"}`))
	s := newV3Server(t, status(http.StatusCreated, dashboardBody))
	_, err := runCLI(t, v3Env(s), "dashboards", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", "file://"+path)
	require.NoError(t, err)
	assertJSONBody(t, `{"title":"From File"}`, s.only(t).Body)
}

func TestDashboardsCreateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "create", "--cli-input-json", `{"title":"x"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestDashboardsUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"title only", `{"title":"T"}`},
		{"remove every widget", `{"widgets":[]}`},
		{
			"keep a widget by id",
			`{"widgets":[{"id":"w1","type":"insights_vis","grid":{"x":0,"y":0,"w":12,"h":3}}]}`,
		},
		{"zero grid position", `{"widgets":[{"id":"w2","type":"errors","grid":{"x":0,"y":0}}]}`},
		{"clear the default range", `{"default_ts":null}`},
		{"nothing", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(dashboardBody))
			out, err := runCLI(t, v3Env(s), "dashboards", "update", "--project-id", "Xk9mZp",
				"--id", "Db4rKn", "--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/dashboards/Db4rKn", req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Dashboard updated successfully")
		})
	}
}

func TestDashboardsUpdateHelpSaysItIsPartial(t *testing.T) {
	assert.Contains(t, dashboardsUpdateCmd.Long, "Only the fields you send change")
	assert.Contains(t, dashboardsUpdateCmd.Long, "replaces the dashboard's widgets")
	assert.NotContains(t, dashboardsUpdateCmd.Long, `"dashboard": {`)
}

func TestDashboardsUpdateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "update", "--project-id", "Xk9mZp",
		"--id", "Db4rKn", "--cli-input-json", `{"dashboard":{"title":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "dashboard" key`)
	assert.Empty(t, s.Requests())
}

func TestDashboardsUpdateRefusesReadOnlyFields(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "update", "--project-id", "Xk9mZp",
		"--id", "Db4rKn", "--cli-input-json", dashboardJSON)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "id"`)
	assert.Empty(t, s.Requests())
}

func TestDashboardsUpdateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "update", "--id", "Db4rKn",
		"--cli-input-json", `{"title":"T"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestDashboardsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "dashboards", "delete", "--project-id", "Xk9mZp",
		"--id", "Db4rKn")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/dashboards/Db4rKn", req.Path)
	assert.Contains(t, out, "Dashboard deleted successfully")
}

func TestDashboardsDeleteRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "dashboards", "delete", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}
