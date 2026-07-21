// Package credentials stores OAuth tokens obtained by `hb auth login` in a
// JSON file, keyed by canonical issuer URL so logins to multiple Honeybadger
// instances (e.g. US and EU) can coexist.
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

// Save writes the credentials file with owner-only permissions. The write is
// atomic (temp file + rename) so a crash can't truncate the file, and the
// secrets are never on disk with permissions wider than 0600.
func Save(path string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("failed to create credentials directory: %w", err)
		}
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name()) // #nosec G703 -- temp file we created beside the credentials file
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("failed to set credentials file permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	// #nosec G703 -- both paths derive from the user's own credentials file location
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	return nil
}
