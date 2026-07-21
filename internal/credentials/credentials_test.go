package credentials

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFile(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	require.NoError(t, err)
	assert.Empty(t, f.Credentials)
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "creds", "credentials.json")
	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	f := &File{Version: 1, Credentials: map[string]*Entry{
		"app.honeybadger.io": {
			ClientID:     "client-us",
			AccessToken:  "access-us",
			RefreshToken: "refresh-us",
			Scope:        "read write",
			ExpiresAt:    expires,
		},
		"eu-app.honeybadger.io": {
			ClientID:    "client-eu",
			AccessToken: "access-eu",
		},
	}}
	require.NoError(t, Save(path, f))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "credentials file must be owner-only")

	loaded, err := Load(path)
	require.NoError(t, err)
	require.Len(t, loaded.Credentials, 2)
	us := loaded.Credentials["app.honeybadger.io"]
	require.NotNil(t, us)
	assert.Equal(t, "access-us", us.AccessToken)
	assert.True(t, us.ExpiresAt.Equal(expires))
	assert.Equal(t, "access-eu", loaded.Credentials["eu-app.honeybadger.io"].AccessToken)
}

func TestSaveTightensPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	// #nosec G306 -- deliberately wide permissions; Save must tighten them
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o644))

	require.NoError(t, Save(path, &File{Version: 1}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestLoadCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse credentials file")
}

func TestPathEnvOverride(t *testing.T) {
	t.Setenv(EnvVar, "/tmp/custom-creds.json")
	p, err := Path()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/custom-creds.json", p)
}

func TestExpired(t *testing.T) {
	now := time.Now()
	skew := time.Minute

	assert.False(t, (&Entry{}).Expired(now, skew), "zero expiry never counts as expired")
	assert.False(t, (&Entry{ExpiresAt: now.Add(time.Hour)}).Expired(now, skew))
	assert.True(t, (&Entry{ExpiresAt: now.Add(30 * time.Second)}).Expired(now, skew),
		"tokens inside the skew window count as expired")
	assert.True(t, (&Entry{ExpiresAt: now.Add(-time.Hour)}).Expired(now, skew))
}
