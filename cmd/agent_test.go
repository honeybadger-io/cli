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

func TestParseTags(t *testing.T) {
	t.Run("parses valid key=value tags", func(t *testing.T) {
		input := []string{"environment=stage", "role=web-1"}
		result, err := parseTags(input)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"environment": "stage",
			"role":        "web-1",
		}, result)
	})

	t.Run("handles values containing equals signs", func(t *testing.T) {
		input := []string{"label=a=b=c"}
		result, err := parseTags(input)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"label": "a=b=c"}, result)
	})

	t.Run("rejects tag without equals sign", func(t *testing.T) {
		input := []string{"badtag"}
		_, err := parseTags(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tag")
	})

	t.Run("rejects tag with empty key", func(t *testing.T) {
		input := []string{"=value"}
		_, err := parseTags(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tag")
	})

	t.Run("returns empty map for no tags", func(t *testing.T) {
		result, err := parseTags(nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("rejects reserved metric field keys", func(t *testing.T) {
		for _, key := range []string{"ts", "event_type", "used_percent", "total_bytes", "mountpoint"} {
			_, err := parseTags([]string{key + "=foo"})
			require.Error(t, err, "expected error for reserved key %q", key)
			assert.Contains(t, err.Error(), "reserved metric field")
		}
	})

	t.Run("allows host as a tag key", func(t *testing.T) {
		result, err := parseTags([]string{"host=custom"})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"host": "custom"}, result)
	})
}

func TestMergeTags(t *testing.T) {
	t.Run("CLI flags override config tags", func(t *testing.T) {
		configTags := map[string]string{"environment": "config-env", "region": "us-east-1"}
		flagTags := map[string]string{"environment": "flag-env"}
		result := mergeTags(configTags, flagTags)
		assert.Equal(t, map[string]string{
			"environment": "flag-env",
			"region":      "us-east-1",
		}, result)
	})

	t.Run("returns flag tags when no config tags", func(t *testing.T) {
		flagTags := map[string]string{"role": "web-1"}
		result := mergeTags(nil, flagTags)
		assert.Equal(t, map[string]string{"role": "web-1"}, result)
	})

	t.Run("returns config tags when no flag tags", func(t *testing.T) {
		configTags := map[string]string{"role": "web-1"}
		result := mergeTags(configTags, nil)
		assert.Equal(t, map[string]string{"role": "web-1"}, result)
	})

	t.Run("returns empty map when both nil", func(t *testing.T) {
		result := mergeTags(nil, nil)
		assert.Empty(t, result)
	})
}

func TestAgentTagsFromConfig(t *testing.T) {
	t.Run("loads tags from viper config", func(t *testing.T) {
		viper.Reset()
		viper.Set("api_key", "test-key")
		viper.Set("agent.tags", map[string]interface{}{
			"environment": "production",
			"role":        "web-1",
		})

		result, err := loadConfigTags()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"environment": "production",
			"role":        "web-1",
		}, result)
	})

	t.Run("returns empty map when no config tags", func(t *testing.T) {
		viper.Reset()
		result, err := loadConfigTags()
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("rejects reserved keys in config tags", func(t *testing.T) {
		viper.Reset()
		viper.Set("agent.tags", map[string]interface{}{
			"event_type": "bad",
		})
		_, err := loadConfigTags()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reserved metric field")
	})
}

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		viper.Set("endpoint", endpoint)

		hostname, err := os.Hostname()
		require.NoError(t, err)

		err = reportMetrics(hostname, nil)
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
		viper.Set("endpoint", endpoint)

		// Set up a short interval for testing
		interval = 1 // 1 second

		// Run the agent command
		err := reportMetrics("test-host", nil)
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

func TestSendMetricWithTags(t *testing.T) {
	t.Run("tags are merged into event JSON", func(t *testing.T) {
		var receivedEvent map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&receivedEvent)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		viper.Reset()
		viper.Set("api_key", "test-key")
		viper.Set("endpoint", server.URL)

		payload := cpuPayload{
			Ts:    "2026-01-01T00:00:00Z",
			Event: "report.system.cpu",
			Host:  "auto-hostname",
		}
		tags := map[string]string{"environment": "stage", "role": "web-1"}

		err := sendMetric(payload, tags)
		require.NoError(t, err)

		assert.Equal(t, "stage", receivedEvent["environment"])
		assert.Equal(t, "web-1", receivedEvent["role"])
		assert.Equal(t, "auto-hostname", receivedEvent["host"])
	})

	t.Run("host tag overrides struct host field", func(t *testing.T) {
		var receivedEvent map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&receivedEvent)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		viper.Reset()
		viper.Set("api_key", "test-key")
		viper.Set("endpoint", server.URL)

		payload := cpuPayload{
			Ts:    "2026-01-01T00:00:00Z",
			Event: "report.system.cpu",
			Host:  "auto-hostname",
		}
		tags := map[string]string{"host": "custom-host"}

		err := sendMetric(payload, tags)
		require.NoError(t, err)

		assert.Equal(t, "custom-host", receivedEvent["host"])
	})

	t.Run("works with no tags", func(t *testing.T) {
		var receivedEvent map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&receivedEvent)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		viper.Reset()
		viper.Set("api_key", "test-key")
		viper.Set("endpoint", server.URL)

		payload := cpuPayload{
			Ts:    "2026-01-01T00:00:00Z",
			Event: "report.system.cpu",
			Host:  "auto-hostname",
		}

		err := sendMetric(payload, nil)
		require.NoError(t, err)

		assert.Equal(t, "auto-hostname", receivedEvent["host"])
		assert.Equal(t, "report.system.cpu", receivedEvent["event_type"])
	})
}

func TestReportMetricsWithTags(t *testing.T) {
	t.Run("tags appear in all submitted events", func(t *testing.T) {
		var receivedEvents []map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&event); err == nil {
				receivedEvents = append(receivedEvents, event)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		viper.Reset()
		viper.Set("api_key", "test-key")
		viper.Set("endpoint", server.URL)

		tags := map[string]string{"environment": "stage", "role": "web-1"}
		err := reportMetrics("auto-host", tags)
		require.NoError(t, err)

		// Should have at least CPU + memory + 1 disk = 3 events
		require.GreaterOrEqual(t, len(receivedEvents), 3)

		for _, event := range receivedEvents {
			eventType := event["event_type"]
			assert.Equal(t, "stage", event["environment"],
				"event_type=%s missing environment tag", eventType)
			assert.Equal(t, "web-1", event["role"],
				"event_type=%s missing role tag", eventType)
			assert.Equal(t, "auto-host", event["host"],
				"event_type=%s has wrong host", eventType)
		}
	})

	t.Run("host tag overrides hostname in all events", func(t *testing.T) {
		var receivedEvents []map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&event); err == nil {
				receivedEvents = append(receivedEvents, event)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		viper.Reset()
		viper.Set("api_key", "test-key")
		viper.Set("endpoint", server.URL)

		tags := map[string]string{"host": "custom-host", "environment": "prod"}
		err := reportMetrics("auto-host", tags)
		require.NoError(t, err)

		require.GreaterOrEqual(t, len(receivedEvents), 3)

		for _, event := range receivedEvents {
			eventType := event["event_type"]
			assert.Equal(t, "custom-host", event["host"],
				"event_type=%s host not overridden", eventType)
			assert.Equal(t, "prod", event["environment"],
				"event_type=%s missing environment tag", eventType)
		}
	})
}
