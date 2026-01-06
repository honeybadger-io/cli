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
	teamsAccountID    int
	teamID            int
	teamsOutputFormat string
	teamName          string
	teamMemberID      int
	teamMemberAdmin   bool
	teamInvitationID  int
	teamCLIInputJSON  string
)

// teamsCmd represents the teams command
var teamsCmd = &cobra.Command{
	Use:     "teams",
	Short:   "Manage Honeybadger teams",
	GroupID: GroupDataAPI,
	Long:    `View and manage teams, team members, and team invitations.`,
}

// teamsListCmd represents the teams list command
var teamsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List teams for an account",
	Long:  `List all teams for a specific account.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamsAccountID == 0 {
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
		teams, err := client.Teams.List(ctx, teamsAccountID)
		if err != nil {
			return fmt.Errorf("failed to list teams: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(teams, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tCREATED")
			for _, team := range teams {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n",
					team.ID,
					team.Name,
					team.CreatedAt.Format("2006-01-02 15:04"))
			}
			_ = w.Flush()
		}

		return nil
	},
}

// teamsGetCmd represents the teams get command
var teamsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a team by ID",
	Long:  `Get detailed information about a specific team.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --id flag")
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
		team, err := client.Teams.Get(ctx, teamID)
		if err != nil {
			return fmt.Errorf("failed to get team: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(team, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Team Details:\n")
			fmt.Printf("  ID: %d\n", team.ID)
			fmt.Printf("  Name: %s\n", team.Name)
			fmt.Printf("  Account ID: %d\n", team.AccountID)
			fmt.Printf("  Created: %s\n", team.CreatedAt.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

// teamsCreateCmd represents the teams create command
var teamsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new team",
	Long:  `Create a new team for an account.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamsAccountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if teamName == "" {
			return fmt.Errorf("team name is required. Set it using --name flag")
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
		team, err := client.Teams.Create(ctx, teamsAccountID, teamName)
		if err != nil {
			return fmt.Errorf("failed to create team: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(team, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Team created successfully!\n")
			fmt.Printf("  ID: %d\n", team.ID)
			fmt.Printf("  Name: %s\n", team.Name)
		}

		return nil
	},
}

// teamsUpdateCmd represents the teams update command
var teamsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing team",
	Long:  `Update an existing team's name.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --id flag")
		}
		if teamName == "" {
			return fmt.Errorf("team name is required. Set it using --name flag")
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
		team, err := client.Teams.Update(ctx, teamID, teamName)
		if err != nil {
			return fmt.Errorf("failed to update team: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(team, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Team updated successfully!\n")
			fmt.Printf("  ID: %d\n", team.ID)
			fmt.Printf("  Name: %s\n", team.Name)
		}

		return nil
	},
}

// teamsDeleteCmd represents the teams delete command
var teamsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a team",
	Long:  `Delete a team by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --id flag")
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
		err := client.Teams.Delete(ctx, teamID)
		if err != nil {
			return fmt.Errorf("failed to delete team: %w", err)
		}

		fmt.Println("Team deleted successfully")
		return nil
	},
}

// teamsMembersCmd is the parent command for team member operations
var teamsMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage team members",
	Long:  `View and manage members of a team.`,
}

// teamsMembersListCmd represents the teams members list command
var teamsMembersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List members of a team",
	Long:  `List all members of a specific team.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
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
		members, err := client.Teams.ListMembers(ctx, teamID)
		if err != nil {
			return fmt.Errorf("failed to list team members: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(members, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tADMIN")
			for _, member := range members {
				admin := " "
				if member.Admin {
					admin = "Yes"
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					member.ID,
					member.Name,
					member.Email,
					admin)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// teamsMembersUpdateCmd represents the teams members update command
var teamsMembersUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a team member's permissions",
	Long:  `Update a team member's admin status.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
		}
		if teamMemberID == 0 {
			return fmt.Errorf("member ID is required. Set it using --member-id flag")
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
		member, err := client.Teams.UpdateMember(ctx, teamID, teamMemberID, teamMemberAdmin)
		if err != nil {
			return fmt.Errorf("failed to update team member: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(member, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Team member updated successfully!\n")
			fmt.Printf("  ID: %d\n", member.ID)
			fmt.Printf("  Name: %s\n", member.Name)
			fmt.Printf("  Admin: %v\n", member.Admin)
		}

		return nil
	},
}

// teamsMembersRemoveCmd represents the teams members remove command
var teamsMembersRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a member from a team",
	Long:  `Remove a member from a team. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
		}
		if teamMemberID == 0 {
			return fmt.Errorf("member ID is required. Set it using --member-id flag")
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
		err := client.Teams.RemoveMember(ctx, teamID, teamMemberID)
		if err != nil {
			return fmt.Errorf("failed to remove team member: %w", err)
		}

		fmt.Println("Team member removed successfully")
		return nil
	},
}

