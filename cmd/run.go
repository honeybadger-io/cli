package cmd

import (
	"bytes"
	"context"
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

const (
	maxOutputSize = 8 * 1024 // 8KB max for stdout/stderr sent to API
	httpTimeout   = 30 * time.Second
	truncatedMsg  = "\n[output truncated]"
)

var (
	checkInID   string
	slug        string
	runExitCode int       // stores exit code from wrapped command
	exitFunc    = os.Exit // injectable for testing
)

type checkInPayload struct {
	CheckIn struct {
		Status   string `json:"status"`
		Duration int64  `json:"duration,omitempty"`
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
  hb run --slug daily-backup -- pg_dump -U postgres mydb

Note: Shell operators such as ">" are interpreted by your shell before hb runs,
so if you need redirection or other shell features, wrap them in a shell script
and invoke that script with "hb run".`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if checkInID == "" && slug == "" {
			return fmt.Errorf("either check-in ID (--id) or slug (--slug) is required")
		}
		if checkInID != "" && slug != "" {
			return fmt.Errorf("cannot specify both check-in ID and slug")
		}

		// API key is only required when using slug
		apiKey := viper.GetString("api_key")
		if slug != "" && apiKey == "" {
			return fmt.Errorf(
				"API key is required when using --slug. " +
					"Set it using --api-key flag or HONEYBADGER_API_KEY environment variable",
			)
		}

		// Prepare command execution
		command := args[0]
		var cmdArgs []string
		if len(args) > 1 {
			cmdArgs = args[1:]
		}

		execCmd := exec.Command(command, cmdArgs...) // nolint:gosec

		// Use MultiWriter to stream output in real-time while capturing it
		var stdout, stderr bytes.Buffer
		execCmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
		execCmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

		// Execute command and measure duration
		startTime := time.Now()
		execErr := execCmd.Run()
		durationMs := time.Since(startTime).Milliseconds()

		// Determine exit code
		runExitCode = 0
		if execErr != nil {
			if exitErr, ok := execErr.(*exec.ExitError); ok {
				runExitCode = exitErr.ExitCode()
			} else {
				// For non-exit errors (like command not found), use -1
				runExitCode = -1
			}
		}

		// Prepare payload
		payload := checkInPayload{}
		payload.CheckIn.Duration = durationMs
		payload.CheckIn.Stdout = truncateOutput(stdout.String())
		payload.CheckIn.Stderr = truncateOutput(stderr.String())
		payload.CheckIn.ExitCode = runExitCode

		if execErr != nil {
			payload.CheckIn.Status = "error"
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

		// Create request with timeout
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		// Send request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send check-in to Honeybadger: %w", err)
		}
		defer resp.Body.Close() // nolint:errcheck

		// Check response status
		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf(
					"unexpected status code: %d, and failed to read response body: %v",
					resp.StatusCode,
					err,
				)
			}
			return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
		}

		fmt.Fprintf(
			os.Stderr,
			"Check-in reported to Honeybadger (duration: %dms, status: %s)\n",
			durationMs,
			payload.CheckIn.Status,
		)

		// Exit with the same code as the wrapped command
		if runExitCode != 0 {
			exitFunc(runExitCode)
		}
		return nil
	},
}

// truncateOutput truncates output to maxOutputSize if necessary
func truncateOutput(s string) string {
	if len(s) <= maxOutputSize {
		return s
	}
	return s[:maxOutputSize-len(truncatedMsg)] + truncatedMsg
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&checkInID, "id", "i", "", "Check-in ID to report")
	runCmd.Flags().StringVarP(&slug, "slug", "s", "", "Check-in slug to report")
}
