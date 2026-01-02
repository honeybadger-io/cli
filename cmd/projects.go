package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	projectOutputFormat      string
	projectAccountID         string
	projectID                int
	projectCLIInputJSON      string
	projectOccurrencesPeriod string
	projectOccurrencesEnv    string
	projectReportType        string
	projectReportStart       string
	projectReportStop        string
	projectReportEnv         string
)

// projectsCmd represents the projects command
var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage Honeybadger projects",
	Long:  `View and manage your Honeybadger projects.`,
}

// projectsListCmd represents the projects list command
var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long:  `List all projects accessible with your API key.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		var response *hbapi.ProjectsResponse
		var err error

		// List projects by account ID if provided
		if projectAccountID != "" {
			response, err = client.Projects.ListByAccountID(ctx, projectAccountID)
		} else {
			response, err = client.Projects.ListAll(ctx)
		}

		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		// Output results based on format
		switch projectOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(response.Results, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			// Table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tACTIVE\tFAULTS\tUNRESOLVED")
			for _, project := range response.Results {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%v\t%d\t%d\n",
					project.ID,
					project.Name,
					project.Active,
					project.FaultCount,
					project.UnresolvedFaultCount)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// projectsGetCmd represents the projects get command
var projectsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a project by ID",
	Long:  `Get detailed information about a specific project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		project, err := client.Projects.Get(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		// Output result based on format
		switch projectOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(project, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			// Detailed text format
			fmt.Printf("Project Details:\n")
			fmt.Printf("  ID: %d\n", project.ID)
			fmt.Printf("  Name: %s\n", project.Name)
			fmt.Printf("  Active: %v\n", project.Active)
			fmt.Printf("  Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Total Faults: %d\n", project.FaultCount)
			fmt.Printf("  Unresolved Faults: %d\n", project.UnresolvedFaultCount)
			fmt.Printf("  Token: %s\n", project.Token)

			if len(project.Environments) > 0 {
				fmt.Printf("  Environments:\n")
				for _, env := range project.Environments {
					fmt.Printf("    - %s\n", env)
				}
			}

			if len(project.Teams) > 0 {
				fmt.Printf("  Teams:\n")
				for _, team := range project.Teams {
					fmt.Printf("    - %s (ID: %d)\n", team.Name, team.ID)
				}
			}

			if len(project.Users) > 0 {
				fmt.Printf("  Users:\n")
				for _, user := range project.Users {
					fmt.Printf("    - %s <%s> (ID: %d)\n", user.Name, user.Email, user.ID)
				}
			}

			if len(project.Sites) > 0 {
				fmt.Printf("  Sites:\n")
				for _, site := range project.Sites {
					fmt.Printf("    - %s: %s (State: %s)\n", site.Name, site.URL, site.State)
				}
			}
		}

		return nil
	},
}

// projectsCreateCmd represents the projects create command
var projectsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Long: `Create a new project in a specific account.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "project": {
    "name": "My Project",
    "resolve_errors_on_deploy": true,
    "disable_public_links": false,
    "language": "ruby",
    "user_url": "https://myapp.com/users/[user_id]",
    "source_url": "https://github.com/myorg/myrepo/blob/main/[filename]#L[line]",
    "purge_days": 90,
    "user_search_field": "user_id"
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectAccountID == "" {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if projectCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		// Parse JSON input
		var jsonData []byte
		var err error

		if len(projectCLIInputJSON) >= 7 && projectCLIInputJSON[:7] == "file://" {
			// Read from file
			filePath := projectCLIInputJSON[7:]
			jsonData, err = os.ReadFile(
				filePath,
			) // #nosec G304 - User-provided file path is expected for CLI
			if err != nil {
				return fmt.Errorf("failed to read JSON file: %w", err)
			}
		} else {
			// Use direct JSON string
			jsonData = []byte(projectCLIInputJSON)
		}

		// Parse the payload structure
		var payload struct {
			Project hbapi.ProjectRequest `json:"project"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		project, err := client.Projects.Create(ctx, projectAccountID, payload.Project)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		// Output result based on format
		switch projectOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(project, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Project created successfully!\n")
			fmt.Printf("  ID: %d\n", project.ID)
			fmt.Printf("  Name: %s\n", project.Name)
			fmt.Printf("  Token: %s\n", project.Token)
		}

		return nil
	},
}

