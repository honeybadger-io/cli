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
	Long:  `List a project's most recent deployments, newest first, with optional filtering.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&deploymentsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		response, err := client.Deploys.List(context.Background(), deploymentsProjectID,
			apiv3.InEnvironment(deploymentsEnvironment),
			apiv3.DeployedBy(deploymentsLocalUser),
			apiv3.Limit(deploymentsLimit))
		if err != nil {
			return dataAPIError("list deployments", err)
		}

		if deploymentsOutputFormat == "json" {
			return printJSON(response.Data)
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
		return w.Flush()
	},
}

// deploymentsGetCmd represents the deployments get command
var deploymentsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a deployment by ID",
	Long:  `Get detailed information about a specific deployment.`,
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

	// Flags for get command
	deploymentsGetCmd.Flags().StringVar(&deploymentID, "id", "", "Deployment ID")
	deploymentsGetCmd.Flags().
		StringVarP(&deploymentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = deploymentsGetCmd.MarkFlagRequired("id")
}
