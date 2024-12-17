package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	checkInID string
	slug      string
)

type checkInPayload struct {
	CheckIn struct {
		Status   string `json:"status"`
		Duration int    `json:"duration,omitempty"`
		Stdout   string `json:"stdout,omitempty"`
		Stderr   string `json:"stderr,omitempty"`
		ExitCode int    `json:"exit_code"`
	} `json:"check_in"`
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command and report its status to Honeybadger",
	Long: `Run a command and report its status to Honeybadger's Reporting API.
This command executes the provided command, captures its output and execution time,
and reports the results using either a check-in ID or slug.

Example:
  hb run --id check-123 -- /usr/local/bin/backup.sh
  hb run --slug daily-backup -- pg_dump -U postgres mydb > backup.sql`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			return fmt.Errorf("API key is required. Set it using --api-key flag or HONEYBADGER_API_KEY environment variable")
		}

		if checkInID == "" && slug == "" {
			return fmt.Errorf("either check-in ID (--id) or slug (--slug) is required")
		}
		if checkInID != "" && slug != "" {
			return fmt.Errorf("cannot specify both check-in ID and slug")
		}

		// Prepare command execution
		command := args[0]
		var cmdArgs []string
		if len(args) > 1 {
			cmdArgs = args[1:]
		}

		execCmd := exec.Command(command, cmdArgs...)
		var stdout, stderr bytes.Buffer
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr

		// Execute command and measure duration
		startTime := time.Now()
		err := execCmd.Run()
		duration := int(time.Since(startTime).Seconds())

		// Prepare payload
		payload := checkInPayload{}
		payload.CheckIn.Duration = duration
		payload.CheckIn.Stdout = stdout.String()
		payload.CheckIn.Stderr = stderr.String()
		payload.CheckIn.ExitCode = 0 // Default to 0 for success

		if err != nil {
			payload.CheckIn.Status = "error"
			if exitErr, ok := err.(*exec.ExitError); ok {
				payload.CheckIn.ExitCode = exitErr.ExitCode()
			} else {
				// For non-exit errors (like command not found), use -1
				payload.CheckIn.ExitCode = -1
			}
		} else {
			payload.CheckIn.Status = "success"
		}

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("error marshaling payload: %w", err)
		}

		apiEndpoint := viper.GetString("endpoint")
		var url string
		if checkInID != "" {
			url = fmt.Sprintf("%s/v1/check_in/%s", apiEndpoint, checkInID)
		} else {
			url = fmt.Sprintf("%s/v1/check_in/%s/%s", apiEndpoint, apiKey, slug)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		// Send request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
		}

		// Print command output to user's terminal
		if stdout.Len() > 0 {
			os.Stdout.Write(stdout.Bytes())
		}
		if stderr.Len() > 0 {
			os.Stderr.Write(stderr.Bytes())
		}

		fmt.Printf("\nCheck-in successfully reported to Honeybadger (duration: %ds)\n", duration)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&checkInID, "id", "i", "", "Check-in ID to report")
	runCmd.Flags().StringVarP(&slug, "slug", "s", "", "Check-in slug to report")
}
