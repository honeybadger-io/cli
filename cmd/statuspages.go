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
	statuspagesAccountID    int
	statuspageID            int
	statuspagesOutputFormat string
	statuspageCLIInputJSON  string
)

// statuspagesCmd represents the statuspages command
var statuspagesCmd = &cobra.Command{
	Use:   "statuspages",
	Short: "Manage status pages",
	Long:  `View and manage status pages for your Honeybadger accounts.`,
}

// statuspagesListCmd represents the statuspages list command
var statuspagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List status pages for an account",
	Long:  `List all status pages configured for a specific account.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if statuspagesAccountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
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
		statusPages, err := client.StatusPages.List(ctx, statuspagesAccountID)
		if err != nil {
			return fmt.Errorf("failed to list status pages: %w", err)
		}

		switch statuspagesOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(statusPages, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tURL\tSITES\tCHECK-INS")
			for _, sp := range statusPages {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\n",
					sp.ID,
					sp.Name,
					sp.URL,
					len(sp.Sites),
					len(sp.CheckIns))
			}
			_ = w.Flush()
		}

		return nil
	},
}

// statuspagesGetCmd represents the statuspages get command
var statuspagesGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a status page by ID",
	Long:  `Get detailed information about a specific status page.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if statuspagesAccountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if statuspageID == 0 {
			return fmt.Errorf("status page ID is required. Set it using --id flag")
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
		statusPage, err := client.StatusPages.Get(ctx, statuspagesAccountID, statuspageID)
		if err != nil {
			return fmt.Errorf("failed to get status page: %w", err)
		}

		switch statuspagesOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(statusPage, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Status Page Details:\n")
			fmt.Printf("  ID: %d\n", statusPage.ID)
			fmt.Printf("  Name: %s\n", statusPage.Name)
			fmt.Printf("  URL: %s\n", statusPage.URL)
			fmt.Printf("  Account ID: %d\n", statusPage.AccountID)
			if statusPage.Domain != nil {
				fmt.Printf("  Domain: %s\n", *statusPage.Domain)
			}
			fmt.Printf("  Created: %s\n", statusPage.CreatedAt.Format("2006-01-02 15:04:05"))
			if statusPage.DomainVerifiedAt != nil {
				fmt.Printf(
					"  Domain Verified: %s\n",
					statusPage.DomainVerifiedAt.Format("2006-01-02 15:04:05"),
				)
			}
			if len(statusPage.Sites) > 0 {
				fmt.Printf("  Sites: %v\n", statusPage.Sites)
			}
			if len(statusPage.CheckIns) > 0 {
				fmt.Printf("  Check-ins: %v\n", statusPage.CheckIns)
			}
		}

		return nil
	},
}

// statuspagesCreateCmd represents the statuspages create command
var statuspagesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new status page",
	Long: `Create a new status page for an account.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "status_page": {
    "name": "My Status Page",
    "domain": "status.example.com",
    "sites": ["site-id-1", "site-id-2"],
    "check_ins": ["checkin-slug-1"],
    "hide_branding": false
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if statuspagesAccountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if statuspageCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(statuspageCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			StatusPage hbapi.StatusPageParams `json:"status_page"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		statusPage, err := client.StatusPages.Create(ctx, statuspagesAccountID, payload.StatusPage)
		if err != nil {
			return fmt.Errorf("failed to create status page: %w", err)
		}

		switch statuspagesOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(statusPage, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Status page created successfully!\n")
			fmt.Printf("  ID: %d\n", statusPage.ID)
			fmt.Printf("  Name: %s\n", statusPage.Name)
			fmt.Printf("  URL: %s\n", statusPage.URL)
		}

		return nil
	},
}

// statuspagesUpdateCmd represents the statuspages update command
var statuspagesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing status page",
	Long: `Update an existing status page's settings.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "status_page": {
    "name": "Updated Status Page",
    "sites": ["site-id-1", "site-id-3"]
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if statuspagesAccountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if statuspageID == 0 {
			return fmt.Errorf("status page ID is required. Set it using --id flag")
		}
		if statuspageCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(statuspageCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			StatusPage hbapi.StatusPageParams `json:"status_page"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		err = client.StatusPages.Update(ctx, statuspagesAccountID, statuspageID, payload.StatusPage)
		if err != nil {
			return fmt.Errorf("failed to update status page: %w", err)
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
		if statuspagesAccountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if statuspageID == 0 {
			return fmt.Errorf("status page ID is required. Set it using --id flag")
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
		err := client.StatusPages.Delete(ctx, statuspagesAccountID, statuspageID)
		if err != nil {
			return fmt.Errorf("failed to delete status page: %w", err)
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

	// Common flags
	statuspagesCmd.PersistentFlags().IntVar(&statuspagesAccountID, "account-id", 0, "Account ID")

	// Flags for list command
	statuspagesListCmd.Flags().
		StringVarP(&statuspagesOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	statuspagesGetCmd.Flags().IntVar(&statuspageID, "id", 0, "Status page ID")
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
	statuspagesUpdateCmd.Flags().IntVar(&statuspageID, "id", 0, "Status page ID")
	statuspagesUpdateCmd.Flags().
		StringVar(&statuspageCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	_ = statuspagesUpdateCmd.MarkFlagRequired("id")
	_ = statuspagesUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for delete command
	statuspagesDeleteCmd.Flags().IntVar(&statuspageID, "id", 0, "Status page ID")
	_ = statuspagesDeleteCmd.MarkFlagRequired("id")
}
