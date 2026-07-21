package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/spf13/cobra"
)

var (
	alarmsProjectID    int
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
		if err := resolveProjectID(&alarmsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		response, err := client.Alarms.List(ctx, alarmsProjectID)
		if err != nil {
			return fmt.Errorf("failed to list alarms: %w", err)
		}

		switch alarmsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(response.Results, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSTATE\tQUERY\tEVAL PERIOD\tLAST CHECKED")
			for _, alarm := range response.Results {
				query := alarm.Query
				if len(query) > 50 {
					query = query[:47] + "..."
				}

				lastChecked := "Never"
				if alarm.LastCheckedAt != nil {
					lastChecked = alarm.LastCheckedAt.Format("2006-01-02 15:04")
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					alarm.ID,
					alarm.Name,
					alarm.State,
					query,
					alarm.EvaluationPeriod,
					lastChecked)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// alarmsGetCmd represents the alarms get command
var alarmsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an alarm by ID",
	Long:  `Get detailed information about a specific Insights alarm.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		alarm, err := client.Alarms.Get(ctx, alarmsProjectID, alarmID)
		if err != nil {
			return fmt.Errorf("failed to get alarm: %w", err)
		}

		switch alarmsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(alarm, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Alarm Details:\n")
			fmt.Printf("  ID: %s\n", alarm.ID)
			fmt.Printf("  Name: %s\n", alarm.Name)
			if alarm.Description != "" {
				fmt.Printf("  Description: %s\n", alarm.Description)
			}
			fmt.Printf("  State: %s\n", alarm.State)
			fmt.Printf("  Query: %s\n", alarm.Query)
			if len(alarm.StreamIDs) > 0 {
				fmt.Printf("  Stream IDs: %s\n", strings.Join(alarm.StreamIDs, ", "))
			}
			fmt.Printf("  Evaluation Period: %s\n", alarm.EvaluationPeriod)
			if alarm.LookbackLag != "" {
				fmt.Printf("  Lookback Lag: %s\n", alarm.LookbackLag)
			}
			if alarm.TriggerConfig != nil {
				if triggerConfigJSON, err := json.Marshal(alarm.TriggerConfig); err == nil {
					fmt.Printf("  Trigger Config: %s\n", string(triggerConfigJSON))
				}
			}
			if alarm.Error != "" {
				fmt.Printf("  Error: %s\n", alarm.Error)
			}
			if alarm.LastCheckedAt != nil {
				fmt.Printf(
					"  Last Checked: %s\n",
					alarm.LastCheckedAt.Format("2006-01-02 15:04:05"),
				)
			}
			if alarm.NextCheckAt != nil {
				fmt.Printf("  Next Check: %s\n", alarm.NextCheckAt.Format("2006-01-02 15:04:05"))
			}
			fmt.Printf("  Created: %s\n", alarm.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Updated: %s\n", alarm.UpdatedAt.Format("2006-01-02 15:04:05"))
			if alarm.URL != "" {
				fmt.Printf("  URL: %s\n", alarm.URL)
			}
		}

		return nil
	},
}

// alarmsCreateCmd represents the alarms create command
var alarmsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new alarm",
	Long: `Create a new Insights alarm for a project.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

stream_ids is required and must be a non-empty array of the project's Insights stream
IDs; an empty or omitted value is rejected by the API. Find the IDs in the Insights UI
or via a BadgerQL query like "stats count() by @stream.id, @stream.name".

Example JSON payload:
{
  "alarm": {
    "name": "High Error Rate",
    "description": "Alert when errors spike",
    "query": "filter event_type::str == \"notice\"",
    "stream_ids": ["<stream-id>"],
    "evaluation_period": "5m",
    "lookback_lag": "1m",
    "trigger_config": {
      "type": "alert_result_count",
      "config": {"operator": "gt", "value": 10}
    }
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		jsonData, err := readJSONInput(alarmCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Alarm hbapi.AlarmRequest `json:"alarm"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		alarm, err := client.Alarms.Create(ctx, alarmsProjectID, payload.Alarm)
		if err != nil {
			return fmt.Errorf("failed to create alarm: %w", err)
		}

		switch alarmsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(alarm, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Alarm created successfully!\n")
			fmt.Printf("  ID: %s\n", alarm.ID)
			fmt.Printf("  Name: %s\n", alarm.Name)
			fmt.Printf("  State: %s\n", alarm.State)
		}

		return nil
	},
}

// alarmsUpdateCmd represents the alarms update command
var alarmsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing alarm",
	Long: `Update an existing Insights alarm's settings.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

stream_ids is required and must be a non-empty array of the project's Insights stream
IDs; an empty or omitted value is rejected by the API. Find the IDs in the Insights UI
or via a BadgerQL query like "stats count() by @stream.id, @stream.name".

Example JSON payload:
{
  "alarm": {
    "name": "Updated Alarm Name",
    "query": "filter event_type::str == \"notice\"",
    "stream_ids": ["<stream-id>"],
    "evaluation_period": "10m",
    "lookback_lag": "1m",
    "trigger_config": {
      "type": "alert_result_count",
      "config": {"operator": "gt", "value": 25}
    }
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}
		if alarmCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		jsonData, err := readJSONInput(alarmCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Alarm hbapi.AlarmRequest `json:"alarm"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		result, err := client.Alarms.Update(ctx, alarmsProjectID, alarmID, payload.Alarm)
		if err != nil {
			return fmt.Errorf("failed to update alarm: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

// alarmsDeleteCmd represents the alarms delete command
var alarmsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an alarm",
	Long:  `Delete an Insights alarm by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := client.Alarms.Delete(ctx, alarmsProjectID, alarmID)
		if err != nil {
			return fmt.Errorf("failed to delete alarm: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

// alarmsHistoryCmd represents the alarms history command
var alarmsHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Get the trigger history for an alarm",
	Long:  `Get the trigger history for a specific Insights alarm.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&alarmsProjectID); err != nil {
			return err
		}
		if alarmID == "" {
			return fmt.Errorf("alarm ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		response, err := client.Alarms.History(ctx, alarmsProjectID, alarmID, alarmHistoryPage)
		if err != nil {
			return fmt.Errorf("failed to get alarm history: %w", err)
		}

		switch alarmsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(response.Triggers, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tSTATE\tCREATED AT")
			for _, trigger := range response.Triggers {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
					trigger.ID,
					trigger.State,
					trigger.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			_ = w.Flush()
		}

		return nil
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
	alarmsCmd.PersistentFlags().IntVar(&alarmsProjectID, "project-id", 0, "Project ID")

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
	alarmsHistoryCmd.Flags().IntVar(&alarmHistoryPage, "page", 0, "Page number for pagination")
	alarmsHistoryCmd.Flags().
		StringVarP(&alarmsOutputFormat, "output", "o", "table", "Output format (table or json)")
	_ = alarmsHistoryCmd.MarkFlagRequired("id")
}
