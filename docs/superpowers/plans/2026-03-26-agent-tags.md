# Agent Tags Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--tag key=value` support to the `hb agent` command so users can annotate metrics with custom metadata (environment, role, host override, etc.)

**Architecture:** Parse tags from CLI flags and YAML config, merge them into each metric event's JSON at the serialization boundary in `sendMetric`. Tags override struct fields, so `--tag host=foo` naturally replaces the auto-detected hostname. Typed structs stay unchanged for metric construction.

**Tech Stack:** Go, Cobra (CLI), Viper (config), existing test patterns with httptest + testify.

**Note:** There is a Go toolchain version mismatch on this machine (go.mod says 1.23, installed is 1.26). Tests may not compile locally. Write correct code and verify manually if needed.

---

### Task 1: Parse and validate `--tag` flags

**Files:**
- Modify: `cmd/agent.go:23` (add tags variable)
- Modify: `cmd/agent.go:101-104` (register flag in init)
- Modify: `cmd/agent.go:68-98` (parse tags in RunE)
- Test: `cmd/agent_test.go`

- [ ] **Step 1: Write the failing test for tag parsing**

Add to `cmd/agent_test.go`:

```go
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
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestParseTags -v`
Expected: FAIL — `parseTags` not defined.

- [ ] **Step 3: Implement parseTags and register the flag**

In `cmd/agent.go`, add the `tagFlags` variable next to the existing `interval` var:

```go
var interval int
var tagFlags []string
```

Add the `parseTags` function after the existing `init()`:

```go
// parseTags converts a slice of "key=value" strings into a map.
// Returns an error if any tag is malformed.
func parseTags(raw []string) (map[string]string, error) {
	tags := make(map[string]string)
	for _, tag := range raw {
		key, value, ok := strings.Cut(tag, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid tag %q: must be in key=value format", tag)
		}
		tags[key] = value
	}
	return tags, nil
}
```

In `init()`, register the flag:

```go
func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.Flags().IntVarP(&interval, "interval", "i", 60, "Reporting interval in seconds")
	agentCmd.Flags().StringArrayVarP(&tagFlags, "tag", "t", nil, "Tag in key=value format (repeatable, e.g. --tag environment=stage)")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestParseTags -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/agent.go cmd/agent_test.go
git commit -m "feat(agent): add --tag flag parsing and validation"
```

---

### Task 2: Load tags from YAML config and merge with CLI flags

**Files:**
- Modify: `cmd/agent.go:68-98` (RunE function — merge config + flag tags)
- Test: `cmd/agent_test.go`

- [ ] **Step 1: Write the failing test for config tag merging**

Add to `cmd/agent_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestMergeTags -v`
Expected: FAIL — `mergeTags` not defined.

- [ ] **Step 3: Implement mergeTags**

Add to `cmd/agent.go`:

```go
// mergeTags combines config tags with CLI flag tags. Flag tags take precedence.
func mergeTags(configTags, flagTags map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range configTags {
		merged[k] = v
	}
	for k, v := range flagTags {
		merged[k] = v
	}
	return merged
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestMergeTags -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/agent.go cmd/agent_test.go
git commit -m "feat(agent): add mergeTags for config + CLI flag precedence"
```

---

### Task 3: Inject tags into sendMetric serialization

**Files:**
- Modify: `cmd/agent.go:106-143` (sendMetric function — accept and merge tags)
- Modify: `cmd/agent.go:145-243` (reportMetrics function — pass tags through)
- Test: `cmd/agent_test.go`

- [ ] **Step 1: Write the failing test for tag injection into metric payloads**

Add to `cmd/agent_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestSendMetricWithTags -v`
Expected: FAIL — `sendMetric` signature mismatch (currently takes only `payload interface{}`).

- [ ] **Step 3: Update sendMetric to accept and merge tags**

Replace the `sendMetric` function in `cmd/agent.go`:

```go
// sendMetric sends a single metric event to Honeybadger.
// Tags are merged into the JSON payload, overriding any existing fields.
func sendMetric(payload interface{}, tags map[string]string) error {
	// Marshal struct to JSON, then unmarshal to map so we can merge tags
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling metrics: %w", err)
	}

	var merged map[string]interface{}
	if err := json.Unmarshal(jsonData, &merged); err != nil {
		return fmt.Errorf("error unmarshaling metrics for tag merge: %w", err)
	}

	for k, v := range tags {
		merged[k] = v
	}

	finalJSON, err := json.Marshal(merged)
	if err != nil {
		return fmt.Errorf("error marshaling final payload: %w", err)
	}

	apiEndpoint := viper.GetString("endpoint")
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/events", apiEndpoint),
		strings.NewReader(string(finalJSON)+"\n"),
	)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", viper.GetString("api_key"))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req) //nolint:gosec // endpoint is intentionally user-configurable
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
```

- [ ] **Step 4: Update reportMetrics to accept and pass tags**

Change the `reportMetrics` signature and all `sendMetric` calls in `cmd/agent.go`:

```go
func reportMetrics(hostname string, tags map[string]string) error {
```

Update each `sendMetric` call to pass `tags`:

