package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const faultBulkBody = `{"data":{"count":2,"dry_run":false,"fault_ids":["Fk7pQ2","Fk8qR3"],"fault_ids_truncated":false}}`

const selectorMessage = "use exactly one of --id, --query or --all"

const timeFilterWithIDMessage = "--created-after, --occurred-after and --occurred-before apply to --query or --all, not --id"

const allNeedsAFilterMessage = "--all needs --created-after, --occurred-after or --occurred-before; it won't change every fault in the project at once"

func TestFaultsBulkSendsTheNamedFaultsToEachAction(t *testing.T) {
	tests := []struct {
		command string
		path    string
		output  string
	}{
		{"resolve", "/v3/projects/Xk9mZp/faults/resolve", "Resolved 2 faults:"},
		{"unresolve", "/v3/projects/Xk9mZp/faults/unresolve", "Unresolved 2 faults:"},
		{"ignore", "/v3/projects/Xk9mZp/faults/ignore", "Ignored 2 faults:"},
		{"unignore", "/v3/projects/Xk9mZp/faults/unignore", "Unignored 2 faults:"},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			s := newV3Server(t, ok(faultBulkBody))
			out, err := runCLI(t, v3Env(s), "faults", tt.command, "--project-id", "Xk9mZp",
				"--id", "Fk7pQ2", "--id", "Fk8qR3")
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, tt.path, req.Path)
			assert.Equal(t, "Bearer hbu_test", req.Auth)
			assertJSONBody(t, `{"fault_ids":["Fk7pQ2","Fk8qR3"]}`, req.Body)
			assert.Contains(t, out, tt.output)
			assert.Contains(t, out, "  Fk7pQ2\n  Fk8qR3\n")
		})
	}
}

func TestFaultsBulkSendsTheQueryAndTimeFilters(t *testing.T) {
	s := newV3Server(t, ok(faultBulkBody))
	_, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
		"--query", "is:unresolved", "--occurred-before", "2026-01-01")
	require.NoError(t, err)
	assertJSONBody(t, `{"q":"is:unresolved","occurred_before":1767225600}`, s.only(t).Body)
}

func TestFaultsBulkSendsEveryTimeFilter(t *testing.T) {
	s := newV3Server(t, ok(faultBulkBody))
	_, err := runCLI(t, v3Env(s), "faults", "ignore", "--project-id", "Xk9mZp",
		"--query", "class:RuntimeError",
		"--created-after", "2026-01-01T00:00:00Z",
		"--occurred-after", "2026-01-02",
		"--occurred-before", "2026-01-03T00:00:00")
	require.NoError(t, err)
	assertJSONBody(
		t,
		`{"q":"class:RuntimeError","created_after":1767225600,"occurred_after":1767312000,"occurred_before":1767398400}`,
		s.only(t).Body,
	)
}

func TestFaultsBulkAllWithATimeFilter(t *testing.T) {
	s := newV3Server(t, ok(faultBulkBody))
	_, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
		"--all", "--occurred-before", "2026-01-01")
	require.NoError(t, err)
	assertJSONBody(t, `{"all":true,"occurred_before":1767225600}`, s.only(t).Body)
}

func TestFaultsBulkRefusesAllWithoutATimeFilter(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"alone", []string{"--all"}},
		{"dry run", []string{"--all", "--dry-run"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(faultBulkBody))
			args := append([]string{"faults", "resolve", "--project-id", "Xk9mZp"}, tt.args...)
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), allNeedsAFilterMessage)
			assert.Empty(t, s.Requests())
		})
	}
}

func TestFaultsBulkHelpSaysAllNeedsATimeFilter(t *testing.T) {
	assert.Contains(t, faultsResolveCmd.Long, "--all needs a time filter")
}

func TestFaultsBulkRefusesAnythingButOneSelector(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"none", nil},
		{"id and query", []string{"--id", "Fk7pQ2", "--query", "x"}},
		{"query and all", []string{"--query", "x", "--all", "--occurred-before", "2026-01-01"}},
		{"id and all", []string{"--id", "Fk7pQ2", "--all"}},
		{"a time filter alone", []string{"--occurred-before", "2026-01-01"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(faultBulkBody))
			args := append([]string{"faults", "unresolve", "--project-id", "Xk9mZp"}, tt.args...)
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), selectorMessage)
			assert.Empty(t, s.Requests())
		})
	}
}

