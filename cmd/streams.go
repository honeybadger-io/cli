package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	streamsProjectID    string
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
and alarms to specific streams.` + v3Reference("streams/list-streams"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&streamsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		streams, err := client.Insights.ListAllStreams(context.Background(), streamsProjectID)
		if err != nil {
			return dataAPIError("list streams", err)
		}

		if streamsOutputFormat == "json" {
			return printJSON(streams)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tINTERNAL\tCREATED")
		for _, stream := range streams {
			internal := " "
			if deref(stream.Internal) {
				internal = "✓"
			}

			created := ""
			if stream.CreatedAt != nil {
				created = stream.CreatedAt.Format("2006-01-02 15:04")
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				deref(stream.Id),
				deref(stream.Name),
				deref(stream.Slug),
				internal,
				created)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(streamsCmd)
	streamsCmd.AddCommand(streamsListCmd)

	// Common flags
	streamsCmd.PersistentFlags().StringVar(&streamsProjectID, "project-id", "", "Project ID")

	// Flags for list command
	streamsListCmd.Flags().
		StringVarP(&streamsOutputFormat, "output", "o", "table", "Output format (table or json)")
}
