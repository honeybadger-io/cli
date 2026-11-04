package cmd

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ingestionKeyJSON = `{"id":"Kp7mZq2x","project_id":"Xk9mZp","key":"hbp_aBcDeFgHiJkLmNoPqRsTuVwXyZ012345aB12Cd","label":"CI","created_at":"2026-01-02T03:04:05Z"}`

const ingestionKeyBody = `{"data":` + ingestionKeyJSON + `}`

const ingestionKeysListBody = `{"data":[` + ingestionKeyJSON + `,{"id":"Kq8nAr3y","project_id":"Xk9mZp","key":"hbp_zYxWvUtSrQpOnMlKjIhGfEdCbA543210Zy98Xw","label":null,"created_at":"2026-01-01T02:03:04Z"}],"pagination":{"page":1,"per_page":100},"links":{"self":"/v3/projects/Xk9mZp/ingestion_keys","next":null}}`

func TestIngestionKeysList(t *testing.T) {
	s := newV3Server(t, ok(ingestionKeysListBody))
	out, err := runCLI(t, v3Env(s), "ingestion-keys", "list", "--project-id", "Xk9mZp")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/ingestion_keys", req.Path)
	assert.Equal(t, "Bearer hbu_test", req.Auth)
	assertQuery(t, url.Values{"page": {"1"}, "per_page": {"100"}}, req.Query)
	assert.Contains(t, out, "ID        LABEL  KEY")
	assert.Contains(
		t,
		out,
		"Kp7mZq2x  CI     hbp_aBcDeFgHiJkLmNoPqRsTuVwXyZ012345aB12Cd  2026-01-02 03:04",
	)
	assert.Contains(t, out, "Kq8nAr3y         hbp_zYxWvUtSrQpOnMlKjIhGfEdCbA543210Zy98Xw")
}

func TestIngestionKeysListJSONPrintsTheData(t *testing.T) {
	s := newV3Server(t, ok(ingestionKeysListBody))
	out, err := runCLI(t, v3Env(s), "ingestion-keys", "list", "--project-id", "Xk9mZp",
		"--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"id": "Kp7mZq2x"`)
	assert.Contains(t, out, `"label": null`)
	assert.NotContains(t, out, `"pagination"`)
}

func TestIngestionKeysListTakesTheProjectIDFromTheEnvironment(t *testing.T) {
	s := newV3Server(t, ok(ingestionKeysListBody))
	env := v3Env(s)
	env["HONEYBADGER_PROJECT_ID"] = "123"
	_, err := runCLI(t, env, "ingestion-keys", "list")
	require.NoError(t, err)
	assert.Equal(t, "/v3/projects/123/ingestion_keys", s.only(t).Path)
}

func TestIngestionKeysListRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestIngestionKeysListRequiresAnAPIToken(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, cliEnv{"HONEYBADGER_ENDPOINT": s.URL},
		"ingestion-keys", "list", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API Token is required")
	assert.Empty(t, s.Requests())
}

func TestIngestionKeysGet(t *testing.T) {
	s := newV3Server(t, ok(ingestionKeyBody))
	out, err := runCLI(t, v3Env(s), "ingestion-keys", "get", "--project-id", "Xk9mZp",
		"--id", "Kp7mZq2x")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/ingestion_keys/Kp7mZq2x", req.Path)
	assert.Contains(t, out, "ID: Kp7mZq2x")
	assert.Contains(t, out, "Project ID: Xk9mZp")
	assert.Contains(t, out, "Label: CI")
	assert.Contains(t, out, "Key: hbp_aBcDeFgHiJkLmNoPqRsTuVwXyZ012345aB12Cd")
	assert.Contains(t, out, "Created: 2026-01-02 03:04:05")
}

func TestIngestionKeysGetJSONPrintsTheKey(t *testing.T) {
	s := newV3Server(t, ok(ingestionKeyBody))
	out, err := runCLI(t, v3Env(s), "ingestion-keys", "get", "--project-id", "Xk9mZp",
		"--id", "Kp7mZq2x", "--output", "json")
	require.NoError(t, err)
	assert.Contains(t, out, `"key": "hbp_aBcDeFgHiJkLmNoPqRsTuVwXyZ012345aB12Cd"`)
}

func TestIngestionKeysGetRequiresAnID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "get", "--project-id", "Xk9mZp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required flag(s) "id" not set`)
	assert.Empty(t, s.Requests())
}

