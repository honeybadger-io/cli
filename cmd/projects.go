package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/honeybadger-io/api-go/apiv3"
	"github.com/spf13/cobra"
)

var (
	projectOutputFormat      string
	projectID                string
	projectCLIInputJSON      string
	projectOccurrencesPeriod string
	projectOccurrencesEnv    string
)

// projectsCmd represents the projects command
var projectsCmd = &cobra.Command{
	Use:     "projects",
	Short:   "Manage Honeybadger projects",
	GroupID: GroupDataAPI,
	Long:    `View and manage your Honeybadger projects.`,
}

// projectsListCmd represents the projects list command
var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long:  `List every project your API Token can reach.` + v3Reference("projects/list-projects"),
	RunE: func(_ *cobra.Command, _ []string) error {
		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		projects, err := client.Projects.ListAll(context.Background())
		if err != nil {
			return dataAPIError("list projects", err)
		}

		if projectOutputFormat == "json" {
			return printJSON(projects)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tACTIVE\tFAULTS\tUNRESOLVED")
		for _, project := range projects {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%v\t%d\t%d\n",
				project.Id,
				project.Name,
				project.Active,
				project.FaultCount,
				project.UnresolvedFaultCount)
		}
		return w.Flush()
	},
}

// projectsGetCmd represents the projects get command
var projectsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a project by ID",
	Long: `Get detailed information about a specific project.` + v3Reference(
		"projects/get-project",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == "" {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		project, err := client.Projects.Get(context.Background(), projectID)
		if err != nil {
			return dataAPIError("get project", err)
		}

		if projectOutputFormat == "json" {
			return printJSON(project)
		}

		fmt.Printf("Project Details:\n")
		fmt.Printf("  ID: %s\n", project.Id)
		fmt.Printf("  Name: %s\n", project.Name)
		fmt.Printf("  Active: %v\n", project.Active)
		fmt.Printf("  Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Total Faults: %d\n", project.FaultCount)
		fmt.Printf("  Unresolved Faults: %d\n", project.UnresolvedFaultCount)
		if len(project.Environments) > 0 {
			fmt.Printf("  Environments:\n")
			for _, env := range project.Environments {
				fmt.Printf("    - %s\n", env)
			}
		}
		return nil
	},
}

// projectsCreateCmd represents the projects create command
var projectsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Long: `Create a new project in the account your API Token belongs to.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.
It takes the project's fields directly; only name is required.

Example JSON payload:
{
  "name": "My Project",
  "resolve_errors_on_deploy": true,
  "disable_public_links": false,
  "language": "ruby",
  "user_url": "https://myapp.com/users/[user_id]",
  "source_url": "https://github.com/myorg/myrepo/blob/main/[file]#L[line]",
  "purge_days": 90,
  "user_search_field": "context.user_id"
}` + v3Reference("projects/create-project"),
	RunE: func(_ *cobra.Command, _ []string) error {
		params, err := decodeInput[apiv3.ProjectCreateParams](projectCLIInputJSON, "project")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		project, err := client.Projects.Create(ctx, params)
		if err != nil {
			return dataAPIError("create project", err)
		}

		if projectOutputFormat == "json" {
			return printJSON(project)
		}
		fmt.Printf("Project created successfully!\n")
		fmt.Printf("  ID: %s\n", project.Id)
		fmt.Printf("  Name: %s\n", project.Name)

		// The project exists now, so a token that can create projects but not
		// read their keys still succeeds; it just can't show the key.
		keys, err := client.IngestionKeys.ListAll(ctx, project.Id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't read the project's Ingestion Key: %v\n", err)
			return nil
		}
		if len(keys) > 0 {
			fmt.Printf("  Ingestion Key: %s\n", keys[0].Key)
		}
		return nil
	},
}

// projectsUpdateCmd represents the projects update command
var projectsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing project",
	Long: `Update an existing project's settings. Only the fields you send change.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "name": "My Updated Project",
  "resolve_errors_on_deploy": false,
  "purge_days": 120
}` + v3Reference("projects/update-project"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == "" {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}
		params, err := decodeInput[apiv3.ProjectParams](projectCLIInputJSON, "project")
		if err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		project, err := client.Projects.Update(context.Background(), projectID, params)
		if err != nil {
			return dataAPIError("update project", err)
		}

		fmt.Printf("Project %s was successfully updated\n", project.Id)
		return nil
	},
}

