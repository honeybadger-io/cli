package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountsIsGone(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		assert.NotEqual(t, "accounts", c.Name())
	}
}

func TestAccountsIsAnUnknownCommand(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "accounts", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown command "accounts"`)
	assert.Empty(t, s.Requests())
}
