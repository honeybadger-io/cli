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
	environmentsProjectID    int
	environmentID            int
	environmentsOutputFormat string
	environmentCLIInputJSON  string
)

// environmentsCmd represents the environments command
var environmentsCmd = &cobra.Command{
	Use:   "environments",
	Short: "Manage project environments",
	Long:  `View and manage environments for your Honeybadger projects.`,
}

// environmentsListCmd represents the environments list command
var environmentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments for a project",
	Long:  `List all environments configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if environmentsProjectID == 0 {
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

		ctx := context.Background()
		environments, err := client.Environments.List(ctx, environmentsProjectID)
		if err != nil {
			return fmt.Errorf("failed to list environments: %w", err)
		}

		switch environmentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(environments, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tNOTIFICATIONS\tCREATED")
			for _, env := range environments {
				notifications := "No"
				if env.Notifications {
					notifications = "Yes"
				}

				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					env.ID,
					env.Name,
					notifications,
					env.CreatedAt.Format("2006-01-02 15:04"))
			}
			_ = w.Flush()
		}

		return nil
	},
}

// environmentsGetCmd represents the environments get command
var environmentsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an environment by ID",
	Long:  `Get detailed information about a specific environment.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if environmentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if environmentID == 0 {
			return fmt.Errorf("environment ID is required. Set it using --id flag")
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
		environment, err := client.Environments.Get(ctx, environmentsProjectID, environmentID)
		if err != nil {
			return fmt.Errorf("failed to get environment: %w", err)
		}

		switch environmentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(environment, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Environment Details:\n")
			fmt.Printf("  ID: %d\n", environment.ID)
			fmt.Printf("  Name: %s\n", environment.Name)
			fmt.Printf("  Project ID: %d\n", environment.ProjectID)
			fmt.Printf("  Notifications: %v\n", environment.Notifications)
			fmt.Printf("  Created: %s\n", environment.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Updated: %s\n", environment.UpdatedAt.Format("2006-01-02 15:04:05"))
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

Example JSON payload:
{
  "environment": {
    "name": "staging",
    "notifications": true
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if environmentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if environmentCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
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

		jsonData, err := readJSONInput(environmentCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Environment hbapi.EnvironmentParams `json:"environment"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		environment, err := client.Environments.Create(
			ctx,
			environmentsProjectID,
			payload.Environment,
		)
		if err != nil {
			return fmt.Errorf("failed to create environment: %w", err)
		}

		switch environmentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(environment, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Environment created successfully!\n")
			fmt.Printf("  ID: %d\n", environment.ID)
			fmt.Printf("  Name: %s\n", environment.Name)
		}

		return nil
	},
}

// environmentsUpdateCmd represents the environments update command
var environmentsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing environment",
	Long: `Update an existing environment's settings.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "environment": {
    "name": "production",
    "notifications": false
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if environmentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if environmentID == 0 {
			return fmt.Errorf("environment ID is required. Set it using --id flag")
		}
		if environmentCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
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

		jsonData, err := readJSONInput(environmentCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Environment hbapi.EnvironmentParams `json:"environment"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		err = client.Environments.Update(
			ctx,
			environmentsProjectID,
			environmentID,
			payload.Environment,
		)
		if err != nil {
			return fmt.Errorf("failed to update environment: %w", err)
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
		if environmentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if environmentID == 0 {
			return fmt.Errorf("environment ID is required. Set it using --id flag")
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
		err := client.Environments.Delete(ctx, environmentsProjectID, environmentID)
		if err != nil {
			return fmt.Errorf("failed to delete environment: %w", err)
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
	environmentsCmd.PersistentFlags().IntVar(&environmentsProjectID, "project-id", 0, "Project ID")

	// Flags for list command
	environmentsListCmd.Flags().
		StringVarP(&environmentsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	environmentsGetCmd.Flags().IntVar(&environmentID, "id", 0, "Environment ID")
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
	environmentsUpdateCmd.Flags().IntVar(&environmentID, "id", 0, "Environment ID")
	environmentsUpdateCmd.Flags().
		StringVar(&environmentCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = environmentsUpdateCmd.MarkFlagRequired("id")
	_ = environmentsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	environmentsDeleteCmd.Flags().IntVar(&environmentID, "id", 0, "Environment ID")
	_ = environmentsDeleteCmd.MarkFlagRequired("id")
}
