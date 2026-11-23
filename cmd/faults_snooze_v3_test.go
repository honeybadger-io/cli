package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var snoozedFaultBody = `{"data":` + strings.Replace(faultJSON, `"resolved":false,`,
	`"resolved":false,"snoozed":true,`, 1) + `}`

var unsnoozedFaultBody = `{"data":` + strings.Replace(faultJSON, `"resolved":false,`,
	`"resolved":false,"snoozed":false,`, 1) + `}`

func TestFaultsSnoozeFor(t *testing.T) {
	for _, duration := range []string{"hour", "day", "week"} {
		t.Run(duration, func(t *testing.T) {
			s := newV3Server(t, ok(snoozedFaultBody))
			out, err := runCLI(t, v3Env(s), "faults", "snooze", "--project-id", "Xk9mZp",
				"--id", "Fk7pQ2", "--for", duration)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/snooze", req.Path)
			assert.Equal(t, "Bearer hbu_test", req.Auth)
			assertJSONBody(t, `{"time":"`+duration+`"}`, req.Body)
			assert.Equal(t, "Fault Fk7pQ2 snoozed for a "+duration+"\n", out)
		})
	}
}

func TestFaultsSnoozeOccurrences(t *testing.T) {
	for _, count := range []string{"10", "100", "1000"} {
		t.Run(count, func(t *testing.T) {
			s := newV3Server(t, ok(snoozedFaultBody))
			out, err := runCLI(t, v3Env(s), "faults", "snooze", "--project-id", "Xk9mZp",
				"--id", "Fk7pQ2", "--occurrences", count)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/snooze", req.Path)
			assertJSONBody(t, `{"count":`+count+`}`, req.Body)
			assert.Equal(t, "Fault Fk7pQ2 snoozed for the next "+count+" occurrences\n", out)
		})
	}
}

func TestFaultsSnoozeRefusesBadInput(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"unknown duration": {
			[]string{"--for", "month"}, `invalid --for "month": use hour, day or week`,
		},
		"unknown count": {
			[]string{"--occurrences", "50"}, `invalid --occurrences 50: use 10, 100 or 1000`,
		},
		"neither": {
			nil, `at least one of the flags in the group [for occurrences] is required`,
		},
		"both": {
			[]string{"--for", "day", "--occurrences", "10"},
			`if any flags in the group [for occurrences] are set none of the others can be`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := newV3Server(t, ok(snoozedFaultBody))
			args := append([]string{"faults", "snooze", "--project-id", "Xk9mZp", "--id", "Fk7pQ2"},
				tc.args...)
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Empty(t, s.Requests())
		})
	}
}

func TestFaultsSnoozeRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "snooze", "--id", "Fk7pQ2", "--for", "day")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestFaultsUnsnooze(t *testing.T) {
	s := newV3Server(t, ok(unsnoozedFaultBody))
	out, err := runCLI(t, v3Env(s), "faults", "unsnooze", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/snooze", req.Path)
	assert.Empty(t, req.Body)
	assert.Equal(t, "Fault Fk7pQ2 unsnoozed\n", out)
}

func TestFaultsUnsnoozeRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "unsnooze", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}
