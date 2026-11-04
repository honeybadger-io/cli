package cmd

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const faultJSON = `{"id":"Fk7pQ2","project_id":"Xk9mZp","klass":"RuntimeError","message":"Something broke","environment":"production","component":"users","action":"show","created_at":"2026-01-02T03:04:05Z","last_notice_at":"2026-01-03T04:05:06Z","notices_count":7,"comments_count":2,"resolved":false,"ignored":false,"assignee":{"id":"Us3rAb","name":"Jane Doe","email":"jane@example.com"},"assignee_id":"Us3rAb","tags":["urgent","billing"],"links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/faults/Fk7pQ2"}}`

const faultBody = `{"data":` + faultJSON + `}`

const faultsListBody = `{"data":[` + faultJSON + `],"pagination":{"page":1,"per_page":25},"links":{"self":"/v3/projects/Xk9mZp/faults","next":null}}`

// assertQuery fails unless the raw query string holds exactly want.
func assertQuery(t *testing.T, want url.Values, raw string) {
	t.Helper()
	got, err := url.ParseQuery(raw)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFaultsList(t *testing.T) {
	s := newV3Server(t, ok(faultsListBody))
	out, err := runCLI(t, v3Env(s), "faults", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults", req.Path)
	assertQuery(t, url.Values{"order": {"recent"}, "per_page": {"25"}}, req.Query)
	assert.Contains(
		t,
		out,
		"ID      CLASS         MESSAGE          ENV         NOTICES  RESOLVED  LAST SEEN",
	)
	assert.Contains(t, out, "Fk7pQ2  RuntimeError  Something broke  production  7")
	assert.Contains(t, out, "2026-01-03 04:05")
}

func TestFaultsListSendsTheLimitAsPerPage(t *testing.T) {
	s := newV3Server(t, ok(faultsListBody))
	_, err := runCLI(t, v3Env(s), "faults", "list", "--project-id", "Xk9mZp", "--limit", "100")
	require.NoError(t, err)
	assertQuery(t, url.Values{"order": {"recent"}, "per_page": {"100"}}, s.only(t).Query)
}

func TestFaultsListSendsTheQueryAndOrder(t *testing.T) {
	s := newV3Server(t, ok(faultsListBody))
	_, err := runCLI(t, v3Env(s), "faults", "list", "--project-id", "Xk9mZp",
		"--query", "x", "--order", "frequent")
	require.NoError(t, err)
	assertQuery(
		t,
		url.Values{"q": {"x"}, "order": {"frequent"}, "per_page": {"25"}},
		s.only(t).Query,
	)
}

func TestFaultsListHelpSaysTheLimitIs100(t *testing.T) {
	assert.Contains(t, faultsListCmd.Flags().Lookup("limit").Usage, "max 100")
}

func TestFaultsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(faultsListBody))
	out, err := runCLI(t, v3Env(s), "faults", "list", "--project-id", "Xk9mZp", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Fk7pQ2"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestFaultsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(faultsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "faults", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/faults", s.only(t).Path)
}

func TestFaultsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "list")
	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"project ID is required. Set it using --project-id flag, HONEYBADGER_PROJECT_ID environment variable, or project_id in your config file",
	)
	assert.Empty(t, s.Requests())
}

func TestFaultsGet(t *testing.T) {
	s := newV3Server(t, ok(faultBody))
	out, err := runCLI(t, v3Env(s), "faults", "get", "--project-id", "Xk9mZp", "--id", "Fk7pQ2")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2", req.Path)
	assert.Contains(t, out, "ID: Fk7pQ2")
	assert.Contains(t, out, "Class: RuntimeError")
	assert.Contains(t, out, "Last Noticed: 2026-01-03 04:05:06")
	assert.Contains(t, out, "Notice Count: 7")
	assert.Contains(t, out, "Assignee: Jane Doe <jane@example.com>")
	assert.Contains(t, out, "URL: https://app.honeybadger.io/projects/Xk9mZp/faults/Fk7pQ2")
	assert.Contains(t, out, "Tags: urgent, billing")
}

