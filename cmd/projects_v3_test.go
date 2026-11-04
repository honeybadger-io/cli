package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectBody = `{"data":{"id":"Xk9mZp","account_id":"Ab3kL9","name":"My Project","active":true,"created_at":"2026-01-02T03:04:05Z","environments":["production"],"fault_count":3,"unresolved_fault_count":1}}`

func TestProjectsList(t *testing.T) {
	s := newV3Server(t, ok(projectsListBody))
	out, err := runCLI(t, v3Env(s), "projects", "list")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects", req.Path)
	assert.Equal(t, "Bearer hbu_test", req.Auth)
	assert.Contains(t, out, "Xk9mZp")
	assert.Contains(t, out, "My Rails App")
}

func TestProjectsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(projectsListBody))
	out, err := runCLI(t, v3Env(s), "projects", "list", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Xk9mZp"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestProjectsListNoLongerTakesAnAccountID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "projects", "list", "--account-id", "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag: --account-id")
	assert.Empty(t, s.Requests())
}

func TestProjectsGet(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	out, err := runCLI(t, v3Env(s), "projects", "get", "--id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp", req.Path)
	assert.Contains(t, out, "Xk9mZp")
	assert.Contains(t, out, "My Project")
}

func TestProjectsGetAcceptsALegacyNumericID(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(t, v3Env(s), "projects", "get", "--id", "123")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123", s.only(t).Path)
}

func TestProjectsCreateSendsTheFlatBody(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, projectBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"create",
		"--cli-input-json",
		`{"name":"My Project"}`,
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects", req.Path)
	assertJSONBody(t, `{"name":"My Project"}`, req.Body)
	assert.Contains(t, out, "Xk9mZp")
}

func TestProjectsCreateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"create",
		"--cli-input-json",
		`{"project":{"name":"x"}}`,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `v3 takes the fields directly: remove the outer "project" key`)
	assert.Empty(t, s.Requests())
}

func TestProjectsCreateRefusesUnknownFields(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"create",
		"--cli-input-json",
		`{"name":"x","bogus":1}`,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Empty(t, s.Requests())
}

func TestProjectsCreateReportsAMissingInputFile(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"create",
		"--cli-input-json",
		"file://"+t.TempDir()+"/missing.json",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read JSON input")
	assert.Empty(t, s.Requests())
}

func TestProjectsCreateNoLongerTakesAnAccountID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"create",
		"--account-id",
		"x",
		"--cli-input-json",
		`{"name":"x"}`,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag: --account-id")
}

func TestProjectsUpdateSendsOnlyTheGivenFields(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"update",
		"--id",
		"Xk9mZp",
		"--cli-input-json",
		`{"name":"New"}`,
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "PATCH", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp", req.Path)
	assertJSONBody(t, `{"name":"New"}`, req.Body)
}

func TestProjectsUpdateSendsAnEmptyObjectAsIs(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(t, v3Env(s), "projects", "update", "--id", "Xk9mZp", "--cli-input-json", `{}`)
	require.NoError(t, err)
	assertJSONBody(t, `{}`, s.only(t).Body)
}

func TestProjectsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	_, err := runCLI(t, v3Env(s), "projects", "delete", "--id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp", req.Path)
}

func TestProjectsOccurrencesForOneProject(t *testing.T) {
	s := newV3Server(
		t,
		ok(`{"data":{"project_id":"123","buckets":[{"timestamp":1767225600,"count":4}]}}`),
	)
	out, err := runCLI(
		t,
		cliEnv{"HONEYBADGER_API_TOKEN": "hbu_test", "HONEYBADGER_ENDPOINT": s.URL},
		"projects",
		"occurrences",
		"--id",
		"123",
		"--period",
		"week",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "/v3/projects/123/occurrences", req.Path)
	assert.Equal(t, "period=week", req.Query)
	assert.Contains(t, out, "2026-01-01T00:00:00Z  4")
}

func TestProjectsReportsAndIntegrationsAreGone(t *testing.T) {
	for _, c := range projectsCmd.Commands() {
		assert.NotContains(t, []string{"reports", "integrations"}, c.Name())
	}
}

func TestProjectsUpdateSendsFalse(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(t, v3Env(s), "projects", "update", "--id", "Xk9mZp",
		"--cli-input-json", `{"resolve_errors_on_deploy":false}`)
	require.NoError(t, err)
	assertJSONBody(t, `{"resolve_errors_on_deploy":false}`, s.only(t).Body)
}

func TestProjectsListJSONPrintsAnEmptyListAsAnArray(t *testing.T) {
	s := newV3Server(
		t,
		ok(
			`{"data":[],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects","next":null}}`,
		),
	)
	out, err := runCLI(t, v3Env(s), "projects", "list", "--output", "json")
	require.NoError(t, err)
	assert.Equal(t, "[]\n", out)
}

func TestProjectsOccurrencesForEveryProject(t *testing.T) {
	s := newV3Server(
		t,
		ok(
			`{"data":[{"project_id":"Xk9mZp","buckets":[{"timestamp":1767225600,"count":4}]}],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/occurrences","next":null}}`,
		),
	)
	out, err := runCLI(t, v3Env(s), "projects", "occurrences", "--period", "day")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "/v3/projects/occurrences", req.Path)
	assert.Contains(t, req.Query, "period=day")
	assert.Contains(t, out, "Project Xk9mZp:")
	assert.Contains(t, out, "2026-01-01T00:00:00Z  4")
}

func TestProjectsUpdateRefusesTrailingInput(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(t, v3Env(s), "projects", "update", "--id", "Xk9mZp",
		"--cli-input-json", `{"name":"New"} {"resolve_errors_on_deploy":false}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON payload")
	assert.Empty(t, s.Requests())
}

func TestProjectsUpdateRefusesATrailingBracket(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"update",
		"--id",
		"Xk9mZp",
		"--cli-input-json",
		`{"name":"New"}]`,
	)
	require.Error(t, err)
	assert.Empty(t, s.Requests())
}

func TestProjectsUpdateAcceptsATrailingNewline(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"update",
		"--id",
		"Xk9mZp",
		"--cli-input-json",
		"{\"name\":\"New\"}\n",
	)
	require.NoError(t, err)
	assertJSONBody(t, `{"name":"New"}`, s.only(t).Body)
}
