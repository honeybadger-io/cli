package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:     "whoami",
	Short:   "Describe the API Token in use",
	GroupID: GroupDataAPI,
	Long: `Show the API Token the Data API commands are using: its kind, name, account,
scopes, the projects it can reach, when it expires and when it was last used.` + v3Reference("tokens/get-token"),
	RunE: func(_ *cobra.Command, _ []string) error {
		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		token, err := client.Tokens.Get(context.Background())
		if err != nil {
			return dataAPIError("describe the API Token", err)
		}

		fmt.Printf("Kind: %s\n", token.Kind)
		fmt.Printf("Name: %s\n", token.Name)
		fmt.Printf("Account: %s\n", token.AccountID)
		fmt.Printf("Scopes: %s\n", listOrNone(token.Scopes))
		fmt.Printf("Projects: %s\n", listOrNone(token.ProjectIDs))
		fmt.Printf("Expires: %s\n", timeOrNever(token.ExpiresAt))
		fmt.Printf("Last used: %s\n", timeOrNever(token.LastUsedAt))
		return nil
	},
}

func listOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func timeOrNever(at *string) string {
	if at == nil {
		return "never"
	}
	return *at
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
