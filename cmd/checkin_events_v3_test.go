package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const checkInEventsBody = `{"data":[{"check_in_id":"Ci5xWq","state":"reporting","created_at":"2026-01-03T04:05:06Z","payload":"{\"rows\":42}"},{"check_in_id":"Ci5xWq","state":"missing","created_at":"2026-01-02T04:20:00Z","payload":null}],"pagination":{"limit":25,"has_older":true,"has_newer":false},"links":{"self":"/v3/projects/Xk9mZp/check_ins/Ci5xWq/events","older":"/v3/projects/Xk9mZp/check_ins/Ci5xWq/events?created_before=1767327600","newer":null}}`

func TestCheckInEvents(t *testing.T) {
	s := newV3Server(t, ok(checkInEventsBody))
	out, err := runCLI(t, v3Env(s), "check-ins", "events", "--project-id", "Xk9mZp",
		"--id", "Ci5xWq")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/check_ins/Ci5xWq/events", req.Path)
	assert.Equal(t, "Bearer hbu_test", req.Auth)
	assert.Empty(t, req.Query)
	assert.Contains(t, out, "TIME                 STATE")
	assert.Contains(t, out, "2026-01-03 04:05:06  reporting")
	assert.Contains(t, out, "2026-01-02 04:20:00  missing")
}

func TestCheckInEventsJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(checkInEventsBody))
	out, err := runCLI(t, v3Env(s), "check-ins", "events", "--project-id", "Xk9mZp",
		"--id", "Ci5xWq", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"state": "reporting"`)
	assert.Contains(t, out, `"payload": "{\"rows\":42}"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestCheckInEventsTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(checkInEventsBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "check-ins", "events", "--id", "Ci5xWq")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/check_ins/Ci5xWq/events", s.only(t).Path)
}

func TestCheckInEventsRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "events", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestCheckInEventsRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "events", "--id", "Ci5xWq")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}
