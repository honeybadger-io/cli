package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/spf13/cobra"
)

var (
	alarmsProjectID    string
	alarmID            string
	alarmsOutputFormat string
	alarmCLIInputJSON  string
	alarmHistoryPage   int
)

// alarmsCmd represents the alarms command
var alarmsCmd = &cobra.Command{
	Use:     "alarms",
	Short:   "Manage Honeybadger Insights alarms",
	GroupID: GroupDataAPI,
	Long:    `View and manage Insights alarms for your Honeybadger projects.`,
}

// alarmsListCmd represents the alarms list command
var alarmsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List alarms for a project",
	Long:  `List all Insights alarms configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&alarmsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		response, err := client.Alarms.List(context.Background(), alarmsProjectID)
		if err != nil {
			return dataAPIError("list alarms", err)
		}

		if alarmsOutputFormat == "json" {
			return printJSON(response.Data)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tSTATE\tQUERY\tEVAL PERIOD\tLAST CHECKED")
		for _, alarm := range response.Data {
			query := alarm.Query.GetOrEmpty()
			if len(query) > 50 {
				query = query[:47] + "..."
			}

			lastChecked := "Never"
			if at, err := alarm.LastCheckedAt.Get(); err == nil {
				lastChecked = at.Format("2006-01-02 15:04")
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				alarm.Id,
				alarm.Name,
				alarm.State,
				query,
				alarm.EvaluationPeriod.GetOrEmpty(),
				lastChecked)
		}
		return w.Flush()
	},
}

// alarmsGetCmd represents the alarms get command
var alarmsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an alarm by ID",
	Long:  `Get detailed information about a specific Insights alarm.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		alarm, err := client.Alarms.Get(context.Background(), alarmsProjectID, alarmID)
		if err != nil {
			return dataAPIError("get alarm", err)
		}

		if alarmsOutputFormat == "json" {
			return printJSON(alarm)
		}

		fmt.Printf("Alarm Details:\n")
		fmt.Printf("  ID: %s\n", alarm.Id)
		fmt.Printf("  Name: %s\n", alarm.Name)
		if description := alarm.Description.GetOrEmpty(); description != "" {
			fmt.Printf("  Description: %s\n", description)
		}
		fmt.Printf("  State: %s\n", alarm.State)
		fmt.Printf("  Query: %s\n", alarm.Query.GetOrEmpty())
		if streamIDs := alarm.StreamIds.GetOrEmpty(); len(streamIDs) > 0 {
			fmt.Printf("  Stream IDs: %s\n", strings.Join(streamIDs, ", "))
		}
		fmt.Printf("  Evaluation Period: %s\n", alarm.EvaluationPeriod.GetOrEmpty())
		if lag := alarm.LookbackLag.GetOrEmpty(); lag != "" {
			fmt.Printf("  Lookback Lag: %s\n", lag)
		}
		if trigger, err := alarm.TriggerConfig.Get(); err == nil {
			if triggerJSON, err := json.Marshal(trigger); err == nil {
				fmt.Printf("  Trigger Config: %s\n", triggerJSON)
			}
		}
		if alarmError := alarm.Error.GetOrEmpty(); alarmError != "" {
			fmt.Printf("  Error: %s\n", alarmError)
		}
		if at, err := alarm.LastCheckedAt.Get(); err == nil {
			fmt.Printf("  Last Checked: %s\n", at.Format("2006-01-02 15:04:05"))
		}
		if at, err := alarm.NextCheckAt.Get(); err == nil {
			fmt.Printf("  Next Check: %s\n", at.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("  Created: %s\n", alarm.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated: %s\n", alarm.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  URL: %s\n", alarm.Links.Web)
		return nil
	},
}

// alarmsCreateCmd represents the alarms create command
var alarmsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new alarm",
	Long: `Create a new Insights alarm for a project.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.
It takes the alarm's fields directly. name, query, evaluation_period, lookback_lag and
trigger_config are required.

stream_ids is optional. Omit it and the alarm queries all of the project's Insights
streams. If you do supply it, use the opaque IDs from 'hb streams list', not the slugs.
It can't be empty, and an ID that isn't one of the project's streams is rejected with a 422.

Example JSON payload:
{
  "name": "High Error Rate",
  "description": "Alert when errors spike",
  "query": "filter event_type::str == \"notice\"",
  "evaluation_period": "5m",
  "lookback_lag": "1m",
  "trigger_config": {
    "type": "alert_result_count",
    "config": {"operator": "gt", "value": 10}
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&alarmsProjectID); err != nil {
			return err
		}
		params, err := decodeInput[apiv3.AlarmCreateParams](alarmCLIInputJSON, "alarm")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		alarm, err := client.Alarms.Create(context.Background(), alarmsProjectID, params)
		if err != nil {
			return dataAPIError("create alarm", err)
		}

		if alarmsOutputFormat == "json" {
			return printJSON(alarm)
		}
		fmt.Printf("Alarm created successfully!\n")
		fmt.Printf("  ID: %s\n", alarm.Id)
		fmt.Printf("  Name: %s\n", alarm.Name)
		fmt.Printf("  State: %s\n", alarm.State)
		return nil
	},
}

// alarmsUpdateCmd represents the alarms update command
var alarmsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing alarm",
	Long: `Update an existing Insights alarm's settings. Only the fields you send change;
