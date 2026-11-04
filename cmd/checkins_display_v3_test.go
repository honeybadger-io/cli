package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// v3 sends an unnamed check-in's name as null and an unset grace period as
// "00:00:00". v2 showed the check-in's id as its name and a blank grace period;
// the text output does the same, and the JSON keeps v3's values.

const unnamedCheckInJSON = `{"id":"Ci0nUl","project_id":"Xk9mZp","name":null,"slug":null,"schedule_type":"simple","state":"pending","reported_at":null,"expected_at":null,"missed_count":0,"grace_period":"00:00:00","report_url":"https://api.honeybadger.io/v1/check_in/Ci0nUl","cron_timezone":"UTC","created_at":"2026-01-02T03:04:05Z","links":{"web":"https://app.honeybadger.io/c"},"report_period":"5 minutes"}`

func TestCheckInsListNamesAnUnnamedCheckInByItsID(t *testing.T) {
	s := newV3Server(t, ok(`{"data":[`+unnamedCheckInJSON+
		`],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/check_ins","next":null}}`))
	out, err := runCLI(t, v3Env(s), "check-ins", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	assert.Contains(t, out, "Ci0nUl  Ci0nUl")
}

func TestCheckInsGetShowsAnUnnamedCheckInsIDAndNoGracePeriod(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+unnamedCheckInJSON+`}`))
	out, err := runCLI(t, v3Env(s), "check-ins", "get", "--project-id", "Xk9mZp",
		"--id", "Ci0nUl")
	require.NoError(t, err)
	assert.Contains(t, out, "  Name: Ci0nUl\n")
	assert.Contains(t, out, "  Grace Period: \n")
}

func TestCheckInsJSONKeepsV3sNameAndGracePeriod(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+unnamedCheckInJSON+`}`))
	out, err := runCLI(t, v3Env(s), "check-ins", "get", "--project-id", "Xk9mZp",
		"--id", "Ci0nUl", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"name": null`)
	assert.Contains(t, out, `"grace_period": "00:00:00"`)
}
