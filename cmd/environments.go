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
	environmentsProjectID    string
	environmentID            string
	environmentsOutputFormat string
	environmentCLIInputJSON  string
)

// environmentsCmd represents the environments command
var environmentsCmd = &cobra.Command{
	Use:     "environments",
	Short:   "Manage project environments",
	GroupID: GroupDataAPI,
	Long:    `View and manage environments for your Honeybadger projects.`,
}

// environmentsListCmd represents the environments list command
var environmentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments for a project",
	Long:  `List all environments configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&environmentsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		environments, err := client.Environments.ListAll(
			context.Background(),
			environmentsProjectID,
		)
		if err != nil {
			return dataAPIError("list environments", err)
		}

		if environmentsOutputFormat == "json" {
			return printJSON(environments)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tNOTIFICATIONS\tCREATED")
		for _, env := range environments {
			notifications := "No"
			if env.Notifications {
				notifications = "Yes"
			}

			created := ""
			if at, err := env.CreatedAt.Get(); err == nil {
				created = at.Format("2006-01-02 15:04")
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				env.Id,
				env.Name,
				notifications,
				created)
		}
		return w.Flush()
	},
}

// environmentsGetCmd represents the environments get command
var environmentsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an environment by ID",
	Long:  `Get detailed information about a specific environment.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&environmentsProjectID); err != nil {
			return err
		}
		if environmentID == "" {
			return fmt.Errorf("environment ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		environment, err := client.Environments.Get(
			context.Background(),
			environmentsProjectID,
			environmentID,
		)
		if err != nil {
			return dataAPIError("get environment", err)
		}

		if environmentsOutputFormat == "json" {
			return printJSON(environment)
		}

		fmt.Printf("Environment Details:\n")
		fmt.Printf("  ID: %s\n", environment.Id)
		fmt.Printf("  Name: %s\n", environment.Name)
		fmt.Printf("  Project ID: %s\n", environment.ProjectId)
		fmt.Printf("  Notifications: %v\n", environment.Notifications)
		if at, err := environment.CreatedAt.Get(); err == nil {
			fmt.Printf("  Created: %s\n", at.Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

// environmentsCreateCmd represents the environments create command
var environmentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new environment",
	Long: `Create a new environment for a project.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.
It takes the environment's fields directly; only name is required.

Example JSON payload:
{
  "name": "staging",
  "notifications": true
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&environmentsProjectID); err != nil {
			return err
		}
		params, err := decodeInput[apiv3.EnvironmentCreateParams](
			environmentCLIInputJSON,
			"environment",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		environment, err := client.Environments.Create(
			context.Background(),
			environmentsProjectID,
			params,
		)
		if err != nil {
			return dataAPIError("create environment", err)
		}

		if environmentsOutputFormat == "json" {
			return printJSON(environment)
		}
		fmt.Printf("Environment created successfully!\n")
		fmt.Printf("  ID: %s\n", environment.Id)
		fmt.Printf("  Name: %s\n", environment.Name)
		return nil
	},
}

// environmentsUpdateCmd represents the environments update command
var environmentsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing environment",
	Long: `Update an existing environment's settings. Only the fields you send change.
An environment can't be renamed.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "notifications": false
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&environmentsProjectID); err != nil {
			return err
		}
		if environmentID == "" {
			return fmt.Errorf("environment ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.EnvironmentUpdateParams](
			environmentCLIInputJSON,
			"environment",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if _, err := client.Environments.Update(
			context.Background(),
			environmentsProjectID,
			environmentID,
			params,
		); err != nil {
			return dataAPIError("update environment", err)
		}

		fmt.Println("Environment updated successfully")
		return nil
	},
}

// environmentsDeleteCmd represents the environments delete command
var environmentsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an environment",
	Long:  `Delete an environment by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&environmentsProjectID); err != nil {
			return err
		}
		if environmentID == "" {
			return fmt.Errorf("environment ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Environments.Delete(
			context.Background(),
			environmentsProjectID,
			environmentID,
		); err != nil {
			return dataAPIError("delete environment", err)
		}

		fmt.Println("Environment deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(environmentsCmd)
	environmentsCmd.AddCommand(environmentsListCmd)
	environmentsCmd.AddCommand(environmentsGetCmd)
	environmentsCmd.AddCommand(environmentsCreateCmd)
	environmentsCmd.AddCommand(environmentsUpdateCmd)
	environmentsCmd.AddCommand(environmentsDeleteCmd)

	// Common flags
	environmentsCmd.PersistentFlags().
		StringVar(&environmentsProjectID, "project-id", "", "Project ID")

	// Flags for list command
	environmentsListCmd.Flags().
		StringVarP(&environmentsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	environmentsGetCmd.Flags().StringVar(&environmentID, "id", "", "Environment ID")
	environmentsGetCmd.Flags().
		StringVarP(&environmentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = environmentsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	environmentsCreateCmd.Flags().
		StringVar(&environmentCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	environmentsCreateCmd.Flags().
		StringVarP(&environmentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = environmentsCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for update command
	environmentsUpdateCmd.Flags().StringVar(&environmentID, "id", "", "Environment ID")
	environmentsUpdateCmd.Flags().
		StringVar(&environmentCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = environmentsUpdateCmd.MarkFlagRequired("id")
	_ = environmentsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	environmentsDeleteCmd.Flags().StringVar(&environmentID, "id", "", "Environment ID")
	_ = environmentsDeleteCmd.MarkFlagRequired("id")
}
