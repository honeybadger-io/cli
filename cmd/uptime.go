package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/spf13/cobra"
)

var (
	uptimeProjectID     string
	uptimeSiteID        string
	uptimeOutputFormat  string
	uptimeCLIInputJSON  string
	uptimeCreatedBefore string
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

// parseSiteID parses --site-id. v3 site ids are UUIDs, so anything else is
// refused here rather than sent.
func parseSiteID(id string) (apiv3.SiteID, error) {
	var siteID apiv3.SiteID
	if err := siteID.UnmarshalText([]byte(id)); err != nil {
		return siteID, errors.New("invalid --site-id: must be a UUID")
	}
	return siteID, nil
}

// uptimeSitesListCmd represents the uptime sites list command
var uptimeSitesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List uptime sites for a project",
	Long:  `List all uptime monitoring sites configured for a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&uptimeProjectID); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		sites, err := client.Sites.ListAll(context.Background(), uptimeProjectID)
		if err != nil {
			return dataAPIError("list uptime sites", err)
		}

		if uptimeOutputFormat == "json" {
			return printJSON(sites)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tURL\tSTATE\tACTIVE\tFREQ")
		for _, site := range sites {
			active := " "
			if site.Active {
				active = "Yes"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%dm\n",
				site.Id,
				site.Name,
				site.Url,
				site.State,
				active,
				site.Frequency)
		}
		return w.Flush()
	},
}

// uptimeSitesGetCmd represents the uptime sites get command
var uptimeSitesGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an uptime site by ID",
	Long:  `Get detailed information about a specific uptime site.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&uptimeProjectID); err != nil {
			return err
		}
		siteID, err := parseSiteID(uptimeSiteID)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		site, err := client.Sites.Get(context.Background(), uptimeProjectID, siteID)
		if err != nil {
			return dataAPIError("get uptime site", err)
		}

		if uptimeOutputFormat == "json" {
			return printJSON(site)
		}

		fmt.Printf("Uptime Site Details:\n")
		fmt.Printf("  ID: %s\n", site.Id)
		fmt.Printf("  Name: %s\n", site.Name)
		fmt.Printf("  URL: %s\n", site.Url)
		fmt.Printf("  State: %s\n", site.State)
		fmt.Printf("  Active: %v\n", site.Active)
		fmt.Printf("  Frequency: %d minutes\n", site.Frequency)
		fmt.Printf("  Match Type: %s\n", site.MatchType)
		if match, err := site.Match.Get(); err == nil {
			fmt.Printf("  Match: %s\n", match)
		}
		if at, err := site.LastCheckedAt.Get(); err == nil {
			fmt.Printf("  Last Checked: %s\n", at.Format("2006-01-02 15:04:05"))
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
It takes the site's fields directly; only url is required.

Example JSON payload:
{
  "name": "My Website",
  "url": "https://example.com",
  "frequency": 5,
  "match_type": "success",
  "locations": ["Virginia", "Oregon"],
  "validate_ssl": true
}

Available options:
- frequency: 1, 2, 5, or 15 (minutes)
- match_type: "success", "exact", "include", "exclude", "jmespath"
- request_method: "GET", "POST", "PUT", "PATCH", "DELETE"
- locations: "Virginia", "Oregon", "Frankfurt", "Singapore", "London"`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&uptimeProjectID); err != nil {
			return err
		}
		params, err := decodeInput[apiv3.SiteCreateParams](uptimeCLIInputJSON, "site")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		site, err := client.Sites.Create(context.Background(), uptimeProjectID, params)
		if err != nil {
			return dataAPIError("create uptime site", err)
		}

		if uptimeOutputFormat == "json" {
			return printJSON(site)
		}
		fmt.Printf("Uptime site created successfully!\n")
		fmt.Printf("  ID: %s\n", site.Id)
		fmt.Printf("  Name: %s\n", site.Name)
		fmt.Printf("  URL: %s\n", site.Url)
		return nil
	},
}

// uptimeSitesUpdateCmd represents the uptime sites update command
var uptimeSitesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing uptime site",
	Long: `Update an existing uptime monitoring site's settings. Only the fields you send
change; a field sent as null is cleared or reset to its default.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "name": "Updated Website",
  "frequency": 15,
  "active": false
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&uptimeProjectID); err != nil {
			return err
		}
		siteID, err := parseSiteID(uptimeSiteID)
		if err != nil {
			return err
		}
		params, err := decodeInput[apiv3.SiteUpdateParams](uptimeCLIInputJSON, "site")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		site, err := client.Sites.Update(context.Background(), uptimeProjectID, siteID, params)
		if err != nil {
			return dataAPIError("update uptime site", err)
		}

		if uptimeOutputFormat == "json" {
			return printJSON(site)
		}
		fmt.Printf("Uptime site updated successfully!\n")
		fmt.Printf("  ID: %s\n", site.Id)
		fmt.Printf("  Name: %s\n", site.Name)
		return nil
	},
}

