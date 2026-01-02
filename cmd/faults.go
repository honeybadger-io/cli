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
	faultsProjectID      int
	faultID              int
	faultQuery           string
	faultEnvironment     string
	faultOrder           string
	faultLimit           int
	faultOutputFormat    string
	faultAffectedUserQuery string
)

// faultsCmd represents the faults command
var faultsCmd = &cobra.Command{
	Use:   "faults",
	Short: "Manage Honeybadger faults",
	Long:  `View and manage faults (errors) in your Honeybadger projects.`,
}

// faultsListCmd represents the faults list command
var faultsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List faults for a project",
	Long:  `List all faults for a specific project with optional filtering.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if faultsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf("auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable")
		}

		endpoint := viper.GetString("endpoint")
		// Faults API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		// Build options
		options := hbapi.FaultListOptions{
			Q:     faultQuery,
			Order: faultOrder,
			Limit: faultLimit,
		}

		ctx := context.Background()
		response, err := client.Faults.List(ctx, faultsProjectID, options)
		if err != nil {
			return fmt.Errorf("failed to list faults: %w", err)
		}

		// Output results based on format
		switch faultOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(response.Results, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			// Table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tCLASS\tMESSAGE\tENV\tNOTICES\tRESOLVED\tLAST SEEN")
			for _, fault := range response.Results {
				lastSeen := "Never"
				if fault.LastNoticeAt != nil {
					lastSeen = fault.LastNoticeAt.Format("2006-01-02 15:04")
				}

				message := fault.Message
				if len(message) > 50 {
					message = message[:47] + "..."
				}

				resolved := " "
				if fault.Resolved {
					resolved = "✓"
				}

				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%s\t%s\n",
					fault.ID,
					fault.Klass,
					message,
					fault.Environment,
					fault.NoticesCount,
					resolved,
					lastSeen)
			}
			w.Flush()
		}

		return nil
	},
}

// faultsGetCmd represents the faults get command
var faultsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a fault by ID",
	Long:  `Get detailed information about a specific fault.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if faultsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if faultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf("auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable")
		}

		endpoint := viper.GetString("endpoint")
		// Faults API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		fault, err := client.Faults.Get(ctx, faultsProjectID, faultID)
		if err != nil {
			return fmt.Errorf("failed to get fault: %w", err)
		}

		// Output result based on format
		switch faultOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(fault, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			// Detailed text format
			fmt.Printf("Fault Details:\n")
			fmt.Printf("  ID: %d\n", fault.ID)
			fmt.Printf("  Class: %s\n", fault.Klass)
			fmt.Printf("  Message: %s\n", fault.Message)
			fmt.Printf("  Environment: %s\n", fault.Environment)
			fmt.Printf("  Component: %s\n", fault.Component)
			fmt.Printf("  Action: %s\n", fault.Action)
			fmt.Printf("  Created: %s\n", fault.CreatedAt.Format("2006-01-02 15:04:05"))

			if fault.LastNoticeAt != nil {
				fmt.Printf("  Last Noticed: %s\n", fault.LastNoticeAt.Format("2006-01-02 15:04:05"))
			}

			fmt.Printf("  Notice Count: %d\n", fault.NoticesCount)
			fmt.Printf("  Comments Count: %d\n", fault.CommentsCount)
			fmt.Printf("  Resolved: %v\n", fault.Resolved)
			fmt.Printf("  Ignored: %v\n", fault.Ignored)
			fmt.Printf("  URL: %s\n", fault.URL)

			if fault.Assignee != nil {
				fmt.Printf("  Assignee: %s <%s>\n", fault.Assignee.Name, fault.Assignee.Email)
			}

			if len(fault.Tags) > 0 {
				fmt.Printf("  Tags: ")
				for i, tag := range fault.Tags {
					if i > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%s", tag)
				}
				fmt.Println()
			}
		}

		return nil
	},
}

