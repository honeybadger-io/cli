package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsightsQueryStreamIDs(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var request map[string]interface{}
			require.NoError(t, json.Unmarshal(body, &request))
			assert.Equal(t, []interface{}{"abc123", "def456"}, request["stream_ids"])

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"meta": {"query": "fields @ts", "fields": [], "rows": 0, "total_rows": 0},
				"results": []
			}`))
		}),
	)
	defer server.Close()

	viper.Reset()
	viper.Set("endpoint", server.URL)
	viper.Set("auth_token", "test-token")

	insightsProjectID = 123
	insightsQuery = "fields @ts"
	insightsTimestamp = ""
	insightsTimezone = ""
	insightsStreamIDs = []string{"abc123", "def456"}
	insightsOutputFormat = "table"
	defer func() { insightsStreamIDs = nil }()

	err := insightsQueryCmd.RunE(insightsQueryCmd, []string{})
	assert.NoError(t, err)
}
