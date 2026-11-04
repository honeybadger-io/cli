package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A comment that records a change (resolved, unresolved, ...) has no body in
// v3. v2's server gave it one, "marked as resolved", which the CLI now writes
// itself in the text output.

const activityCommentJSON = `{"id":"Cm9rNz","fault_id":"Fk7pQ2","author":{"id":"Us3rAb","name":"Jane Doe","email":"jane@example.com"},"body":null,"source":"api","event":"resolved","created_at":"2026-01-04T05:06:07Z","links":{"web":"https://app.honeybadger.io/c"}}`

func TestCommentsListSaysWhatAnActivityCommentMarked(t *testing.T) {
	s := newV3Server(t, ok(`{"data":[`+activityCommentJSON+
		`],"pagination":{"limit":100,"has_older":false,"has_newer":false},"links":{"self":"/v3/projects/Xk9mZp/faults/Fk7pQ2/comments","older":null,"newer":null}}`))
	out, err := runCLI(t, v3Env(s), "comments", "list", "--project-id", "Xk9mZp",
		"--fault-id", "Fk7pQ2")
	require.NoError(t, err)
	assert.Contains(t, out, "resolved  2026-01-04 05:06  marked as resolved")
}

func TestCommentsGetSaysWhatAnActivityCommentMarked(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+activityCommentJSON+`}`))
	out, err := runCLI(t, v3Env(s), "comments", "get", "--project-id", "Xk9mZp",
		"--fault-id", "Fk7pQ2", "--id", "Cm9rNz")
	require.NoError(t, err)
	assert.Contains(t, out, "  Body:\n    marked as resolved\n")
}

func TestCommentsJSONKeepsAnActivityCommentsBodyNull(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+activityCommentJSON+`}`))
	out, err := runCLI(t, v3Env(s), "comments", "get", "--project-id", "Xk9mZp",
		"--fault-id", "Fk7pQ2", "--id", "Cm9rNz", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"body": null`)
}
