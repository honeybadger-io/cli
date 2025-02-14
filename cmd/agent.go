package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

var (
	interval int
)

type diskMetrics struct {
	Mountpoint  string  `json:"mountpoint"`
	Device      string  `json:"device"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Free        uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type memoryMetrics struct {
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Free        uint64  `json:"free_bytes"`
	Available   uint64  `json:"available_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type cpuMetrics struct {
	UsedPercent float64 `json:"used_percent"`
	LoadAvg1    float64 `json:"load_avg_1"`
	LoadAvg5    float64 `json:"load_avg_5"`
	LoadAvg15   float64 `json:"load_avg_15"`
	NumCPUs     int     `json:"num_cpus"`
}

type metricsPayload struct {
	Ts     string        `json:"ts"`
	Event  string        `json:"event"`
	Host   string        `json:"host"`
	CPU    cpuMetrics    `json:"cpu"`
	Memory memoryMetrics `json:"memory"`
	Disks  []diskMetrics `json:"disks"`
}

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Start a metrics reporting agent",
	Long: `Start a persistent process that periodically reports host metrics to Honeybadger's Insights API.
This command collects and reports system metrics such as CPU usage, memory usage, disk usage, and load averages.
Metrics are aggregated and reported at a configurable interval (default: 60 seconds).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check for API key before starting
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			return fmt.Errorf("API key not configured. Use --api-key flag or set HONEYBADGER_API_KEY environment variable")
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

func reportMetrics(hostname string) error {
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("error getting CPU metrics: %w", err)
	}

	virtualMem, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error getting memory metrics: %w", err)
	}

	parts, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("error getting disk partitions: %w", err)
	}

	// Get usage for all partitions
	var disks []diskMetrics
	for _, part := range parts {
		// Skip pseudo filesystems
		if part.Fstype == "devfs" || part.Fstype == "autofs" || part.Fstype == "nullfs" ||
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

		disks = append(disks, diskMetrics{
			Mountpoint:  part.Mountpoint,
			Device:      part.Device,
			Fstype:      part.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	loadAvg, err := load.Avg()
	if err != nil {
		return fmt.Errorf("error getting load average: %w", err)
	}

	// Get number of CPUs
	numCPU, err := cpu.Counts(true)
	if err != nil {
		numCPU = 0 // fallback if we can't get the count
	}

	payload := metricsPayload{
		Ts:    time.Now().UTC().Format(time.RFC3339),
		Event: "system.metrics",
		Host:  hostname,
		CPU: cpuMetrics{
			UsedPercent: cpuPercent[0],
			LoadAvg1:    loadAvg.Load1,
			LoadAvg5:    loadAvg.Load5,
			LoadAvg15:   loadAvg.Load15,
			NumCPUs:     numCPU,
		},
		Memory: memoryMetrics{
			Total:       virtualMem.Total,
			Used:        virtualMem.Used,
			Free:        virtualMem.Free,
			Available:   virtualMem.Available,
			UsedPercent: virtualMem.UsedPercent,
		},
		Disks: disks,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling metrics: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/events", endpoint), strings.NewReader(string(jsonData)+"\n"))
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
