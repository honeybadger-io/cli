package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/spf13/viper"
)

// apiTokenPrefixes are the prefixes of the credentials the v3 Data API accepts:
// a personal API Token (hbu_, or hbt_ for ones minted before hbu_) or an
// account API Token (hba_).
var apiTokenPrefixes = []string{"hbu_", "hbt_", "hba_"}

// ingestionKeyPrefix marks an Ingestion Key, which only sends errors and events.
const ingestionKeyPrefix = "hbp_"

const unauthorizedHint = "Check that the API Token hasn't been revoked or expired, " +
	"and that it's for this region (US: app.honeybadger.io, EU: eu-app.honeybadger.io)."

// dataAPIAppURL is the app host the Data API is served from, derived from the
// configured endpoint (api.honeybadger.io serves ingestion, app.honeybadger.io
// the Data API).
func dataAPIAppURL() string {
	return convertEndpointForDataAPI(viper.GetString("endpoint"))
}

// newDataAPIClient builds the v3 client every Data API command uses. It refuses,
// before any request, a missing token and anything that isn't an API Token.
func newDataAPIClient() (*apiv3.Client, error) {
	token, err := apiToken()
	if err != nil {
		return nil, err
	}
	return apiv3.NewClient().
		WithBaseURL(dataAPIAppURL()).
		WithBearerToken(token).
		WithUserAgent("hb-cli/" + rootCmd.Version), nil
}

func apiToken() (string, error) {
	token := strings.TrimSpace(viper.GetString("api_token"))
	createOne := fmt.Sprintf("create one at %s/users/edit#authentication", dataAPIAppURL())

	switch {
	case token == "" && strings.TrimSpace(viper.GetString("auth_token")) != "":
		return "", fmt.Errorf(
			"auth_token is no longer used. The Data API now needs an API Token: %s "+
				"and set it with --api-token, HONEYBADGER_API_TOKEN, or api_token in your config file",
			createOne,
		)
	case token == "":
		return "", errors.New(
			"API Token is required. Set it with --api-token, HONEYBADGER_API_TOKEN, or api_token in your config file",
		)
	case strings.HasPrefix(token, ingestionKeyPrefix):
		//nolint:staticcheck // a sentence shown to the user, not an error fragment
		return "", fmt.Errorf("That's an Ingestion Key, which only sends errors and events. "+
			"The Data API needs an API Token: %s", createOne)
	case !hasAPITokenPrefix(token):
		//nolint:staticcheck // a sentence shown to the user, not an error fragment
		return "", fmt.Errorf(
			"That isn't an API Token. The v3 Data API doesn't accept v2 personal auth tokens: %s",
			createOne,
		)
	}
	return token, nil
}

func hasAPITokenPrefix(token string) bool {
	for _, p := range apiTokenPrefixes {
		if strings.HasPrefix(token, p) {
			return true
		}
	}
	return false
}

// resolveDataAPIProjectID fills in projectID from project_id (the
// HONEYBADGER_PROJECT_ID environment variable or the config file) when the
// --project-id flag wasn't given. v3 project ids are strings; a legacy numeric
// id works too.
func resolveDataAPIProjectID(projectID *string) error {
	if *projectID == "" {
		*projectID = strings.TrimSpace(viper.GetString("project_id"))
	}
	if *projectID == "" {
		return errors.New(
			"project ID is required. Set it using --project-id flag, HONEYBADGER_PROJECT_ID environment variable, or project_id in your config file",
		)
	}
	return nil
}

// dataAPIError reports a failed Data API call. A 401 gets a hint, since v3 says
// only "Access denied" and the usual causes are a revoked token or one from the
// other region.
func dataAPIError(action string, err error) error {
	if errors.Is(err, apiv3.ErrUnauthorized) {
		return fmt.Errorf("failed to %s: %w\n%s", action, err, unauthorizedHint)
	}
	return fmt.Errorf("failed to %s: %w", action, err)
}

// decodeInput reads --cli-input-json (a JSON string or file://path) into T, v3's
// flat request body. It refuses v2's shape, the body wrapped in the resource
// name, and any field T doesn't have, so nothing is silently dropped.
func decodeInput[T any](input, wrapper string) (T, error) {
	var v T
	if input == "" {
		return v, errors.New("JSON payload is required. Set it using --cli-input-json flag")
	}
	data, err := readJSONInput(input)
	if err != nil {
		return v, fmt.Errorf("failed to read JSON input: %w", err)
	}

	var top map[string]json.RawMessage
	if json.Unmarshal(data, &top) == nil && len(top) == 1 {
		if _, wrapped := top[wrapper]; wrapped {
			return v, fmt.Errorf("v3 takes the fields directly: remove the outer %q key", wrapper)
		}
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, fmt.Errorf("failed to parse JSON payload: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return v, errors.New(
			"failed to parse JSON payload: unexpected content after the JSON object",
		)
	}
	return v, nil
}

// printJSON prints v as indented JSON.
func printJSON(v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(out))
	return nil
}
