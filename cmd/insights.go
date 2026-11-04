package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/spf13/cobra"
)

var (
	insightsProjectID    string
	insightsQuery        string
	insightsTimestamp    string
	insightsTimezone     string
	insightsStreamIDs    []string
	insightsOutputFormat string
)

// insightsQueryData is the query service's result, which v3 passes through and
// apiv3 returns untyped.
type insightsQueryData struct {
	Results []map[string]any `json:"results"`
	Meta    struct {
		Query      string   `json:"query"`
		Fields     []string `json:"fields"`
		RowCount   int      `json:"row_count"`
		TotalCount int      `json:"total_count"`
		StartAt    string   `json:"start_at"`
		EndAt      string   `json:"end_at"`
	} `json:"meta"`
}

// insightsCmd represents the insights command
var insightsCmd = &cobra.Command{
	Use:     "insights",
	Short:   "Query Honeybadger Insights data",
	GroupID: GroupDataAPI,
	Long:    `Execute BadgerQL queries against your Honeybadger Insights data.`,
}

// insightsQueryCmd represents the insights query command
var insightsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Execute a BadgerQL query",
	Long: `Execute a BadgerQL query against your project's Insights data.

Examples:
  # Query CPU usage
  hb insights query --project-id Xk9mZp --query "SELECT AVG(used_percent) FROM report.system.cpu"

  # Query with timezone
  hb insights query --project-id Xk9mZp --query "SELECT * FROM report.system.memory" --timezone "America/New_York"

  # Query the last 7 days
  hb insights query --project-id Xk9mZp --query "SELECT * FROM report.system.disk" --ts P7D

  # Restrict the query to specific streams (find IDs with 'hb streams list')
  hb insights query --project-id Xk9mZp --query "fields @ts, @message" --stream-ids "a3f19c02b7de,b4e20d13c8ef"` + v3Reference("insights/run-insights-query"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&insightsProjectID); err != nil {
			return err
		}
		if insightsQuery == "" {
			return fmt.Errorf("query is required. Set it using --query flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		result, err := client.Insights.Query(context.Background(), insightsProjectID,
			apiv3.InsightsQuery{
				Query:     insightsQuery,
				Ts:        insightsTimestamp,
				Timezone:  insightsTimezone,
				StreamIDs: insightsStreamIDs,
			})
		if err != nil {
			return dataAPIError("execute query", err)
		}

		if insightsOutputFormat == "json" {
			return printJSON(result.Data)
		}

		raw, err := json.Marshal(result.Data)
		if err != nil {
			return fmt.Errorf("failed to read query result: %w", err)
		}
		var data insightsQueryData
		if err := json.Unmarshal(raw, &data); err != nil {
			return fmt.Errorf("failed to read query result: %w", err)
		}

		if len(data.Results) == 0 {
			fmt.Println("No results found")
			return nil
		}

		fmt.Printf("Query: %s\n", data.Meta.Query)
		fmt.Printf("Rows: %d (Total: %d)\n", data.Meta.RowCount, data.Meta.TotalCount)
		if data.Meta.StartAt != "" {
			fmt.Printf("Time Range: %s to %s\n", data.Meta.StartAt, data.Meta.EndAt)
		}
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if len(data.Meta.Fields) > 0 {
			for i, field := range data.Meta.Fields {
				if i > 0 {
					_, _ = fmt.Fprint(w, "\t")
				}
				_, _ = fmt.Fprint(w, field)
			}
			_, _ = fmt.Fprintln(w)
		}

		for _, row := range data.Results {
			for i, field := range data.Meta.Fields {
				if i > 0 {
					_, _ = fmt.Fprint(w, "\t")
				}
				_, _ = fmt.Fprint(w, formatInsightsValue(row[field]))
			}
			_, _ = fmt.Fprintln(w)
		}
		return w.Flush()
	},
}

// formatInsightsValue renders one result cell: whole numbers without decimals,
// other numbers to two places, and RFC3339 timestamps as 2006-01-02 15:04:05.
func formatInsightsValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "NULL"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func init() {
	rootCmd.AddCommand(insightsCmd)
	insightsCmd.AddCommand(insightsQueryCmd)

	// Flags for query command
	insightsQueryCmd.Flags().StringVar(&insightsProjectID, "project-id", "", "Project ID")
	insightsQueryCmd.Flags().
		StringVarP(&insightsQuery, "query", "q", "", "BadgerQL query to execute")
	insightsQueryCmd.Flags().
		StringVar(&insightsTimestamp, "ts", "", "Time range for the query, such as '1h' or 'P7D'")
	insightsQueryCmd.Flags().
		StringVar(&insightsTimezone, "timezone", "", "Timezone for the query (e.g., 'America/New_York')")
	insightsQueryCmd.Flags().
		StringSliceVar(&insightsStreamIDs, "stream-ids", nil, "Restrict the query to specific stream IDs (comma-separated; see 'hb streams list')")
	insightsQueryCmd.Flags().
		StringVarP(&insightsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Mark required flags
	if err := insightsQueryCmd.MarkFlagRequired("query"); err != nil {
		fmt.Printf("error marking query flag as required: %v\n", err)
	}
}
