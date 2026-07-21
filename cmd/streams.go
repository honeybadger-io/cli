package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	streamsProjectID    int
	streamsOutputFormat string
)

// streamsCmd represents the streams command
var streamsCmd = &cobra.Command{
	Use:     "streams",
	Short:   "Manage Insights streams",
	GroupID: GroupDataAPI,
	Long:    `View Insights data streams for your Honeybadger projects.`,
}

// streamsListCmd represents the streams list command
var streamsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List streams for a project",
	Long: `List all Insights data streams for a specific project.

Stream IDs can be used to scope Insights queries (hb insights query --stream-ids)
and alarms to specific streams.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&streamsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		streams, err := client.Streams.List(ctx, streamsProjectID)
		if err != nil {
			return fmt.Errorf("failed to list streams: %w", err)
		}

		switch streamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(streams, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tINTERNAL\tCREATED")
			for _, stream := range streams {
				internal := " "
				if stream.Internal {
					internal = "✓"
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					stream.ID,
					stream.Name,
					stream.Slug,
					internal,
					stream.CreatedAt.Format("2006-01-02 15:04"))
			}
			_ = w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(streamsCmd)
	streamsCmd.AddCommand(streamsListCmd)

	// Common flags
	streamsCmd.PersistentFlags().IntVar(&streamsProjectID, "project-id", 0, "Project ID")

	// Flags for list command
	streamsListCmd.Flags().
		StringVarP(&streamsOutputFormat, "output", "o", "table", "Output format (table or json)")
}
