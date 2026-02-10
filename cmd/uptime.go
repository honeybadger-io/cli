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
	uptimeProjectID     int
	uptimeSiteID        string
	uptimeOutputFormat  string
	uptimeCLIInputJSON  string
	uptimeCreatedAfter  int64
	uptimeCreatedBefore int64
	uptimeLimit         int
)

// uptimeCmd represents the uptime command
var uptimeCmd = &cobra.Command{
	Use:     "uptime",
	Short:   "Manage uptime monitoring",
	GroupID: GroupDataAPI,
	Long:    `View and manage uptime monitoring sites, outages, and checks for your Honeybadger projects.`,
}

// uptimeSitesCmd represents the uptime sites parent command
var uptimeSitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "Manage uptime sites",
	Long:  `View and manage uptime monitoring sites.`,
}

// uptimeSitesListCmd represents the uptime sites list command
var uptimeSitesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List uptime sites for a project",
	Long:  `List all uptime monitoring sites configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if uptimeProjectID == 0 {
			uptimeProjectID = viper.GetInt("project_id")
		}
		if uptimeProjectID == 0 {
			return fmt.Errorf(
				"project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable",
			)
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
		sites, err := client.Uptime.List(ctx, uptimeProjectID)
		if err != nil {
			return fmt.Errorf("failed to list uptime sites: %w", err)
		}

		switch uptimeOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(sites, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tURL\tSTATE\tACTIVE\tFREQ")
			for _, site := range sites {
				active := " "
				if site.Active {
					active = "Yes"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%dm\n",
					site.ID,
					site.Name,
					site.URL,
					site.State,
					active,
					site.Frequency)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// uptimeSitesGetCmd represents the uptime sites get command
var uptimeSitesGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an uptime site by ID",
	Long:  `Get detailed information about a specific uptime site.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if uptimeProjectID == 0 {
			uptimeProjectID = viper.GetInt("project_id")
		}
		if uptimeProjectID == 0 {
			return fmt.Errorf(
				"project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable",
			)
		}
		if uptimeSiteID == "" {
			return fmt.Errorf("site ID is required. Set it using --site-id flag")
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
		site, err := client.Uptime.Get(ctx, uptimeProjectID, uptimeSiteID)
		if err != nil {
			return fmt.Errorf("failed to get uptime site: %w", err)
		}

		switch uptimeOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(site, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Uptime Site Details:\n")
			fmt.Printf("  ID: %s\n", site.ID)
			fmt.Printf("  Name: %s\n", site.Name)
			fmt.Printf("  URL: %s\n", site.URL)
			fmt.Printf("  State: %s\n", site.State)
			fmt.Printf("  Active: %v\n", site.Active)
			fmt.Printf("  Frequency: %d minutes\n", site.Frequency)
			fmt.Printf("  Match Type: %s\n", site.MatchType)
			if site.Match != nil {
				fmt.Printf("  Match: %s\n", *site.Match)
			}
			if site.LastCheckedAt != nil {
				fmt.Printf("  Last Checked: %s\n", site.LastCheckedAt.Format("2006-01-02 15:04:05"))
			}
		}

		return nil
	},
}

