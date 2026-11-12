package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v2's project create printed the new project's key, which an app needs to
// report errors; v3's create response doesn't carry it, so the CLI reads it
// from the project's Ingestion Keys.

func TestProjectsCreatePrintsTheNewProjectsIngestionKey(t *testing.T) {
	s := newV3Routes(t, map[string]v3Response{
		"POST /v3/projects":                      status(http.StatusCreated, projectBody),
		"GET /v3/projects/Xk9mZp/ingestion_keys": ok(ingestionKeysListBody),
	})
	out, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"create",
		"--cli-input-json",
		`{"name":"My Project"}`,
	)
	require.NoError(t, err)
	assert.Equal(t, "Project created successfully!\n"+
		"  ID: Xk9mZp\n"+
		"  Name: My Project\n"+
		"  Ingestion Key: hbp_aBcDeFgHiJkLmNoPqRsTuVwXyZ012345aB12Cd\n", out)
	assert.Len(t, s.Requests(), 2)
}

func TestProjectsCreateSucceedsWhenTheTokenCantReadIngestionKeys(t *testing.T) {
	s := newV3Routes(t, map[string]v3Response{
		"POST /v3/projects": status(http.StatusCreated, projectBody),
		"GET /v3/projects/Xk9mZp/ingestion_keys": status(
			http.StatusForbidden,
			`{"error":{"code":"insufficient_scope","message":"Insufficient scope","details":{"required_scope":"projects:read","token_scopes":["projects:create"]}}}`,
		),
	})
	out, err := runCLI(
		t,
		v3Env(s),
		"projects",
		"create",
		"--cli-input-json",
		`{"name":"My Project"}`,
	)
	require.NoError(t, err)
	assert.Contains(t, out, "Project created successfully!")
	assert.NotContains(t, out, "Ingestion Key")
}

func TestProjectsCreateJSONPrintsOnlyTheProject(t *testing.T) {
	s := newV3Routes(t, map[string]v3Response{
		"POST /v3/projects": status(http.StatusCreated, projectBody),
	})
	out, err := runCLI(t, v3Env(s), "projects", "create", "--cli-input-json",
		`{"name":"My Project"}`, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Xk9mZp"`)
	assert.Len(t, s.Requests(), 1)
}