// faultsNoticesCmd represents the faults notices command
var faultsNoticesCmd = &cobra.Command{
	Use:   "notices",
	Short: "List notices for a fault",
	Long:  `List individual error occurrences (notices) for a specific fault.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if faultsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if faultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf("auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable")
		}

		endpoint := viper.GetString("endpoint")
		// Faults API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		// Build options
		options := hbapi.FaultListNoticesOptions{
			Limit: faultLimit,
		}

		ctx := context.Background()
		response, err := client.Faults.ListNotices(ctx, faultsProjectID, faultID, options)
		if err != nil {
			return fmt.Errorf("failed to list notices: %w", err)
		}

		// Output results based on format
		switch faultOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(response.Results, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			// Table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tMESSAGE\tENVIRONMENT\tHOSTNAME\tCREATED")
			for _, notice := range response.Results {
				message := notice.Message
				if len(message) > 60 {
					message = message[:57] + "..."
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					notice.ID,
					message,
					notice.EnvironmentName,
					notice.Environment.Hostname,
					notice.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			w.Flush()
		}

		return nil
	},
}

// faultsCountsCmd represents the faults counts command
var faultsCountsCmd = &cobra.Command{
	Use:   "counts",
	Short: "Get fault counts for a project",
	Long:  `Get summary counts of faults grouped by environment and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if faultsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf("auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable")
		}

		endpoint := viper.GetString("endpoint")
		// Faults API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		counts, err := client.Faults.GetCounts(ctx, faultsProjectID, hbapi.FaultListOptions{})
		if err != nil {
			return fmt.Errorf("failed to get fault counts: %w", err)
		}

		// Output results based on format
		switch faultOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(counts, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Total Faults: %d\n\n", counts.Total)

			if len(counts.Environments) > 0 {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ENVIRONMENT\tRESOLVED\tIGNORED\tCOUNT")
				for _, env := range counts.Environments {
					fmt.Fprintf(w, "%s\t%v\t%v\t%d\n",
						env.Environment,
						env.Resolved,
						env.Ignored,
						env.Count)
				}
				w.Flush()
			}
		}

		return nil
	},
}

// faultsAffectedUsersCmd represents the faults affected-users command
var faultsAffectedUsersCmd = &cobra.Command{
	Use:   "affected-users",
	Short: "List users affected by a fault",
	Long:  `List all users who have been affected by a specific fault.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if faultsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag")
		}
		if faultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf("auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable")
		}

		endpoint := viper.GetString("endpoint")
		// Faults API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		// Build options
		options := hbapi.FaultListAffectedUsersOptions{
			Q: faultAffectedUserQuery,
		}

		ctx := context.Background()
		users, err := client.Faults.ListAffectedUsers(ctx, faultsProjectID, faultID, options)
		if err != nil {
			return fmt.Errorf("failed to list affected users: %w", err)
		}

		// Output results based on format
		switch faultOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(users, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			// Table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "USER\tOCCURRENCES")
			for _, user := range users {
				fmt.Fprintf(w, "%s\t%d\n", user.User, user.Count)
			}
			w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(faultsCmd)
	faultsCmd.AddCommand(faultsListCmd)
	faultsCmd.AddCommand(faultsGetCmd)
	faultsCmd.AddCommand(faultsNoticesCmd)
	faultsCmd.AddCommand(faultsCountsCmd)
	faultsCmd.AddCommand(faultsAffectedUsersCmd)

	// Common flags
	faultsCmd.PersistentFlags().IntVar(&faultsProjectID, "project-id", 0, "Project ID")

	// Flags for list command
	faultsListCmd.Flags().StringVarP(&faultQuery, "query", "q", "", "Search query to filter faults")
	faultsListCmd.Flags().StringVar(&faultOrder, "order", "recent", "Order faults by 'recent' or 'frequent'")
	faultsListCmd.Flags().IntVar(&faultLimit, "limit", 25, "Maximum number of faults to return (max 25)")
	faultsListCmd.Flags().StringVarP(&faultOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	faultsGetCmd.Flags().IntVar(&faultID, "id", 0, "Fault ID")
	faultsGetCmd.Flags().StringVarP(&faultOutputFormat, "output", "o", "text", "Output format (text or json)")

	// Flags for notices command
	faultsNoticesCmd.Flags().IntVar(&faultID, "id", 0, "Fault ID")
	faultsNoticesCmd.Flags().IntVar(&faultLimit, "limit", 25, "Maximum number of notices to return (max 25)")
	faultsNoticesCmd.Flags().StringVarP(&faultOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for counts command
	faultsCountsCmd.Flags().StringVarP(&faultOutputFormat, "output", "o", "text", "Output format (text or json)")

	// Flags for affected-users command
	faultsAffectedUsersCmd.Flags().IntVar(&faultID, "id", 0, "Fault ID")
	faultsAffectedUsersCmd.Flags().StringVarP(&faultAffectedUserQuery, "query", "q", "", "Search query to filter users")
	faultsAffectedUsersCmd.Flags().StringVarP(&faultOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Mark required flags
	if err := faultsGetCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := faultsNoticesCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := faultsAffectedUsersCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
}
