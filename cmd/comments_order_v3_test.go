package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v2 listed a fault's comments oldest first, as the fault's history reads; v3
// pages them newest first, so the CLI reads every page and reverses them.

const commentsPath = "/v3/projects/Xk9mZp/faults/Fk7pQ2/comments"

func commentJSONAt(id, day string) string {
	return `{"id":"` + id + `","fault_id":"Fk7pQ2","author":{"id":"Us3rAb","name":"Jane Doe","email":"jane@example.com"},"body":"note ` + id + `","source":"web","event":null,"created_at":"2026-01-` + day + `T04:05:06Z","links":{"web":"https://app.honeybadger.io/c"}}`
}

var commentsNewestPage = `{"data":[` + commentJSONAt(
	"Cm3",
	"03",
) + `,` + commentJSONAt(
	"Cm2",
	"02",
) +
	`],"pagination":{"limit":100,"has_older":true,"has_newer":false},"links":{"self":"` + commentsPath +
	`","older":"` + commentsPath + `?created_before=1767326706","newer":null}}`

var commentsOldestPage = `{"data":[` + commentJSONAt("Cm1", "01") +
	`],"pagination":{"limit":100,"has_older":false,"has_newer":true},"links":{"self":"` + commentsPath +
	`?created_before=1767326706","older":null,"newer":null}}`

func commentsRoutes() map[string]v3Response {
	return map[string]v3Response{
		"GET " + commentsPath + "?limit=100":                 ok(commentsNewestPage),
		"GET " + commentsPath + "?created_before=1767326706": ok(commentsOldestPage),
	}
}

func TestCommentsListPrintsEveryCommentOldestFirst(t *testing.T) {
	s := newV3Routes(t, commentsRoutes())
	out, err := runCLI(t, v3Env(s), "comments", "list", "--project-id", "Xk9mZp",
		"--fault-id", "Fk7pQ2")
	require.NoError(t, err)
	assert.Len(t, s.Requests(), 2)
	first, second, third := strings.Index(out, "Cm1"), strings.Index(out, "Cm2"),
		strings.Index(out, "Cm3")
	require.True(t, first > 0 && second > 0 && third > 0, out)
	assert.True(t, first < second && second < third, "want Cm1, Cm2, Cm3 in order:\n%s", out)
}

func TestCommentsListJSONIsOldestFirstToo(t *testing.T) {
	s := newV3Routes(t, commentsRoutes())
	out, err := runCLI(t, v3Env(s), "comments", "list", "--project-id", "Xk9mZp",
		"--fault-id", "Fk7pQ2", "--output", "json")
	require.NoError(t, err)
	first, third := strings.Index(out, `"id": "Cm1"`), strings.Index(out, `"id": "Cm3"`)
	require.True(t, first > 0 && third > 0, out)
	assert.Less(t, first, third)
}
