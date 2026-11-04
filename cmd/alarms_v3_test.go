package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const alarmJSON = `{"id":"Qx7LmP2vRt9K","project_id":"Xk9mZp","name":"High Error Rate","description":"Alert when errors spike","state":"alarm","query":"filter event_type::str == \"notice\" | stats count() as count","stream_ids":["St1aBc"],"evaluation_period":"10m","lookback_lag":"1m","trigger_config":{"type":"alert_result_count","config":{"operator":"gt","value":10}},"error":null,"last_checked_at":"2026-01-03T04:05:06Z","next_check_at":"2026-01-03T04:15:06Z","created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T05:06:07Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/insights/alarms/Qx7LmP2vRt9K"}}`

const newAlarmJSON = `{"id":"Rt9KQx7LmP2v","project_id":"Xk9mZp","name":"Slow Requests","description":null,"state":"initial","query":"filter duration::int > 1000","stream_ids":null,"evaluation_period":"5m","lookback_lag":"1m","trigger_config":null,"error":null,"last_checked_at":null,"next_check_at":null,"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/insights/alarms/Rt9KQx7LmP2v"}}`

const alarmBody = `{"data":` + alarmJSON + `}`

const alarmsListBody = `{"data":[` + alarmJSON + `,` + newAlarmJSON + `]}`

const alarmHistoryBody = `{"data":[{"id":"Tr1gGr","alarm_id":"Qx7LmP2vRt9K","alarm_version_id":"Av1sIo","status":"alarm","created_at":"2026-01-03T04:05:06Z","evaluation_started_at":"2026-01-03T03:55:06Z","evaluation_result":12},{"id":"Tr2gGr","alarm_id":"Qx7LmP2vRt9K","alarm_version_id":"Av1sIo","status":null,"created_at":"2026-01-03T03:05:06Z","evaluation_started_at":"2026-01-03T02:55:06Z","evaluation_result":null}],"pagination":{"page":2,"per_page":25},"links":{"self":"/v3/projects/Xk9mZp/alarms/Qx7LmP2vRt9K/history?page=2","next":null}}`

func TestAlarmsList(t *testing.T) {
	s := newV3Server(t, ok(alarmsListBody))
	out, err := runCLI(t, v3Env(s), "alarms", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/alarms", req.Path)
	assert.Empty(t, req.Query)
	assert.Contains(t, out, "ID            NAME             STATE    QUERY")
	assert.Contains(t, out, "EVAL PERIOD  LAST CHECKED")
	assert.Contains(
		t,
		out,
		`Qx7LmP2vRt9K  High Error Rate  alarm    filter event_type::str == "notice" | stats coun...  10m          2026-01-03 04:05`,
	)
	assert.Contains(
		t,
		out,
		"Rt9KQx7LmP2v  Slow Requests    initial  filter duration::int > 1000                         5m           Never",
	)
}

func TestAlarmsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(alarmsListBody))
	out, err := runCLI(t, v3Env(s), "alarms", "list", "--project-id", "Xk9mZp", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Qx7LmP2vRt9K"`)
	assert.Contains(t, out, `"evaluation_period": "10m"`)
	assert.NotContains(t, out, `"data"`)
}

func TestAlarmsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(alarmsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "alarms", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/alarms", s.only(t).Path)
}

func TestAlarmsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestAlarmsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"alarms", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestAlarmsGet(t *testing.T) {
	s := newV3Server(t, ok(alarmBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"alarms",
		"get",
		"--project-id",
		"Xk9mZp",
		"--id",
		"Qx7LmP2vRt9K",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/alarms/Qx7LmP2vRt9K", req.Path)
	assert.Contains(t, out, "ID: Qx7LmP2vRt9K")
	assert.Contains(t, out, "Name: High Error Rate")
	assert.Contains(t, out, "Description: Alert when errors spike")
	assert.Contains(t, out, "State: alarm")
	assert.Contains(t, out, `Query: filter event_type::str == "notice" | stats count() as count`)
	assert.Contains(t, out, "Stream IDs: St1aBc")
	assert.Contains(t, out, "Evaluation Period: 10m")
	assert.Contains(t, out, "Lookback Lag: 1m")
	assert.Contains(t, out,
		`Trigger Config: {"config":{"operator":"gt","value":10},"type":"alert_result_count"}`)
	assert.Contains(t, out, "Last Checked: 2026-01-03 04:05:06")
	assert.Contains(t, out, "Next Check: 2026-01-03 04:15:06")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
	assert.Contains(t, out, "Updated: 2026-01-02 05:06:07")
	assert.Contains(
		t,
		out,
		"URL: https://app.honeybadger.io/projects/Xk9mZp/insights/alarms/Qx7LmP2vRt9K",
	)
	assert.NotContains(t, out, "Error:")
}

func TestAlarmsGetLeavesOutNullFields(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+newAlarmJSON+`}`))
	out, err := runCLI(
		t,
		v3Env(s),
		"alarms",
		"get",
		"--project-id",
		"Xk9mZp",
		"--id",
		"Rt9KQx7LmP2v",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "State: initial")
	assert.NotContains(t, out, "Description")
	assert.NotContains(t, out, "Stream IDs")
	assert.NotContains(t, out, "Trigger Config")
	assert.NotContains(t, out, "Last Checked")
}

