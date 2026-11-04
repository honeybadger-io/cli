package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const environmentJSON = `{"id":"En6tRm","project_id":"Xk9mZp","name":"production","notifications":true,"created_at":"2026-01-02T03:04:05Z"}`

const environmentBody = `{"data":` + environmentJSON + `}`

const environmentsListBody = `{"data":[` + environmentJSON + `,{"id":"En7uSn","project_id":"Xk9mZp","name":"staging","notifications":false,"created_at":null}],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/environments","next":null}}`

func TestEnvironmentsList(t *testing.T) {
	s := newV3Server(t, ok(environmentsListBody))
	out, err := runCLI(t, v3Env(s), "environments", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/environments", req.Path)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID      NAME        NOTIFICATIONS  CREATED")
	assert.Contains(t, out, "En6tRm  production  Yes            2026-01-02 03:04")
	assert.Contains(t, out, "En7uSn  staging     No")
}

func TestEnvironmentsListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(environmentsListBody))
	out, err := runCLI(t, v3Env(s), "environments", "list", "--project-id", "Xk9mZp",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "En6tRm"`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestEnvironmentsListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(environmentsListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "environments", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/environments", s.only(t).Path)
}

func TestEnvironmentsListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "environments", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestEnvironmentsListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"environments", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestEnvironmentsGet(t *testing.T) {
	s := newV3Server(t, ok(environmentBody))
	out, err := runCLI(
		t,
		v3Env(s),
		"environments",
		"get",
		"--project-id",
		"Xk9mZp",
		"--id",
		"En6tRm",
	)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/environments/En6tRm", req.Path)
	assert.Contains(t, out, "ID: En6tRm")
	assert.Contains(t, out, "Name: production")
	assert.Contains(t, out, "Project ID: Xk9mZp")
	assert.Contains(t, out, "Notifications: true")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
}

func TestEnvironmentsGetAcceptsALegacyNumericID(t *testing.T) {
	s := newV3Server(t, ok(environmentBody))
	_, err := runCLI(t, v3Env(s), "environments", "get", "--project-id", "123", "--id", "456")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/environments/456", s.only(t).Path)
}

func TestEnvironmentsGetJSONPrintsTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(environmentBody))
	out, err := runCLI(t, v3Env(s), "environments", "get", "--project-id", "Xk9mZp",
		"--id", "En6tRm", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "En6tRm"`)
	assert.Contains(t, out, `"notifications": true`)
}

func TestEnvironmentsCreateSendsTheFlatBody(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, environmentBody))
	out, err := runCLI(t, v3Env(s), "environments", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"name":"production","notifications":true}`)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/environments", req.Path)
	assertJSONBody(t, `{"name":"production","notifications":true}`, req.Body)
	assert.Contains(t, out, "Environment created successfully!")
	assert.Contains(t, out, "ID: En6tRm")
	assert.Contains(t, out, "Name: production")
}

func TestEnvironmentsCreateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "environments", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"environment":{"name":"staging"}}`)
	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		`v3 takes the fields directly: remove the outer "environment" key`,
	)
	assert.Empty(t, s.Requests())
}

func TestEnvironmentsUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"turn notifications off", `{"notifications":false}`},
		{"reset notifications", `{"notifications":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(environmentBody))
			out, err := runCLI(t, v3Env(s), "environments", "update", "--project-id", "Xk9mZp",
				"--id", "En6tRm", "--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/environments/En6tRm", req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Environment updated successfully")
		})
	}
}

func TestEnvironmentsUpdateRefusesTheV2Wrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "environments", "update", "--project-id", "Xk9mZp",
		"--id", "En6tRm", "--cli-input-json", `{"environment":{"notifications":false}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "environment" key`)
	assert.Empty(t, s.Requests())
}

func TestEnvironmentsUpdateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "environments", "update", "--id", "En6tRm",
		"--cli-input-json", `{"notifications":false}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestEnvironmentsDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "environments", "delete", "--project-id", "Xk9mZp",
		"--id", "En6tRm")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/environments/En6tRm", req.Path)
	assert.Contains(t, out, "Environment deleted successfully")
}