// projectsUpdateCmd represents the projects update command
var projectsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing project",
	Long: `Update an existing project's settings.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "project": {
    "name": "My Updated Project",
    "resolve_errors_on_deploy": false,
    "purge_days": 120
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}
		if projectCLIInputJSON == "" {
			return fmt.Errorf("JSON payload is required. Set it using --cli-input-json flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		// Parse JSON input
		var jsonData []byte
		var err error

		if len(projectCLIInputJSON) >= 7 && projectCLIInputJSON[:7] == "file://" {
			// Read from file
			filePath := projectCLIInputJSON[7:]
			jsonData, err = os.ReadFile(
				filePath,
			) // #nosec G304 - User-provided file path is expected for CLI
			if err != nil {
				return fmt.Errorf("failed to read JSON file: %w", err)
			}
		} else {
			// Use direct JSON string
			jsonData = []byte(projectCLIInputJSON)
		}

		// Parse the payload structure
		var payload struct {
			Project hbapi.ProjectRequest `json:"project"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		result, err := client.Projects.Update(ctx, projectID, payload.Project)
		if err != nil {
			return fmt.Errorf("failed to update project: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

// projectsDeleteCmd represents the projects delete command
var projectsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a project",
	Long:  `Delete a project by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		result, err := client.Projects.Delete(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to delete project: %w", err)
		}

		fmt.Println(result.Message)
		return nil
	},
}

// projectsOccurrencesCmd represents the projects occurrences command
var projectsOccurrencesCmd = &cobra.Command{
	Use:   "occurrences",
	Short: "Get occurrence counts for projects",
	Long:  `Get occurrence counts for all projects or a specific project over time.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		options := hbapi.ProjectGetOccurrenceCountsOptions{
			Period:      projectOccurrencesPeriod,
			Environment: projectOccurrencesEnv,
		}

		ctx := context.Background()

		// If project ID is specified, get occurrences for that project
		// Otherwise get occurrences for all projects
		if projectID > 0 {
			result, err := client.Projects.GetOccurrenceCounts(ctx, projectID, options)
			if err != nil {
				return fmt.Errorf("failed to get occurrence counts: %w", err)
			}

			// Output results based on format
			switch projectOutputFormat {
			case "json":
				jsonBytes, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(jsonBytes))
			default:
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintln(w, "TIMESTAMP\tCOUNT")
				for _, count := range result {
					_, _ = fmt.Fprintf(w, "%d\t%d\n", count[0], count[1])
				}
				_ = w.Flush()
			}
		} else {
			result, err := client.Projects.GetAllOccurrenceCounts(ctx, options)
			if err != nil {
				return fmt.Errorf("failed to get occurrence counts: %w", err)
			}

			// Output results based on format
			switch projectOutputFormat {
			case "json":
				jsonBytes, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(jsonBytes))
			default:
				for projectIDStr, counts := range result {
					fmt.Printf("Project %s:\n", projectIDStr)
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					_, _ = fmt.Fprintln(w, "  TIMESTAMP\tCOUNT")
					for _, count := range counts {
						_, _ = fmt.Fprintf(w, "  %d\t%d\n", count[0], count[1])
					}
					_ = w.Flush()
					fmt.Println()
				}
			}
		}

		return nil
	},
}

