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

	t.Run("reports metrics successfully", func(t *testing.T) {
		// Reset viper config
		viper.Reset()

		// Set up test server
		var receivedPayload metricsPayload
		var receivedAPIKey string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request headers
			receivedAPIKey = r.Header.Get("X-API-Key")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Parse and validate payload
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&receivedPayload)
			require.NoError(t, err)

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

		// Validate received payload
		assert.Equal(t, "test-api-key", receivedAPIKey)
		assert.Equal(t, "system.metrics", receivedPayload.Event)
		assert.NotEmpty(t, receivedPayload.Ts)
		assert.NotEmpty(t, receivedPayload.Host)

		// Validate CPU metrics
		assert.NotZero(t, receivedPayload.CPU.NumCPUs)
		assert.GreaterOrEqual(t, receivedPayload.CPU.UsedPercent, float64(0))
		assert.LessOrEqual(t, receivedPayload.CPU.UsedPercent, float64(100))

		// Validate memory metrics
		assert.NotZero(t, receivedPayload.Memory.Total)
		assert.NotZero(t, receivedPayload.Memory.Used)
		assert.GreaterOrEqual(t, receivedPayload.Memory.UsedPercent, float64(0))
		assert.LessOrEqual(t, receivedPayload.Memory.UsedPercent, float64(100))

		// Validate disk metrics
		assert.NotEmpty(t, receivedPayload.Disks)
		for _, disk := range receivedPayload.Disks {
			assert.NotEmpty(t, disk.Mountpoint)
			assert.NotEmpty(t, disk.Device)
			assert.NotEmpty(t, disk.Fstype)
			assert.NotZero(t, disk.Total)
			assert.GreaterOrEqual(t, disk.UsedPercent, float64(0))
			assert.LessOrEqual(t, disk.UsedPercent, float64(100))
		}
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

	t.Run("collects metrics without error", func(t *testing.T) {
		// Configure viper
		viper.Reset()
		viper.Set("api_key", "test-api-key")

		hostname, err := os.Hostname()
		require.NoError(t, err)

		err = reportMetrics(hostname)
		require.NoError(t, err)
	})

	t.Run("handles missing API key", func(t *testing.T) {
		// Reset viper config and ensure API key is not set
		viper.Reset()
		viper.Set("api_key", "")

		hostname, err := os.Hostname()
		require.NoError(t, err)

		err = reportMetrics(hostname)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key not configured")
	})

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
}
