package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCommand(t *testing.T) {
	// Save original values
	originalClient := http.DefaultClient
	originalEnvAPIKey := os.Getenv("HONEYBADGER_API_KEY")
	defer func() {
		// Restore original values after test
		http.DefaultClient = originalClient
		if err := os.Setenv("HONEYBADGER_API_KEY", originalEnvAPIKey); err != nil {
			t.Errorf("error restoring environment variable: %v", err)
		}
	}()

	// Unset environment variable for tests
	if err := os.Unsetenv("HONEYBADGER_API_KEY"); err != nil {
		t.Errorf("error unsetting environment variable: %v", err)
	}

	t.Run("requires API key", func(t *testing.T) {
		// Reset viper config
		viper.Reset()

		cmd := agentCmd
		err := cmd.RunE(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key not configured")
	})
}

func TestMetricsCollection(t *testing.T) {
	// Save original values
	originalEndpoint := endpoint
	originalEnvAPIKey := os.Getenv("HONEYBADGER_API_KEY")
	defer func() {
		// Restore original values after test
		endpoint = originalEndpoint
		if err := os.Setenv("HONEYBADGER_API_KEY", originalEnvAPIKey); err != nil {
			t.Errorf("error restoring environment variable: %v", err)
		}
	}()

	// Unset environment variable for tests
	if err := os.Unsetenv("HONEYBADGER_API_KEY"); err != nil {
		t.Errorf("error unsetting environment variable: %v", err)
	}

	// Set up test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use test server endpoint
	endpoint = server.URL

	t.Run("handles invalid endpoint", func(t *testing.T) {
		// Configure viper
		viper.Reset()
		viper.Set("api_key", "test-api-key")

		// Set invalid endpoint
		endpoint = "http://invalid-endpoint"

		hostname, err := os.Hostname()
		require.NoError(t, err)

		err = reportMetrics(hostname)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error sending metrics")
	})

	t.Run("reports metrics successfully", func(t *testing.T) {
		// Reset viper config
		viper.Reset()

		// Set up test server
		var receivedEvents []map[string]interface{}
		var receivedAPIKey string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request headers
			receivedAPIKey = r.Header.Get("X-API-Key")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Parse and validate payload
			var event map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&event)
			require.NoError(t, err)

			// Store the event
			receivedEvents = append(receivedEvents, event)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Configure viper
		viper.Set("api_key", "test-api-key")
		endpoint = server.URL

		// Set up a short interval for testing
		interval = 1 // 1 second

		// Run the agent command
		err := reportMetrics("test-host")
		require.NoError(t, err)

		// Wait for metrics to be reported
		time.Sleep(2 * time.Second)

		// Validate API key
		assert.Equal(t, "test-api-key", receivedAPIKey)

		// Ensure we received at least 3 events (CPU, memory, and at least one disk)
		assert.GreaterOrEqual(t, len(receivedEvents), 3)

		// Helper function to find event by type
		findEvent := func(eventType string) map[string]interface{} {
			for _, event := range receivedEvents {
				if event["event_type"].(string) == eventType {
					return event
				}
			}
			return nil
		}

		// Validate CPU metrics
		cpuEvent := findEvent("report.system.cpu")
		assert.NotNil(t, cpuEvent)
		assert.Equal(t, "test-host", cpuEvent["host"])
		assert.NotEmpty(t, cpuEvent["ts"])
		assert.NotZero(t, cpuEvent["num_cpus"])
		assert.GreaterOrEqual(t, cpuEvent["used_percent"].(float64), float64(0))
		assert.LessOrEqual(t, cpuEvent["used_percent"].(float64), float64(100))

		// Validate memory metrics
		memoryEvent := findEvent("report.system.memory")
		assert.NotNil(t, memoryEvent)
		assert.Equal(t, "test-host", memoryEvent["host"])
		assert.NotEmpty(t, memoryEvent["ts"])
		assert.NotZero(t, memoryEvent["total_bytes"])
		assert.NotZero(t, memoryEvent["used_bytes"])
		assert.GreaterOrEqual(t, memoryEvent["used_percent"].(float64), float64(0))
		assert.LessOrEqual(t, memoryEvent["used_percent"].(float64), float64(100))

		// Validate disk metrics
		var foundDiskEvent bool
		for _, event := range receivedEvents {
			if event["event_type"].(string) == "report.system.disk" {
				foundDiskEvent = true
				assert.Equal(t, "test-host", event["host"])
				assert.NotEmpty(t, event["ts"])
				assert.NotEmpty(t, event["mountpoint"])
				assert.NotEmpty(t, event["device"])
				assert.NotEmpty(t, event["fstype"])
				assert.NotZero(t, event["total_bytes"])
				assert.GreaterOrEqual(t, event["used_percent"].(float64), float64(0))
				assert.LessOrEqual(t, event["used_percent"].(float64), float64(100))
			}
		}
		assert.True(t, foundDiskEvent, "No disk metrics were received")
	})
}
