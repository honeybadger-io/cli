package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFaultsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "faults", "delete", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2", req.Path)
	assert.Equal(t, "Bearer hbu_test", req.Auth)
	assert.Contains(t, out, "Fault deleted successfully")
}

func TestFaultsDeleteReportsAMissingFault(t *testing.T) {
	s := newV3Server(t, status(http.StatusNotFound,
		`{"error":{"code":"not_found","message":"Fault not found"}}`))
	_, err := runCLI(t, v3Env(s), "faults", "delete", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete fault")
	assert.Contains(t, err.Error(), "Fault not found")
}

func TestFaultsDeleteRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "delete", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestFaultsDeleteRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "delete", "--id", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}