// projectsDeleteCmd represents the projects delete command
var projectsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a project",
	Long: `Delete a project by ID. This action cannot be undone.` + v3Reference(
		"projects/delete-project",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if projectID == "" {
			return fmt.Errorf("project ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Projects.Delete(context.Background(), projectID); err != nil {
			return dataAPIError("delete project", err)
		}

		fmt.Printf("Project %s deleted successfully\n", projectID)
		return nil
	},
}

// projectsOccurrencesCmd represents the projects occurrences command
var projectsOccurrencesCmd = &cobra.Command{
	Use:   "occurrences",
	Short: "Get occurrence counts for projects",
	Long: `Get occurrence counts for all projects or a specific project over time.` + v3Reference(
		"projects/get-project-occurrences",
		"projects/list-account-occurrences",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		options := apiv3.OccurrenceOptions{
			Period:      apiv3.OccurrencePeriod(projectOccurrencesPeriod),
			Environment: projectOccurrencesEnv,
		}
		ctx := context.Background()

		var series []apiv3.OccurrenceSeries
		if projectID != "" {
			one, err := client.Projects.Occurrences(ctx, projectID, options)
			if err != nil {
				return dataAPIError("get occurrence counts", err)
			}
			series = []apiv3.OccurrenceSeries{*one}
		} else {
			series, err = client.Projects.AccountOccurrences(ctx, options)
			if err != nil {
				return dataAPIError("get occurrence counts", err)
			}
		}

		if projectOutputFormat == "json" {
			if projectID != "" {
				return printJSON(series[0])
			}
			return printJSON(series)
		}

		if projectID != "" {
			return printOccurrenceTable(series[0], "")
		}
		// One table per project, indented under its heading as v2 printed it.
		for i, s := range series {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("Project %s:\n", s.ProjectId)
			if err := printOccurrenceTable(s, "  "); err != nil {
				return err
			}
		}
		return nil
	},
}

func printOccurrenceTable(s apiv3.OccurrenceSeries, indent string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, indent+"TIMESTAMP\tCOUNT")
	for _, b := range s.Buckets {
		_, _ = fmt.Fprintf(w, "%s%s\t%d\n", indent,
			time.Unix(b.Timestamp, 0).UTC().Format(time.RFC3339), b.Count)
	}
	return w.Flush()
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsGetCmd)
	projectsCmd.AddCommand(projectsCreateCmd)
	projectsCmd.AddCommand(projectsUpdateCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)
	projectsCmd.AddCommand(projectsOccurrencesCmd)

	// Flags for list command
	projectsListCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	projectsGetCmd.Flags().StringVar(&projectID, "id", "", "Project ID")
	projectsGetCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "text", "Output format (text or json)")

	if err := projectsGetCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}

	// Flags for create command
	projectsCreateCmd.Flags().
		StringVar(&projectCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	projectsCreateCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "text", "Output format (text or json)")

	if err := projectsCreateCmd.MarkFlagRequired("cli-input-json"); err != nil {
		fmt.Printf("error marking cli-input-json flag as required: %v\n", err)
	}

	// Flags for update command
	projectsUpdateCmd.Flags().StringVar(&projectID, "id", "", "Project ID")
	projectsUpdateCmd.Flags().
		StringVar(&projectCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")

	if err := projectsUpdateCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}
	if err := projectsUpdateCmd.MarkFlagRequired("cli-input-json"); err != nil {
		fmt.Printf("error marking cli-input-json flag as required: %v\n", err)
	}

	// Flags for delete command
	projectsDeleteCmd.Flags().StringVar(&projectID, "id", "", "Project ID")

	if err := projectsDeleteCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("error marking id flag as required: %v\n", err)
	}

	// Flags for occurrences command
	projectsOccurrencesCmd.Flags().
		StringVar(&projectID, "id", "", "Project ID (optional - if not specified, shows all projects)")
	projectsOccurrencesCmd.Flags().
		StringVar(&projectOccurrencesPeriod, "period", "day", "Time period (hour, day, week, or month)")
	projectsOccurrencesCmd.Flags().
		StringVar(&projectOccurrencesEnv, "environment", "", "Filter by environment")
	projectsOccurrencesCmd.Flags().
		StringVarP(&projectOutputFormat, "output", "o", "table", "Output format (table or json)")
}