// projectsIntegrationsCmd represents the projects integrations command
var projectsIntegrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "Get integrations for a project",
	Long:  `Get all notification channels/integrations configured for a project.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		integrations, err := client.Projects.GetIntegrations(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to get integrations: %w", err)
		}

		// Output results based on format
		switch projectOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(integrations, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tTYPE\tACTIVE\tEVENTS")
			for _, integration := range integrations {
				active := " "
				if integration.Active {
					active = "✓"
				}
				events := "none"
				if len(integration.Events) > 0 {
					events = fmt.Sprintf("%v", integration.Events)
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					integration.ID,
					integration.Type,
					active,
					events)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// projectsReportsCmd represents the projects reports command
var projectsReportsCmd = &cobra.Command{
	Use:   "reports",
	Short: "Get report data for a project",
	Long: `Get report data for a project. Available report types:
  - notices_by_class: Group notices by error class
  - notices_by_location: Group notices by location
  - notices_by_user: Group notices by user
  - notices_per_day: Count notices per day`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}
		if projectReportType == "" {
			return fmt.Errorf("report type is required. Set it using --type flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := viper.GetString("endpoint")
		// Projects API uses app.honeybadger.io, not api.honeybadger.io
		if endpoint == "https://api.honeybadger.io" {
			endpoint = "https://app.honeybadger.io"
		}

		// Create API client
		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		options := hbapi.ProjectGetReportOptions{
			Environment: projectReportEnv,
		}

		// Parse start and stop times if provided
		if projectReportStart != "" {
			startTime, err := time.Parse(time.RFC3339, projectReportStart)
			if err != nil {
				return fmt.Errorf("invalid start time format (use RFC3339): %w", err)
			}
			options.Start = &startTime
		}
		if projectReportStop != "" {
			stopTime, err := time.Parse(time.RFC3339, projectReportStop)
			if err != nil {
				return fmt.Errorf("invalid stop time format (use RFC3339): %w", err)
			}
			options.Stop = &stopTime
		}

		ctx := context.Background()
		report, err := client.Projects.GetReport(
			ctx,
			projectID,
			hbapi.ProjectReportType(projectReportType),
			options,
		)
		if err != nil {
			return fmt.Errorf("failed to get report: %w", err)
		}

		// Output results based on format
		switch projectOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, row := range report {
				for i, col := range row {
					if i > 0 {
						_, _ = fmt.Fprint(w, "\t")
					}
					_, _ = fmt.Fprintf(w, "%v", col)
				}
				_, _ = fmt.Fprintln(w)
			}
			_ = w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsGetCmd)
	projectsCmd.AddCommand(projectsCreateCmd)
	projectsCmd.AddCommand(projectsUpdateCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)
	projectsCmd.AddCommand(projectsOccurrencesCmd)
	projectsCmd.AddCommand(projectsIntegrationsCmd)
	projectsCmd.AddCommand(projectsReportsCmd)

	// Flags for list command
	projectsListCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "table", "Output format (table or json)")
	projectsListCmd.Flags().
		StringVar(&projectAccountID, "account-id", "", "Filter projects by account ID")

	// Flags for get command
	projectsGetCmd.Flags().IntVar(&projectID, "id", 0, "Project ID")
	projectsGetCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "text", "Output format (text or json)")

	if err := projectsGetCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}

	// Flags for create command
	projectsCreateCmd.Flags().
		StringVar(&projectAccountID, "account-id", "", "Account ID to create project in")
	projectsCreateCmd.Flags().
		StringVar(&projectCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	projectsCreateCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "text", "Output format (text or json)")

	if err := projectsCreateCmd.MarkFlagRequired("account-id"); err != nil {
		fmt.Printf("error marking account-id flag as required: %v\n", err)
	}
	if err := projectsCreateCmd.MarkFlagRequired("cli-input-json"); err != nil {
		fmt.Printf("error marking cli-input-json flag as required: %v\n", err)
	}

	// Flags for update command
	projectsUpdateCmd.Flags().IntVar(&projectID, "id", 0, "Project ID")
	projectsUpdateCmd.Flags().
		StringVar(&projectCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")

	if err := projectsUpdateCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := projectsUpdateCmd.MarkFlagRequired("cli-input-json"); err != nil {
		fmt.Printf("error marking cli-input-json flag as required: %v\n", err)
	}

	// Flags for delete command
	projectsDeleteCmd.Flags().IntVar(&projectID, "id", 0, "Project ID")

	if err := projectsDeleteCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}

	// Flags for occurrences command
	projectsOccurrencesCmd.Flags().
		IntVar(&projectID, "id", 0, "Project ID (optional - if not specified, shows all projects)")
	projectsOccurrencesCmd.Flags().
		StringVar(&projectOccurrencesPeriod, "period", "day", "Time period (hour, day, week, or month)")
	projectsOccurrencesCmd.Flags().
		StringVar(&projectOccurrencesEnv, "environment", "", "Filter by environment")
	projectsOccurrencesCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for integrations command
	projectsIntegrationsCmd.Flags().IntVar(&projectID, "id", 0, "Project ID")
	projectsIntegrationsCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "table", "Output format (table or json)")

	if err := projectsIntegrationsCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}

	// Flags for reports command
	projectsReportsCmd.Flags().IntVar(&projectID, "id", 0, "Project ID")
	projectsReportsCmd.Flags().
		StringVar(&projectReportType, "type", "", "Report type (notices_by_class, notices_by_location, notices_by_user, notices_per_day)")
	projectsReportsCmd.Flags().
		StringVar(&projectReportStart, "start", "", "Start time (RFC3339 format)")
	projectsReportsCmd.Flags().
		StringVar(&projectReportStop, "stop", "", "Stop time (RFC3339 format)")
	projectsReportsCmd.Flags().
		StringVar(&projectReportEnv, "environment", "", "Filter by environment")
	projectsReportsCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "table", "Output format (table or json)")

	if err := projectsReportsCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := projectsReportsCmd.MarkFlagRequired("type"); err != nil {
		fmt.Printf("error marking type flag as required: %v\n", err)
	}
}
