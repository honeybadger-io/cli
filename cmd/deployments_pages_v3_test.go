package cmd

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v3 has no date filters for deploys, so older ones are reached by cursor:
// each page's oldest and newest cursors are printed (to stderr, keeping the
// table and JSON clean) for --before and --after to fetch the next page.

const deploysMiddlePage = `{"data":[` + deployJSON + `],"pagination":{"limit":1,"has_older":true,"has_newer":true,"oldest_cursor":"Old1","newest_cursor":"New1"},"links":{"self":"/v3/projects/Xk9mZp/deploys","older":"/v3/projects/Xk9mZp/deploys?before=Old1","newer":"/v3/projects/Xk9mZp/deploys?after=New1"}}`

func TestDeploymentsListSendsTheCursor(t *testing.T) {
	for _, flag := range []string{"before", "after"} {
		t.Run(flag, func(t *testing.T) {
			s := newV3Server(t, ok(deploysListBody))
			_, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp",
				"--"+flag, "Cur9")
			require.NoError(t, err)
			assertQuery(t, url.Values{"limit": {"25"}, flag: {"Cur9"}}, s.only(t).Query)
		})
	}
}

func TestDeploymentsListRefusesBothCursors(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp",
		"--before", "Old1", "--after", "New1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "[after before] were all set")
	assert.Empty(t, s.Requests())
}

func TestDeploymentsListSaysHowToReachTheNextPages(t *testing.T) {
	for _, output := range []string{"table", "json"} {
		t.Run(output, func(t *testing.T) {
			s := newV3Server(t, ok(deploysMiddlePage))
			stderr := captureStderr(t)
			out, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp",
				"--output", output)
			require.NoError(t, err)
			hints := stderr()
			assert.Contains(
				t,
				hints,
				"Older deployments: rerun with --before Old1 in place of any --before or --after",
			)
			assert.Contains(
				t,
				hints,
				"Newer deployments: rerun with --after New1 in place of any --before or --after",
			)
			assert.NotContains(t, out, "Old1")
			if output == "json" {
				var deploys []map[string]any
				require.NoError(t, json.Unmarshal([]byte(out), &deploys))
				assert.Len(t, deploys, 1)
			}
		})
	}
}

func TestDeploymentsListSaysNothingMoreOnTheOnlyPage(t *testing.T) {
	s := newV3Server(t, ok(deploysListBody))
	stderr := captureStderr(t)
	_, err := runCLI(t, v3Env(s), "deployments", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	assert.Empty(t, stderr())
}
