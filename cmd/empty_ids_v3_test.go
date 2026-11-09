package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cobra's required-flag check accepts --id "", as a script passing an unset
// variable would. Like get and update, these commands refuse the empty id
// themselves instead of sending a request with an empty path segment.
func TestCommandsRefuseAnEmptyID(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{
			[]string{"faults", "delete", "--id", ""},
			"fault ID is required. Set it using --id flag",
		},
		{
			[]string{"faults", "merge", "--id", "", "--into", "Fk8rS3"},
			"fault ID is required. Set it using --id flag",
		},
		{
			[]string{"faults", "merge", "--id", "Fk7pQ2", "--into", ""},
			"fault ID to merge into is required. Set it using --into flag",
		},
		{
			[]string{"faults", "pause-recording", "--id", "", "--for", "hour"},
			"fault ID is required. Set it using --id flag",
		},
		{
			[]string{"faults", "resume-recording", "--id", ""},
			"fault ID is required. Set it using --id flag",
		},
		{
			[]string{"check-ins", "events", "--id", ""},
			"check-in ID is required. Set it using --id flag",
		},
	}
	for _, tt := range tests {
		t.Run(tt.args[0]+" "+tt.args[1], func(t *testing.T) {
			s := newV3Server(t)
			args := append(tt.args, "--project-id", "Xk9mZp")
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			assert.Empty(t, s.Requests())
		})
	}
}
