package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/spf13/cobra"
)

var (
	integrationsProjectID    string
	integrationID            string
	integrationsOutputFormat string
	integrationCLIInputJSON  string
)

// integrationsCmd represents the integrations command
var integrationsCmd = &cobra.Command{
	Use:     "integrations",
	Short:   "Manage notification integrations",
	GroupID: GroupDataAPI,
	Long: `View and manage a project's notification integrations: Slack, webhooks,
email, PagerDuty and the rest.`,
}

// integrationEvents lists an integration's events for display.
func integrationEvents(events []string) string {
	if len(events) == 0 {
		return "none"
	}
	return strings.Join(events, ", ")
}

// integrationsListCmd represents the integrations list command
var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List integrations for a project",
	Long:  `List all notification integrations configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&integrationsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		integrations, err := client.Integrations.ListAll(
			context.Background(),
			integrationsProjectID,
		)
		if err != nil {
			return dataAPIError("list integrations", err)
		}

		if integrationsOutputFormat == "json" {
			return printJSON(integrations)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tTYPE\tACTIVE\tEVENTS")
		for _, integration := range integrations {
			active := "No"
			if integration.Active {
				active = "Yes"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				integration.Id,
				integration.Type,
				active,
				integrationEvents(integration.Events))
		}
		return w.Flush()
	},
}

// integrationsGetCmd represents the integrations get command
var integrationsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an integration by ID",
	Long:  `Get detailed information about a specific notification integration.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&integrationsProjectID); err != nil {
			return err
		}
		if integrationID == "" {
			return fmt.Errorf("integration ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		integration, err := client.Integrations.Get(
			context.Background(),
			integrationsProjectID,
			integrationID,
		)
		if err != nil {
			return dataAPIError("get integration", err)
		}

		if integrationsOutputFormat == "json" {
			return printJSON(integration)
		}

		fmt.Printf("Integration Details:\n")
		fmt.Printf("  ID: %s\n", integration.Id)
		fmt.Printf("  Project ID: %s\n", integration.ProjectId)
		fmt.Printf("  Type: %s\n", integration.Type)
		fmt.Printf("  Active: %v\n", integration.Active)
		if integration.Connected != nil {
			fmt.Printf("  Connected: %v\n", *integration.Connected)
		}
		if deref(integration.Error) {
			fmt.Printf("  Error: %s\n", deref(integration.ErrorReason))
		}
		fmt.Printf("  Events: %s\n", integrationEvents(integration.Events))
		fmt.Printf("  Created: %s\n", integration.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  URL: %s\n", integration.Links.Web)
		return nil
	},
}

// integrationsCreateCmd represents the integrations create command
var integrationsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new integration",
	Long: `Create a new notification integration for a project.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.
It takes the integration's fields directly. type picks the integration and config
holds its settings, listed per type at
https://docs.honeybadger.io/api/v3/integrations/types/.

OAuth integrations (Slack, GitHub and the like) are created unconnected: connect
one in the web UI at the URL this command prints.

Example JSON payload:
{
  "type": "WebHook",
  "config": {
    "url": "https://example.com/hook"
  },
  "events": ["occurred", "resolved"]
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&integrationsProjectID); err != nil {
			return err
		}
		params, err := decodeInput[apiv3.IntegrationCreateParams](
			integrationCLIInputJSON,
			"integration",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		integration, err := client.Integrations.Create(
			context.Background(),
			integrationsProjectID,
			params,
		)
		if err != nil {
			return dataAPIError("create integration", err)
		}

		if integrationsOutputFormat == "json" {
			return printJSON(integration)
		}
		fmt.Printf("Integration created successfully!\n")
		fmt.Printf("  ID: %s\n", integration.Id)
		fmt.Printf("  Type: %s\n", integration.Type)
		fmt.Printf("  URL: %s\n", integration.Links.Web)
		return nil
	},
}

// integrationsUpdateCmd represents the integrations update command
var integrationsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing integration",
	Long: `Update an existing integration's settings. Only the fields you send change;
a field sent as null is cleared or reset to its default. config is merged into the
stored settings, so send only the settings you're changing. An integration's type
can't be changed.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "active": false
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&integrationsProjectID); err != nil {
			return err
		}
		if integrationID == "" {
			return fmt.Errorf("integration ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.IntegrationUpdateParams](
			integrationCLIInputJSON,
			"integration",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if _, err := client.Integrations.Update(
			context.Background(),
			integrationsProjectID,
			integrationID,
			params,
		); err != nil {
			return dataAPIError("update integration", err)
		}

		fmt.Println("Integration updated successfully")
		return nil
	},
}

// integrationsDeleteCmd represents the integrations delete command
var integrationsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an integration",
	Long:  `Delete a notification integration by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&integrationsProjectID); err != nil {
			return err
		}
		if integrationID == "" {
			return fmt.Errorf("integration ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Integrations.Delete(
			context.Background(),
			integrationsProjectID,
			integrationID,
		); err != nil {
			return dataAPIError("delete integration", err)
		}

		fmt.Println("Integration deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(integrationsCmd)
	integrationsCmd.AddCommand(integrationsListCmd)
	integrationsCmd.AddCommand(integrationsGetCmd)
	integrationsCmd.AddCommand(integrationsCreateCmd)
	integrationsCmd.AddCommand(integrationsUpdateCmd)
	integrationsCmd.AddCommand(integrationsDeleteCmd)

	// Common flags
	integrationsCmd.PersistentFlags().
		StringVar(&integrationsProjectID, "project-id", "", "Project ID")

	// Flags for list command
	integrationsListCmd.Flags().
		StringVarP(&integrationsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	integrationsGetCmd.Flags().StringVar(&integrationID, "id", "", "Integration ID")
	integrationsGetCmd.Flags().
		StringVarP(&integrationsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = integrationsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	integrationsCreateCmd.Flags().
		StringVar(&integrationCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	integrationsCreateCmd.Flags().
		StringVarP(&integrationsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = integrationsCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for update command
	integrationsUpdateCmd.Flags().StringVar(&integrationID, "id", "", "Integration ID")
	integrationsUpdateCmd.Flags().
		StringVar(&integrationCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = integrationsUpdateCmd.MarkFlagRequired("id")
	_ = integrationsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	integrationsDeleteCmd.Flags().StringVar(&integrationID, "id", "", "Integration ID")
	_ = integrationsDeleteCmd.MarkFlagRequired("id")
}