func TestIngestionKeysCreateSendsTheFlatBody(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, ingestionKeyBody))
	out, err := runCLI(t, v3Env(s), "ingestion-keys", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"label":"CI"}`)
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/ingestion_keys", req.Path)
	assertJSONBody(t, `{"label":"CI"}`, req.Body)
	assert.Contains(t, out, "Ingestion Key created successfully!")
	assert.Contains(t, out, "ID: Kp7mZq2x")
	assert.Contains(t, out, "Key: hbp_aBcDeFgHiJkLmNoPqRsTuVwXyZ012345aB12Cd")
}

func TestIngestionKeysCreateJSONPrintsTheKey(t *testing.T) {
	s := newV3Server(t, status(http.StatusCreated, ingestionKeyBody))
	out, err := runCLI(t, v3Env(s), "ingestion-keys", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{}`, "--output", "json")
	require.NoError(t, err)
	assertJSONBody(t, `{}`, s.only(t).Body)
	assert.Contains(t, out, `"id": "Kp7mZq2x"`)
}

func TestIngestionKeysCreateRefusesTheWrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"ingestion_key":{"label":"CI"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(),
		`v3 takes the fields directly: remove the outer "ingestion_key" key`)
	assert.Empty(t, s.Requests())
}

func TestIngestionKeysCreateRefusesTheKeyValue(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "create", "--project-id", "Xk9mZp",
		"--cli-input-json", `{"label":"CI","key":"hbp_mine"}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "key"`)
	assert.Empty(t, s.Requests())
}

func TestIngestionKeysCreateRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "create", "--cli-input-json", `{}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}

func TestIngestionKeysUpdateSendsOnlyTheGivenFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"rename", `{"label":"Deploys"}`},
		{"clear the label", `{"label":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(ingestionKeyBody))
			out, err := runCLI(t, v3Env(s), "ingestion-keys", "update", "--project-id", "Xk9mZp",
				"--id", "Kp7mZq2x", "--cli-input-json", tt.input)
			require.NoError(t, err)
			req := s.only(t)
			assert.Equal(t, "PATCH", req.Method)
			assert.Equal(t, "/v3/projects/Xk9mZp/ingestion_keys/Kp7mZq2x", req.Path)
			assertJSONBody(t, tt.input, req.Body)
			assert.Contains(t, out, "Ingestion Key updated successfully")
		})
	}
}

func TestIngestionKeysUpdateRefusesTheWrapper(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "update", "--project-id", "Xk9mZp",
		"--id", "Kp7mZq2x", "--cli-input-json", `{"ingestion_key":{"label":"CI"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `remove the outer "ingestion_key" key`)
	assert.Empty(t, s.Requests())
}

func TestIngestionKeysDelete(t *testing.T) {
	s := newV3Server(t, status(http.StatusNoContent, ""))
	out, err := runCLI(t, v3Env(s), "ingestion-keys", "delete", "--project-id", "Xk9mZp",
		"--id", "Kp7mZq2x")
	require.NoError(t, err)
	req := s.only(t)
	assert.Equal(t, "DELETE", req.Method)
	assert.Equal(t, "/v3/projects/Xk9mZp/ingestion_keys/Kp7mZq2x", req.Path)
	assert.Contains(t, out, "Ingestion Key deleted successfully")
}

func TestIngestionKeysDeleteReportsTheLastKeyRefusal(t *testing.T) {
	s := newV3Server(t, status(
		http.StatusUnprocessableEntity,
		`{"error":{"code":"validation_error","message":"A project's last ingestion key can't be deleted"}}`,
	))
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "delete", "--project-id", "Xk9mZp",
		"--id", "Kp7mZq2x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete Ingestion Key")
	assert.Contains(t, err.Error(), "last ingestion key")
}

func TestIngestionKeysDeleteRequiresAProjectID(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "ingestion-keys", "delete", "--id", "Kp7mZq2x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project ID is required")
	assert.Empty(t, s.Requests())
}
