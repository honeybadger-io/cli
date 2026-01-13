package cmd

import (
	"fmt"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/cli/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:     "tui",
	Short:   "Start the interactive terminal UI",
	GroupID: GroupDataAPI,
	Long: `Start an interactive terminal UI for browsing your Honeybadger data.

The TUI allows you to navigate through your accounts, projects, faults,
deployments, uptime sites, and more using a keyboard-driven interface.

Navigation:
  ↑/k        Move up
  ↓/j        Move down
  Enter/→/l  Select/Drill down
  Esc/←/h    Go back
  r          Refresh current view
  q          Quit (or go back)
  ?          Show help

This command requires an auth token. Set it using --auth-token flag or
HONEYBADGER_AUTH_TOKEN environment variable.`,
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

		app := tui.NewApp(client)
		return app.Run()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
