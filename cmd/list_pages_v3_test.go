package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// streams, check-ins and environments list used to print only the first page;
// they now read every page, in text output and in JSON.

// twoPages answers page 1 and page 2 of path, each with one item, page 1
// linking to page 2.
func twoPages(path, first, second string) map[string]v3Response {
	return map[string]v3Response{
		"GET " + path + "?page=1&per_page=100": ok(`{"data":[` + first +
			`],"pagination":{"page":1,"per_page":100},"links":{"self":"` + path +
			`","next":"` + path + `?page=2&per_page=100"}}`),
		"GET " + path + "?page=2&per_page=100": ok(`{"data":[` + second +
			`],"pagination":{"page":2,"per_page":100},"links":{"self":"` + path +
			`?page=2&per_page=100","next":null}}`),
	}
}

func TestListsReadEveryPage(t *testing.T) {
	tests := []struct {
		command           string
		routes            map[string]v3Response
		firstID, secondID string
	}{
		{
			command: "streams",
			routes: twoPages(
				"/v3/projects/Xk9mZp/streams",
				`{"id":"a3f19c02b7de","name":"Default","slug":"default","internal":false,"project_id":"Xk9mZp","created_at":"2026-01-02T03:04:05Z"}`,
				`{"id":"b4e20d13c8ef","name":"Internal","slug":"internal","internal":true,"project_id":"Xk9mZp","created_at":"2026-01-02T03:04:06Z"}`,
			),
			firstID:  "a3f19c02b7de",
			secondID: "b4e20d13c8ef",
		},
		{
			command:  "check-ins",
			routes:   twoPages("/v3/projects/Xk9mZp/check_ins", checkInJSON, cronCheckInJSON),
			firstID:  "Ci5xWq",
			secondID: "Ci6yZr",
		},
		{
			command: "environments",
			routes: twoPages(
				"/v3/projects/Xk9mZp/environments",
				environmentJSON,
				`{"id":"En7uSn","project_id":"Xk9mZp","name":"staging","notifications":false,"created_at":null}`,
			),
			firstID:  "En6tRm",
			secondID: "En7uSn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			s := newV3Routes(t, tt.routes)
			out, err := runCLI(t, v3Env(s), tt.command, "list", "--project-id", "Xk9mZp")
			require.NoError(t, err)
			assert.Len(t, s.Requests(), 2)
			assert.Contains(t, out, tt.firstID)
			assert.Contains(t, out, tt.secondID)

			s = newV3Routes(t, tt.routes)
			out, err = runCLI(t, v3Env(s), tt.command, "list", "--project-id", "Xk9mZp",
				"--output", "json")
			require.NoError(t, err)
			assert.Contains(t, out, `"id": "`+tt.firstID+`"`)
			assert.Contains(t, out, `"id": "`+tt.secondID+`"`)
		})
	}
}
