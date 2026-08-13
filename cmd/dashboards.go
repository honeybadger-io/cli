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
	dashboardsProjectID    int
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
	Long:  `List all Insights dashboards configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&dashboardsProjectID); err != nil {
			return err
		}

		client, err := newDashboardsClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		response, err := client.Dashboards.List(ctx, dashboardsProjectID)
		if err != nil {
			return fmt.Errorf("failed to list dashboards: %w", err)
		}

		switch dashboardsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(response.Results, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tTITLE\tWIDGETS\tDEFAULT\tSHARED\tUPDATED")
			for _, dashboard := range response.Results {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
					dashboard.ID,
					dashboard.Title,
					len(dashboard.Widgets),
					checkmark(dashboard.IsDefault),
					checkmark(dashboard.Shared),
					dashboard.UpdatedAt.Format("2006-01-02 15:04"))
			}
			_ = w.Flush()
		}

		return nil
	},
}

// dashboardsGetCmd represents the dashboards get command
var dashboardsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a dashboard by ID",
	Long: `Get detailed information about a specific Insights dashboard.

Use --output json to get the full widget definitions, which is the form
'hb dashboards update --cli-input-json' expects.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		if dashboardID == "" {
			return fmt.Errorf("dashboard ID is required. Set it using --id flag")
		}

		client, err := newDashboardsClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		dashboard, err := client.Dashboards.Get(ctx, dashboardsProjectID, dashboardID)
		if err != nil {
			return fmt.Errorf("failed to get dashboard: %w", err)
		}

		switch dashboardsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(dashboard, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Dashboard Details:\n")
			fmt.Printf("  ID: %s\n", dashboard.ID)
			fmt.Printf("  Title: %s\n", dashboard.Title)
			fmt.Printf("  Project ID: %d\n", dashboard.ProjectID)
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
					_, _ = fmt.Fprintf(w, "  %s\t%s\t%s\n",
						widgetString(widget, "id"),
						widgetString(widget, "type"),
						widgetTitle(widget))
				}
				_ = w.Flush()
			}
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

Widgets are laid out on a 12-column grid via their grid object (x, y, w, h); widgets
must not overlap. Omit a widget's id and the server assigns one.

Example JSON payload:
{
  "dashboard": {
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
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		if dashboardCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
		}

		client, err := newDashboardsClient()
		if err != nil {
			return err
		}

		request, err := parseDashboardRequest(dashboardCLIInputJSON)
		if err != nil {
			return err
		}

		ctx := context.Background()
		dashboard, err := client.Dashboards.Create(ctx, dashboardsProjectID, request)
		if err != nil {
			return fmt.Errorf("failed to create dashboard: %w", err)
		}

		switch dashboardsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(dashboard, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Dashboard created successfully!\n")
			fmt.Printf("  ID: %s\n", dashboard.ID)
			fmt.Printf("  Title: %s\n", dashboard.Title)
			fmt.Printf("  Widgets: %d\n", len(dashboard.Widgets))
		}

		return nil
	},
}

// dashboardsUpdateCmd represents the dashboards update command
var dashboardsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing dashboard",
	Long: `Update an existing Insights dashboard.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

The request replaces the dashboard, so send the complete widget list -- any widget you
omit is dropped. Fetch the current state first with:

  hb dashboards get --id <id> --output json

Keep each existing widget's id in the payload so it is updated in place rather than
replaced by a newly assigned one.

Example JSON payload:
{
  "dashboard": {
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
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		if dashboardID == "" {
			return fmt.Errorf("dashboard ID is required. Set it using --id flag")
		}
		if dashboardCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
		}

		client, err := newDashboardsClient()
		if err != nil {
			return err
		}

		request, err := parseDashboardRequest(dashboardCLIInputJSON)
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := client.Dashboards.Update(ctx, dashboardsProjectID, dashboardID, request)
		if err != nil {
			return fmt.Errorf("failed to update dashboard: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

// dashboardsDeleteCmd represents the dashboards delete command
var dashboardsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a dashboard",
	Long:  `Delete an Insights dashboard by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveProjectID(&dashboardsProjectID); err != nil {
			return err
		}
		if dashboardID == "" {
			return fmt.Errorf("dashboard ID is required. Set it using --id flag")
		}

		client, err := newDashboardsClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		result, err := client.Dashboards.Delete(ctx, dashboardsProjectID, dashboardID)
		if err != nil {
			return fmt.Errorf("failed to delete dashboard: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

// newDashboardsClient builds a Data API client, returning an error when no auth token is set.
func newDashboardsClient() (*hbapi.Client, error) {
	authToken := viper.GetString("auth_token")
	if authToken == "" {
		return nil, fmt.Errorf(
			"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
		)
	}

	endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

	return hbapi.NewClient().
		WithBaseURL(endpoint).
		WithAuthToken(authToken), nil
}

// parseDashboardRequest reads and unmarshals a dashboard payload from a JSON string or file://path.
func parseDashboardRequest(input string) (hbapi.DashboardRequest, error) {
	jsonData, err := readJSONInput(input)
	if err != nil {
		return hbapi.DashboardRequest{}, fmt.Errorf("failed to read JSON input: %w", err)
	}

	var payload struct {
		Dashboard hbapi.DashboardRequest `json:"dashboard"`
	}
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return hbapi.DashboardRequest{}, fmt.Errorf("failed to parse JSON payload: %w", err)
	}

	return payload.Dashboard, nil
}

// checkmark renders a boolean as a table-friendly marker.
func checkmark(value bool) string {
	if value {
		return "✓"
	}
	return " "
}

// widgetString safely reads a top-level string field from a widget definition.
func widgetString(widget map[string]interface{}, key string) string {
	value, _ := widget[key].(string)
	return value
}

// widgetTitle safely reads presentation.title from a widget definition.
func widgetTitle(widget map[string]interface{}) string {
	presentation, ok := widget["presentation"].(map[string]interface{})
	if !ok {
		return ""
	}
	title, _ := presentation["title"].(string)
	return title
}

func init() {
	rootCmd.AddCommand(dashboardsCmd)
	dashboardsCmd.AddCommand(dashboardsListCmd)
	dashboardsCmd.AddCommand(dashboardsGetCmd)
	dashboardsCmd.AddCommand(dashboardsCreateCmd)
	dashboardsCmd.AddCommand(dashboardsUpdateCmd)
	dashboardsCmd.AddCommand(dashboardsDeleteCmd)

	// Common flags
	dashboardsCmd.PersistentFlags().IntVar(&dashboardsProjectID, "project-id", 0, "Project ID")

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