// uptimeSitesCreateCmd represents the uptime sites create command
var uptimeSitesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new uptime site",
	Long: `Create a new uptime monitoring site for a project.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "site": {
    "name": "My Website",
    "url": "https://example.com",
    "frequency": 5,
    "match_type": "success",
    "locations": ["Virginia", "Oregon"],
    "validate_ssl": true
  }
}

Available options:
- frequency: 1, 5, or 15 (minutes)
- match_type: "success", "exact", "include", "exclude"
- request_method: "GET", "POST", "PUT", "PATCH", "DELETE"
- locations: "Virginia", "Oregon", "Frankfurt", "Singapore", "London"`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if uptimeProjectID == 0 {
			uptimeProjectID = viper.GetInt("project_id")
		}
		if uptimeProjectID == 0 {
			return fmt.Errorf(
				"project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable",
			)
		}
		if uptimeCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(uptimeCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Site hbapi.SiteParams `json:"site"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		site, err := client.Uptime.Create(ctx, uptimeProjectID, payload.Site)
		if err != nil {
			return fmt.Errorf("failed to create uptime site: %w", err)
		}

		switch uptimeOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(site, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Uptime site created successfully!\n")
			fmt.Printf("  ID: %s\n", site.ID)
			fmt.Printf("  Name: %s\n", site.Name)
			fmt.Printf("  URL: %s\n", site.URL)
		}

		return nil
	},
}

// uptimeSitesUpdateCmd represents the uptime sites update command
var uptimeSitesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing uptime site",
	Long: `Update an existing uptime monitoring site's settings.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "site": {
    "name": "Updated Website",
    "frequency": 15,
    "active": false
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if uptimeProjectID == 0 {
			uptimeProjectID = viper.GetInt("project_id")
		}
		if uptimeProjectID == 0 {
			return fmt.Errorf(
				"project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable",
			)
		}
		if uptimeSiteID == "" {
			return fmt.Errorf("site ID is required. Set it using --site-id flag")
		}
		if uptimeCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(uptimeCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Site hbapi.SiteParams `json:"site"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		site, err := client.Uptime.Update(ctx, uptimeProjectID, uptimeSiteID, payload.Site)
		if err != nil {
			return fmt.Errorf("failed to update uptime site: %w", err)
		}

		switch uptimeOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(site, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Uptime site updated successfully!\n")
			fmt.Printf("  ID: %s\n", site.ID)
			fmt.Printf("  Name: %s\n", site.Name)
		}

		return nil
	},
}

// uptimeSitesDeleteCmd represents the uptime sites delete command
var uptimeSitesDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an uptime site",
	Long:  `Delete an uptime monitoring site by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if uptimeProjectID == 0 {
			uptimeProjectID = viper.GetInt("project_id")
		}
		if uptimeProjectID == 0 {
			return fmt.Errorf(
				"project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable",
			)
		}
		if uptimeSiteID == "" {
			return fmt.Errorf("site ID is required. Set it using --site-id flag")
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
		err := client.Uptime.Delete(ctx, uptimeProjectID, uptimeSiteID)
		if err != nil {
			return fmt.Errorf("failed to delete uptime site: %w", err)
		}

		fmt.Println("Uptime site deleted successfully")
		return nil
	},
}

// uptimeOutagesCmd represents the uptime outages command
var uptimeOutagesCmd = &cobra.Command{
	Use:   "outages",
	Short: "List outages for a site",
	Long:  `List outages recorded for a specific uptime monitoring site.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if uptimeProjectID == 0 {
			uptimeProjectID = viper.GetInt("project_id")
		}
		if uptimeProjectID == 0 {
			return fmt.Errorf(
				"project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable",
			)
		}
		if uptimeSiteID == "" {
			return fmt.Errorf("site ID is required. Set it using --site-id flag")
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

		options := hbapi.OutageListOptions{
			CreatedAfter:  uptimeCreatedAfter,
			CreatedBefore: uptimeCreatedBefore,
			Limit:         uptimeLimit,
		}

		ctx := context.Background()
		outages, err := client.Uptime.ListOutages(ctx, uptimeProjectID, uptimeSiteID, options)
		if err != nil {
			return fmt.Errorf("failed to list outages: %w", err)
		}

		switch uptimeOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(outages, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "DOWN AT\tUP AT\tSTATUS\tREASON")
			for _, outage := range outages {
				upAt := "Still down"
				if outage.UpAt != nil {
					upAt = outage.UpAt.Format("2006-01-02 15:04")
				}

				reason := outage.Reason
				if len(reason) > 40 {
					reason = reason[:37] + "..."
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
					outage.DownAt.Format("2006-01-02 15:04"),
					upAt,
					outage.Status,
					reason)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// uptimeChecksCmd represents the uptime checks command
var uptimeChecksCmd = &cobra.Command{
	Use:   "checks",
	Short: "List uptime checks for a site",
	Long:  `List individual uptime checks performed for a specific site.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if uptimeProjectID == 0 {
			uptimeProjectID = viper.GetInt("project_id")
		}
		if uptimeProjectID == 0 {
			return fmt.Errorf(
				"project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable",
			)
		}
		if uptimeSiteID == "" {
			return fmt.Errorf("site ID is required. Set it using --site-id flag")
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

		options := hbapi.UptimeCheckListOptions{
			CreatedAfter:  uptimeCreatedAfter,
			CreatedBefore: uptimeCreatedBefore,
			Limit:         uptimeLimit,
		}

		ctx := context.Background()
		checks, err := client.Uptime.ListUptimeChecks(ctx, uptimeProjectID, uptimeSiteID, options)
		if err != nil {
			return fmt.Errorf("failed to list uptime checks: %w", err)
		}

		switch uptimeOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(checks, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "CREATED\tLOCATION\tUP\tDURATION")
			for _, check := range checks {
				up := "No"
				if check.Up {
					up = "Yes"
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%dms\n",
					check.CreatedAt.Format("2006-01-02 15:04:05"),
					check.Location,
					up,
					check.Duration)
			}
			_ = w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uptimeCmd)

	// Add subcommands
	uptimeCmd.AddCommand(uptimeSitesCmd)
	uptimeCmd.AddCommand(uptimeOutagesCmd)
	uptimeCmd.AddCommand(uptimeChecksCmd)

	// Sites subcommands
	uptimeSitesCmd.AddCommand(uptimeSitesListCmd)
	uptimeSitesCmd.AddCommand(uptimeSitesGetCmd)
	uptimeSitesCmd.AddCommand(uptimeSitesCreateCmd)
	uptimeSitesCmd.AddCommand(uptimeSitesUpdateCmd)
	uptimeSitesCmd.AddCommand(uptimeSitesDeleteCmd)

	// Common flags
	uptimeCmd.PersistentFlags().IntVar(&uptimeProjectID, "project-id", 0, "Project ID")

	// Flags for sites list
	uptimeSitesListCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for sites get
	uptimeSitesGetCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID")
	uptimeSitesGetCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = uptimeSitesGetCmd.MarkFlagRequired("site-id")

	// Flags for sites create
	uptimeSitesCreateCmd.Flags().
		StringVar(&uptimeCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	uptimeSitesCreateCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = uptimeSitesCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for sites update
	uptimeSitesUpdateCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID")
	uptimeSitesUpdateCmd.Flags().
		StringVar(&uptimeCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	uptimeSitesUpdateCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = uptimeSitesUpdateCmd.MarkFlagRequired("site-id")
	_ = uptimeSitesUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for sites delete
	uptimeSitesDeleteCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID")
	_ = uptimeSitesDeleteCmd.MarkFlagRequired("site-id")

	// Flags for outages
	uptimeOutagesCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID")
	uptimeOutagesCmd.Flags().
		Int64Var(&uptimeCreatedAfter, "created-after", 0, "Filter by creation time (Unix timestamp)")
	uptimeOutagesCmd.Flags().
		Int64Var(&uptimeCreatedBefore, "created-before", 0, "Filter by creation time (Unix timestamp)")
	uptimeOutagesCmd.Flags().
		IntVar(&uptimeLimit, "limit", 25, "Maximum number of outages to return (max 25)")
	uptimeOutagesCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "table", "Output format (table or json)")
	_ = uptimeOutagesCmd.MarkFlagRequired("site-id")

	// Flags for checks
	uptimeChecksCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID")
	uptimeChecksCmd.Flags().
		Int64Var(&uptimeCreatedAfter, "created-after", 0, "Filter by creation time (Unix timestamp)")
	uptimeChecksCmd.Flags().
		Int64Var(&uptimeCreatedBefore, "created-before", 0, "Filter by creation time (Unix timestamp)")
	uptimeChecksCmd.Flags().
		IntVar(&uptimeLimit, "limit", 25, "Maximum number of checks to return (max 25)")
	uptimeChecksCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "table", "Output format (table or json)")
	_ = uptimeChecksCmd.MarkFlagRequired("site-id")
}
