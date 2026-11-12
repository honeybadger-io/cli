package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/spf13/cobra"
)

var (
	deploymentsProjectID    string
	deploymentID            string
	deploymentsOutputFormat string
	deploymentsEnvironment  string
	deploymentsLocalUser    string
	deploymentsLimit        int
	deploymentsBefore       string
	deploymentsAfter        string
)

// deploymentsCmd represents the deployments command
var deploymentsCmd = &cobra.Command{
	Use:     "deployments",
	Short:   "View deployments",
	GroupID: GroupDataAPI,
	Long:    `View deployment records for your Honeybadger projects.`,
}

// deploymentsListCmd represents the deployments list command
var deploymentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployments for a project",
	Long: `List a project's most recent deployments, newest first, with optional filtering.

When there are more, it prints the cursor for the next page to stderr: pass it
with --before for older deployments, or --after for newer ones.

Example:
  hb deployments list --project-id Xk9mZp --limit 100
  hb deployments list --project-id Xk9mZp --limit 100 --before <cursor>` + v3Reference(
		"deploys/list-deploys",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&deploymentsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		opts := []apiv3.Option{
			apiv3.InEnvironment(deploymentsEnvironment),
			apiv3.DeployedBy(deploymentsLocalUser),
			apiv3.Limit(deploymentsLimit),
		}
		// Before and After each set both cursors, so only the one given is passed.
		if deploymentsBefore != "" {
			opts = append(opts, apiv3.Before(deploymentsBefore))
		}
		if deploymentsAfter != "" {
			opts = append(opts, apiv3.After(deploymentsAfter))
		}
		response, err := client.Deploys.List(context.Background(), deploymentsProjectID, opts...)
		if err != nil {
			return dataAPIError("list deployments", err)
		}

		if deploymentsOutputFormat == "json" {
			if err := printJSON(response.Data); err != nil {
				return err
			}
			printDeployCursors(response.TimeSeries)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tENVIRONMENT\tREVISION\tUSER\tCREATED")
		for _, d := range response.Data {
			revision := d.Revision.GetOrEmpty()
			if len(revision) > 12 {
				revision = revision[:12]
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				d.Id,
				d.Environment.GetOrEmpty(),
				revision,
				d.LocalUsername.GetOrEmpty(),
				d.CreatedAt.Format("2006-01-02 15:04"))
		}
		if err := w.Flush(); err != nil {
			return err
		}
		printDeployCursors(response.TimeSeries)
		return nil
	},
}

// printDeployCursors tells the user how to fetch the pages either side of
// this one. v3 has no date filters for deploys, so the cursors are the only
// way to older ones. They go to stderr to keep the table and JSON clean.
func printDeployCursors(page *apiv3.TimeSeriesPagination) {
	if page == nil {
		return
	}
	if cursor := page.OldestCursor.GetOrEmpty(); page.HasOlder && cursor != "" {
		fmt.Fprintf(
			os.Stderr,
			"Older deployments: rerun with --before %s in place of any --before or --after\n",
			cursor,
		)
	}
	if cursor := page.NewestCursor.GetOrEmpty(); page.HasNewer && cursor != "" {
		fmt.Fprintf(
			os.Stderr,
			"Newer deployments: rerun with --after %s in place of any --before or --after\n",
			cursor,
		)
	}
}

// deploymentsGetCmd represents the deployments get command
var deploymentsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a deployment by ID",
	Long: `Get detailed information about a specific deployment.` + v3Reference(
		"deploys/get-deploy",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&deploymentsProjectID); err != nil {
			return err
		}
		if deploymentID == "" {
			return fmt.Errorf("deployment ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		deployment, err := client.Deploys.Get(
			context.Background(),
			deploymentsProjectID,
			deploymentID,
		)
		if err != nil {
			return dataAPIError("get deployment", err)
		}

		if deploymentsOutputFormat == "json" {
			return printJSON(deployment)
		}

		fmt.Printf("Deployment Details:\n")
		fmt.Printf("  ID: %s\n", deployment.Id)
		fmt.Printf("  Environment: %s\n", deployment.Environment.GetOrEmpty())
		fmt.Printf("  Revision: %s\n", deployment.Revision.GetOrEmpty())
		fmt.Printf("  Repository: %s\n", deployment.Repository.GetOrEmpty())
		fmt.Printf("  Local Username: %s\n", deployment.LocalUsername.GetOrEmpty())
		fmt.Printf("  Project ID: %s\n", deployment.ProjectId)
		fmt.Printf("  Created: %s\n", deployment.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deploymentsCmd)
	deploymentsCmd.AddCommand(deploymentsListCmd)
	deploymentsCmd.AddCommand(deploymentsGetCmd)

	// Common flags
	deploymentsCmd.PersistentFlags().
		StringVar(&deploymentsProjectID, "project-id", "", "Project ID")

	// Flags for list command
	deploymentsListCmd.Flags().
		StringVarP(&deploymentsOutputFormat, "output", "o", "table", "Output format (table or json)")
	deploymentsListCmd.Flags().
		StringVarP(&deploymentsEnvironment, "environment", "e", "", "Filter by environment")
	deploymentsListCmd.Flags().
		StringVar(&deploymentsLocalUser, "local-user", "", "Filter by local username")
	deploymentsListCmd.Flags().
		IntVar(&deploymentsLimit, "limit", 25, "Maximum number of deployments to return (max 100)")
	deploymentsListCmd.Flags().
		StringVar(&deploymentsBefore, "before", "", "Show older deployments: the cursor a previous page printed")
	deploymentsListCmd.Flags().
		StringVar(&deploymentsAfter, "after", "", "Show newer deployments: the cursor a previous page printed")
	deploymentsListCmd.MarkFlagsMutuallyExclusive("before", "after")

	// Flags for get command
	deploymentsGetCmd.Flags().StringVar(&deploymentID, "id", "", "Deployment ID")
	deploymentsGetCmd.Flags().
		StringVarP(&deploymentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = deploymentsGetCmd.MarkFlagRequired("id")
}
