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
	ingestionKeysProjectID    string
	ingestionKeyID            string
	ingestionKeysOutputFormat string
	ingestionKeyCLIInputJSON  string
)

// ingestionKeysCmd represents the ingestion-keys command
var ingestionKeysCmd = &cobra.Command{
	Use:     "ingestion-keys",
	Short:   "Manage project Ingestion Keys",
	GroupID: GroupDataAPI,
	Long: `View and manage a project's Ingestion Keys (hbp_), the keys an app uses to
send errors and events to Honeybadger.`,
}

// ingestionKeysListCmd represents the ingestion-keys list command
var ingestionKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Ingestion Keys for a project",
	Long: `List all Ingestion Keys for a specific project.` + v3Reference(
		"ingestion-keys/list-ingestion-keys",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&ingestionKeysProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		keys, err := client.IngestionKeys.ListAll(context.Background(), ingestionKeysProjectID)
		if err != nil {
			return dataAPIError("list Ingestion Keys", err)
		}

		if ingestionKeysOutputFormat == "json" {
			return printJSON(keys)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tLABEL\tKEY\tCREATED")
		for _, key := range keys {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				key.Id,
				key.Label.GetOrEmpty(),
				key.Key,
				key.CreatedAt.Format("2006-01-02 15:04"))
		}
		return w.Flush()
	},
}

// ingestionKeysGetCmd represents the ingestion-keys get command
var ingestionKeysGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an Ingestion Key by ID",
	Long: `Get detailed information about a specific Ingestion Key, including its value.` + v3Reference(
		"ingestion-keys/get-ingestion-key",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&ingestionKeysProjectID); err != nil {
			return err
		}
		if ingestionKeyID == "" {
			return fmt.Errorf("ingestion key ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		key, err := client.IngestionKeys.Get(
			context.Background(),
			ingestionKeysProjectID,
			ingestionKeyID,
		)
		if err != nil {
			return dataAPIError("get Ingestion Key", err)
		}

		if ingestionKeysOutputFormat == "json" {
			return printJSON(key)
		}

		fmt.Printf("Ingestion Key Details:\n")
		fmt.Printf("  ID: %s\n", key.Id)
		fmt.Printf("  Project ID: %s\n", key.ProjectId)
		fmt.Printf("  Label: %s\n", key.Label.GetOrEmpty())
		fmt.Printf("  Key: %s\n", key.Key)
		fmt.Printf("  Created: %s\n", key.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

// ingestionKeysCreateCmd represents the ingestion-keys create command
var ingestionKeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Ingestion Key",
	Long: `Create a new Ingestion Key for a project. A project can hold a limited number
of keys.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.
It takes the key's fields directly; the only one is an optional label. Send {} for a
key without a label.

Example JSON payload:
{
  "label": "CI"
}` + v3Reference("ingestion-keys/create-ingestion-key"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&ingestionKeysProjectID); err != nil {
			return err
		}
		params, err := decodeInput[apiv3.IngestionKeyParams](
			ingestionKeyCLIInputJSON,
			"ingestion_key",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		key, err := client.IngestionKeys.Create(
			context.Background(),
			ingestionKeysProjectID,
			params,
		)
		if err != nil {
			return dataAPIError("create Ingestion Key", err)
		}

		if ingestionKeysOutputFormat == "json" {
			return printJSON(key)
		}
		fmt.Printf("Ingestion Key created successfully!\n")
		fmt.Printf("  ID: %s\n", key.Id)
		fmt.Printf("  Label: %s\n", key.Label.GetOrEmpty())
		fmt.Printf("  Key: %s\n", key.Key)
		return nil
	},
}

// ingestionKeysUpdateCmd represents the ingestion-keys update command
var ingestionKeysUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an Ingestion Key's label",
	Long: `Update an Ingestion Key's label. The key's value can't be changed; a label sent
as null is cleared.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "label": "Deploys"
}` + v3Reference("ingestion-keys/update-ingestion-key"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&ingestionKeysProjectID); err != nil {
			return err
		}
		if ingestionKeyID == "" {
			return fmt.Errorf("ingestion key ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.IngestionKeyParams](
			ingestionKeyCLIInputJSON,
			"ingestion_key",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if _, err := client.IngestionKeys.Update(
			context.Background(),
			ingestionKeysProjectID,
			ingestionKeyID,
			params,
		); err != nil {
			return dataAPIError("update Ingestion Key", err)
		}

		fmt.Println("Ingestion Key updated successfully")
		return nil
	},
}

// ingestionKeysDeleteCmd represents the ingestion-keys delete command
var ingestionKeysDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an Ingestion Key",
	Long: `Delete an Ingestion Key by ID. Apps still sending with it stop reporting.
A project's last key can't be deleted. This action cannot be undone.` + v3Reference("ingestion-keys/delete-ingestion-key"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&ingestionKeysProjectID); err != nil {
			return err
		}
		if ingestionKeyID == "" {
			return fmt.Errorf("ingestion key ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.IngestionKeys.Delete(
			context.Background(),
			ingestionKeysProjectID,
			ingestionKeyID,
		); err != nil {
			return dataAPIError("delete Ingestion Key", err)
		}

		fmt.Println("Ingestion Key deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ingestionKeysCmd)
	ingestionKeysCmd.AddCommand(ingestionKeysListCmd)
	ingestionKeysCmd.AddCommand(ingestionKeysGetCmd)
	ingestionKeysCmd.AddCommand(ingestionKeysCreateCmd)
	ingestionKeysCmd.AddCommand(ingestionKeysUpdateCmd)
	ingestionKeysCmd.AddCommand(ingestionKeysDeleteCmd)

	// Common flags
	ingestionKeysCmd.PersistentFlags().
		StringVar(&ingestionKeysProjectID, "project-id", "", "Project ID")

	// Flags for list command
	ingestionKeysListCmd.Flags().
		StringVarP(&ingestionKeysOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	ingestionKeysGetCmd.Flags().StringVar(&ingestionKeyID, "id", "", "Ingestion Key ID")
	ingestionKeysGetCmd.Flags().
		StringVarP(&ingestionKeysOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = ingestionKeysGetCmd.MarkFlagRequired("id")

	// Flags for create command
	ingestionKeysCreateCmd.Flags().
		StringVar(&ingestionKeyCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	ingestionKeysCreateCmd.Flags().
		StringVarP(&ingestionKeysOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = ingestionKeysCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for update command
	ingestionKeysUpdateCmd.Flags().StringVar(&ingestionKeyID, "id", "", "Ingestion Key ID")
	ingestionKeysUpdateCmd.Flags().
		StringVar(&ingestionKeyCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = ingestionKeysUpdateCmd.MarkFlagRequired("id")
	_ = ingestionKeysUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	ingestionKeysDeleteCmd.Flags().StringVar(&ingestionKeyID, "id", "", "Ingestion Key ID")
	_ = ingestionKeysDeleteCmd.MarkFlagRequired("id")
}