func TestAlarmsGetJSONPrintsTheAlarm(t *testing.T) {
	s := newV3Server(t, ok(alarmBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"alarms",
		"get",
		"--project-id",
		"Xk9mZp",
		"--id",
		"Qx7LmP2vRt9K",
		"--output",
		"json",
	)
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Qx7LmP2vRt9K"`)
	assert.Contains(t, out, `"lookback_lag": "1m"`)
}

func TestAlarmsGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "get", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestAlarmsCreateSendsTheFlatBody(t *testing.T) {
	input := `{"name":"High Error Rate","description":"Alert when errors spike","query":"filter event_type::str == \"notice\"","evaluation_period":"5m","lookback_lag":"1m","trigger_config":{"type":"alert_result_count","config":{"operator":"gt","value":10}}}`
	s := newV3Server(t, status(http.StatusCreated, alarmBody))
	out, err := runCLI(t, v3Env(s), "alarms", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", input)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/alarms", req.Path)
	assertJSONBody(t, input, req.Body)
	assert.Contains(t, out, "Alarm created successfully!")
	assert.Contains(t, out, "ID: Qx7LmP2vRt9K")
	assert.Contains(t, out, "Name: High Error Rate")
	assert.Contains(t, out, "State: alarm")
}

func TestAlarmsCreateJSONPrintsTheAlarm(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, alarmBody))
	out, err := runCLI(t, v3Env(s), "alarms", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"name":"x"}`, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Qx7LmP2vRt9K"`)
}

func TestAlarmsCreateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"alarm":{"name":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `v3 takes the fields directly: remove the outer "alarm" key`)
	assert.Empty(t, s.Requests())
}

func TestAlarmsCreateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "create", "--cli-input-json", `{"name":"x"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestAlarmsUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"one field", `{"name":"Renamed"}`},
		{
			"zero threshold",
			`{"trigger_config":{"type":"alert_result_count","config":{"operator":"eq","value":0}}}`,
		},
		{"clear the description", `{"description":null}`},
		{"reset the streams", `{"stream_ids":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(alarmBody))
			out, err := runCLI(t, v3Env(s), "alarms", "update", "--project-id", "Xk9mZp",
				"--id", "Qx7LmP2vRt9K", "--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/alarms/Qx7LmP2vRt9K", req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Alarm updated successfully")
		})
	}
}

func TestAlarmsUpdateHelpSaysATriggerIsReplaced(t *testing.T) {
	assert.Contains(t, alarmsUpdateCmd.Long, "replaces the stored one")
}

func TestAlarmsUpdateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "update", "--project-id", "Xk9mZp",
		"--id", "Qx7LmP2vRt9K", "--cli-input-json", `{"alarm":{"name":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "alarm" key`)
	assert.Empty(t, s.Requests())
}

func TestAlarmsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "alarms", "delete", "--project-id", "Xk9mZp",
		"--id", "Qx7LmP2vRt9K")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/alarms/Qx7LmP2vRt9K", req.Path)
	assert.Contains(t, out, "Alarm deleted successfully")
}

func TestAlarmsDeleteRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "delete", "--id", "Qx7LmP2vRt9K")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestAlarmsHistory(t *testing.T) {
	s := newV3Server(t, ok(alarmHistoryBody))
	out, err := runCLI(t, v3Env(s), "alarms", "history", "--project-id", "Xk9mZp",
		"--id", "Qx7LmP2vRt9K")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/alarms/Qx7LmP2vRt9K/history", req.Path)
	assert.Empty(t, req.Query)
	assert.Contains(t, out, "ID      STATE  CREATED AT")
	assert.Contains(t, out, "Tr1gGr  alarm  2026-01-03 04:05:06")
	assert.Contains(t, out, "Tr2gGr         2026-01-03 03:05:06")
}

func TestAlarmsHistorySendsThePage(t *testing.T) {
	s := newV3Server(t, ok(alarmHistoryBody))
	_, err := runCLI(t, v3Env(s), "alarms", "history", "--project-id", "Xk9mZp",
		"--id", "Qx7LmP2vRt9K", "--page", "2")
	require.NoError(t, err)
	assertQuery(t, url.Values{"page": {"2"}}, s.only(t).Query)
}

func TestAlarmsHistoryJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(alarmHistoryBody))
	out, err := runCLI(t, v3Env(s), "alarms", "history", "--project-id", "Xk9mZp",
		"--id", "Qx7LmP2vRt9K", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Tr1gGr"`)
	assert.Contains(t, out, `"evaluation_result": 12`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestAlarmsHistoryRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "history", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}
