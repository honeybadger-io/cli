package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const statusPageJSON = `{"id":"Sp2wQx","account_id":"Ab3kL9","name":"Acme Status","url":"https://status.example.com","features":{},"hide_branding":false,"password_protected":false,"username":null,"links":{"web":"https://app.honeybadger.io/accounts/Ab3kL9/status_pages/Sp2wQx"},"domain":"status.example.com","created_at":"2026-01-02T03:04:05Z","domain_verified_at":"2026-01-03T04:05:06Z","sites":[{"site_id":"9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47","display_name":"Website","description":null}],"check_ins":[{"check_in_id":"Ci5xWq","display_name":"Nightly Backup","description":"Runs at 2am"}],"incidents_enabled":true,"search_engine_indexing_disabled":false,"message_enabled":false,"message_title":null,"message_body":null,"google_analytics_id":null}`

const bareStatusPageJSON = `{"id":"Sp3xRy","account_id":"Ab3kL9","name":"Internal","url":"https://hb-status.example.com/Sp3xRy","features":{},"hide_branding":false,"password_protected":false,"username":null,"links":{"web":"https://app.honeybadger.io/accounts/Ab3kL9/status_pages/Sp3xRy"},"domain":null,"created_at":"2026-01-02T03:04:05Z","domain_verified_at":null,"sites":[],"check_ins":[],"incidents_enabled":false,"search_engine_indexing_disabled":false,"message_enabled":false,"message_title":null,"message_body":null,"google_analytics_id":null}`

const statusPageBody = `{"data":` + statusPageJSON + `}`

const statusPagesListBody = `{"data":[` + statusPageJSON + `,` + bareStatusPageJSON + `],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/status_pages","next":null}}`

func TestStatusPagesList(t *testing.T) {
	s := newV3Server(t, ok(statusPagesListBody))
	out, err := runCLI(t, v3Env(s), "statuspages", "list")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/status_pages", req.Path)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(
		t,
		out,
		"ID      NAME         URL                                   SITES  CHECK-INS",
	)
	assert.Contains(t, out, "Sp2wQx  Acme Status  https://status.example.com            1      1")
	assert.Contains(t, out, "Sp3xRy  Internal     https://hb-status.example.com/Sp3xRy  0      0")
}

func TestStatusPagesListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(statusPagesListBody))
	out, err := runCLI(t, v3Env(s), "statuspages", "list", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Sp2wQx"`)
	assert.Contains(t, out, `"incidents_enabled": true`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestStatusPagesListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL}, "statuspages", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestStatusPagesHaveNoAccountIDFlag(t *testing.T) {
	commands := [][]string{
		{"statuspages", "list"},
		{"statuspages", "get", "--id", "Sp2wQx"},
		{"statuspages", "create", "--cli-input-json", `{"name":"x"}`},
		{"statuspages", "update", "--id", "Sp2wQx", "--cli-input-json", `{"name":"x"}`},
		{"statuspages", "delete", "--id", "Sp2wQx"},
	}
	for _, args := range commands {
		t.Run(args[1], func(t *testing.T) {
			s := newV3Server(t)
			_, err := runCLI(t, v3Env(s), append(args, "--account-id", "Ab3kL9")...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unknown flag: --account-id")
			assert.Empty(t, s.Requests())
		})
	}
}

func TestStatusPagesGet(t *testing.T) {
	s := newV3Server(t, ok(statusPageBody))
	out, err := runCLI(t, v3Env(s), "statuspages", "get", "--id", "Sp2wQx")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/status_pages/Sp2wQx", req.Path)
	assert.Contains(t, out, "ID: Sp2wQx")
	assert.Contains(t, out, "Name: Acme Status")
	assert.Contains(t, out, "URL: https://status.example.com")
	assert.Contains(t, out, "Account ID: Ab3kL9")
	assert.Contains(t, out, "Domain: status.example.com")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
	assert.Contains(t, out, "Domain Verified: 2026-01-03 04:05:06")
	assert.Contains(t, out, "Sites:\n    - Website (9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47)")
	assert.Contains(t, out, "Check-ins:\n    - Nightly Backup (Ci5xWq)")
}

func TestStatusPagesGetLeavesOutNullFields(t *testing.T) {
	s := newV3Server(t, ok(`{"data":`+bareStatusPageJSON+`}`))
	out, err := runCLI(t, v3Env(s), "statuspages", "get", "--id", "Sp3xRy")
	require.NoError(t, err)
	assert.Contains(t, out, "Name: Internal")
	assert.NotContains(t, out, "Domain")
	assert.NotContains(t, out, "Sites:")
	assert.NotContains(t, out, "Check-ins:")
}

func TestStatusPagesGetJSONPrintsTheStatusPage(t *testing.T) {
	s := newV3Server(t, ok(statusPageBody))
	out, err := runCLI(t, v3Env(s), "statuspages", "get", "--id", "Sp2wQx", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Sp2wQx"`)
	assert.Contains(t, out, `"check_in_id": "Ci5xWq"`)
}

func TestStatusPagesGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "statuspages", "get")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestStatusPagesCreateSendsTheFlatBody(t *testing.T) {
	input := `{"name":"Acme Status","domain":"status.example.com","sites":[{"site_id":"9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47","display_name":"Website"}],"check_ins":[{"check_in_id":"Ci5xWq"}],"hide_branding":false}`
	s := newV3Server(t, status(http.StatusCreated, statusPageBody))
	out, err := runCLI(t, v3Env(s), "statuspages", "create", "--cli-input-json", input)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/status_pages", req.Path)
	assertJSONBody(t, input, req.Body)
	assert.Contains(t, out, "Status page created successfully!")
	assert.Contains(t, out, "ID: Sp2wQx")
	assert.Contains(t, out, "Name: Acme Status")
	assert.Contains(t, out, "URL: https://status.example.com")
}

func TestStatusPagesCreateJSONPrintsTheStatusPage(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, statusPageBody))
	out, err := runCLI(t, v3Env(s), "statuspages", "create", "--cli-input-json", `{"name":"x"}`,
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Sp2wQx"`)
}

func TestStatusPagesCreateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "statuspages", "create",
		"--cli-input-json", `{"status_page":{"name":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(),
		`v3 takes the fields directly: remove the outer "status_page" key`)
	assert.Empty(t, s.Requests())
}

func TestStatusPagesCreateRefusesV2SiteIDStrings(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "statuspages", "create",
		"--cli-input-json", `{"name":"x","sites":["site-id-1"]}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON payload")
	assert.Empty(t, s.Requests())
}

func TestStatusPagesUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"one field", `{"name":"Renamed"}`},
		{"turn the banner off", `{"message_enabled":false}`},
		{"remove every site", `{"sites":[]}`},
		{"clear the domain", `{"domain":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(statusPageBody))
			out, err := runCLI(t, v3Env(s), "statuspages", "update", "--id", "Sp2wQx",
				"--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/status_pages/Sp2wQx", req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Status page updated successfully")
		})
	}
}

func TestStatusPagesUpdateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "statuspages", "update", "--id", "Sp2wQx",
		"--cli-input-json", `{"status_page":{"name":"x"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "status_page" key`)
	assert.Empty(t, s.Requests())
}

func TestStatusPagesDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "statuspages", "delete", "--id", "Sp2wQx")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/status_pages/Sp2wQx", req.Path)
	assert.Contains(t, out, "Status page deleted successfully")
}
