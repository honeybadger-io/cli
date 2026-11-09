package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --cli-input-json refuses a field the request doesn't have at any depth, not
// only at the top: api-go decodes a nullable field's contents without the
// unknown-field check, so a typo there would otherwise vanish before v3 could
// refuse it.

func TestStatusPagesUpdateRefusesATypoInsideFeatures(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "statuspages", "update", "--id", "Sp2wQx",
		"--cli-input-json", `{"features":{"home_lnik":"https://example.com"}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "features.home_lnik"`)
	assert.Empty(t, s.Requests())
}

func TestIntegrationsUpdateRefusesATypoInsideAFilter(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "integrations", "update", "--project-id", "Xk9mZp",
		"--id", "Ch9wLp", "--cli-input-json",
		`{"filters":[{"event":"all","query":"class:Foo"},{"event":"all","qeury":"class:Bar"}]}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "filters[1].qeury"`)
	assert.Empty(t, s.Requests())
}

// A null that would be sent as nothing can't be told apart from an unknown
// field, since neither survives the encode; the error names both.
func TestNullsThatWouldBeDroppedAreRefused(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		input string
		want  string
	}{
		{
			"a field that can't be null",
			[]string{"integrations", "update", "--project-id", "Xk9mZp", "--id", "Ch9wLp"},
			`{"active":null}`,
			`"active": unknown field, or one that can't be null`,
		},
		{
			"a typo set to null",
			[]string{"statuspages", "update", "--id", "Sp2wQx"},
			`{"features":{"home_lnik":null}}`,
			`"features.home_lnik": unknown field, or one that can't be null`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t)
			args := append(tt.args, "--cli-input-json", tt.input)
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			assert.Empty(t, s.Requests())
		})
	}
}

// A null on a field that always has a value would otherwise be sent as its
// zero value: here, an alarm threshold of 0.
func TestANullSentAsZeroIsRefused(t *testing.T) {
	s := newV3Server(t)
	_, err := runCLI(t, v3Env(s), "alarms", "update", "--project-id", "Xk9mZp",
		"--id", "Qx7LmP2vRt9K", "--cli-input-json",
		`{"trigger_config":{"type":"alert_result_count","config":{"operator":"gt","value":null}}}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `field "trigger_config.config.value" can't be null`)
	assert.Empty(t, s.Requests())
}

// A null list item is sent as its zero value too: an integration's events
// [null] went out as [""], which v3 drops, leaving the integration with no
// events. A null payload went out as {}, and ingestion-keys create made a key.
func TestNullsInListsAndANullPayloadAreRefused(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		input string
		want  string
	}{
		{
			"a null event",
			[]string{"integrations", "update", "--project-id", "Xk9mZp", "--id", "Ch9wLp"},
			`{"events":["occurred",null]}`,
			`field "events[1]" can't be null`,
		},
		{
			"a null filter",
			[]string{"integrations", "update", "--project-id", "Xk9mZp", "--id", "Ch9wLp"},
			`{"filters":[null]}`,
			`field "filters[0]" can't be null`,
		},
		{
			"a null payload",
			[]string{"ingestion-keys", "create", "--project-id", "Xk9mZp"},
			`null`,
			"the JSON payload must be an object, not null",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t)
			args := append(tt.args, "--cli-input-json", tt.input)
			_, err := runCLI(t, v3Env(s), args...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			assert.Empty(t, s.Requests())
		})
	}
}

func TestNestedFieldsThatExistAreSentAsGiven(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		input string
		body  string
	}{
		{
			"a status page feature",
			[]string{"statuspages", "update", "--id", "Sp2wQx"},
			`{"features":{"home_link":"https://example.com","custom_css":null}}`,
			statusPageBody,
		},
		{
			"an integration filter",
			[]string{"integrations", "update", "--project-id", "Xk9mZp", "--id", "Ch9wLp"},
			`{"filters":[{"event":"all","query":"class:Foo"}]}`,
			integrationBody,
		},
		{
			"an integration's free-form config",
			[]string{"integrations", "update", "--project-id", "Xk9mZp", "--id", "Ch9wLp"},
			`{"config":{"url":"https://example.com/hook","headers":{"X-Token":"abc"},"label":null}}`,
			integrationBody,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newV3Server(t, ok(tt.body))
			args := append(tt.args, "--cli-input-json", tt.input)
			_, err := runCLI(t, v3Env(s), args...)
			require.NoError(t, err)
			assertJSONBody(t, tt.input, s.only(t).Body)
		})
	}
}
