package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	deploymentsProjectID     int
	deploymentID             int
	deploymentsOutputFormat  string
	deploymentsEnvironment   string
	deploymentsLocalUser     string
	deploymentsCreatedAfter  int64
	deploymentsCreatedBefore int64
	deploymentsLimit         int
)

// deploymentsCmd represents the deployments command
var deploymentsCmd = &cobra.Command{
	Use:     "deployments",
	Short:   "View and manage deployments",
	GroupID: GroupDataAPI,
	Long:    `View and manage deployment records for your Honeybadger projects.`,
}

// deploymentsListCmd represents the deployments list command
var deploymentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployments for a project",
	Long:  `List all deployments for a specific project with optional filtering.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if deploymentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		options := hbapi.DeploymentListOptions{
			Environment:   deploymentsEnvironment,
			LocalUsername: deploymentsLocalUser,
			CreatedAfter:  deploymentsCreatedAfter,
			CreatedBefore: deploymentsCreatedBefore,
			Limit:         deploymentsLimit,
		}

		ctx := context.Background()
		deployments, err := client.Deployments.List(ctx, deploymentsProjectID, options)
		if err != nil {
			return fmt.Errorf("failed to list deployments: %w", err)
		}

		switch deploymentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(deployments, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tENVIRONMENT\tREVISION\tUSER\tCREATED")
			for _, d := range deployments {
				revision := d.Revision
				if len(revision) > 12 {
					revision = revision[:12]
				}

				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					d.ID,
					d.Environment,
					revision,
					d.LocalUsername,
					d.CreatedAt.Format("2006-01-02 15:04"))
			}
			_ = w.Flush()
		}

		return nil
	},
}

// deploymentsGetCmd represents the deployments get command
var deploymentsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a deployment by ID",
	Long:  `Get detailed information about a specific deployment.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if deploymentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if deploymentID == 0 {
			return fmt.Errorf("deployment ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		deployment, err := client.Deployments.Get(ctx, deploymentsProjectID, deploymentID)
		if err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}

		switch deploymentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(deployment, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Deployment Details:\n")
			fmt.Printf("  ID: %d\n", deployment.ID)
			fmt.Printf("  Environment: %s\n", deployment.Environment)
			fmt.Printf("  Revision: %s\n", deployment.Revision)
			fmt.Printf("  Repository: %s\n", deployment.Repository)
			fmt.Printf("  Local Username: %s\n", deployment.LocalUsername)
			fmt.Printf("  Project ID: %d\n", deployment.ProjectID)
			fmt.Printf("  Created: %s\n", deployment.CreatedAt.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

// deploymentsDeleteCmd represents the deployments delete command
var deploymentsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a deployment",
	Long:  `Delete a deployment record by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if deploymentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if deploymentID == 0 {
			return fmt.Errorf("deployment ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		err := client.Deployments.Delete(ctx, deploymentsProjectID, deploymentID)
		if err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}

		fmt.Println("Deployment deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deploymentsCmd)
	deploymentsCmd.AddCommand(deploymentsListCmd)
	deploymentsCmd.AddCommand(deploymentsGetCmd)
	deploymentsCmd.AddCommand(deploymentsDeleteCmd)

	// Common flags
	deploymentsCmd.PersistentFlags().IntVar(&deploymentsProjectID, "project-id", 0, "Project ID")

	// Flags for list command
	deploymentsListCmd.Flags().
		StringVarP(&deploymentsOutputFormat, "output", "o", "table", "Output format (table or json)")
	deploymentsListCmd.Flags().
		StringVarP(&deploymentsEnvironment, "environment", "e", "", "Filter by environment")
	deploymentsListCmd.Flags().
		StringVar(&deploymentsLocalUser, "local-user", "", "Filter by local username")
	deploymentsListCmd.Flags().
		Int64Var(&deploymentsCreatedAfter, "created-after", 0, "Filter by creation time (Unix timestamp)")
	deploymentsListCmd.Flags().
		Int64Var(&deploymentsCreatedBefore, "created-before", 0, "Filter by creation time (Unix timestamp)")
	deploymentsListCmd.Flags().
		IntVar(&deploymentsLimit, "limit", 25, "Maximum number of deployments to return (max 25)")

	// Flags for get command
	deploymentsGetCmd.Flags().IntVar(&deploymentID, "id", 0, "Deployment ID")
	deploymentsGetCmd.Flags().
		StringVarP(&deploymentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = deploymentsGetCmd.MarkFlagRequired("id")

	// Flags for delete command
	deploymentsDeleteCmd.Flags().IntVar(&deploymentID, "id", 0, "Deployment ID")
	_ = deploymentsDeleteCmd.MarkFlagRequired("id")
}
