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
	accountsOutputFormat string
	accountID            int
	accountUserID        int
	accountUserRole      string
	accountInvitationID  int
	accountCLIInputJSON  string
)

// accountsCmd represents the accounts command
var accountsCmd = &cobra.Command{
	Use:     "accounts",
	Short:   "Manage Honeybadger accounts",
	GroupID: GroupDataAPI,
	Long:    `View and manage your Honeybadger accounts, users, and invitations.`,
}

// accountsListCmd represents the accounts list command
var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	Long:  `List all accounts accessible with your auth token.`,
	RunE: func(_ *cobra.Command, _ []string) error {
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
		accounts, err := client.Accounts.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(accounts, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL")
			for _, account := range accounts {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n",
					account.ID,
					account.Name,
					account.Email)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// accountsGetCmd represents the accounts get command
var accountsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an account by ID",
	Long:  `Get detailed information about a specific account including quota and API stats.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --id flag")
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
		account, err := client.Accounts.Get(ctx, accountID)
		if err != nil {
			return fmt.Errorf("failed to get account: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(account, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Account Details:\n")
			fmt.Printf("  ID: %d\n", account.ID)
			fmt.Printf("  Name: %s\n", account.Name)
			fmt.Printf("  Email: %s\n", account.Email)
			if account.Active != nil {
				fmt.Printf("  Active: %v\n", *account.Active)
			}
			if account.Parked != nil {
				fmt.Printf("  Parked: %v\n", *account.Parked)
			}
			if account.QuotaConsumed != nil {
				fmt.Printf("  Quota Consumed: %.2f%%\n", *account.QuotaConsumed)
			}
		}

		return nil
	},
}

// accountsUsersListCmd represents the accounts users list command
var accountsUsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users for an account",
	Long:  `List all users associated with an account.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
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
		users, err := client.Accounts.ListUsers(ctx, accountID)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(users, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tNAME\tEMAIL\tROLE")
			for _, user := range users {
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					user.ID,
					user.Name,
					user.Email,
					user.Role)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// accountsUsersGetCmd represents the accounts users get command
var accountsUsersGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a user by ID",
	Long:  `Get detailed information about a specific user in an account.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if accountUserID == 0 {
			return fmt.Errorf("user ID is required. Set it using --user-id flag")
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
		user, err := client.Accounts.GetUser(ctx, accountID, accountUserID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(user, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("User Details:\n")
			fmt.Printf("  ID: %d\n", user.ID)
			fmt.Printf("  Name: %s\n", user.Name)
			fmt.Printf("  Email: %s\n", user.Email)
			fmt.Printf("  Role: %s\n", user.Role)
		}

		return nil
	},
}

// accountsUsersUpdateCmd represents the accounts users update command
var accountsUsersUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a user's role",
	Long:  `Update a user's role in an account. Valid roles: Member, Billing, Admin, Owner.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if accountUserID == 0 {
			return fmt.Errorf("user ID is required. Set it using --user-id flag")
		}
		if accountUserRole == "" {
			return fmt.Errorf("role is required. Set it using --role flag")
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
		user, err := client.Accounts.UpdateUser(ctx, accountID, accountUserID, accountUserRole)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(user, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("User updated successfully!\n")
			fmt.Printf("  ID: %d\n", user.ID)
			fmt.Printf("  Name: %s\n", user.Name)
			fmt.Printf("  Role: %s\n", user.Role)
		}

		return nil
	},
}

// accountsUsersRemoveCmd represents the accounts users remove command
var accountsUsersRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a user from an account",
	Long:  `Remove a user from an account. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if accountUserID == 0 {
			return fmt.Errorf("user ID is required. Set it using --user-id flag")
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
		err := client.Accounts.RemoveUser(ctx, accountID, accountUserID)
		if err != nil {
			return fmt.Errorf("failed to remove user: %w", err)
		}

		fmt.Println("User removed successfully")
		return nil
	},
}

// accountsUsersCmd is the parent command for user operations
var accountsUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage account users",
	Long:  `View and manage users in your Honeybadger account.`,
}

// accountsInvitationsListCmd represents the accounts invitations list command
var accountsInvitationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List invitations for an account",
	Long:  `List all pending invitations for an account.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
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
		invitations, err := client.Accounts.ListInvitations(ctx, accountID)
		if err != nil {
			return fmt.Errorf("failed to list invitations: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitations, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tEMAIL\tROLE\tCREATED\tACCEPTED")
			for _, inv := range invitations {
				accepted := "No"
				if inv.AcceptedAt != nil {
					accepted = inv.AcceptedAt.Format("2006-01-02 15:04")
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					inv.ID,
					inv.Email,
					inv.Role,
					inv.CreatedAt.Format("2006-01-02 15:04"),
					accepted)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// accountsInvitationsGetCmd represents the accounts invitations get command
var accountsInvitationsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an invitation by ID",
	Long:  `Get detailed information about a specific invitation.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if accountInvitationID == 0 {
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
		invitation, err := client.Accounts.GetInvitation(ctx, accountID, accountInvitationID)
		if err != nil {
			return fmt.Errorf("failed to get invitation: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitation, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Invitation Details:\n")
			fmt.Printf("  ID: %d\n", invitation.ID)
			fmt.Printf("  Email: %s\n", invitation.Email)
			fmt.Printf("  Role: %s\n", invitation.Role)
			fmt.Printf("  Token: %s\n", invitation.Token)
			fmt.Printf("  Created: %s\n", invitation.CreatedAt.Format("2006-01-02 15:04:05"))
			if invitation.AcceptedAt != nil {
				fmt.Printf("  Accepted: %s\n", invitation.AcceptedAt.Format("2006-01-02 15:04:05"))
			}
			if len(invitation.TeamIDs) > 0 {
				fmt.Printf("  Team IDs: %v\n", invitation.TeamIDs)
			}
		}

		return nil
	},
}

// accountsInvitationsCreateCmd represents the accounts invitations create command
var accountsInvitationsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new invitation",
	Long: `Create a new invitation to join an account.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "invitation": {
    "email": "user@example.com",
    "role": "Member",
    "team_ids": [123, 456]
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if accountCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(accountCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Invitation hbapi.AccountInvitationParams `json:"invitation"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		invitation, err := client.Accounts.CreateInvitation(ctx, accountID, payload.Invitation)
		if err != nil {
			return fmt.Errorf("failed to create invitation: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitation, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Invitation created successfully!\n")
			fmt.Printf("  ID: %d\n", invitation.ID)
			fmt.Printf("  Email: %s\n", invitation.Email)
			fmt.Printf("  Role: %s\n", invitation.Role)
			fmt.Printf("  Token: %s\n", invitation.Token)
		}

		return nil
	},
}

// accountsInvitationsUpdateCmd represents the accounts invitations update command
var accountsInvitationsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an invitation",
	Long: `Update an existing invitation.

The --cli-input-json flag accepts either a JSON string or a file path prefixed with 'file://'.

Example JSON payload:
{
  "invitation": {
    "role": "Admin",
    "team_ids": [123]
  }
}`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if accountInvitationID == 0 {
			return fmt.Errorf("invitation ID is required. Set it using --invitation-id flag")
		}
		if accountCLIInputJSON == "" {
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

		jsonData, err := readJSONInput(accountCLIInputJSON)
		if err != nil {
			return fmt.Errorf("failed to read JSON input: %w", err)
		}

		var payload struct {
			Invitation hbapi.AccountInvitationParams `json:"invitation"`
		}
		if err := json.Unmarshal(jsonData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON payload: %w", err)
		}

		ctx := context.Background()
		invitation, err := client.Accounts.UpdateInvitation(
			ctx,
			accountID,
			accountInvitationID,
			payload.Invitation,
		)
		if err != nil {
			return fmt.Errorf("failed to update invitation: %w", err)
		}

		switch accountsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(invitation, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Invitation updated successfully!\n")
			fmt.Printf("  ID: %d\n", invitation.ID)
			fmt.Printf("  Email: %s\n", invitation.Email)
			fmt.Printf("  Role: %s\n", invitation.Role)
		}

		return nil
	},
}

// accountsInvitationsDeleteCmd represents the accounts invitations delete command
var accountsInvitationsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an invitation",
	Long:  `Delete a pending invitation. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if accountID == 0 {
			return fmt.Errorf("account ID is required. Set it using --account-id flag")
		}
		if accountInvitationID == 0 {
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
		err := client.Accounts.DeleteInvitation(ctx, accountID, accountInvitationID)
		if err != nil {
			return fmt.Errorf("failed to delete invitation: %w", err)
		}

		fmt.Println("Invitation deleted successfully")
		return nil
	},
}

// accountsInvitationsCmd is the parent command for invitation operations
var accountsInvitationsCmd = &cobra.Command{
	Use:   "invitations",
	Short: "Manage account invitations",
	Long:  `View and manage invitations to your Honeybadger account.`,
}

func init() {
	rootCmd.AddCommand(accountsCmd)

	// Add subcommands
	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsGetCmd)
	accountsCmd.AddCommand(accountsUsersCmd)
	accountsCmd.AddCommand(accountsInvitationsCmd)

	// Users subcommands
	accountsUsersCmd.AddCommand(accountsUsersListCmd)
	accountsUsersCmd.AddCommand(accountsUsersGetCmd)
	accountsUsersCmd.AddCommand(accountsUsersUpdateCmd)
	accountsUsersCmd.AddCommand(accountsUsersRemoveCmd)

	// Invitations subcommands
	accountsInvitationsCmd.AddCommand(accountsInvitationsListCmd)
	accountsInvitationsCmd.AddCommand(accountsInvitationsGetCmd)
	accountsInvitationsCmd.AddCommand(accountsInvitationsCreateCmd)
	accountsInvitationsCmd.AddCommand(accountsInvitationsUpdateCmd)
	accountsInvitationsCmd.AddCommand(accountsInvitationsDeleteCmd)

	// Flags for list command
	accountsListCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	accountsGetCmd.Flags().IntVar(&accountID, "id", 0, "Account ID")
	accountsGetCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = accountsGetCmd.MarkFlagRequired("id")

	// Common account ID flag for users subcommands
	accountsUsersCmd.PersistentFlags().IntVar(&accountID, "account-id", 0, "Account ID")

	// Flags for users list
	accountsUsersListCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for users get
	accountsUsersGetCmd.Flags().IntVar(&accountUserID, "user-id", 0, "User ID")
	accountsUsersGetCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = accountsUsersGetCmd.MarkFlagRequired("user-id")

	// Flags for users update
	accountsUsersUpdateCmd.Flags().IntVar(&accountUserID, "user-id", 0, "User ID")
	accountsUsersUpdateCmd.Flags().
		StringVar(&accountUserRole, "role", "", "New role (Member, Billing, Admin, Owner)")
	accountsUsersUpdateCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = accountsUsersUpdateCmd.MarkFlagRequired("user-id")
	_ = accountsUsersUpdateCmd.MarkFlagRequired("role")

	// Flags for users remove
	accountsUsersRemoveCmd.Flags().IntVar(&accountUserID, "user-id", 0, "User ID")
	_ = accountsUsersRemoveCmd.MarkFlagRequired("user-id")

	// Common account ID flag for invitations subcommands
	accountsInvitationsCmd.PersistentFlags().IntVar(&accountID, "account-id", 0, "Account ID")

	// Flags for invitations list
	accountsInvitationsListCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for invitations get
	accountsInvitationsGetCmd.Flags().
		IntVar(&accountInvitationID, "invitation-id", 0, "Invitation ID")
	accountsInvitationsGetCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = accountsInvitationsGetCmd.MarkFlagRequired("invitation-id")

	// Flags for invitations create
	accountsInvitationsCreateCmd.Flags().
		StringVar(&accountCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	accountsInvitationsCreateCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = accountsInvitationsCreateCmd.MarkFlagRequired("cli-input-json")

	// Flags for invitations update
	accountsInvitationsUpdateCmd.Flags().
		IntVar(&accountInvitationID, "invitation-id", 0, "Invitation ID")
	accountsInvitationsUpdateCmd.Flags().
		StringVar(&accountCLIInputJSON, "cli-input-json", "", "JSON payload (string or file://path)")
	accountsInvitationsUpdateCmd.Flags().
		StringVarP(&accountsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = accountsInvitationsUpdateCmd.MarkFlagRequired("invitation-id")
	_ = accountsInvitationsUpdateCmd.MarkFlagRequired("cli-input-json")

	// Flags for invitations delete
	accountsInvitationsDeleteCmd.Flags().
		IntVar(&accountInvitationID, "invitation-id", 0, "Invitation ID")
	_ = accountsInvitationsDeleteCmd.MarkFlagRequired("invitation-id")
}
