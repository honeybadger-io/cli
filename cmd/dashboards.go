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
	dashboardsProjectID    string
	dashboardID            string
	dashboardsOutputFormat string
	dashboardCLIInputJSON  string
)

// dashboardsCmd represents the dashboards command
var dashboardsCmd = &cobra.Command{
	Use:     "dashboards",
	Short:   "Manage Insights dashboards",
	GroupID: GroupDataAPI,
	Long:    `View and manage Insights dashboards for your Honeybadger projects.`,
}

// dashboardsListCmd represents the dashboards list command
var dashboardsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List dashboards for a project",
	Long: `List all Insights dashboards configured for a specific project.` + v3Reference(
		"dashboards/list-dashboards",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&dashboardsProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		dashboards, err := client.Dashboards.ListAll(context.Background(), dashboardsProjectID)
		if err != nil {
			return dataAPIError("list dashboards", err)
		}

		if dashboardsOutputFormat == "json" {
			return printJSON(dashboards)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tTITLE\tWIDGETS\tDEFAULT\tSHARED\tUPDATED")
		for _, dashboard := range dashboards {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
				dashboard.Id,
				dashboard.Title,
				len(dashboard.Widgets),
				checkmark(dashboard.IsDefault),
				checkmark(dashboard.Shared),
				dashboard.UpdatedAt.Format("2006-01-02 15:04"))
		}
		return w.Flush()
	},
}

// dashboardsGetCmd represents the dashboards get command
var dashboardsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a dashboard by ID",
	Long: `Get detailed information about a specific Insights dashboard.

Use --output json to get the full widget definitions. Its widgets list is in the
form 'hb dashboards update --cli-input-json' takes, so you can edit it and send it
back as {"widgets": [...]}.` + v3Reference("dashboards/get-dashboard"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		if dashboardID == "" {
			return fmt.Errorf("dashboard ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		dashboard, err := client.Dashboards.Get(
			context.Background(),
			dashboardsProjectID,
			dashboardID,
		)
		if err != nil {
			return dataAPIError("get dashboard", err)
		}

		if dashboardsOutputFormat == "json" {
			return printJSON(dashboard)
		}

		fmt.Printf("Dashboard Details:\n")
		fmt.Printf("  ID: %s\n", dashboard.Id)
		fmt.Printf("  Title: %s\n", dashboard.Title)
		fmt.Printf("  Project ID: %s\n", dashboard.ProjectId)
		fmt.Printf("  Default: %t\n", dashboard.IsDefault)
		fmt.Printf("  Shared: %t\n", dashboard.Shared)
		fmt.Printf("  Widgets: %d\n", len(dashboard.Widgets))
		fmt.Printf("  Created: %s\n", dashboard.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated: %s\n", dashboard.UpdatedAt.Format("2006-01-02 15:04:05"))

		if len(dashboard.Widgets) > 0 {
			fmt.Printf("\n")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "  WIDGET ID\tTYPE\tTITLE")
			for _, widget := range dashboard.Widgets {
				title := ""
				if widget.Presentation != nil {
					title = deref(widget.Presentation.Title)
				}
				_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\n", deref(widget.Id), widget.Type, title)
			}
			return w.Flush()
		}
		return nil
	},
}

// dashboardsCreateCmd represents the dashboards create command
var dashboardsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new dashboard",
	Long: `Create a new Insights dashboard for a project.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.
It takes the dashboard's fields directly; only title is required.

Widgets are laid out on a 12-column grid via their grid object (x, y, w, h); widgets
must not overlap. Omit a widget's id and the server assigns one.

Example JSON payload:
{
  "title": "Request Health",
  "default_ts": "P1D",
  "widgets": [
    {
      "type": "insights_vis",
      "grid": {"x": 0, "y": 0, "w": 6, "h": 4},
      "presentation": {"title": "Errors Over Time"},
      "config": {
        "streams": ["default"],
        "query": "filter event_type::str == \"notice\" | stats count() as count by bin(1h)",
        "vis": {"view": "line"}
      }
    }
  ]
}` + v3Reference("dashboards/create-dashboard"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		params, err := decodeInput[apiv3.DashboardCreateParams](dashboardCLIInputJSON, "dashboard")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		dashboard, err := client.Dashboards.Create(
			context.Background(),
			dashboardsProjectID,
			params,
		)
		if err != nil {
			return dataAPIError("create dashboard", err)
		}

		if dashboardsOutputFormat == "json" {
			return printJSON(dashboard)
		}
		fmt.Printf("Dashboard created successfully!\n")
		fmt.Printf("  ID: %s\n", dashboard.Id)
		fmt.Printf("  Title: %s\n", dashboard.Title)
		fmt.Printf("  Widgets: %d\n", len(dashboard.Widgets))
		return nil
	},
}

