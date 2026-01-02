package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile         string
	apiKey          string
	authToken       string
	endpoint        string
	defaultEndpoint = "https://api.honeybadger.io"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hb",
	Short: "Honeybadger CLI tool",
	Long: `A command line interface for interacting with Honeybadger's Reporting API.
This tool allows you to manage deployments and other reporting features.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is config/honeybadger.yml)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "Honeybadger API key (for Reporting API)")
	rootCmd.PersistentFlags().StringVar(&authToken, "auth-token", "", "Honeybadger personal auth token (for Data API)")
	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", defaultEndpoint, "Honeybadger endpoint")

	if err := viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key")); err != nil {
		fmt.Printf("error binding api-key flag: %v\n", err)
	}
	if err := viper.BindPFlag("auth_token", rootCmd.PersistentFlags().Lookup("auth-token")); err != nil {
		fmt.Printf("error binding auth-token flag: %v\n", err)
	}
	if err := viper.BindPFlag("endpoint", rootCmd.PersistentFlags().Lookup("endpoint")); err != nil {
		fmt.Printf("error binding endpoint flag: %v\n", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in config directory
		viper.AddConfigPath("config")
		viper.SetConfigType("yml")
		viper.SetConfigName("honeybadger")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("HONEYBADGER")
	viper.SetDefault("endpoint", defaultEndpoint)

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
