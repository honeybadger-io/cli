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
	teamID            string
	teamsOutputFormat string
	teamName          string
)

// teamsCmd represents the teams command
var teamsCmd = &cobra.Command{
	Use:     "teams",
	Short:   "Manage Honeybadger teams",
	GroupID: GroupDataAPI,
	Long:    `View and manage the teams of the account your API Token belongs to.`,
}

// teamsListCmd represents the teams list command
var teamsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List teams for the account",
	Long: `List all teams in the account your API Token belongs to.` + v3Reference(
		"teams/list-teams",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		teams, err := client.Teams.ListAll(context.Background())
		if err != nil {
			return dataAPIError("list teams", err)
		}

		if teamsOutputFormat == "json" {
			return printJSON(teams)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tNAME\tCREATED")
		for _, team := range teams {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
				team.Id,
				team.Name,
				team.CreatedAt.Format("2006-01-02 15:04"))
		}
		return w.Flush()
	},
}

// teamsGetCmd represents the teams get command
var teamsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a team by ID",
	Long:  `Get detailed information about a specific team.` + v3Reference("teams/get-team"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == "" {
			return fmt.Errorf("team ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		team, err := client.Teams.Get(context.Background(), teamID)
		if err != nil {
			return dataAPIError("get team", err)
		}

		if teamsOutputFormat == "json" {
			return printJSON(team)
		}

		fmt.Printf("Team Details:\n")
		fmt.Printf("  ID: %s\n", team.Id)
		fmt.Printf("  Name: %s\n", team.Name)
		fmt.Printf("  Account ID: %s\n", team.AccountId)
		fmt.Printf("  Created: %s\n", team.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

// teamsCreateCmd represents the teams create command
var teamsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new team",
	Long: `Create a new team in the account your API Token belongs to.` + v3Reference(
		"teams/create-team",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamName == "" {
			return fmt.Errorf("team name is required. Set it using --name flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		team, err := client.Teams.Create(
			context.Background(),
			apiv3.TeamCreateParams{Name: teamName},
		)
		if err != nil {
			return dataAPIError("create team", err)
		}

		if teamsOutputFormat == "json" {
			return printJSON(team)
		}
		fmt.Printf("Team created successfully!\n")
		fmt.Printf("  ID: %s\n", team.Id)
		fmt.Printf("  Name: %s\n", team.Name)
		return nil
	},
}

// teamsUpdateCmd represents the teams update command
var teamsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing team",
	Long:  `Update an existing team's name.` + v3Reference("teams/update-team"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == "" {
			return fmt.Errorf("team ID is required. Set it using --id flag")
		}
		if teamName == "" {
			return fmt.Errorf("team name is required. Set it using --name flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		team, err := client.Teams.Update(
			context.Background(),
			teamID,
			apiv3.TeamUpdateParams{Name: &teamName},
		)
		if err != nil {
			return dataAPIError("update team", err)
		}

		if teamsOutputFormat == "json" {
			return printJSON(team)
		}
		fmt.Printf("Team updated successfully!\n")
		fmt.Printf("  ID: %s\n", team.Id)
		fmt.Printf("  Name: %s\n", team.Name)
		return nil
	},
}

// teamsDeleteCmd represents the teams delete command
var teamsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a team",
	Long:  `Delete a team by ID. This action cannot be undone.` + v3Reference("teams/delete-team"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == "" {
			return fmt.Errorf("team ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Teams.Delete(context.Background(), teamID); err != nil {
			return dataAPIError("delete team", err)
		}

		fmt.Println("Team deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(teamsCmd)

	// Add subcommands
	teamsCmd.AddCommand(teamsListCmd)
	teamsCmd.AddCommand(teamsGetCmd)
	teamsCmd.AddCommand(teamsCreateCmd)
	teamsCmd.AddCommand(teamsUpdateCmd)
	teamsCmd.AddCommand(teamsDeleteCmd)

	// Flags for list command
	teamsListCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	teamsGetCmd.Flags().StringVar(&teamID, "id", "", "Team ID")
	teamsGetCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	teamsCreateCmd.Flags().StringVar(&teamName, "name", "", "Team name")
	teamsCreateCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsCreateCmd.MarkFlagRequired("name")

	// Flags for update command
	teamsUpdateCmd.Flags().StringVar(&teamID, "id", "", "Team ID")
	teamsUpdateCmd.Flags().StringVar(&teamName, "name", "", "New team name")
	teamsUpdateCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsUpdateCmd.MarkFlagRequired("id")
	_ = teamsUpdateCmd.MarkFlagRequired("name")

	// Flags for delete command
	teamsDeleteCmd.Flags().StringVar(&teamID, "id", "", "Team ID")
	_ = teamsDeleteCmd.MarkFlagRequired("id")
}
