package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const siteID = "9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47"

const siteJSON = `{"id":"` + siteID + `","project_id":"Xk9mZp","name":"Production App","url":"https://app.example.com","active":true,"links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/sites/` + siteID + `"},"state":"up","last_checked_at":"2026-01-03T04:05:06Z","match_type":"include","match":"OK","request_method":"GET","frequency":5,"locations":["Virginia","Oregon"],"validate_ssl":true,"request_headers":{},"request_body":null,"outage_threshold":null,"timeout":null,"created_at":"2026-01-02T03:04:05Z"}`

const pausedSiteJSON = `{"id":"3d7a1c94-6f28-4e5b-8a13-c5e9b204d6f8","project_id":"Xk9mZp","name":"Staging","url":"https://staging.example.com","active":false,"links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/sites/3d7a1c94-6f28-4e5b-8a13-c5e9b204d6f8"},"state":"pending","last_checked_at":null,"match_type":"success","match":null,"request_method":"GET","frequency":15,"locations":[],"validate_ssl":true,"request_headers":{},"request_body":null,"outage_threshold":null,"timeout":null,"created_at":"2026-01-02T03:04:05Z"}`

const siteBody = `{"data":` + siteJSON + `}`

const sitesListBody = `{"data":[` + siteJSON + `,` + pausedSiteJSON + `],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/sites","next":null}}`

const outagesBody = `{"data":[{"id":"0b6c1d2e-3f40-4a5b-8c6d-7e8f9a0b1c2d","site_id":"` + siteID + `","down_at":"2026-01-03T04:05:06Z","up_at":"2026-01-03T04:15:06Z","status":503,"reason":"Service Unavailable from the load balancer in front of the app","headers":{"content-type":"text/html"},"duration_seconds":600,"created_at":"2026-01-03T04:05:06Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/sites/` + siteID + `"}},{"id":"1c7d2e3f-4051-4b6c-9d7e-8f9a0b1c2d3e","site_id":"` + siteID + `","down_at":"2026-01-04T04:05:06Z","up_at":null,"status":null,"reason":null,"headers":{},"duration_seconds":null,"created_at":"2026-01-04T04:05:06Z","links":{"web":"https://app.honeybadger.io/projects/Xk9mZp/sites/` + siteID + `"}}],"pagination":{"limit":25,"has_older":false,"has_newer":false},"links":{"self":"/v3/projects/Xk9mZp/sites/` + siteID + `/outages","older":null,"newer":null}}`

const uptimeChecksBody = `{"data":[{"created_at":"2026-01-03T04:05:06Z","duration":123,"location":"Virginia","up":true},{"created_at":"2026-01-03T04:00:06Z","duration":null,"location":"Oregon","up":false}],"pagination":{"limit":25,"has_older":false,"has_newer":false},"links":{"self":"/v3/projects/Xk9mZp/sites/` + siteID + `/checks","older":null,"newer":null}}`

func TestUptimeSitesList(t *testing.T) {
	s := newV3Server(t, ok(sitesListBody))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/sites", req.Path)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID                                    NAME            URL")
	assert.Contains(
		t,
		out,
		siteID+"  Production App  https://app.example.com      up       Yes     5m",
	)
	assert.Contains(
		t,
		out,
		"3d7a1c94-6f28-4e5b-8a13-c5e9b204d6f8  Staging         https://staging.example.com  pending          15m",
	)
}

func TestUptimeSitesListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(sitesListBody))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "list", "--project-id", "Xk9mZp",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "`+siteID+`"`)
	assert.Contains(t, out, `"match_type": "include"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestUptimeSitesListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(sitesListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "uptime", "sites", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/sites", s.only(t).Path)
}

func TestUptimeSitesListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "uptime", "sites", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestUptimeSitesListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"uptime", "sites", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestUptimeSitesGet(t *testing.T) {
	s := newV3Server(t, ok(siteBody))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "get", "--project-id", "Xk9mZp",
		"--site-id", siteID)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/sites/"+siteID, req.Path)
	assert.Contains(t, out, "ID: "+siteID)
	assert.Contains(t, out, "Name: Production App")
	assert.Contains(t, out, "URL: https://app.example.com")
	assert.Contains(t, out, "State: up")
	assert.Contains(t, out, "Active: true")
	assert.Contains(t, out, "Frequency: 5 minutes")
	assert.Contains(t, out, "Match Type: include")
	assert.Contains(t, out, "Match: OK")
	assert.Contains(t, out, "Last Checked: 2026-01-03 04:05:06")
}

func TestUptimeSitesGetLeavesOutNullFields(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+pausedSiteJSON+`}`))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "get", "--project-id", "Xk9mZp",
		"--site-id", "3d7a1c94-6f28-4e5b-8a13-c5e9b204d6f8")
	require.NoError(t, err)
	assert.Contains(t, out, "Active: false")
	assert.NotContains(t, out, "Match:")
	assert.NotContains(t, out, "Last Checked")
}

func TestUptimeSitesGetJSONPrintsTheSite(t *testing.T) {
	s := newV3Server(t, ok(siteBody))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "get", "--project-id", "Xk9mZp",
		"--site-id", siteID, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "`+siteID+`"`)
	assert.Contains(t, out, `"frequency": 5`)
}

func TestUptimeRefusesASiteIDThatIsNotAUUID(t *testing.T) {
	commands := [][]string{
		{"uptime", "sites", "get"},
		{"uptime", "sites", "update", "--cli-input-json", `{"name":"x"}`},
		{"uptime", "sites", "delete"},
		{"uptime", "outages"},
		{"uptime", "checks"},
	}
	for _, args := range commands {
		t.Run(args[1], func(t *testing.T) {
			s := newV3Server(t)
			args = append(args, "--project-id", "Xk9mZp", "--site-id", "not-a-uuid")
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid --site-id: must be a UUID")
			assert.Empty(t, s.Requests())
		})
	}
}

func TestUptimeSitesCreateSendsTheFlatBody(t *testing.T) {
	input := `{"name":"Production App","url":"https://app.example.com","frequency":5,"match_type":"include","match":"OK","locations":["Virginia","Oregon"],"validate_ssl":true}`
	s := newV3Server(t, status(http.StatusCreated, siteBody))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", input)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/sites", req.Path)
	assertJSONBody(t, input, req.Body)
	assert.Contains(t, out, "Uptime site created successfully!")
	assert.Contains(t, out, "ID: "+siteID)
	assert.Contains(t, out, "Name: Production App")
	assert.Contains(t, out, "URL: https://app.example.com")
}

func TestUptimeSitesCreateJSONPrintsTheSite(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, siteBody))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"url":"https://app.example.com"}`, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "`+siteID+`"`)
}

func TestUptimeSitesCreateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "uptime", "sites", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"site":{"url":"https://x.example.com"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `v3 takes the fields directly: remove the outer "site" key`)
	assert.Empty(t, s.Requests())
}

func TestUptimeSitesCreateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "uptime", "sites", "create",
		"--cli-input-json", `{"url":"https://x.example.com"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestUptimeSitesUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"one field", `{"name":"Renamed"}`},
		{"pause monitoring", `{"active":false}`},
		{"clear the match", `{"match_type":"success","match":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(siteBody))
			out, err := runCLI(t, v3Env(s), "uptime", "sites", "update", "--project-id", "Xk9mZp",
				"--site-id", siteID, "--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/sites/"+siteID, req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Uptime site updated successfully!")
			assert.Contains(t, out, "ID: "+siteID)
			assert.Contains(t, out, "Name: Production App")
		})
	}
}

func TestUptimeSitesUpdateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "uptime", "sites", "update", "--project-id", "Xk9mZp",
		"--site-id", siteID, "--cli-input-json", `{"site":{"name":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "site" key`)
	assert.Empty(t, s.Requests())
}

func TestUptimeSitesDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "uptime", "sites", "delete", "--project-id", "Xk9mZp",
		"--site-id", siteID)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/sites/"+siteID, req.Path)
	assert.Contains(t, out, "Uptime site deleted successfully")
}

func TestUptimeOutages(t *testing.T) {
	s := newV3Server(t, ok(outagesBody))
	out, err := runCLI(t, v3Env(s), "uptime", "outages", "--project-id", "Xk9mZp",
		"--site-id", siteID)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/sites/"+siteID+"/outages", req.Path)
	assertQuery(t, url.Values{"limit": {"25"}}, req.Query)
	assert.Contains(t, out, "DOWN AT           UP AT             STATUS  REASON")
	assert.Contains(t, out,
		"2026-01-03 04:05  2026-01-03 04:15  503     Service Unavailable from the load bal...")
	assert.Contains(t, out, "2026-01-04 04:05  Still down")
}

func TestUptimeOutagesSendsLimitAndCreatedBefore(t *testing.T) {
	s := newV3Server(t, ok(outagesBody))
	_, err := runCLI(t, v3Env(s), "uptime", "outages", "--project-id", "Xk9mZp",
		"--site-id", siteID, "--limit", "100", "--created-before", "2024-01-02")
	require.NoError(t, err)
	assertQuery(t, url.Values{"limit": {"100"}, "created_before": {"1704153600"}}, s.only(t).Query)
}

func TestUptimeOutagesJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(outagesBody))
	out, err := runCLI(t, v3Env(s), "uptime", "outages", "--project-id", "Xk9mZp",
		"--site-id", siteID, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"status": 503`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestUptimeChecks(t *testing.T) {
	s := newV3Server(t, ok(uptimeChecksBody))
	out, err := runCLI(t, v3Env(s), "uptime", "checks", "--project-id", "Xk9mZp",
		"--site-id", siteID)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/sites/"+siteID+"/checks", req.Path)
	assertQuery(t, url.Values{"limit": {"25"}}, req.Query)
	assert.Contains(t, out, "CREATED              LOCATION  UP   DURATION")
	assert.Contains(t, out, "2026-01-03 04:05:06  Virginia  Yes  123ms")
	assert.Contains(t, out, "2026-01-03 04:00:06  Oregon    No")
}

func TestUptimeChecksSendsLimitAndCreatedBefore(t *testing.T) {
	s := newV3Server(t, ok(uptimeChecksBody))
	_, err := runCLI(t, v3Env(s), "uptime", "checks", "--project-id", "Xk9mZp",
		"--site-id", siteID, "--limit", "50", "--created-before", "2024-01-02T00:00:00Z")
	require.NoError(t, err)
	assertQuery(t, url.Values{"limit": {"50"}, "created_before": {"1704153600"}}, s.only(t).Query)
}

func TestUptimeChecksJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(uptimeChecksBody))
	out, err := runCLI(t, v3Env(s), "uptime", "checks", "--project-id", "Xk9mZp",
		"--site-id", siteID, "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"duration": 123`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestUptimeOutagesAndChecksRefuseABadCreatedBefore(t *testing.T) {
	for _, sub := range []string{"outages", "checks"} {
		t.Run(sub, func(t *testing.T) {
			s := newV3Server(t)
			_, err := runCLI(t, v3Env(s), "uptime", sub, "--project-id", "Xk9mZp",
				"--site-id", siteID, "--created-before", "yesterday")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid --created-before")
			assert.Empty(t, s.Requests())
		})
	}
}

func TestUptimeOutagesAndChecksHaveNoCreatedAfter(t *testing.T) {
	for _, sub := range []string{"outages", "checks"} {
		t.Run(sub, func(t *testing.T) {
			s := newV3Server(t)
			_, err := runCLI(t, v3Env(s), "uptime", sub, "--project-id", "Xk9mZp",
				"--site-id", siteID, "--created-after", "2024-01-01")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown flag: --created-after")
			assert.Empty(t, s.Requests())
		})
	}
}
