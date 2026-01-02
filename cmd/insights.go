package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	insightsProjectID    int
	insightsQuery        string
	insightsTimestamp    string
	insightsTimezone     string
	insightsOutputFormat string
)

// insightsCmd represents the insights command
var insightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "Query Honeybadger Insights data",
	Long:  `Execute BadgerQL queries against your Honeybadger Insights data.`,
}

// insightsQueryCmd represents the insights query command
var insightsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Execute a BadgerQL query",
	Long: `Execute a BadgerQL query against your project's Insights data.

Examples:
  # Query CPU usage
  hb insights query --project-id 12345 --query "SELECT AVG(used_percent) FROM report.system.cpu"

  # Query with timezone
  hb insights query --project-id 12345 --query "SELECT * FROM report.system.memory" --timezone "America/New_York"

  # Query at a specific timestamp
  hb insights query --project-id 12345 --query "SELECT * FROM report.system.disk" --ts "2024-01-01T00:00:00Z"`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if insightsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if insightsQuery == "" {
			return fmt.Errorf("query is required. Set it using --query flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Insights API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		// Build request
		request := hbapi.InsightsQueryRequest{
			Query:    insightsQuery,
			Ts:       insightsTimestamp,
			Timezone: insightsTimezone,
		}

		ctx := context.Background()
		response, err := client.Insights.Query(ctx, insightsProjectID, request)
		if err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}

		// Output results based on format
		switch insightsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			// Table format
			if len(response.Results) == 0 {
				fmt.Println("No results found")
				return nil
			}

			// Print metadata
			fmt.Printf("Query: %s\n", response.Meta.Query)
			fmt.Printf("Rows: %d (Total: %d)\n", response.Meta.Rows, response.Meta.TotalRows)
			if response.Meta.StartAt != "" {
				startTime, _ := time.Parse(time.RFC3339, response.Meta.StartAt)
				endTime, _ := time.Parse(time.RFC3339, response.Meta.EndAt)
				fmt.Printf(
					"Time Range: %s to %s\n",
					startTime.Format("2006-01-02 15:04:05"),
					endTime.Format("2006-01-02 15:04:05"),
				)
			}
			fmt.Println()

			// Print results as table
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			// Print header
			if len(response.Meta.Fields) > 0 {
				for i, field := range response.Meta.Fields {
					if i > 0 {
						_, _ = fmt.Fprint(w, "\t")
					}
					_, _ = fmt.Fprint(w, field)
				}
				_, _ = fmt.Fprintln(w)
			}

			// Print rows
			for _, row := range response.Results {
				for i, field := range response.Meta.Fields {
					if i > 0 {
						_, _ = fmt.Fprint(w, "\t")
					}
					// Format the value based on type
					value := row[field]
					if value == nil {
						_, _ = fmt.Fprint(w, "NULL")
					} else {
						switch v := value.(type) {
						case float64:
							// Round to 2 decimal places if it's a float
							if v == float64(int64(v)) {
								_, _ = fmt.Fprintf(w, "%d", int64(v))
							} else {
								_, _ = fmt.Fprintf(w, "%.2f", v)
							}
						case string:
							// Try to parse as timestamp for better display
							if t, err := time.Parse(time.RFC3339, v); err == nil {
								_, _ = fmt.Fprint(w, t.Format("2006-01-02 15:04:05"))
							} else {
								_, _ = fmt.Fprint(w, v)
							}
						default:
							_, _ = fmt.Fprintf(w, "%v", v)
						}
					}
				}
				_, _ = fmt.Fprintln(w)
			}
			_ = w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(insightsCmd)
	insightsCmd.AddCommand(insightsQueryCmd)

	// Flags for query command
	insightsQueryCmd.Flags().IntVar(&insightsProjectID, "project-id", 0, "Project ID")
	insightsQueryCmd.Flags().
		StringVarP(&insightsQuery, "query", "q", "", "BadgerQL query to execute")
	insightsQueryCmd.Flags().
		StringVar(&insightsTimestamp, "ts", "", "Timestamp for the query (RFC3339 format)")
	insightsQueryCmd.Flags().
		StringVar(&insightsTimezone, "timezone", "", "Timezone for the query (e.g., 'America/New_York')")
	insightsQueryCmd.Flags().
		StringVarP(&insightsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Mark required flags
	if err := insightsQueryCmd.MarkFlagRequired("project-id"); err != nil {
		fmt.Printf("error marking project-id flag as required: %v\n", err)
	}
	if err := insightsQueryCmd.MarkFlagRequired("query"); err != nil {
		fmt.Printf("error marking query flag as required: %v\n", err)
	}
}
