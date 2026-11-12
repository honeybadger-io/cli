package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tokenBody = `{"data":{"kind":"user","name":"Laptop CLI","scopes":["faults:read","faults:write"],"account_id":"Ab3kL9","project_ids":["Xk9mZp","Yl0nAq"],"expires_at":null,"last_used_at":"2026-01-03T04:05:06Z"}}`

func TestWhoami(t *testing.T) {
	s := newV3Server(t, ok(tokenBody))
	out, err := runCLI(t, v3Env(s), "whoami")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/token", req.Path)
	assert.Equal(t, "Bearer hbu_test", req.Auth)
	assert.Equal(t, `Kind: user
Name: Laptop CLI
Account: Ab3kL9
Scopes: faults:read, faults:write
Projects: Xk9mZp, Yl0nAq
Expires: never
Last used: 2026-01-03T04:05:06Z
`, out)
}

func TestWhoamiAnAccountTokenWithAnExpiry(t *testing.T) {
	s := newV3Server(
		t,
		ok(
			`{"data":{"kind":"account","name":null,"scopes":[],"account_id":"Ab3kL9","project_ids":[],"expires_at":"2026-06-01T00:00:00Z","last_used_at":null}}`,
		),
	)
	out, err := runCLI(
		t,
		cliEnv{"HONEYBADGER_API_TOKEN": "hba_test", "HONEYBADGER_ENDPOINT": s.URL},
		"whoami",
	)
	require.NoError(t, err)
	assert.Equal(t, "Bearer hba_test", s.only(t).Auth)
	assert.Contains(t, out, "Kind: account\n")
	assert.Contains(t, out, "Scopes: none\n")
	assert.Contains(t, out, "Projects: none\n")
	assert.Contains(t, out, "Expires: 2026-06-01T00:00:00Z\n")
	assert.Contains(t, out, "Last used: never\n")
}

func TestWhoamiDoesNotNeedAProject(t *testing.T) {
	s := newV3Server(t, ok(tokenBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "whoami")
	require.NoError(t, err)
	assert.Equal(t, "/v3/token", s.only(t).Path)
}

func TestWhoamiRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL}, "whoami")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestWhoamiAddsTheRegionHintToARevokedToken(t *testing.T) {
	s := newV3Server(t, status(http.StatusUnauthorized,
		`{"error":{"code":"unauthorized","message":"Access denied"}}`))
	_, err := runCLI(t, v3Env(s), "whoami")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to describe the API Token")
	assert.Contains(t, err.Error(), "for this region")
}

func TestWhoamiIsADataAPICommand(t *testing.T) {
	assert.Equal(t, GroupDataAPI, whoamiCmd.GroupID)
}
