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
	statuspageID            string
	statuspagesOutputFormat string
	statuspageCLIInputJSON  string
)

// statuspagesCmd represents the statuspages command
var statuspagesCmd = &cobra.Command{
	Use:     "statuspages",
	Short:   "Manage status pages",
	GroupID: GroupDataAPI,
	Long:    `View and manage the status pages of the account your API Token belongs to.`,
}

// statuspagesListCmd represents the statuspages list command
var statuspagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List status pages for the account",
	Long:  `List all status pages in the account your API Token belongs to.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		statusPages, err := client.StatusPages.ListAll(context.Background())
		if err != nil {
			return dataAPIError("list status pages", err)
		}

		if statuspagesOutputFormat == "json" {
			return printJSON(statusPages)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tURL\tSITES\tCHECK-INS")
		for _, sp := range statusPages {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n",
				sp.Id,
				sp.Name,
				sp.Url,
				len(sp.Sites),
				len(sp.CheckIns))
		}
		return w.Flush()
	},
}

// statuspagesGetCmd represents the statuspages get command
var statuspagesGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a status page by ID",
	Long:  `Get detailed information about a specific status page.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if statuspageID == "" {
			return fmt.Errorf("status page ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		statusPage, err := client.StatusPages.Get(context.Background(), statuspageID)
		if err != nil {
			return dataAPIError("get status page", err)
		}

		if statuspagesOutputFormat == "json" {
			return printJSON(statusPage)
		}

		fmt.Printf("Status Page Details:\n")
		fmt.Printf("  ID: %s\n", statusPage.Id)
		fmt.Printf("  Name: %s\n", statusPage.Name)
		fmt.Printf("  URL: %s\n", statusPage.Url)
		fmt.Printf("  Account ID: %s\n", statusPage.AccountId)
		if domain, err := statusPage.Domain.Get(); err == nil {
			fmt.Printf("  Domain: %s\n", domain)
		}
		fmt.Printf("  Created: %s\n", statusPage.CreatedAt.Format("2006-01-02 15:04:05"))
		if at, err := statusPage.DomainVerifiedAt.Get(); err == nil {
			fmt.Printf("  Domain Verified: %s\n", at.Format("2006-01-02 15:04:05"))
		}
		if len(statusPage.Sites) > 0 {
			fmt.Printf("  Sites:\n")
			for _, site := range statusPage.Sites {
				fmt.Printf("    - %s (%s)\n", site.DisplayName.GetOrEmpty(), site.SiteId)
			}
		}
		if len(statusPage.CheckIns) > 0 {
			fmt.Printf("  Check-ins:\n")
			for _, ci := range statusPage.CheckIns {
				fmt.Printf("    - %s (%s)\n", ci.DisplayName.GetOrEmpty(), ci.CheckInId)
			}
		}
		return nil
	},
}

// statuspagesCreateCmd represents the statuspages create command
var statuspagesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new status page",
	Long: `Create a new status page in the account your API Token belongs to.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.
It takes the status page's fields directly; only name is required. sites and check_ins
are listed in page order, each naming a site (by its UUID) or check-in by id, with an
optional display_name and description.

Example JSON payload:
{
  "name": "My Status Page",
  "domain": "status.example.com",
  "sites": [{"site_id": "9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47", "display_name": "Website"}],
  "check_ins": [{"check_in_id": "Ci5xWq"}],
  "hide_branding": false
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		params, err := decodeInput[apiv3.StatusPageCreateParams](
			statuspageCLIInputJSON,
			"status_page",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		statusPage, err := client.StatusPages.Create(context.Background(), params)
		if err != nil {
			return dataAPIError("create status page", err)
		}

		if statuspagesOutputFormat == "json" {
			return printJSON(statusPage)
		}
		fmt.Printf("Status page created successfully!\n")
		fmt.Printf("  ID: %s\n", statusPage.Id)
		fmt.Printf("  Name: %s\n", statusPage.Name)
		fmt.Printf("  URL: %s\n", statusPage.Url)
		return nil
	},
}

// statuspagesUpdateCmd represents the statuspages update command
var statuspagesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing status page",
	Long: `Update an existing status page's settings. Only the fields you send change;
a field sent as null is cleared or reset to its default. A sites or check_ins list
replaces the page's list.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "name": "Updated Status Page",
  "sites": [{"site_id": "9f8b6d2e-4c1a-4b7f-9e35-2a6c8d0f1b47"}]
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if statuspageID == "" {
			return fmt.Errorf("status page ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.StatusPageUpdateParams](
			statuspageCLIInputJSON,
			"status_page",
		)
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if _, err := client.StatusPages.Update(
			context.Background(),
			statuspageID,
			params,
		); err != nil {
			return dataAPIError("update status page", err)
		}

		fmt.Println("Status page updated successfully")
		return nil
	},
}

// statuspagesDeleteCmd represents the statuspages delete command
var statuspagesDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a status page",
	Long:  `Delete a status page by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if statuspageID == "" {
			return fmt.Errorf("status page ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.StatusPages.Delete(context.Background(), statuspageID); err != nil {
			return dataAPIError("delete status page", err)
		}

		fmt.Println("Status page deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statuspagesCmd)
	statuspagesCmd.AddCommand(statuspagesListCmd)
	statuspagesCmd.AddCommand(statuspagesGetCmd)
	statuspagesCmd.AddCommand(statuspagesCreateCmd)
	statuspagesCmd.AddCommand(statuspagesUpdateCmd)
	statuspagesCmd.AddCommand(statuspagesDeleteCmd)

	// Flags for list command
	statuspagesListCmd.Flags().
		StringVarP(&statuspagesOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	statuspagesGetCmd.Flags().StringVar(&statuspageID, "id", "", "Status page ID")
	statuspagesGetCmd.Flags().
		StringVarP(&statuspagesOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = statuspagesGetCmd.MarkFlagRequired("id")

	// Flags for create command
	statuspagesCreateCmd.Flags().
		StringVar(&statuspageCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	statuspagesCreateCmd.Flags().
		StringVarP(&statuspagesOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = statuspagesCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for update command
	statuspagesUpdateCmd.Flags().StringVar(&statuspageID, "id", "", "Status page ID")
	statuspagesUpdateCmd.Flags().
		StringVar(&statuspageCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = statuspagesUpdateCmd.MarkFlagRequired("id")
	_ = statuspagesUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	statuspagesDeleteCmd.Flags().StringVar(&statuspageID, "id", "", "Status page ID")
	_ = statuspagesDeleteCmd.MarkFlagRequired("id")
}
