package cmd

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const deployJSON = `{"id":"Dp2yNk","project_id":"Xk9mZp","environment":"production","revision":"a1b2c3d4e5f6a7b8","repository":"acme/app","local_username":"jane","created_at":"2026-01-02T03:04:05Z"}`

const deployBody = `{"data":` + deployJSON + `}`

const deploysListBody = `{"data":[` + deployJSON + `,{"id":"Dp3zPm","project_id":"Xk9mZp","environment":null,"revision":null,"repository":null,"local_username":null,"created_at":"2026-01-01T02:03:04Z"}],"pagination":{"limit":25,"has_older":false,"has_newer":false},"links":{"self":"/v3/projects/Xk9mZp/deploys","older":null,"newer":null}}`

func TestDeploymentsList(t *testing.T) {
	s := newV3Server(t, ok(deploysListBody))
	out, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/deploys", req.Path)
	assertQuery(t, url.Values{"limit": {"25"}}, req.Query)
	assert.Contains(t, out, "ID      ENVIRONMENT  REVISION      USER  CREATED")
	assert.Contains(t, out, "Dp2yNk  production   a1b2c3d4e5f6  jane  2026-01-02 03:04")
	assert.Contains(t, out, "Dp3zPm")
}

func TestDeploymentsListSendsTheFilters(t *testing.T) {
	s := newV3Server(t, ok(deploysListBody))
	_, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp",
		"--environment", "production", "--local-user", "jane", "--limit", "100")
	require.NoError(t, err)
	assertQuery(
		t,
		url.Values{
			"environment":    {"production"},
			"local_username": {"jane"},
			"limit":          {"100"},
		},
		s.only(t).Query,
	)
}

func TestDeploymentsListHelpSaysTheLimitIs100(t *testing.T) {
	assert.Contains(t, deploymentsListCmd.Flags().Lookup("limit").Usage, "max 100")
}

func TestDeploymentsListNoLongerTakesTimeFilters(t *testing.T) {
	for _, flag := range []string{"--created-after", "--created-before"} {
		t.Run(flag, func(t *testing.T) {
			s := newV3Server(t)
			_, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp",
				flag, "2026-01-01")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown flag: "+flag)
			assert.Empty(t, s.Requests())
		})
	}
}

func TestDeploymentsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(deploysListBody))
	out, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Dp2yNk"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestDeploymentsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(deploysListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "deployments", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/deploys", s.only(t).Path)
}

func TestDeploymentsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "deployments", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestDeploymentsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"deployments", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestDeploymentsGet(t *testing.T) {
	s := newV3Server(t, ok(deployBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"deployments",
		"get",
		"--project-id",
		"Xk9mZp",
		"--id",
		"Dp2yNk",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/deploys/Dp2yNk", req.Path)
	assert.Contains(t, out, "ID: Dp2yNk")
	assert.Contains(t, out, "Environment: production")
	assert.Contains(t, out, "Revision: a1b2c3d4e5f6a7b8")
	assert.Contains(t, out, "Repository: acme/app")
	assert.Contains(t, out, "Local Username: jane")
	assert.Contains(t, out, "Project ID: Xk9mZp")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
}

func TestDeploymentsGetAcceptsALegacyNumericID(t *testing.T) {
	s := newV3Server(t, ok(deployBody))
	_, err := runCLI(t, v3Env(s), "deployments", "get", "--project-id", "123", "--id", "456")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/deploys/456", s.only(t).Path)
}

func TestDeploymentsGetJSONPrintsTheDeploy(t *testing.T) {
	s := newV3Server(t, ok(deployBody))
	out, err := runCLI(t, v3Env(s), "deployments", "get", "--project-id", "Xk9mZp",
		"--id", "Dp2yNk", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Dp2yNk"`)
	assert.Contains(t, out, `"local_username": "jane"`)
}

func TestDeploymentsGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "deployments", "get", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestDeploymentsDeleteIsGone(t *testing.T) {
	for _, c := range deploymentsCmd.Commands() {
		assert.NotEqual(t, "delete", c.Name())
	}
}