func TestFaultsGetJSONPrintsTheFault(t *testing.T) {
	s := newV3Server(t, ok(faultBody))
	out, err := runCLI(t, v3Env(s), "faults", "get", "--project-id", "Xk9mZp", "--id", "Fk7pQ2",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Fk7pQ2"`)
	assert.Contains(t, out, `"klass": "RuntimeError"`)
}

func TestFaultsUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name string
		args []string
		body string
	}{
		{"resolve", []string{"--resolved"}, `{"resolved":true}`},
		{"un-resolve", []string{"--resolved=false"}, `{"resolved":false}`},
		{"ignore", []string{"--ignored"}, `{"ignored":true}`},
		{"resolve on deploy", []string{"--resolve-on-deploy"}, `{"resolve_on_deploy":true}`},
		{"assign", []string{"--assignee-id", "Xk9mZp"}, `{"assignee_id":"Xk9mZp"}`},
		{"unassign", []string{"--unassign"}, `{"assignee_id":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(faultBody))
			args := append([]string{"faults", "update", "--project-id", "Xk9mZp", "--id", "Fk7pQ2"},
				tt.args...)
			out, err := runCLI(t, v3Env(s), args...)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2", req.Path)
			assertJSONBody(t, tt.body, req.Body)
			assert.Contains(t, out, "Fault Fk7pQ2 updated")
		})
	}
}

func TestFaultsUpdateRefusesAssigneeAndUnassignTogether(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "update", "--project-id", "Xk9mZp", "--id", "Fk7pQ2",
		"--assignee-id", "x", "--unassign")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--assignee-id and --unassign are mutually exclusive")
	assert.Empty(t, s.Requests())
}

func TestFaultsUpdateRefusesNothingToUpdate(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "update", "--project-id", "Xk9mZp", "--id", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
	assert.Empty(t, s.Requests())
}

func TestFaultsUpdateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "update", "--id", "Fk7pQ2", "--resolved")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestFaultsUpdateRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "update", "--project-id", "Xk9mZp", "--resolved")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestFaultsUpdateRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"faults", "update", "--project-id", "Xk9mZp", "--id", "Fk7pQ2", "--resolved")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

const faultSummaryBody = `{"data":{"total":142,"environments":[{"environment":"production","resolved":false,"ignored":false,"count":140},{"environment":null,"resolved":true,"ignored":false,"count":2}]}}`

func TestFaultsCounts(t *testing.T) {
	s := newV3Server(t, ok(faultSummaryBody))
	out, err := runCLI(t, v3Env(s), "faults", "counts", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/summary", req.Path)
	assert.Empty(t, req.Query)
	assert.Contains(t, out, "Total Faults: 142")
	assert.Contains(t, out, "production   false     false    140")
}

func TestFaultsCountsJSONPrintsTheSummary(t *testing.T) {
	s := newV3Server(t, ok(faultSummaryBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"faults",
		"counts",
		"--project-id",
		"Xk9mZp",
		"--output",
		"json",
	)
	require.NoError(t, err)
	assert.Contains(t, out, `"total": 142`)
}

const noticesBody = `{"data":[{"id":"1e2ba1f0-4f3c-4b1a-9c7e-6f2f5c0f1a3b","fault_id":"Fk7pQ2","project_id":"Xk9mZp","message":"Something broke","environment":"production","created_at":"2026-01-03T04:05:06Z"}],"pagination":{"limit":10,"has_older":false,"has_newer":false},"links":{"self":"/v3/projects/Xk9mZp/faults/Fk7pQ2/notices","older":null,"newer":null}}`

func TestFaultsNotices(t *testing.T) {
	s := newV3Server(t, ok(noticesBody))
	out, err := runCLI(t, v3Env(s), "faults", "notices", "--project-id", "Xk9mZp", "--id", "Fk7pQ2",
		"--limit", "10")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/notices", req.Path)
	assertQuery(t, url.Values{"limit": {"10"}}, req.Query)
	assert.Contains(t, out, "1e2ba1f0-4f3c-4b1a-9c7e-6f2f5c0f1a3b  Something broke  production")
	assert.Contains(t, out, "2026-01-03 04:05:06")
}

func TestFaultsNoticesJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(noticesBody))
	out, err := runCLI(t, v3Env(s), "faults", "notices", "--project-id", "Xk9mZp", "--id", "Fk7pQ2",
		"--output", "json")
	require.NoError(t, err)
	assertQuery(t, url.Values{"limit": {"25"}}, s.only(t).Query)
	assert.Contains(t, out, `"id": "1e2ba1f0-4f3c-4b1a-9c7e-6f2f5c0f1a3b"`)
	assert.NotContains(t, out, `"pagination"`)
}

const affectedUsersBody = `{"data":[{"user":"jane@example.com","count":3}]}`

func TestFaultsAffectedUsers(t *testing.T) {
	s := newV3Server(t, ok(affectedUsersBody))
	out, err := runCLI(t, v3Env(s), "faults", "affected-users", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--query", "jane")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/affected_users", req.Path)
	assertQuery(t, url.Values{"q": {"jane"}}, req.Query)
	assert.Contains(t, out, "jane@example.com  3")
}

func TestFaultsAffectedUsersJSON(t *testing.T) {
	s := newV3Server(t, ok(affectedUsersBody))
	out, err := runCLI(t, v3Env(s), "faults", "affected-users", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--output", "json")
	require.NoError(t, err)
	assert.Empty(t, s.only(t).Query)
	assert.Contains(t, out, `"user": "jane@example.com"`)
}
