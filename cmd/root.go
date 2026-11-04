package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile         string
	apiKey          string
	apiTokenFlag    string
	endpoint        string
	defaultEndpoint = "https://api.honeybadger.io"

	// Version is the version string set via ldflags during build
	Version string
	// Commit is the git commit hash set via ldflags during build
	Commit string
	// Date is the build date set via ldflags during build
	Date string
)

// Command group IDs
const (
	GroupReportingAPI = "reporting"
	GroupDataAPI      = "data"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "hb",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return checkIDFlags(cmd)
	},
	Short: "Honeybadger CLI tool",
	Long: `A command line interface for interacting with Honeybadger.

This tool provides access to two APIs:

  Reporting API - For sending data to Honeybadger (deployments, metrics)
                  Authenticate with an Ingestion Key: --api-key or HONEYBADGER_API_KEY

  Data API      - For reading and managing your Honeybadger data
                  Authenticate with an API Token: --api-token or HONEYBADGER_API_TOKEN`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	if Version != "" {
		rootCmd.Version = Version
	} else {
		rootCmd.Version = "dev"
	}

	// Add command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    GroupReportingAPI,
		Title: "Reporting API Commands (use --api-key with an Ingestion Key):",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    GroupDataAPI,
		Title: "Data API Commands (use --api-token):",
	})

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is ~/.honeybadger-cli.yaml)")
	rootCmd.PersistentFlags().
		StringVar(&apiKey, "api-key", "", "Honeybadger Ingestion Key (for Reporting API)")
	rootCmd.PersistentFlags().
		StringVar(&apiTokenFlag, "api-token", "", "Honeybadger API Token (for Data API)")
	rootCmd.PersistentFlags().
		StringVar(&endpoint, "endpoint", defaultEndpoint, "Honeybadger endpoint")

	err := viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	if err != nil {
		fmt.Printf("error binding api-key flag: %v\n", err)
	}
	if err := viper.BindPFlag(
		"api_token",
		rootCmd.PersistentFlags().Lookup("api-token"),
	); err != nil {
		fmt.Printf("error binding api-token flag: %v\n", err)
	}
	if err := viper.BindPFlag(
		"endpoint",
		rootCmd.PersistentFlags().Lookup("endpoint"),
	); err != nil {
		fmt.Printf("error binding endpoint flag: %v\n", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in home directory
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
		viper.SetConfigType("yaml")
		viper.SetConfigName(".honeybadger-cli")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("HONEYBADGER")
	viper.SetDefault("endpoint", defaultEndpoint)

	// Register project_id for env var lookup (HONEYBADGER_PROJECT_ID).
	// Unlike api_key/api_token/endpoint, project_id has no root-level flag
	// to bind, so we use BindEnv to make viper aware of it.
	_ = viper.BindEnv("project_id")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Override with explicit flags if they were provided
	// This gives command-line flags precedence over environment variables
	if rootCmd.PersistentFlags().Changed("api-key") {
		viper.Set("api_key", apiKey)
	}
	if rootCmd.PersistentFlags().Changed("api-token") {
		viper.Set("api_token", apiTokenFlag)
	}
	if rootCmd.PersistentFlags().Changed("endpoint") {
		viper.Set("endpoint", endpoint)
	}
}

// convertEndpointForDataAPI converts api.honeybadger.io to app.honeybadger.io for Data API calls
func convertEndpointForDataAPI(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return endpoint
	}

	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		switch parsed.Host {
		case "api.honeybadger.io":
			parsed.Host = "app.honeybadger.io"
		case "eu-api.honeybadger.io":
			parsed.Host = "eu-app.honeybadger.io"
		}
		return parsed.String()
	}

	normalized := strings.TrimRight(trimmed, "/")
	switch normalized {
	case "https://api.honeybadger.io":
		return "https://app.honeybadger.io"
	case "https://eu-api.honeybadger.io":
		return "https://eu-app.honeybadger.io"
	default:
		return trimmed
	}
}

// parseTimeFlag parses a user-provided time string into a time.Time.
// Accepts RFC3339 (2024-01-01T00:00:00Z), date-only (2024-01-01),
// or datetime without zone (2024-01-01T00:00:00, treated as UTC).
func parseTimeFlag(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"invalid time format %q (expected YYYY-MM-DD or RFC3339 like 2024-01-01T00:00:00Z)", value,
	)
}

// readJSONInput reads JSON from either a direct string or a file path prefixed with 'file://'
func readJSONInput(input string) ([]byte, error) {
	if strings.HasPrefix(input, "file://") {
		// Read from file
		filePath := input[7:]
		return os.ReadFile(filePath) // #nosec G304 - User-provided file path is expected for CLI
	}
	// Use direct JSON string
	return []byte(input), nil
}