// teamsInvitationsCmd is the parent command for team invitation operations
var teamsInvitationsCmd = &cobra.Command{
	Use:   "invitations",
	Short: "Manage team invitations",
	Long:  `View and manage invitations to join a team.`,
}

// teamsInvitationsListCmd represents the teams invitations list command
var teamsInvitationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List invitations for a team",
	Long:  `List all pending invitations for a team.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
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
		invitations, err := client.Teams.ListInvitations(ctx, teamID)
		if err != nil {
			return fmt.Errorf("failed to list team invitations: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitations, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tEMAIL\tADMIN\tCREATED\tACCEPTED")
			for _, inv := range invitations {
				admin := " "
				if inv.Admin {
					admin = "Yes"
				}
				accepted := "No"
				if inv.AcceptedAt != nil {
					accepted = inv.AcceptedAt.Format("2006-01-02 15:04")
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					inv.ID,
					inv.Email,
					admin,
					inv.CreatedAt.Format("2006-01-02 15:04"),
					accepted)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// teamsInvitationsGetCmd represents the teams invitations get command
var teamsInvitationsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a team invitation by ID",
	Long:  `Get detailed information about a specific team invitation.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
		}
		if teamInvitationID == 0 {
			return fmt.Errorf("invitation ID is required. Set it using --invitation-id flag")
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
		invitation, err := client.Teams.GetInvitation(ctx, teamID, teamInvitationID)
		if err != nil {
			return fmt.Errorf("failed to get team invitation: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitation, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Team Invitation Details:\n")
			fmt.Printf("  ID: %d\n", invitation.ID)
			fmt.Printf("  Email: %s\n", invitation.Email)
			fmt.Printf("  Admin: %v\n", invitation.Admin)
			fmt.Printf("  Token: %s\n", invitation.Token)
			fmt.Printf("  Created: %s\n", invitation.CreatedAt.Format("2006-01-02 15:04:05"))
			if invitation.AcceptedAt != nil {
				fmt.Printf("  Accepted: %s\n", invitation.AcceptedAt.Format("2006-01-02 15:04:05"))
			}
			if invitation.Message != nil {
				fmt.Printf("  Message: %s\n", *invitation.Message)
			}
		}

		return nil
	},
}

// teamsInvitationsCreateCmd represents the teams invitations create command
var teamsInvitationsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new team invitation",
	Long: `Create a new invitation to join a team.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "team_invitation": {
    "email": "user@example.com",
    "admin": false,
    "message": "Welcome to the team!"
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
		}
		if teamCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(teamCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			TeamInvitation hbapi.TeamInvitationParams `json:"team_invitation"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		invitation, err := client.Teams.CreateInvitation(ctx, teamID, payload.TeamInvitation)
		if err != nil {
			return fmt.Errorf("failed to create team invitation: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitation, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Team invitation created successfully!\n")
			fmt.Printf("  ID: %d\n", invitation.ID)
			fmt.Printf("  Email: %s\n", invitation.Email)
			fmt.Printf("  Token: %s\n", invitation.Token)
		}

		return nil
	},
}

// teamsInvitationsUpdateCmd represents the teams invitations update command
var teamsInvitationsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a team invitation",
	Long: `Update an existing team invitation.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "team_invitation": {
    "admin": true
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
		}
		if teamInvitationID == 0 {
			return fmt.Errorf("invitation ID is required. Set it using --invitation-id flag")
		}
		if teamCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(teamCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			TeamInvitation hbapi.TeamInvitationParams `json:"team_invitation"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		invitation, err := client.Teams.UpdateInvitation(
			ctx,
			teamID,
			teamInvitationID,
			payload.TeamInvitation,
		)
		if err != nil {
			return fmt.Errorf("failed to update team invitation: %w", err)
		}

		switch teamsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitation, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Team invitation updated successfully!\n")
			fmt.Printf("  ID: %d\n", invitation.ID)
			fmt.Printf("  Email: %s\n", invitation.Email)
			fmt.Printf("  Admin: %v\n", invitation.Admin)
		}

		return nil
	},
}