func TestFaultsBulkRefusesTimeFiltersWithIDs(t *testing.T) {
	for _, filter := range []string{"--created-after", "--occurred-after", "--occurred-before"} {
		t.Run(filter, func(t *testing.T) {
			s := newV3Server(t, ok(faultBulkBody))
			_, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
				"--id", "Fk7pQ2", filter, "2026-01-01")
			require.Error(t, err)
			assert.Contains(t, err.Error(), timeFilterWithIDMessage)
			assert.Empty(t, s.Requests())
		})
	}
}

func TestFaultsBulkRefusesABadTime(t *testing.T) {
	s := newV3Server(t, ok(faultBulkBody))
	_, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
		"--query", "x", "--occurred-before", "yesterday")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--occurred-before")
	assert.Contains(t, err.Error(), `invalid time format "yesterday"`)
	assert.Empty(t, s.Requests())
}

func TestFaultsBulkDryRun(t *testing.T) {
	s := newV3Server(
		t,
		ok(
			`{"data":{"count":1,"dry_run":true,"fault_ids":["Fk7pQ2"],"fault_ids_truncated":false}}`,
		),
	)
	out, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
		"--query", "is:unresolved", "--dry-run")
	require.NoError(t, err)
	assertJSONBody(t, `{"q":"is:unresolved","dry_run":true}`, s.only(t).Body)
	assert.Contains(t, out, "Dry run, nothing changed. Would resolve 1 fault:")
	assert.Contains(t, out, "  Fk7pQ2\n")
}

func TestFaultsBulkSaysWhenTheListIsTruncated(t *testing.T) {
	s := newV3Server(
		t,
		ok(
			`{"data":{"count":250,"dry_run":false,"fault_ids":["Fk7pQ2","Fk8qR3"],"fault_ids_truncated":true}}`,
		),
	)
	out, err := runCLI(t, v3Env(s), "faults", "ignore", "--project-id", "Xk9mZp",
		"--all", "--created-after", "2026-01-01")
	require.NoError(t, err)
	assert.Contains(t, out, "Ignored 250 faults:")
	assert.Contains(t, out, "  …and 248 more\n")
}

func TestFaultsBulkReportsNoChange(t *testing.T) {
	s := newV3Server(
		t,
		ok(`{"data":{"count":0,"dry_run":false,"fault_ids":[],"fault_ids_truncated":false}}`),
	)
	out, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2")
	require.NoError(t, err)
	assert.Equal(t, "Resolved 0 faults\n", out)
}

func TestFaultsBulkJSONPrintsTheResult(t *testing.T) {
	s := newV3Server(t, ok(faultBulkBody))
	out, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"count": 2`)
	assert.Contains(t, out, `"fault_ids": [`)
	assert.NotContains(t, out, `"data"`)
}

func TestFaultsBulkRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "resolve", "--id", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestFaultsBulkRefusesExtraArguments(t *testing.T) {
	s := newV3Server(t, ok(faultBulkBody))
	_, err := runCLI(t, v3Env(s), "faults", "ignore", "--project-id", "Xk9mZp",
		"--query", "class:RuntimeError", "environment:production")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown command "environment:production"`)
	assert.Empty(t, s.Requests())
}

func TestFaultsBulkRefusesATimeAtOrBeforeTheEpoch(t *testing.T) {
	for _, flag := range []string{"--created-after", "--occurred-after", "--occurred-before"} {
		t.Run(flag, func(t *testing.T) {
			s := newV3Server(t, ok(faultBulkBody))
			_, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp",
				"--all", flag, "1970-01-01")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "must be after 1970-01-01")
			assert.Empty(t, s.Requests())
		})
	}
}

func TestFaultsBulkRefusesABlankQuery(t *testing.T) {
	s := newV3Server(t, ok(faultBulkBody))
	_, err := runCLI(t, v3Env(s), "faults", "resolve", "--project-id", "Xk9mZp", "--query", " ")
	require.Error(t, err)
	assert.Empty(t, s.Requests())
}
