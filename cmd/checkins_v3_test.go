package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const checkInJSON = `{"id":"Ci5xWq","project_id":"Xk9mZp","name":"Daily Backup","slug":"daily-backup","schedule_type":"simple","state":"reporting","reported_at":"2026-01-03T04:05:06Z","expected_at":"2026-01-04T04:05:06Z","missed_count":0,"grace_period":"15 minutes","report_url":"https://api.honeybadger.io/v1/check_in/hbp_x/daily-backup","created_at":"2026-01-02T03:04:05Z","cron_schedule":null,"cron_timezone":"UTC","report_period":"1 day","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/check_ins/Ci5xWq"}}`

const cronCheckInJSON = `{"id":"Ci6yZr","project_id":"Xk9mZp","name":"Hourly Task","slug":"hourly-task","schedule_type":"cron","state":"pending","reported_at":null,"expected_at":null,"missed_count":0,"grace_period":"00:00:00","report_url":"https://api.honeybadger.io/v1/check_in/hbp_x/hourly-task","created_at":"2026-01-02T03:04:05Z","cron_schedule":"0 * * * *","cron_timezone":"Eastern Time (US & Canada)","report_period":null,"links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/check_ins/Ci6yZr"}}`

const checkInBody = `{"data":` + checkInJSON + `}`

const checkInsListBody = `{"data":[` + checkInJSON + `,` + cronCheckInJSON + `],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/check_ins","next":null}}`

func TestCheckInsList(t *testing.T) {
	s := newV3Server(t, ok(checkInsListBody))
	out, err := runCLI(t, v3Env(s), "check-ins", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/check_ins", req.Path)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID      NAME          SLUG          TYPE    SCHEDULE   LAST CHECK-IN")
	assert.Contains(
		t,
		out,
		"Ci5xWq  Daily Backup  daily-backup  simple  1 day      2026-01-03 04:05",
	)
	assert.Contains(t, out, "Ci6yZr  Hourly Task   hourly-task   cron    0 * * * *  Never")
}

func TestCheckInsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(checkInsListBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"check-ins",
		"list",
		"--project-id",
		"Xk9mZp",
		"--output",
		"json",
	)
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Ci5xWq"`)
	assert.Contains(t, out, `"report_period": "1 day"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestCheckInsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(checkInsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "check-ins", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/check_ins", s.only(t).Path)
}

func TestCheckInsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestCheckInsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"check-ins", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestCheckInsGet(t *testing.T) {
	s := newV3Server(t, ok(checkInBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"check-ins",
		"get",
		"--project-id",
		"Xk9mZp",
		"--id",
		"Ci5xWq",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/check_ins/Ci5xWq", req.Path)
	assert.Contains(t, out, "ID: Ci5xWq")
	assert.Contains(t, out, "Name: Daily Backup")
	assert.Contains(t, out, "Slug: daily-backup")
	assert.Contains(t, out, "State: reporting")
	assert.Contains(t, out, "Schedule Type: simple")
	assert.Contains(t, out, "Report Period: 1 day")
	assert.Contains(t, out, "Grace Period: 15 minutes")
	assert.Contains(t, out, "Last Check-in: 2026-01-03 04:05:06")
}

func TestCheckInsGetCronSchedule(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+cronCheckInJSON+`}`))
	out, err := runCLI(t, v3Env(s), "check-ins", "get", "--project-id", "Xk9mZp", "--id", "Ci6yZr")
	require.NoError(t, err)
	assert.Contains(t, out, "Cron Schedule: 0 * * * *")
	assert.Contains(t, out, "Cron Timezone: Eastern Time (US & Canada)")
	assert.NotContains(t, out, "Report Period")
	assert.NotContains(t, out, "Last Check-in")
}

func TestCheckInsGetJSONPrintsTheCheckIn(t *testing.T) {
	s := newV3Server(t, ok(checkInBody))
	out, err := runCLI(t, v3Env(s), "check-ins", "get", "--project-id", "Xk9mZp", "--id", "Ci5xWq",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Ci5xWq"`)
	assert.Contains(t, out, `"schedule_type": "simple"`)
}

func TestCheckInsGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "get", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestCheckInsCreateSendsTheFlatBody(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, checkInBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"check-ins",
		"create",
		"--project-id",
		"Xk9mZp",
		"--cli-input-json",
		`{"name":"Daily Backup","slug":"daily-backup","schedule_type":"simple","report_period":"1 day","grace_period":"15 minutes"}`,
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/check_ins", req.Path)
	assertJSONBody(
		t,
		`{"name":"Daily Backup","slug":"daily-backup","schedule_type":"simple","report_period":"1 day","grace_period":"15 minutes"}`,
		req.Body,
	)
	assert.Contains(t, out, "Check-in created successfully!")
	assert.Contains(t, out, "ID: Ci5xWq")
	assert.Contains(t, out, "Slug: daily-backup")
}

func TestCheckInsCreateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"check_in":{"name":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `v3 takes the fields directly: remove the outer "check_in" key`)
	assert.Empty(t, s.Requests())
}

func TestCheckInsCreateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "create", "--cli-input-json", `{"name":"x"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestCheckInsUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"one field", `{"name":"Nightly Backup"}`},
		{"zero grace period", `{"grace_period":"00:00:00"}`},
		{"clear the name", `{"name":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(checkInBody))
			out, err := runCLI(t, v3Env(s), "check-ins", "update", "--project-id", "Xk9mZp",
				"--id", "Ci5xWq", "--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/check_ins/Ci5xWq", req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Check-in updated successfully!")
			assert.Contains(t, out, "ID: Ci5xWq")
		})
	}
}

func TestCheckInsUpdateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "update", "--project-id", "Xk9mZp",
		"--id", "Ci5xWq", "--cli-input-json", `{"check_in":{"name":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "check_in" key`)
	assert.Empty(t, s.Requests())
}

func TestCheckInsUpdateJSONPrintsTheCheckIn(t *testing.T) {
	s := newV3Server(t, ok(checkInBody))
	out, err := runCLI(t, v3Env(s), "check-ins", "update", "--project-id", "Xk9mZp",
		"--id", "Ci5xWq", "--cli-input-json", `{"name":"x"}`, "--output", "json")
	require.NoError(t, err)
	s.only(t)
	assert.Contains(t, out, `"id": "Ci5xWq"`)
}

func TestCheckInsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(
		t,
		v3Env(s),
		"check-ins",
		"delete",
		"--project-id",
		"Xk9mZp",
		"--id",
		"Ci5xWq",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/check_ins/Ci5xWq", req.Path)
	assert.Contains(t, out, "Check-in deleted successfully")
}

func TestCheckInsDeleteRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "check-ins", "delete", "--id", "Ci5xWq")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}
