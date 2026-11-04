package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const faultMergeBody = `{"data":{"batch_id":"Bt4mGx","source_id":"Fk7pQ2","target_id":"Fk8qR3"}}`

func TestFaultsMerge(t *testing.T) {
	s := newV3Server(t, status(http.StatusAccepted, faultMergeBody))
	out, err := runCLI(t, v3Env(s), "faults", "merge", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--into", "Fk8qR3")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/merge", req.Path)
	assert.Equal(t, "Bearer hbu_test", req.Auth)
	assertJSONBody(t, `{"target_fault_id":"Fk8qR3"}`, req.Body)
	assert.Contains(t, out, "Merging fault Fk7pQ2 into Fk8qR3 (batch Bt4mGx)")
	assert.Contains(t, out, "in the background")
}

func TestFaultsMergeRequiresBothFaults(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
	}{
		{"no source", []string{"--into", "Fk8qR3"}, "id"},
		{"no target", []string{"--id", "Fk7pQ2"}, "into"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t)
			args := append([]string{"faults", "merge", "--project-id", "Xk9mZp"}, tt.args...)
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `required flag(s) "`+tt.flag+`" not set`)
			assert.Empty(t, s.Requests())
		})
	}
}

func TestFaultsMergeRefusesMergingAFaultIntoItself(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "merge", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--into", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to merge fault")
	assert.Contains(t, err.Error(), "cannot be merged into itself")
	assert.Empty(t, s.Requests())
}

func TestFaultsMergeReportsAnAlreadyMergedFault(t *testing.T) {
	s := newV3Server(t, status(
		http.StatusConflict,
		`{"error":{"code":"fault_merged","message":"This fault was merged into another","details":{"merged_into":"Fk9sT4"}}}`,
	))
	_, err := runCLI(t, v3Env(s), "faults", "merge", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--into", "Fk8qR3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to merge fault")
	assert.Contains(t, err.Error(), "merged into another")
}

func TestFaultsMergeRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "merge", "--id", "Fk7pQ2", "--into", "Fk8qR3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}
