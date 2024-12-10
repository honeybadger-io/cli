package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConfigurationLoading(t *testing.T) {
	// Save original environment variables
	originalAPIKey := os.Getenv("HONEYBADGER_API_KEY")
	originalEndpoint := os.Getenv("HONEYBADGER_ENDPOINT")
	originalConfigFile := cfgFile

	// Create a temporary directory for test config files
	tmpDir, err := os.MkdirTemp("", "honeybadger-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test config file
	configContent := `
api_key: config-api-key
endpoint: https://config.honeybadger.io
`
	configPath := filepath.Join(tmpDir, "honeybadger.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	tests := []struct {
		name           string
		envAPIKey     string
		envEndpoint   string
		useConfigFile bool
		wantAPIKey    string
		wantEndpoint  string
	}{
		{
			name:           "environment variables take precedence over config file",
			envAPIKey:      "env-api-key",
			envEndpoint:    "https://env.honeybadger.io",
			useConfigFile:  true,
			wantAPIKey:     "env-api-key",
			wantEndpoint:   "https://env.honeybadger.io",
		},
		{
			name:           "config file values used when no environment variables",
			envAPIKey:      "",
			envEndpoint:    "",
			useConfigFile:  true,
			wantAPIKey:     "config-api-key",
			wantEndpoint:   "https://config.honeybadger.io",
		},
		{
			name:           "default values used when no config",
			envAPIKey:      "",
			envEndpoint:    "",
			useConfigFile:  false,
			wantAPIKey:     "",
			wantEndpoint:   "https://api.honeybadger.io",
		},
		{
			name:           "partial environment override",
			envAPIKey:      "env-api-key",
			envEndpoint:    "",
			useConfigFile:  true,
			wantAPIKey:     "env-api-key",
			wantEndpoint:   "https://config.honeybadger.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			// Restore original environment after test
			defer func() {
				os.Setenv("HONEYBADGER_API_KEY", originalAPIKey)
				os.Setenv("HONEYBADGER_ENDPOINT", originalEndpoint)
				cfgFile = originalConfigFile
			}()

			// Set up environment variables
			if err := os.Setenv("HONEYBADGER_API_KEY", tt.envAPIKey); err != nil {
				t.Fatalf("Failed to set API key env var: %v", err)
			}
			if err := os.Setenv("HONEYBADGER_ENDPOINT", tt.envEndpoint); err != nil {
				t.Fatalf("Failed to set endpoint env var: %v", err)
			}

			// Set up config file
			if tt.useConfigFile {
				cfgFile = configPath
			} else {
				cfgFile = ""
			}

			// Initialize configuration
			initConfig()

			// Test API key
			assert.Equal(t, tt.wantAPIKey, viper.GetString("api_key"), "API key mismatch")

			// Test endpoint
			assert.Equal(t, tt.wantEndpoint, viper.GetString("endpoint"), "Endpoint mismatch")
		})
	}
}