// uptimeSitesDeleteCmd represents the uptime sites delete command
var uptimeSitesDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an uptime site",
	Long:  `Delete an uptime monitoring site by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&uptimeProjectID); err != nil {
			return err
		}
		siteID, err := parseSiteID(uptimeSiteID)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Sites.Delete(context.Background(), uptimeProjectID, siteID); err != nil {
			return dataAPIError("delete uptime site", err)
		}

		fmt.Println("Uptime site deleted successfully")
		return nil
	},
}

// uptimeListOptions turns --limit and --created-before into the options
// v3's outage and uptime check listings take.
func uptimeListOptions() ([]apiv3.Option, error) {
	opts := []apiv3.Option{apiv3.Limit(uptimeLimit)}
	if uptimeCreatedBefore != "" {
		t, err := parseTimeFlag(uptimeCreatedBefore)
		if err != nil {
			return nil, fmt.Errorf("invalid --created-before: %w", err)
		}
		opts = append(opts, apiv3.OlderThan(float64(t.UnixNano())/float64(time.Second)))
	}
	return opts, nil
}

// uptimeOutagesCmd represents the uptime outages command
var uptimeOutagesCmd = &cobra.Command{
	Use:   "outages",
	Short: "List outages for a site",
	Long:  `List outages recorded for a specific uptime monitoring site, newest first.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&uptimeProjectID); err != nil {
			return err
		}
		siteID, err := parseSiteID(uptimeSiteID)
		if err != nil {
			return err
		}
		opts, err := uptimeListOptions()
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		response, err := client.Sites.ListOutages(
			context.Background(),
			uptimeProjectID,
			siteID,
			opts...,
		)
		if err != nil {
			return dataAPIError("list outages", err)
		}

		if uptimeOutputFormat == "json" {
			return printJSON(response.Data)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "DOWN AT\tUP AT\tSTATUS\tREASON")
		for _, outage := range response.Data {
			upAt := "Still down"
			if at, err := outage.UpAt.Get(); err == nil {
				upAt = at.Format("2006-01-02 15:04")
			}

			statusCode := ""
			if code, err := outage.Status.Get(); err == nil {
				statusCode = fmt.Sprint(code)
			}

			reason := outage.Reason.GetOrEmpty()
			if len(reason) > 40 {
				reason = reason[:37] + "..."
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				outage.DownAt.Format("2006-01-02 15:04"),
				upAt,
				statusCode,
				reason)
		}
		return w.Flush()
	},
}

// uptimeChecksCmd represents the uptime checks command
var uptimeChecksCmd = &cobra.Command{
	Use:   "checks",
	Short: "List uptime checks for a site",
	Long:  `List individual uptime checks performed for a specific site, newest first.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := resolveDataAPIProjectID(&uptimeProjectID); err != nil {
			return err
		}
		siteID, err := parseSiteID(uptimeSiteID)
		if err != nil {
			return err
		}
		opts, err := uptimeListOptions()
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		response, err := client.Sites.ListUptimeChecks(
			context.Background(),
			uptimeProjectID,
			siteID,
			opts...,
		)
		if err != nil {
			return dataAPIError("list uptime checks", err)
		}

		if uptimeOutputFormat == "json" {
			return printJSON(response.Data)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "CREATED\tLOCATION\tUP\tDURATION")
		for _, check := range response.Data {
			up := ""
			if isUp, err := check.Up.Get(); err == nil {
				up = "No"
				if isUp {
					up = "Yes"
				}
			}

			duration := ""
			if ms, err := check.Duration.Get(); err == nil {
				duration = fmt.Sprintf("%dms", ms)
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				check.CreatedAt.Format("2006-01-02 15:04:05"),
				check.Location.GetOrEmpty(),
				up,
				duration)
		}
		return w.Flush()
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
	uptimeCmd.PersistentFlags().StringVar(&uptimeProjectID, "project-id", "", "Project ID")

	// Flags for sites list
	uptimeSitesListCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for sites get
	uptimeSitesGetCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID (a UUID)")
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
	uptimeSitesUpdateCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID (a UUID)")
	uptimeSitesUpdateCmd.Flags().
		StringVar(&uptimeCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	uptimeSitesUpdateCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = uptimeSitesUpdateCmd.MarkFlagRequired("site-id")
	_ = uptimeSitesUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for sites delete
	uptimeSitesDeleteCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID (a UUID)")
	_ = uptimeSitesDeleteCmd.MarkFlagRequired("site-id")

	// Flags for outages
	uptimeOutagesCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID (a UUID)")
	uptimeOutagesCmd.Flags().
		StringVar(&uptimeCreatedBefore, "created-before", "", "Only outages created before this time (YYYY-MM-DD or RFC3339)")
	uptimeOutagesCmd.Flags().
		IntVar(&uptimeLimit, "limit", 25, "Maximum number of outages to return (max 100)")
	uptimeOutagesCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "table", "Output format (table or json)")
	_ = uptimeOutagesCmd.MarkFlagRequired("site-id")

	// Flags for checks
	uptimeChecksCmd.Flags().StringVar(&uptimeSiteID, "site-id", "", "Site ID (a UUID)")
	uptimeChecksCmd.Flags().
		StringVar(&uptimeCreatedBefore, "created-before", "", "Only checks created before this time (YYYY-MM-DD or RFC3339)")
	uptimeChecksCmd.Flags().
		IntVar(&uptimeLimit, "limit", 25, "Maximum number of checks to return (max 100)")
	uptimeChecksCmd.Flags().
		StringVarP(&uptimeOutputFormat, "output", "o", "table", "Output format (table or json)")
	_ = uptimeChecksCmd.MarkFlagRequired("site-id")
}
