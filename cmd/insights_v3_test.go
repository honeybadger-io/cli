package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insightsQueryBody is a v3 query result. data is the query service's
// json_events_full result, passed through unchanged.
const insightsQueryBody = `{"data":{"results":[{"count":42,"bucket":"2026-07-29T00:00:00Z","avg":1.256},{"count":7,"bucket":"2026-07-30T00:00:00Z","avg":null}],"meta":{"query":"stats count() by bin(1d) as bucket","fields":["count","bucket","avg"],"schema":[],"row_count":2,"total_count":2,"start_at":"2026-07-28 00:00:00","end_at":"2026-07-31 00:00:00"}},"links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/insights?query=stats"},"meta":{"request_id":"req_q"}}`

const insightsEmptyBody = `{"data":{"results":[],"meta":{"query":"fields @ts","fields":["@ts"],"schema":[],"row_count":0,"total_count":0,"start_at":"2026-07-28 00:00:00","end_at":"2026-07-31 00:00:00"}},"links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/insights?query=fields"}}`

func TestInsightsQuery(t *testing.T) {
	s := newV3Server(t, ok(insightsQueryBody))
	out, err := runCLI(t, v3Env(s), "insights", "query", "--project-id", "Xk9mZp",
		"--query", "stats count() by bin(1d) as bucket")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/insights/queries", req.Path)
	assertJSONBody(t, `{"query":"stats count() by bin(1d) as bucket"}`, req.Body)
	assert.Contains(t, out, "Query: stats count() by bin(1d) as bucket")
	assert.Contains(t, out, "Rows: 2 (Total: 2)")
	assert.Contains(t, out, "Time Range: 2026-07-28 00:00:00 to 2026-07-31 00:00:00")
	assert.Contains(t, out, "count  bucket               avg")
	assert.Contains(t, out, "42     2026-07-29 00:00:00  1.26")
	assert.Contains(t, out, "7      2026-07-30 00:00:00  NULL")
}

func TestInsightsQuerySendsTheTimeRange(t *testing.T) {
	s := newV3Server(t, ok(insightsQueryBody))
	_, err := runCLI(t, v3Env(s), "insights", "query", "--project-id", "Xk9mZp",
		"--query", "fields @ts", "--ts", "P7D")
	require.NoError(t, err)
	assertJSONBody(t, `{"query":"fields @ts","ts":"P7D"}`, s.only(t).Body)
}

func TestInsightsQueryHelpDescribesTheTimeRange(t *testing.T) {
	usage := insightsQueryCmd.Flags().Lookup("ts").Usage
	assert.Contains(t, usage, "1h")
	assert.Contains(t, usage, "P7D")
	assert.NotContains(t, usage, "RFC3339")
	assert.NotContains(t, insightsQueryCmd.Long, "2024-01-01T00:00:00Z")
}

func TestInsightsQuerySendsTheStreamsAndTimezone(t *testing.T) {
	s := newV3Server(t, ok(insightsQueryBody))
	_, err := runCLI(t, v3Env(s), "insights", "query", "--project-id", "Xk9mZp",
		"--query", "fields @ts", "--stream-ids", "a3f19c02b7de,b4e20d13c8ef", "--timezone", "UTC")
	require.NoError(t, err)
	assertJSONBody(
		t,
		`{"query":"fields @ts","stream_ids":["a3f19c02b7de","b4e20d13c8ef"],"timezone":"UTC"}`,
		s.only(t).Body,
	)
}

func TestInsightsQueryWithNoResults(t *testing.T) {
	s := newV3Server(t, ok(insightsEmptyBody))
	out, err := runCLI(t, v3Env(s), "insights", "query", "--project-id", "Xk9mZp",
		"--query", "fields @ts")
	require.NoError(t, err)
	assert.Contains(t, out, "No results found")
}

func TestInsightsQueryJSONPrintsTheResult(t *testing.T) {
	s := newV3Server(t, ok(insightsQueryBody))
	out, err := runCLI(t, v3Env(s), "insights", "query", "--project-id", "Xk9mZp",
		"--query", "fields @ts", "--output", "json")
	require.NoError(t, err)
	// v3's data is {results, meta}, the same top level v2 printed, so scripts
	// reading .results keep working; the envelope's links and meta stay out.
	assert.True(
		t,
		strings.HasPrefix(out, "{\n  \"meta\"") || strings.HasPrefix(out, "{\n  \"results\""),
		out,
	)
	assert.Contains(t, out, `"results": [`)
	assert.Contains(t, out, `"row_count": 2`)
	assert.NotContains(t, out, `"links"`)
	assert.NotContains(t, out, `"request_id"`)
}

func TestInsightsQueryTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(insightsQueryBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "insights", "query", "--query", "fields @ts")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/insights/queries", s.only(t).Path)
}

func TestInsightsQueryRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "insights", "query", "--query", "fields @ts")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestInsightsQueryRequiresAQuery(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "insights", "query", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "query" not set`)
	assert.Empty(t, s.Requests())
}

func TestInsightsQueryRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"insights", "query", "--project-id", "Xk9mZp", "--query", "fields @ts")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}
