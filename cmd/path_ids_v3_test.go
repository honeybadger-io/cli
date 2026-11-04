package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An id becomes a path segment, and "." or ".." would resolve to a different
// resource: deleting Ingestion Key ".." would send DELETE /v3/projects/P/,
// which deletes the project.
func TestDotIDsAreRefusedBeforeAnyRequest(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		args []string
		flag string
	}{
		{
			name: "ingestion key ..",
			args: []string{"ingestion-keys", "delete", "--project-id", "Xk9mZp", "--id", ".."},
			flag: "--id",
		},
		{name: "project .", args: []string{"projects", "delete", "--id", "."}, flag: "--id"},
		{
			name: "project id ..",
			args: []string{"faults", "delete", "--project-id", "..", "--id", "Fk7pQ2"},
			flag: "--project-id",
		},
		{
			name: "merge target ..",
			args: []string{
				"faults",
				"merge",
				"--project-id",
				"Xk9mZp",
				"--id",
				"Fk7pQ2",
				"--into",
				"..",
			},
			flag: "--into",
		},
		{
			name: "comment fault id ..",
			args: []string{"comments", "list", "--project-id", "Xk9mZp", "--fault-id", ".."},
			flag: "--fault-id",
		},
		{
			name: "repeated bulk id",
			args: []string{
				"faults",
				"resolve",
				"--project-id",
				"Xk9mZp",
				"--id",
				"Fk7pQ2",
				"--id",
				"..",
			},
			flag: "--id",
		},
		{
			name: "project id from the environment",
			env:  map[string]string{"HONEYBADGER_PROJECT_ID": ".."},
			args: []string{"faults", "delete", "--id", "Fk7pQ2"},
			flag: "project ID",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t)
			env := v3Env(s)
			for k, v := range tt.env {
				env[k] = v
			}
			_, err := runCLI(t, env, tt.args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.flag)
			assert.Contains(t, err.Error(), "isn't an ID")
			assert.Empty(t, s.Requests())
		})
	}
}

func TestIDsContainingDotsAreSent(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	_, err := runCLI(t, v3Env(s), "projects", "get", "--id", "a.b")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/a.b", s.only(t).Path)
}
