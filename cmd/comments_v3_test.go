package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const commentJSON = `{"id":"Cm7pLx","fault_id":"Fk7pQ2","author":{"id":"Us3rAb","name":"Jane Doe","email":"jane@example.com"},"body":"Looking into it","source":"web","event":null,"created_at":"2026-01-03T04:05:06Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/faults/Fk7pQ2#comment_Cm7pLx"}}`

const commentResponse = `{"data":` + commentJSON + `}`

const commentsListBody = `{"data":[` + commentJSON + `,{"id":"Cm8qMy","fault_id":"Fk7pQ2","author":null,"body":"Resolved by deploy","source":"system","event":"resolved","created_at":"2026-01-02T03:04:05Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/faults/Fk7pQ2#comment_Cm8qMy"}}],"pagination":{"limit":25,"has_older":false,"has_newer":false},"links":{"self":"/v3/projects/Xk9mZp/faults/Fk7pQ2/comments","older":null,"newer":null}}`

func TestCommentsList(t *testing.T) {
	s := newV3Server(t, ok(commentsListBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"comments",
		"list",
		"--project-id",
		"Xk9mZp",
		"--fault-id",
		"Fk7pQ2",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/comments", req.Path)
	assert.Empty(t, req.Query)
	assert.Contains(t, out, "Cm7pLx  Jane Doe")
	assert.Contains(t, out, "Looking into it")
	assert.Contains(t, out, "Cm8qMy  System")
	assert.Contains(t, out, "resolved")
}

func TestCommentsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(commentsListBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"comments",
		"list",
		"--project-id",
		"Xk9mZp",
		"--fault-id",
		"Fk7pQ2",
		"--output",
		"json",
	)
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Cm7pLx"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestCommentsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(commentsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "comments", "list", "--fault-id", "456")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/faults/456/comments", s.only(t).Path)
}

func TestCommentsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "comments", "list", "--fault-id", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestCommentsListRequiresAFaultID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "comments", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fault ID is required. Set it using --fault-id flag")
	assert.Empty(t, s.Requests())
}

func TestCommentsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"comments", "list", "--project-id", "Xk9mZp", "--fault-id", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestCommentsGet(t *testing.T) {
	s := newV3Server(t, ok(commentResponse))
	out, err := runCLI(
		t,
		v3Env(s),
		"comments",
		"get",
		"--project-id",
		"Xk9mZp",
		"--fault-id",
		"Fk7pQ2",
		"--id",
		"Cm7pLx",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/comments/Cm7pLx", req.Path)
	assert.Contains(t, out, "ID: Cm7pLx")
	assert.Contains(t, out, "Fault ID: Fk7pQ2")
	assert.Contains(t, out, "Author: Jane Doe")
	assert.Contains(t, out, "Looking into it")
}

func TestCommentsCreate(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, commentResponse))
	out, err := runCLI(
		t,
		v3Env(s),
		"comments",
		"create",
		"--project-id",
		"Xk9mZp",
		"--fault-id",
		"Fk7pQ2",
		"--body",
		"Looking into it",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/comments", req.Path)
	assertJSONBody(t, `{"body":"Looking into it"}`, req.Body)
	assert.Contains(t, out, "ID: Cm7pLx")
}

func TestCommentsUpdate(t *testing.T) {
	s := newV3Server(t, ok(commentResponse))
	out, err := runCLI(
		t,
		v3Env(s),
		"comments",
		"update",
		"--project-id",
		"Xk9mZp",
		"--fault-id",
		"Fk7pQ2",
		"--id",
		"Cm7pLx",
		"--body",
		"Looking into it",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "PATCH", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/comments/Cm7pLx", req.Path)
	assertJSONBody(t, `{"body":"Looking into it"}`, req.Body)
	assert.Contains(t, out, "ID: Cm7pLx")
}

func TestCommentsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(
		t,
		v3Env(s),
		"comments",
		"delete",
		"--project-id",
		"Xk9mZp",
		"--fault-id",
		"Fk7pQ2",
		"--id",
		"Cm7pLx",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/comments/Cm7pLx", req.Path)
	assert.Contains(t, out, "Comment deleted successfully")
}
