package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Without --id, v2 indented each project's table under its heading.

func TestProjectsOccurrencesIndentsEachProjectsTable(t *testing.T) {
	s := newV3Server(t, ok(
		`{"data":[{"project_id":"Xk9mZp","buckets":[{"timestamp":1767225600,"count":4}]},{"project_id":"Pq2rSt","buckets":[{"timestamp":1767225600,"count":0}]}],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/occurrences","next":null}}`,
	))
	out, err := runCLI(t, v3Env(s), "projects", "occurrences")
	require.NoError(t, err)
	assert.Equal(t, "Project Xk9mZp:\n"+
		"  TIMESTAMP             COUNT\n"+
		"  2026-01-01T00:00:00Z  4\n"+
		"\n"+
		"Project Pq2rSt:\n"+
		"  TIMESTAMP             COUNT\n"+
		"  2026-01-01T00:00:00Z  0\n", out)
}