// dashboardsUpdateCmd represents the dashboards update command
var dashboardsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing dashboard",
	Long: `Update an existing Insights dashboard. Only the fields you send change;
a default_ts sent as null is cleared.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

A widgets list you send replaces the dashboard's widgets: any widget you leave out is
removed, and "widgets": [] removes them all. Keep each existing widget's id so it is
updated in place; a widget without an id is added as a new one. To edit the current
widgets, fetch them with:

  hb dashboards get --id <id> --output json

and send back its widgets list (not the whole dashboard: id, project_id and the other
read-only fields are refused).

Example JSON payload:
{
  "title": "Request Health",
  "widgets": [
    {
      "id": "existing-widget-id",
      "type": "insights_vis",
      "grid": {"x": 0, "y": 0, "w": 6, "h": 4},
      "presentation": {"title": "Errors Over Time"},
      "config": {
        "streams": ["default"],
        "query": "filter event_type::str == \"notice\" | stats count() as count by bin(1h)",
        "vis": {"view": "line"}
      }
    }
  ]
}` + v3Reference("dashboards/update-dashboard"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		if dashboardID == "" {
			return fmt.Errorf("dashboard ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.DashboardUpdateParams](dashboardCLIInputJSON, "dashboard")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if _, err := client.Dashboards.Update(
			context.Background(),
			dashboardsProjectID,
			dashboardID,
			params,
		); err != nil {
			return dataAPIError("update dashboard", err)
		}

		fmt.Println("Dashboard updated successfully")
		return nil
	},
}

// dashboardsDeleteCmd represents the dashboards delete command
var dashboardsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a dashboard",
	Long: `Delete an Insights dashboard by ID. This action cannot be undone.` + v3Reference(
		"dashboards/delete-dashboard",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		if dashboardID == "" {
			return fmt.Errorf("dashboard ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Dashboards.Delete(
			context.Background(),
			dashboardsProjectID,
			dashboardID,
		); err != nil {
			return dataAPIError("delete dashboard", err)
		}

		fmt.Println("Dashboard deleted successfully")
		return nil
	},
}

// checkmark renders a boolean as a table-friendly marker.
func checkmark(value bool) string {
	if value {
		return "✓"
	}
	return " "
}

func init() {
	rootCmd.AddCommand(dashboardsCmd)
	dashboardsCmd.AddCommand(dashboardsListCmd)
	dashboardsCmd.AddCommand(dashboardsGetCmd)
	dashboardsCmd.AddCommand(dashboardsCreateCmd)
	dashboardsCmd.AddCommand(dashboardsUpdateCmd)
	dashboardsCmd.AddCommand(dashboardsDeleteCmd)

	// Common flags
	dashboardsCmd.PersistentFlags().
		StringVar(&dashboardsProjectID, "project-id", "", "Project ID")

	// Flags for list command
	dashboardsListCmd.Flags().
		StringVarP(&dashboardsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	dashboardsGetCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID")
	dashboardsGetCmd.Flags().
		StringVarP(&dashboardsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = dashboardsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	dashboardsCreateCmd.Flags().
		StringVar(&dashboardCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	dashboardsCreateCmd.Flags().
		StringVarP(&dashboardsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = dashboardsCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for update command
	dashboardsUpdateCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID")
	dashboardsUpdateCmd.Flags().
		StringVar(&dashboardCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = dashboardsUpdateCmd.MarkFlagRequired("id")
	_ = dashboardsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	dashboardsDeleteCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID")
	_ = dashboardsDeleteCmd.MarkFlagRequired("id")
}
