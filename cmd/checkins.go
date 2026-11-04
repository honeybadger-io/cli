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
	checkinsProjectID    string
	checkinID            string
	checkinsOutputFormat string
	checkinCLIInputJSON  string
)

// checkinsCmd represents the checkins command
var checkinsCmd = &cobra.Command{
	Use:     "check-ins",
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
		if err := resolveDataAPIProjectID(&checkinsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		checkIns, err := client.CheckIns.ListAll(context.Background(), checkinsProjectID)
		if err != nil {
			return dataAPIError("list check-ins", err)
		}

		if checkinsOutputFormat == "json" {
			return printJSON(checkIns)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tSLUG\tTYPE\tSCHEDULE\tLAST CHECK-IN")
		for _, ci := range checkIns {
			schedule := ""
			switch ci.ScheduleType {
			case apiv3.ScheduleSimple:
				schedule = ci.ReportPeriod.GetOrEmpty()
			case apiv3.ScheduleCron:
				schedule = ci.CronSchedule.GetOrEmpty()
			}

			lastCheckIn := "Never"
			if at, err := ci.ReportedAt.Get(); err == nil {
				lastCheckIn = at.Format("2006-01-02 15:04")
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				ci.Id,
				ci.Name.GetOrEmpty(),
				ci.Slug.GetOrEmpty(),
				ci.ScheduleType,
				schedule,
				lastCheckIn)
		}
		return w.Flush()
	},
}

// checkinsGetCmd represents the checkins get command
var checkinsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a check-in by ID",
	Long:  `Get detailed information about a specific check-in.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&checkinsProjectID); err != nil {
			return err
		}
		if checkinID == "" {
			return fmt.Errorf("check-in ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		checkIn, err := client.CheckIns.Get(context.Background(), checkinsProjectID, checkinID)
		if err != nil {
			return dataAPIError("get check-in", err)
		}

		if checkinsOutputFormat == "json" {
			return printJSON(checkIn)
		}

		fmt.Printf("Check-in Details:\n")
		fmt.Printf("  ID: %s\n", checkIn.Id)
		fmt.Printf("  Name: %s\n", checkIn.Name.GetOrEmpty())
		fmt.Printf("  Slug: %s\n", checkIn.Slug.GetOrEmpty())
		fmt.Printf("  State: %s\n", checkIn.State)
		fmt.Printf("  Schedule Type: %s\n", checkIn.ScheduleType)
		if period, err := checkIn.ReportPeriod.Get(); err == nil {
			fmt.Printf("  Report Period: %s\n", period)
		}
		fmt.Printf("  Grace Period: %s\n", checkIn.GracePeriod)
		if schedule, err := checkIn.CronSchedule.Get(); err == nil {
			fmt.Printf("  Cron Schedule: %s\n", schedule)
			fmt.Printf("  Cron Timezone: %s\n", checkIn.CronTimezone)
		}
		if at, err := checkIn.ReportedAt.Get(); err == nil {
			fmt.Printf("  Last Check-in: %s\n", at.Format("2006-01-02 15:04:05"))
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
It takes the check-in's fields directly. A simple schedule needs report_period; a cron
schedule needs cron_schedule. cron_timezone is a Rails time zone name such as
"Eastern Time (US & Canada)", not an IANA name such as "America/New_York".

Example JSON payload for simple schedule:
{
  "name": "Daily Backup",
  "slug": "daily-backup",
  "schedule_type": "simple",
  "report_period": "1 day",
  "grace_period": "15 minutes"
}

Example JSON payload for cron schedule:
{
  "name": "Hourly Task",
  "slug": "hourly-task",
  "schedule_type": "cron",
  "cron_schedule": "0 * * * *",
  "cron_timezone": "Eastern Time (US & Canada)"
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&checkinsProjectID); err != nil {
			return err
		}
		params, err := decodeInput[apiv3.CheckInCreateParams](checkinCLIInputJSON, "check_in")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		checkIn, err := client.CheckIns.Create(context.Background(), checkinsProjectID, params)
		if err != nil {
			return dataAPIError("create check-in", err)
		}

		if checkinsOutputFormat == "json" {
			return printJSON(checkIn)
		}
		fmt.Printf("Check-in created successfully!\n")
		fmt.Printf("  ID: %s\n", checkIn.Id)
		fmt.Printf("  Name: %s\n", checkIn.Name.GetOrEmpty())
		fmt.Printf("  Slug: %s\n", checkIn.Slug.GetOrEmpty())
		return nil
	},
}

// checkinsUpdateCmd represents the checkins update command
var checkinsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing check-in",
	Long: `Update an existing check-in's settings. Only the fields you send change;
a field sent as null is cleared.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "name": "Updated Name",
  "report_period": "2 days"
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&checkinsProjectID); err != nil {
			return err
		}
		if checkinID == "" {
			return fmt.Errorf("check-in ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.CheckInUpdateParams](checkinCLIInputJSON, "check_in")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		checkIn, err := client.CheckIns.Update(
			context.Background(),
			checkinsProjectID,
			checkinID,
			params,
		)
		if err != nil {
			return dataAPIError("update check-in", err)
		}

		if checkinsOutputFormat == "json" {
			return printJSON(checkIn)
		}
		fmt.Printf("Check-in updated successfully!\n")
		fmt.Printf("  ID: %s\n", checkIn.Id)
		fmt.Printf("  Name: %s\n", checkIn.Name.GetOrEmpty())
		fmt.Printf("  Slug: %s\n", checkIn.Slug.GetOrEmpty())
		return nil
	},
}

// checkinsDeleteCmd represents the checkins delete command
var checkinsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a check-in",
	Long:  `Delete a check-in by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&checkinsProjectID); err != nil {
			return err
		}
		if checkinID == "" {
			return fmt.Errorf("check-in ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.CheckIns.Delete(
			context.Background(),
			checkinsProjectID,
			checkinID,
		); err != nil {
			return dataAPIError("delete check-in", err)
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
	checkinsCmd.PersistentFlags().StringVar(&checkinsProjectID, "project-id", "", "Project ID")

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
