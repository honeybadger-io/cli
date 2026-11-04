package cmd

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v2 printed the server's confirmation for these commands; v3 sends none, so
// the CLI prints the same words itself.

func TestProjectsUpdateSaysWhichProjectWasUpdated(t *testing.T) {
	s := newV3Server(t, ok(projectBody))
	out, err := runCLI(t, v3Env(s), "projects", "update", "--id", "Xk9mZp",
		"--cli-input-json", `{"name":"New"}`)
	require.NoError(t, err)
	assert.Equal(t, "Project Xk9mZp was successfully updated\n", out)
}

func TestProjectsDeleteSaysWhichProjectWasDeleted(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "projects", "delete", "--id", "Xk9mZp")
	require.NoError(t, err)
	assert.Equal(t, "Project Xk9mZp deleted successfully\n", out)
}
