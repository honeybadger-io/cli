package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const teamJSON = `{"id":"Tm8qJv","account_id":"Ab3kL9","name":"Backend Team","links":{"web":"https://app.honeybadger.io/accounts/Ab3kL9/teams/Tm8qJv"},"created_at":"2026-01-02T03:04:05Z","member_count":3,"project_ids":["Xk9mZp"]}`

const teamBody = `{"data":` + teamJSON + `}`

const teamsListBody = `{"data":[` + teamJSON + `,{"id":"Tm9rKw","account_id":"Ab3kL9","name":"Frontend Team","links":{"web":"https://app.honeybadger.io/accounts/Ab3kL9/teams/Tm9rKw"},"created_at":"2026-01-03T04:05:06Z","member_count":0,"project_ids":[]}],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/teams","next":null}}`

func TestTeamsList(t *testing.T) {
	s := newV3Server(t, ok(teamsListBody))
	out, err := runCLI(t, v3Env(s), "teams", "list")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/teams", req.Path)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID      NAME           CREATED")
	assert.Contains(t, out, "Tm8qJv  Backend Team   2026-01-02 03:04")
	assert.Contains(t, out, "Tm9rKw  Frontend Team  2026-01-03 04:05")
}

func TestTeamsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(teamsListBody))
	out, err := runCLI(t, v3Env(s), "teams", "list", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Tm8qJv"`)
	assert.Contains(t, out, `"member_count": 3`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestTeamsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL}, "teams", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestTeamsHaveNoAccountIDFlag(t *testing.T) {
	commands := [][]string{
		{"teams", "list"},
		{"teams", "create", "--name", "x"},
	}
	for _, args := range commands {
		t.Run(args[1], func(t *testing.T) {
			s := newV3Server(t)
			_, err := runCLI(t, v3Env(s), append(args, "--account-id", "Ab3kL9")...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown flag: --account-id")
			assert.Empty(t, s.Requests())
		})
	}
}

func TestTeamsMembersAndInvitationsAreGone(t *testing.T) {
	for _, c := range teamsCmd.Commands() {
		assert.NotContains(t, []string{"members", "invitations"}, c.Name())
	}
}

func TestTeamsGet(t *testing.T) {
	s := newV3Server(t, ok(teamBody))
	out, err := runCLI(t, v3Env(s), "teams", "get", "--id", "Tm8qJv")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/teams/Tm8qJv", req.Path)
	assert.Contains(t, out, "ID: Tm8qJv")
	assert.Contains(t, out, "Name: Backend Team")
	assert.Contains(t, out, "Account ID: Ab3kL9")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
}

func TestTeamsGetTakesALegacyNumericID(t *testing.T) {
	s := newV3Server(t, ok(teamBody))
	_, err := runCLI(t, v3Env(s), "teams", "get", "--id", "42")
	require.NoError(t, err)
	assert.Equal(t, "/v3/teams/42", s.only(t).Path)
}

func TestTeamsGetJSONPrintsTheTeam(t *testing.T) {
	s := newV3Server(t, ok(teamBody))
	out, err := runCLI(t, v3Env(s), "teams", "get", "--id", "Tm8qJv", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Tm8qJv"`)
	assert.Contains(t, out, `"account_id": "Ab3kL9"`)
}

func TestTeamsGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "teams", "get")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestTeamsCreate(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, teamBody))
	out, err := runCLI(t, v3Env(s), "teams", "create", "--name", "Backend Team")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/teams", req.Path)
	assertJSONBody(t, `{"name":"Backend Team"}`, req.Body)
	assert.Contains(t, out, "Team created successfully!")
	assert.Contains(t, out, "ID: Tm8qJv")
	assert.Contains(t, out, "Name: Backend Team")
}

func TestTeamsCreateJSONPrintsTheTeam(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, teamBody))
	out, err := runCLI(t, v3Env(s), "teams", "create", "--name", "Backend Team",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Tm8qJv"`)
}

func TestTeamsCreateRequiresAName(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "teams", "create")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "name" not set`)
	assert.Empty(t, s.Requests())
}

func TestTeamsUpdateSendsTheName(t *testing.T) {
	s := newV3Server(t, ok(teamBody))
	out, err := runCLI(t, v3Env(s), "teams", "update", "--id", "Tm8qJv", "--name", "Platform")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "PATCH", req.Method)
	assert.Equal(t, "/v3/teams/Tm8qJv", req.Path)
	assertJSONBody(t, `{"name":"Platform"}`, req.Body)
	assert.Contains(t, out, "Team updated successfully!")
	assert.Contains(t, out, "ID: Tm8qJv")
}

func TestTeamsUpdateJSONPrintsTheTeam(t *testing.T) {
	s := newV3Server(t, ok(teamBody))
	out, err := runCLI(t, v3Env(s), "teams", "update", "--id", "Tm8qJv", "--name", "Platform",
		"--output", "json")
	require.NoError(t, err)
	s.only(t)
	assert.Contains(t, out, `"id": "Tm8qJv"`)
}

func TestTeamsUpdateRefusesAnEmptyName(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "teams", "update", "--id", "Tm8qJv", "--name", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team name is required")
	assert.Empty(t, s.Requests())
}

func TestTeamsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "teams", "delete", "--id", "Tm8qJv")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/teams/Tm8qJv", req.Path)
	assert.Contains(t, out, "Team deleted successfully")
}
