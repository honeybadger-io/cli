package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An empty --project-id, as a script passes when its variable is unset, is
// refused rather than taken as "not given": otherwise the command would act on
// the default project from HONEYBADGER_PROJECT_ID or the config file.
func TestAnEmptyProjectIDIsRefusedRatherThanReplacedByTheDefault(t *testing.T) {
	tests := [][]string{
		{"faults", "resolve", "--all", "--occurred-before", "2026-01-01"},
		{"streams", "list"},
	}
	for _, args := range tests {
		t.Run(args[0]+" "+args[1], func(t *testing.T) {
			s := newV3Server(t)
			env := v3Env(s)
			env["HONEYBADGER_PROJECT_ID"] = "Xk9mZp"
			_, err := runCLI(t, env, append(args, "--project-id", "")...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "--project-id is empty")
			assert.Empty(t, s.Requests())
		})
	}
}
