package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honeybadger-io/cli/internal/credentials"
)

// TestMain points the credentials store at a temp file so tests never read or
// write a developer's real ~/.honeybadger-cli-credentials.json.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "hb-cli-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv(credentials.EnvVar, filepath.Join(dir, "credentials.json")); err != nil {
		panic(err)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