```go
	if err := sendMetric(cpuPayload, tags); err != nil {
```

```go
	if err := sendMetric(memoryPayload, tags); err != nil {
```

```go
		if err := sendMetric(diskPayload, tags); err != nil {
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./cmd/ -run TestSendMetricWithTags -v`
Expected: FAIL — compilation error because `reportMetrics` callers and existing tests still use old signatures. That's fine, we fix those in the next step.

- [ ] **Step 6: Update existing tests and the RunE caller**

In the `RunE` function in `cmd/agent.go`, update the `reportMetrics` call (we'll wire up real tag loading in Task 4, for now pass `nil`):

```go
			if err := reportMetrics(hostname, nil); err != nil {
```

In `cmd/agent_test.go`, update the existing `TestMetricsCollection` calls to `reportMetrics`:

In `TestMetricsCollection` "handles invalid endpoint" subtest, change:
```go
		err = reportMetrics(hostname)
```
to:
```go
		err = reportMetrics(hostname, nil)
```

In `TestMetricsCollection` "reports metrics successfully" subtest, change:
```go
		err := reportMetrics("test-host")
```
to:
```go
		err := reportMetrics("test-host", nil)
```

- [ ] **Step 7: Run all tests to verify everything passes**

Run: `go test ./cmd/ -run "TestSendMetricWithTags|TestMetricsCollection|TestAgentCommand" -v`
Expected: All PASS

- [ ] **Step 8: Commit**

```bash
git add cmd/agent.go cmd/agent_test.go
git commit -m "feat(agent): merge tags into metric event JSON payloads"
```

---

### Task 4: Wire up tag loading in RunE (CLI flags + YAML config)

**Files:**
- Modify: `cmd/agent.go:68-98` (RunE function — load and merge tags from both sources)
- Test: `cmd/agent_test.go`

- [ ] **Step 1: Write the failing test for YAML config tag loading**

Add to `cmd/agent_test.go`:

```go
func TestAgentTagsFromConfig(t *testing.T) {
	t.Run("loads tags from viper config", func(t *testing.T) {
		viper.Reset()
		viper.Set("api_key", "test-key")
		viper.Set("agent.tags", map[string]interface{}{
			"environment": "production",
			"role":        "web-1",
		})

		result := loadConfigTags()
		assert.Equal(t, map[string]string{
			"environment": "production",
			"role":        "web-1",
		}, result)
	})

	t.Run("returns empty map when no config tags", func(t *testing.T) {
		viper.Reset()
		result := loadConfigTags()
		assert.Empty(t, result)
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestAgentTagsFromConfig -v`
Expected: FAIL — `loadConfigTags` not defined.

- [ ] **Step 3: Implement loadConfigTags**

Add to `cmd/agent.go`:

```go
// loadConfigTags reads tags from the "agent.tags" section of the config file.
func loadConfigTags() map[string]string {
	raw := viper.GetStringMapString("agent.tags")
	if len(raw) == 0 {
		return nil
	}
	return raw
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestAgentTagsFromConfig -v`
Expected: PASS

- [ ] **Step 5: Wire up tag loading in RunE**

In `cmd/agent.go`, replace the `RunE` function body to load and merge tags before entering the ticker loop:

```go
	RunE: func(_ *cobra.Command, _ []string) error {
		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			return fmt.Errorf(
				"API key not configured. Use --api-key flag or set HONEYBADGER_API_KEY environment variable",
			)
		}

		// Parse CLI flag tags
		flagTags, err := parseTags(tagFlags)
		if err != nil {
			return err
		}

		// Load config file tags and merge (CLI flags take precedence)
		configTags := loadConfigTags()
		tags := mergeTags(configTags, flagTags)

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
				if err := reportMetrics(hostname, tags); err != nil {
					fmt.Fprintf(os.Stderr, "Error reporting metrics: %v\n", err)
				}
			}
		}
	},
```

- [ ] **Step 6: Run all tests**

Run: `go test ./cmd/ -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/agent.go cmd/agent_test.go
git commit -m "feat(agent): wire up tag loading from CLI flags and YAML config"
```

---

### Task 5: End-to-end test — tags appear in submitted metrics

**Files:**
- Modify: `cmd/agent_test.go`

- [ ] **Step 1: Write end-to-end test**

Add to `cmd/agent_test.go`:

```go
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
			assert.Equal(t, "stage", event["environment"], "event_type=%s missing environment tag", event["event_type"])
			assert.Equal(t, "web-1", event["role"], "event_type=%s missing role tag", event["event_type"])
			assert.Equal(t, "auto-host", event["host"], "event_type=%s has wrong host", event["event_type"])
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
			assert.Equal(t, "custom-host", event["host"], "event_type=%s host not overridden", event["event_type"])
			assert.Equal(t, "prod", event["environment"], "event_type=%s missing environment tag", event["event_type"])
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./cmd/ -run TestReportMetricsWithTags -v`
Expected: PASS (implementation is already wired up from previous tasks)

- [ ] **Step 3: Run full test suite**

Run: `go test ./cmd/ -v`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/agent_test.go
git commit -m "test(agent): add end-to-end tests for tag injection in metrics"
```
