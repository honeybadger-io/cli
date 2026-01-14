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
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	maxOutputSize = 16 * 1024 // 16KB max combined for stdout/stderr sent to API
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

type sharedLimiter struct {
	mu        sync.Mutex
	remaining int
}

type limitedBuffer struct {
	limiter   *sharedLimiter
	buffer    bytes.Buffer
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	b.limiter.mu.Lock()
	defer b.limiter.mu.Unlock()
	if b.limiter.remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}

	toWrite := len(p)
	if toWrite > b.limiter.remaining {
		reserve := 0
		if !b.truncated {
			reserve = len(truncatedMsg)
		}
		allowed := b.limiter.remaining - reserve
		if allowed < 0 {
			allowed = 0
		}
		if allowed > 0 {
			_, _ = b.buffer.Write(p[:allowed])
		}
		if reserve > 0 && b.limiter.remaining >= reserve {
			_, _ = b.buffer.WriteString(truncatedMsg)
		}
		b.truncated = true
		b.limiter.remaining = 0
		return len(p), nil
	}

	_, _ = b.buffer.Write(p)
	b.limiter.remaining -= toWrite
	return len(p), nil
}

func (b *limitedBuffer) String() string {
	return b.buffer.String()
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
so redirection works as usual. If you need more complex shell features, wrap
them in a shell script and invoke that script with "hb run".`,
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
		limiter := &sharedLimiter{remaining: maxOutputSize}
		stdout := &limitedBuffer{limiter: limiter}
		stderr := &limitedBuffer{limiter: limiter}
		execCmd.Stdout = io.MultiWriter(os.Stdout, stdout)
		execCmd.Stderr = io.MultiWriter(os.Stderr, stderr)

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
		payload.CheckIn.Stdout = stdout.String()
		payload.CheckIn.Stderr = stderr.String()
		payload.CheckIn.ExitCode = runExitCode

		if execErr != nil {
			payload.CheckIn.Status = "error"
		} else {
			payload.CheckIn.Status = "success"
		}

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal check-in payload: %v\n", err)
		} else {
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
				fmt.Fprintf(os.Stderr, "Failed to create check-in request: %v\n", err)
			} else {
				req.Header.Set("Content-Type", "application/json")

				// Send request
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to report check-in to Honeybadger: %v\n", err)
				} else {
					defer resp.Body.Close() // nolint:errcheck
					if resp.StatusCode != http.StatusOK {
						body, err := io.ReadAll(resp.Body)
						if err != nil {
							fmt.Fprintf(
								os.Stderr,
								"Unexpected status code: %d; failed to read response body: %v\n",
								resp.StatusCode,
								err,
							)
						} else {
							fmt.Fprintf(
								os.Stderr,
								"Unexpected status code: %d, body: %s\n",
								resp.StatusCode,
								body,
							)
						}
					} else {
						fmt.Fprintf(
							os.Stderr,
							"Check-in reported to Honeybadger (duration: %dms, status: %s)\n",
							durationMs,
							payload.CheckIn.Status,
						)
					}
				}
			}
		}

		// Exit with the same code as the wrapped command
		if runExitCode != 0 {
			exitFunc(runExitCode)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&checkInID, "id", "i", "", "Check-in ID to report")
	runCmd.Flags().StringVarP(&slug, "slug", "s", "", "Check-in slug to report")
}