a description sent as null is cleared.

A trigger_config you send replaces the stored one rather than merging into it,
so send its type and config together.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

stream_ids left out keeps the alarm's streams; a list replaces them (use the opaque IDs
from 'hb streams list', not the slugs; it can't be empty), and null resets them to all of
the project's Insights streams.

Example JSON payload:
{
  "name": "Updated Alarm Name",
  "evaluation_period": "10m",
  "trigger_config": {
    "type": "alert_result_count",
    "config": {"operator": "gt", "value": 25}
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.AlarmUpdateParams](alarmCLIInputJSON, "alarm")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if _, err := client.Alarms.Update(
			context.Background(),
			alarmsProjectID,
			alarmID,
			params,
		); err != nil {
			return dataAPIError("update alarm", err)
		}

		fmt.Println("Alarm updated successfully")
		return nil
	},
}

// alarmsDeleteCmd represents the alarms delete command
var alarmsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an alarm",
	Long:  `Delete an Insights alarm by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Alarms.Delete(context.Background(), alarmsProjectID, alarmID); err != nil {
			return dataAPIError("delete alarm", err)
		}

		fmt.Println("Alarm deleted successfully")
		return nil
	},
}

// alarmsHistoryCmd represents the alarms history command
var alarmsHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Get the trigger history for an alarm",
	Long: `Get the trigger history for a specific Insights alarm, newest first,
25 entries per page.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		response, err := client.Alarms.ListHistory(
			context.Background(),
			alarmsProjectID,
			alarmID,
			apiv3.Page(alarmHistoryPage, 0),
		)
		if err != nil {
			return dataAPIError("get alarm history", err)
		}

		if alarmsOutputFormat == "json" {
			return printJSON(response.Data)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tSTATE\tCREATED AT")
		for _, entry := range response.Data {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
				entry.Id,
				entry.Status.GetOrEmpty(),
				entry.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(alarmsCmd)
	alarmsCmd.AddCommand(alarmsListCmd)
	alarmsCmd.AddCommand(alarmsGetCmd)
	alarmsCmd.AddCommand(alarmsCreateCmd)
	alarmsCmd.AddCommand(alarmsUpdateCmd)
	alarmsCmd.AddCommand(alarmsDeleteCmd)
	alarmsCmd.AddCommand(alarmsHistoryCmd)

	// Common flags
	alarmsCmd.PersistentFlags().StringVar(&alarmsProjectID, "project-id", "", "Project ID")

	// Flags for list command
	alarmsListCmd.Flags().
		StringVarP(&alarmsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	alarmsGetCmd.Flags().StringVar(&alarmID, "id", "", "Alarm ID")
	alarmsGetCmd.Flags().
		StringVarP(&alarmsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = alarmsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	alarmsCreateCmd.Flags().
		StringVar(&alarmCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	alarmsCreateCmd.Flags().
		StringVarP(&alarmsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = alarmsCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for update command
	alarmsUpdateCmd.Flags().StringVar(&alarmID, "id", "", "Alarm ID")
	alarmsUpdateCmd.Flags().
		StringVar(&alarmCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = alarmsUpdateCmd.MarkFlagRequired("id")
	_ = alarmsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	alarmsDeleteCmd.Flags().StringVar(&alarmID, "id", "", "Alarm ID")
	_ = alarmsDeleteCmd.MarkFlagRequired("id")

	// Flags for history command
	alarmsHistoryCmd.Flags().StringVar(&alarmID, "id", "", "Alarm ID")
	alarmsHistoryCmd.Flags().
		IntVar(&alarmHistoryPage, "page", 0, "Page number for pagination (25 entries per page)")
	alarmsHistoryCmd.Flags().
		StringVarP(&alarmsOutputFormat, "output", "o", "table", "Output format (table or json)")
	_ = alarmsHistoryCmd.MarkFlagRequired("id")
}
