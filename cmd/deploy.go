package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	environment string
	repository  string
	revision    string
	localUser   string
	apiEndpoint = "https://api.honeybadger.io"
)

type deployPayload struct {
	Deploy struct {
		Environment string `json:"environment,omitempty"`
		Repository  string `json:"repository,omitempty"`
		Revision    string `json:"revision,omitempty"`
		LocalUser   string `json:"local_username,omitempty"`
		Timestamp   string `json:"timestamp,omitempty"`
	} `json:"deploy"`
}

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Report a deployment to Honeybadger",
	Long: `Report a deployment to Honeybadger's Reporting API.
This command sends deployment information including environment, repository,
revision, and the local username of the person deploying.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			return fmt.Errorf("API key is required. Set it using --api-key flag or HONEYBADGER_API_KEY environment variable")
		}

		if environment == "" {
			return fmt.Errorf("environment is required. Set it using --environment flag")
		}

		payload := deployPayload{}
		payload.Deploy.Environment = environment
		payload.Deploy.Repository = repository
		payload.Deploy.Revision = revision
		payload.Deploy.LocalUser = localUser
		payload.Deploy.Timestamp = time.Now().UTC().Format(time.RFC3339)

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("error marshaling payload: %w", err)
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/deploys", apiEndpoint), bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error sending request: %w", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				fmt.Printf("error closing response body: %v\n", err)
			}
		}()

		if resp.StatusCode != http.StatusCreated {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("error reading response body: %w", err)
			}
			return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		}

		fmt.Println("Deployment successfully reported to Honeybadger")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment being deployed to")
	deployCmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository being deployed")
	deployCmd.Flags().StringVarP(&revision, "revision", "v", "", "Revision being deployed")
	deployCmd.Flags().StringVarP(&localUser, "user", "u", "", "Local username")

	if err := deployCmd.MarkFlagRequired("environment"); err != nil {
		fmt.Printf("error marking environment flag as required: %v\n", err)
	}
}