// teamsInvitationsDeleteCmd represents the teams invitations delete command
var teamsInvitationsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a team invitation",
	Long:  `Delete a pending team invitation. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if teamID == 0 {
			return fmt.Errorf("team ID is required. Set it using --team-id flag")
		}
		if teamInvitationID == 0 {
			return fmt.Errorf("invitation ID is required. Set it using --invitation-id flag")
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
		err := client.Teams.DeleteInvitation(ctx, teamID, teamInvitationID)
		if err != nil {
			return fmt.Errorf("failed to delete team invitation: %w", err)
		}

		fmt.Println("Team invitation deleted successfully")
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
	teamsCmd.AddCommand(teamsMembersCmd)
	teamsCmd.AddCommand(teamsInvitationsCmd)

	// Members subcommands
	teamsMembersCmd.AddCommand(teamsMembersListCmd)
	teamsMembersCmd.AddCommand(teamsMembersUpdateCmd)
	teamsMembersCmd.AddCommand(teamsMembersRemoveCmd)

	// Invitations subcommands
	teamsInvitationsCmd.AddCommand(teamsInvitationsListCmd)
	teamsInvitationsCmd.AddCommand(teamsInvitationsGetCmd)
	teamsInvitationsCmd.AddCommand(teamsInvitationsCreateCmd)
	teamsInvitationsCmd.AddCommand(teamsInvitationsUpdateCmd)
	teamsInvitationsCmd.AddCommand(teamsInvitationsDeleteCmd)

	// Flags for list command
	teamsListCmd.Flags().IntVar(&teamsAccountID, "account-id", 0, "Account ID")
	teamsListCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "table", "Output format (table or json)")
	_ = teamsListCmd.MarkFlagRequired("account-id")

	// Flags for get command
	teamsGetCmd.Flags().IntVar(&teamID, "id", 0, "Team ID")
	teamsGetCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	teamsCreateCmd.Flags().IntVar(&teamsAccountID, "account-id", 0, "Account ID")
	teamsCreateCmd.Flags().StringVar(&teamName, "name", "", "Team name")
	teamsCreateCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsCreateCmd.MarkFlagRequired("account-id")
	_ = teamsCreateCmd.MarkFlagRequired("name")

	// Flags for update command
	teamsUpdateCmd.Flags().IntVar(&teamID, "id", 0, "Team ID")
	teamsUpdateCmd.Flags().StringVar(&teamName, "name", "", "New team name")
	teamsUpdateCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsUpdateCmd.MarkFlagRequired("id")
	_ = teamsUpdateCmd.MarkFlagRequired("name")

	// Flags for delete command
	teamsDeleteCmd.Flags().IntVar(&teamID, "id", 0, "Team ID")
	_ = teamsDeleteCmd.MarkFlagRequired("id")

	// Common team ID flag for members subcommands
	teamsMembersCmd.PersistentFlags().IntVar(&teamID, "team-id", 0, "Team ID")

	// Flags for members list
	teamsMembersListCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for members update
	teamsMembersUpdateCmd.Flags().IntVar(&teamMemberID, "member-id", 0, "Member ID")
	teamsMembersUpdateCmd.Flags().BoolVar(&teamMemberAdmin, "admin", false, "Set admin status")
	teamsMembersUpdateCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsMembersUpdateCmd.MarkFlagRequired("member-id")

	// Flags for members remove
	teamsMembersRemoveCmd.Flags().IntVar(&teamMemberID, "member-id", 0, "Member ID")
	_ = teamsMembersRemoveCmd.MarkFlagRequired("member-id")

	// Common team ID flag for invitations subcommands
	teamsInvitationsCmd.PersistentFlags().IntVar(&teamID, "team-id", 0, "Team ID")

	// Flags for invitations list
	teamsInvitationsListCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for invitations get
	teamsInvitationsGetCmd.Flags().IntVar(&teamInvitationID, "invitation-id", 0, "Invitation ID")
	teamsInvitationsGetCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsInvitationsGetCmd.MarkFlagRequired("invitation-id")

	// Flags for invitations create
	teamsInvitationsCreateCmd.Flags().
		StringVar(&teamCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	teamsInvitationsCreateCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsInvitationsCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for invitations update
	teamsInvitationsUpdateCmd.Flags().IntVar(&teamInvitationID, "invitation-id", 0, "Invitation ID")
	teamsInvitationsUpdateCmd.Flags().
		StringVar(&teamCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	teamsInvitationsUpdateCmd.Flags().
		StringVarP(&teamsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = teamsInvitationsUpdateCmd.MarkFlagRequired("invitation-id")
	_ = teamsInvitationsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for invitations delete
	teamsInvitationsDeleteCmd.Flags().IntVar(&teamInvitationID, "invitation-id", 0, "Invitation ID")
	_ = teamsInvitationsDeleteCmd.MarkFlagRequired("invitation-id")
}
