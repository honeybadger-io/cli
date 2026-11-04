package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pausedFaultBody = `{"data":` + strings.Replace(faultJSON, `"resolved":false,`,
	`"resolved":false,"recording_paused_until":"2026-01-10T03:04:05Z",`, 1) + `}`

var resumedFaultBody = `{"data":` + strings.Replace(faultJSON, `"resolved":false,`,
	`"resolved":false,"recording_paused_until":null,`, 1) + `}`

func TestFaultsPauseRecording(t *testing.T) {
	for _, duration := range []string{"hour", "day", "week"} {
		t.Run(duration, func(t *testing.T) {
			s := newV3Server(t, ok(pausedFaultBody))
			out, err := runCLI(t, v3Env(s), "faults", "pause-recording", "--project-id", "Xk9mZp",
				"--id", "Fk7pQ2", "--for", duration)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "POST", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/pause_recording", req.Path)
			assert.Equal(t, "Bearer hbu_test", req.Auth)
			assertJSONBody(t, `{"time":"`+duration+`"}`, req.Body)
			assert.Contains(t, out,
				"Recording paused for fault Fk7pQ2 until 2026-01-10 03:04:05 UTC")
		})
	}
}

// The API leaves out recording_paused_until for a fault on an inactive project.
func TestFaultsPauseRecordingWithoutAnEndTime(t *testing.T) {
	s := newV3Server(t, ok(faultBody))
	out, err := runCLI(t, v3Env(s), "faults", "pause-recording", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--for", "hour")
	require.NoError(t, err)
	assert.Equal(t, "Recording paused for fault Fk7pQ2\n", out)
}

func TestFaultsPauseRecordingRefusesAnUnknownDuration(t *testing.T) {
	s := newV3Server(t, ok(pausedFaultBody))
	_, err := runCLI(t, v3Env(s), "faults", "pause-recording", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2", "--for", "month")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid --for "month": use hour, day or week`)
	assert.Empty(t, s.Requests())
}

func TestFaultsPauseRecordingRequiresADuration(t *testing.T) {
	s := newV3Server(t, ok(pausedFaultBody))
	_, err := runCLI(t, v3Env(s), "faults", "pause-recording", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "for" not set`)
	assert.Empty(t, s.Requests())
}

func TestFaultsPauseRecordingRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "pause-recording", "--id", "Fk7pQ2", "--for", "day")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestFaultsResumeRecording(t *testing.T) {
	s := newV3Server(t, ok(resumedFaultBody))
	out, err := runCLI(t, v3Env(s), "faults", "resume-recording", "--project-id", "Xk9mZp",
		"--id", "Fk7pQ2")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/faults/Fk7pQ2/resume_recording", req.Path)
	assert.Empty(t, req.Body)
	assert.Contains(t, out, "Recording resumed for fault Fk7pQ2")
}

func TestFaultsResumeRecordingRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "faults", "resume-recording", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}
