package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	checkInCmdID   string
	checkInCmdSlug string
)

// checkInCmd represents the check-in command
var checkInCmd = &cobra.Command{
	Use:   "check-in",
	Short: "Report a check-in to Honeybadger",
	Long: `Report a check-in to Honeybadger's Reporting API.
This command sends a simple GET request to mark a check-in as successful.

Example:
  hb check-in --id XyZZy
  hb check-in --slug daily-backup --api-key your-project-api-key`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if checkInCmdID == "" && checkInCmdSlug == "" {
			return fmt.Errorf("either check-in ID (--id) or slug (--slug) is required")
		}
		if checkInCmdID != "" && checkInCmdSlug != "" {
			return fmt.Errorf("cannot specify both check-in ID and slug")
		}

		// API key is only required when using slug
		apiKey := viper.GetString("api_key")
		if checkInCmdSlug != "" && apiKey == "" {
			return fmt.Errorf(
				"API key is required when using --slug. " +
					"Set it using --api-key flag or HONEYBADGER_API_KEY environment variable",
			)
		}

		apiEndpoint := viper.GetString("endpoint")
		var url string
		if checkInCmdID != "" {
			url = fmt.Sprintf("%s/v1/check_in/%s", apiEndpoint, checkInCmdID)
		} else {
			url = fmt.Sprintf("%s/v1/check_in/%s/%s", apiEndpoint, apiKey, checkInCmdSlug)
		}

		// Create request with timeout
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		// Send request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send check-in to Honeybadger: %w", err)
		}
		defer resp.Body.Close() // nolint:errcheck

		// Check response status
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
		}

		fmt.Fprintln(os.Stderr, "Check-in reported to Honeybadger")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkInCmd)
	checkInCmd.Flags().StringVarP(&checkInCmdID, "id", "i", "", "Check-in ID to report")
	checkInCmd.Flags().StringVarP(&checkInCmdSlug, "slug", "s", "", "Check-in slug to report")
}
