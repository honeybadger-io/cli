// Package credentials stores OAuth tokens obtained by `hb auth login` in a
// JSON file, keyed by authorization server host so logins to multiple
// Honeybadger instances (e.g. US and EU) can coexist.
package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EnvVar overrides the credentials file location when set.
const EnvVar = "HONEYBADGER_CREDENTIALS_FILE"

const defaultFileName = ".honeybadger-cli-credentials.json"

// Entry holds the OAuth client and tokens for one authorization server.
type Entry struct {
	ClientID           string    `json:"client_id,omitempty"`
	RedirectURI        string    `json:"redirect_uri,omitempty"`
	GrantType          string    `json:"grant_type,omitempty"`
	TokenEndpoint      string    `json:"token_endpoint,omitempty"`
	RevocationEndpoint string    `json:"revocation_endpoint,omitempty"`
	AccessToken        string    `json:"access_token,omitempty"`  // #nosec G117 -- stored with 0600 perms; storing tokens is this package's purpose
	RefreshToken       string    `json:"refresh_token,omitempty"` // #nosec G117
	TokenType          string    `json:"token_type,omitempty"`
	Scope              string    `json:"scope,omitempty"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// Expired reports whether the access token is expired (or expires within the
// skew). A zero ExpiresAt means the server reported no expiry.
func (e *Entry) Expired(now time.Time, skew time.Duration) bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return !now.Add(skew).Before(e.ExpiresAt)
}

// File is the on-disk credentials document.
type File struct {
	Version     int               `json:"version"`
	Credentials map[string]*Entry `json:"credentials"`
}

// Path returns the credentials file location: $HONEYBADGER_CREDENTIALS_FILE
// when set, otherwise ~/.honeybadger-cli-credentials.json.
func Path() (string, error) {
	if p := os.Getenv(EnvVar); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate home directory for credentials file: %w", err)
	}
	return filepath.Join(home, defaultFileName), nil
}

// Load reads the credentials file. A missing file yields an empty File.
func Load(path string) (*File, error) {
	f := &File{Version: 1, Credentials: map[string]*Entry{}}

	data, err := os.ReadFile(path) // #nosec G304 - path is the user's own credentials file
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file %s: %w", path, err)
	}
	if f.Credentials == nil {
		f.Credentials = map[string]*Entry{}
	}
	return f, nil
}

// Save writes the credentials file with owner-only permissions.
func Save(path string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}
	data = append(data, '\n')

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("failed to create credentials directory: %w", err)
		}
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	// Tighten permissions in case the file pre-existed with a wider mode.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("failed to set credentials file permissions: %w", err)
	}
	return nil
}
