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
	checkinsProjectID    int
	checkinID            string
	checkinsOutputFormat string
	checkinCLIInputJSON  string
)

// checkinsCmd represents the checkins command
var checkinsCmd = &cobra.Command{
	Use:     "checkins",
	Short:   "Manage Honeybadger check-ins",
	GroupID: GroupDataAPI,
	Long:    `View and manage check-ins (cron job monitoring) for your Honeybadger projects.`,
}

// checkinsListCmd represents the checkins list command
var checkinsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List check-ins for a project",
	Long:  `List all check-ins configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if checkinsProjectID == 0 {
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
		checkIns, err := client.CheckIns.List(ctx, checkinsProjectID)
		if err != nil {
			return fmt.Errorf("failed to list check-ins: %w", err)
		}

		switch checkinsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(checkIns, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tTYPE\tSCHEDULE\tLAST CHECK-IN")
			for _, ci := range checkIns {
				schedule := ""
				if ci.ScheduleType == "simple" && ci.ReportPeriod != nil {
					schedule = *ci.ReportPeriod
				} else if ci.ScheduleType == "cron" && ci.CronSchedule != nil {
					schedule = *ci.CronSchedule
				}

				lastCheckIn := "Never"
				if ci.ReportedAt != nil {
					lastCheckIn = ci.ReportedAt.Format("2006-01-02 15:04")
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					ci.ID,
					ci.Name,
					ci.Slug,
					ci.ScheduleType,
					schedule,
					lastCheckIn)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// checkinsGetCmd represents the checkins get command
var checkinsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a check-in by ID",
	Long:  `Get detailed information about a specific check-in.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if checkinsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if checkinID == "" {
			return fmt.Errorf("check-in ID is required. Set it using --id flag")
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
		checkIn, err := client.CheckIns.Get(ctx, checkinsProjectID, checkinID)
		if err != nil {
			return fmt.Errorf("failed to get check-in: %w", err)
		}

		switch checkinsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(checkIn, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Check-in Details:\n")
			fmt.Printf("  ID: %s\n", checkIn.ID)
			fmt.Printf("  Name: %s\n", checkIn.Name)
			fmt.Printf("  Slug: %s\n", checkIn.Slug)
			fmt.Printf("  State: %s\n", checkIn.State)
			fmt.Printf("  Schedule Type: %s\n", checkIn.ScheduleType)
			if checkIn.ReportPeriod != nil {
				fmt.Printf("  Report Period: %s\n", *checkIn.ReportPeriod)
			}
			if checkIn.GracePeriod != nil {
				fmt.Printf("  Grace Period: %s\n", *checkIn.GracePeriod)
			}
			if checkIn.CronSchedule != nil {
				fmt.Printf("  Cron Schedule: %s\n", *checkIn.CronSchedule)
			}
			if checkIn.CronTimezone != nil {
				fmt.Printf("  Cron Timezone: %s\n", *checkIn.CronTimezone)
			}
			if checkIn.ReportedAt != nil {
				fmt.Printf(
					"  Last Check-in: %s\n",
					checkIn.ReportedAt.Format("2006-01-02 15:04:05"),
				)
			}
		}

		return nil
	},
}

// checkinsCreateCmd represents the checkins create command
var checkinsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new check-in",
	Long: `Create a new check-in for a project.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload for simple schedule:
{
  "check_in": {
    "name": "Daily Backup",
    "slug": "daily-backup",
    "schedule_type": "simple",
    "report_period": "1 day",
    "grace_period": "15 minutes"
  }
}

Example JSON payload for cron schedule:
{
  "check_in": {
    "name": "Hourly Task",
    "slug": "hourly-task",
    "schedule_type": "cron",
    "cron_schedule": "0 * * * *",
    "cron_timezone": "America/New_York"
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if checkinsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if checkinCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(checkinCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			CheckIn hbapi.CheckInParams `json:"check_in"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		checkIn, err := client.CheckIns.Create(ctx, checkinsProjectID, payload.CheckIn)
		if err != nil {
			return fmt.Errorf("failed to create check-in: %w", err)
		}

		switch checkinsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(checkIn, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Check-in created successfully!\n")
			fmt.Printf("  ID: %s\n", checkIn.ID)
			fmt.Printf("  Name: %s\n", checkIn.Name)
			fmt.Printf("  Slug: %s\n", checkIn.Slug)
		}

		return nil
	},
}

// checkinsUpdateCmd represents the checkins update command
var checkinsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing check-in",
	Long: `Update an existing check-in's settings.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "check_in": {
    "name": "Updated Name",
    "report_period": "2 days"
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if checkinsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if checkinID == "" {
			return fmt.Errorf("check-in ID is required. Set it using --id flag")
		}
		if checkinCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(checkinCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			CheckIn hbapi.CheckInParams `json:"check_in"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		checkIn, err := client.CheckIns.Update(ctx, checkinsProjectID, checkinID, payload.CheckIn)
		if err != nil {
			return fmt.Errorf("failed to update check-in: %w", err)
		}

		switch checkinsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(checkIn, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Check-in updated successfully!\n")
			fmt.Printf("  ID: %s\n", checkIn.ID)
			fmt.Printf("  Name: %s\n", checkIn.Name)
			fmt.Printf("  Slug: %s\n", checkIn.Slug)
		}

		return nil
	},
}

// checkinsDeleteCmd represents the checkins delete command
var checkinsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a check-in",
	Long:  `Delete a check-in by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if checkinsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if checkinID == "" {
			return fmt.Errorf("check-in ID is required. Set it using --id flag")
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
		err := client.CheckIns.Delete(ctx, checkinsProjectID, checkinID)
		if err != nil {
			return fmt.Errorf("failed to delete check-in: %w", err)
		}

		fmt.Println("Check-in deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkinsCmd)
	checkinsCmd.AddCommand(checkinsListCmd)
	checkinsCmd.AddCommand(checkinsGetCmd)
	checkinsCmd.AddCommand(checkinsCreateCmd)
	checkinsCmd.AddCommand(checkinsUpdateCmd)
	checkinsCmd.AddCommand(checkinsDeleteCmd)

	// Common flags
	checkinsCmd.PersistentFlags().IntVar(&checkinsProjectID, "project-id", 0, "Project ID")

	// Flags for list command
	checkinsListCmd.Flags().
		StringVarP(&checkinsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	checkinsGetCmd.Flags().StringVar(&checkinID, "id", "", "Check-in ID")
	checkinsGetCmd.Flags().
		StringVarP(&checkinsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = checkinsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	checkinsCreateCmd.Flags().
		StringVar(&checkinCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	checkinsCreateCmd.Flags().
		StringVarP(&checkinsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = checkinsCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for update command
	checkinsUpdateCmd.Flags().StringVar(&checkinID, "id", "", "Check-in ID")
	checkinsUpdateCmd.Flags().
		StringVar(&checkinCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	checkinsUpdateCmd.Flags().
		StringVarP(&checkinsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = checkinsUpdateCmd.MarkFlagRequired("id")
	_ = checkinsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	checkinsDeleteCmd.Flags().StringVar(&checkinID, "id", "", "Check-in ID")
	_ = checkinsDeleteCmd.MarkFlagRequired("id")
}
