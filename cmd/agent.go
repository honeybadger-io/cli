// Package cmd provides command-line interface commands for the Honeybadger CLI.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var interval int

type cpuPayload struct {
	Ts          string  `json:"ts"`
	Event       string  `json:"event_type"`
	Host        string  `json:"host"`
	UsedPercent float64 `json:"used_percent"`
	LoadAvg1    float64 `json:"load_avg_1"`
	LoadAvg5    float64 `json:"load_avg_5"`
	LoadAvg15   float64 `json:"load_avg_15"`
	NumCPUs     int     `json:"num_cpus"`
}

type memoryPayload struct {
	Ts          string  `json:"ts"`
	Event       string  `json:"event_type"`
	Host        string  `json:"host"`
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Free        uint64  `json:"free_bytes"`
	Available   uint64  `json:"available_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type diskPayload struct {
	Ts          string  `json:"ts"`
	Event       string  `json:"event_type"`
	Host        string  `json:"host"`
	Mountpoint  string  `json:"mountpoint"`
	Device      string  `json:"device"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Free        uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:     "agent",
	Short:   "Start a metrics reporting agent",
	GroupID: GroupReportingAPI,
	Long: `Start a persistent process that periodically reports host metrics to Honeybadger's Insights API.
This command collects and reports system metrics such as CPU usage, memory usage, disk usage, and load averages.
Metrics are aggregated and reported at a configurable interval (default: 60 seconds).`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Check for API key before starting
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			return fmt.Errorf(
				"API key not configured. Use --api-key flag or set HONEYBADGER_API_KEY environment variable",
			)
		}

		ctx := context.Background()
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		fmt.Printf("Starting metrics agent, reporting every %d seconds...\n", interval)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := reportMetrics(hostname); err != nil {
					fmt.Fprintf(os.Stderr, "Error reporting metrics: %v\n", err)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.Flags().IntVarP(&interval, "interval", "i", 60, "Reporting interval in seconds")
}

// sendMetric sends a single metric event to Honeybadger
func sendMetric(payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling metrics: %w", err)
	}

	apiEndpoint := viper.GetString("endpoint")
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/events", apiEndpoint),
		strings.NewReader(string(jsonData)+"\n"),
	)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", viper.GetString("api_key"))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending metrics: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing response body: %v\n", cerr)
		}
	}()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received error response: %s\n%s", resp.Status, body)
	}

	return nil
}

func reportMetrics(hostname string) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Collect and send CPU metrics
	cpuPercent, err := cpu.Percent(time.Second, false)
	var usedPercent float64
	if err != nil {
		// cpu.Percent may fail on macOS with CGO_ENABLED=0; use -1 to indicate unavailable
		usedPercent = -1
	} else if len(cpuPercent) > 0 {
		usedPercent = math.Round(cpuPercent[0]*100) / 100
	}

	loadAvg, err := load.Avg()
	if err != nil {
		return fmt.Errorf("error getting load average: %w", err)
	}

	numCPU, err := cpu.Counts(true)
	if err != nil {
		numCPU = 0 // fallback if we can't get the count
	}

	cpuPayload := cpuPayload{
		Ts:          timestamp,
		Event:       "report.system.cpu",
		Host:        hostname,
		UsedPercent: usedPercent,
		LoadAvg1:    loadAvg.Load1,
		LoadAvg5:    loadAvg.Load5,
		LoadAvg15:   loadAvg.Load15,
		NumCPUs:     numCPU,
	}
	if err := sendMetric(cpuPayload); err != nil {
		return fmt.Errorf("error sending CPU metrics: %w", err)
	}

	// Collect and send memory metrics
	virtualMem, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error getting memory metrics: %w", err)
	}

	memoryPayload := memoryPayload{
		Ts:          timestamp,
		Event:       "report.system.memory",
		Host:        hostname,
		Total:       virtualMem.Total,
		Used:        virtualMem.Used,
		Free:        virtualMem.Free,
		Available:   virtualMem.Available,
		UsedPercent: math.Round(virtualMem.UsedPercent*100) / 100,
	}
	if err := sendMetric(memoryPayload); err != nil {
		return fmt.Errorf("error sending memory metrics: %w", err)
	}

	// Collect and send disk metrics
	parts, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("error getting disk partitions: %w", err)
	}

	// Send metrics for each disk partition
	for _, part := range parts {
		// Skip pseudo filesystems
		if part.Fstype == "devfs" || part.Fstype == "autofs" || part.Fstype == "nullfs" ||
			part.Fstype == "squashfs" ||
			strings.HasPrefix(part.Fstype, "fuse.") ||
			strings.Contains(part.Mountpoint, "/System/Volumes") {
			continue
		}

		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			// Log error but continue with other partitions
			fmt.Fprintf(os.Stderr, "Error getting disk usage for %s: %v\n", part.Mountpoint, err)
			continue
		}

		diskPayload := diskPayload{
			Ts:          timestamp,
			Event:       "report.system.disk",
			Host:        hostname,
			Mountpoint:  part.Mountpoint,
			Device:      part.Device,
			Fstype:      part.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: math.Round(usage.UsedPercent*100) / 100,
		}
		if err := sendMetric(diskPayload); err != nil {
			return fmt.Errorf("error sending disk metrics for %s: %w", part.Mountpoint, err)
		}
	}

	return nil
}
