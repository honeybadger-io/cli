package cmd

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const streamsListBody = `{"data":[{"id":"a3f19c02b7de","name":"Default","slug":"default","internal":false,"project_id":"Xk9mZp","created_at":"2026-01-02T03:04:05Z"},{"id":"b4e20d13c8ef","name":"Internal","slug":"internal","internal":true,"project_id":"Xk9mZp","created_at":"2026-01-02T03:04:06Z"}],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/streams","next":null}}`

func TestStreamsList(t *testing.T) {
	s := newV3Server(t, ok(streamsListBody))
	out, err := runCLI(t, v3Env(s), "streams", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/streams", req.Path)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID            NAME      SLUG      INTERNAL  CREATED")
	assert.Contains(t, out, "a3f19c02b7de  Default   default             2026-01-02 03:04")
	assert.Contains(t, out, "b4e20d13c8ef  Internal  internal  ✓         2026-01-02 03:04")
}

func TestStreamsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(streamsListBody))
	out, err := runCLI(t, v3Env(s), "streams", "list", "--project-id", "Xk9mZp",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "a3f19c02b7de"`)
	assert.Contains(t, out, `"internal": true`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestStreamsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(streamsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "streams", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/streams", s.only(t).Path)
}

func TestStreamsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "streams", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestStreamsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"streams", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}
