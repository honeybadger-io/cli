package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const projectsListBody = `{"data":[{"id":"Xk9mZp","account_id":"Ab3kL9","name":"My Rails App","active":true,"fault_count":3,"unresolved_fault_count":1}],"pagination":{"page":1,"per_page":25},"links":{"self":"/v3/projects","next":null}}`

func TestDataAPIRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL}, "projects", "list")
	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"API Token is required. Set it with --api-token, HONEYBADGER_API_TOKEN, or api_token in your config file",
	)
	assert.Empty(t, s.Requests())
}

func TestDataAPIRefusesTheOldAuthTokenSetting(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(
		t,
		cliEnv{"HONEYBADGER_AUTH_TOKEN": "hbu_x", "HONEYBADGER_ENDPOINT": s.URL},
		"projects",
		"list",
	)
	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"auth_token is no longer used. The Data API now needs an API Token",
	)
	assert.Contains(t, err.Error(), s.URL+"/users/edit#authentication")
	assert.Empty(t, s.Requests())
}

func TestDataAPIUsesAPITokenWhenTheOldSettingIsAlsoSet(t *testing.T) {
	s := newV3Server(t, ok(projectsListBody))
	env := v3Env(s)
	env["HONEYBADGER_AUTH_TOKEN"] = "old"
	_, err := runCLI(t, env, "projects", "list")
	require.NoError(t, err)
	assert.Equal(t, "Bearer hbu_test", s.only(t).Auth)
}

func TestDataAPIRefusesAnIngestionKey(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(
		t,
		cliEnv{"HONEYBADGER_API_TOKEN": "hbp_abc", "HONEYBADGER_ENDPOINT": s.URL},
		"projects",
		"list",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "That's an Ingestion Key, which only sends errors and events")
	assert.Contains(t, err.Error(), "create one at "+s.URL+"/users/edit#authentication")
	assert.Empty(t, s.Requests())
}

func TestDataAPIRefusesAV2PersonalAuthToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(
		t,
		cliEnv{"HONEYBADGER_API_TOKEN": "abc123", "HONEYBADGER_ENDPOINT": s.URL},
		"projects",
		"list",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(),
		"That isn't an API Token. The v3 Data API doesn't accept v2 personal auth tokens")
	assert.Contains(t, err.Error(), "create one at "+s.URL+"/users/edit#authentication")
	assert.Empty(t, s.Requests())
}

func TestDataAPISendsEachKindOfAPIToken(t *testing.T) {
	for _, token := range []string{"hbu_abc", "hba_abc", "hbt_abc"} {
		t.Run(token, func(t *testing.T) {
			s := newV3Server(t, ok(projectsListBody))
			_, err := runCLI(
				t,
				cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
				"projects",
				"list",
				"--api-token",
				token,
			)
			require.NoError(t, err)
			assert.Equal(t, "Bearer "+token, s.only(t).Auth)
		})
	}
}

func TestDataAPITokenLinkFollowsTheRegion(t *testing.T) {
	_, err := runCLI(
		t,
		cliEnv{
			"HONEYBADGER_API_TOKEN": "abc123",
			"HONEYBADGER_ENDPOINT":  "https://eu-api.honeybadger.io",
		},
		"projects",
		"list",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://eu-app.honeybadger.io/users/edit#authentication")
}

func TestDataAPITokenFromConfigFile(t *testing.T) {
	s := newV3Server(t, ok(projectsListBody))
	cfg := t.TempDir() + "/hb.yaml"
	require.NoError(t, writeFile(cfg, "api_token: hba_fromfile\nendpoint: "+s.URL+"\n"))
	_, err := runCLI(t, nil, "projects", "list", "--config", cfg)
	require.NoError(t, err)
	assert.Equal(t, "Bearer hba_fromfile", s.only(t).Auth)
}

func TestDataAPIUnauthorizedAddsTheRegionHint(t *testing.T) {
	s := newV3Server(
		t,
		status(
			http.StatusUnauthorized,
			`{"error":{"code":"unauthorized","message":"Access denied"}}`,
		),
	)
	_, err := runCLI(t, v3Env(s), "projects", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Access denied")
	assert.Contains(
		t,
		err.Error(),
		"Check that the API Token hasn't been revoked or expired, and that it's for this region (US: app.honeybadger.io, EU: eu-app.honeybadger.io).",
	)
}

func TestDataAPIInsufficientScopeNamesTheScope(t *testing.T) {
	s := newV3Server(t, status(
		http.StatusForbidden,
		`{"error":{"code":"insufficient_scope","message":"Insufficient scope","details":{"required_scope":"projects:read","token_scopes":["faults:read"]}}}`,
	))
	_, err := runCLI(t, v3Env(s), "projects", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires projects:read")
	assert.NotContains(t, err.Error(), "for this region")
}

func TestDataAPITrimsWhitespaceAroundTheToken(t *testing.T) {
	s := newV3Server(t, ok(projectsListBody))
	_, err := runCLI(
		t,
		cliEnv{"HONEYBADGER_API_TOKEN": " hbu_test\n", "HONEYBADGER_ENDPOINT": s.URL},
		"projects",
		"list",
	)
	require.NoError(t, err)
	assert.Equal(t, "Bearer hbu_test", s.only(t).Auth)
}
