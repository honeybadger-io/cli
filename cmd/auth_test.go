package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/honeybadger-io/cli/internal/credentials"
	"github.com/honeybadger-io/cli/internal/oauth"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOAuthServer is a minimal authorization server plus Data API endpoint
// for exercising the auth commands end to end.
type fakeOAuthServer struct {
	t      *testing.T
	server *httptest.Server

	supportsDevice bool
	registrations  int
	revokedTokens  []string
	refreshCalls   int
	apiAuthHeaders []string
}

func newFakeOAuthServer(t *testing.T, supportsDevice bool) *fakeOAuthServer {
	f := &fakeOAuthServer{t: t, supportsDevice: supportsDevice}
	mux := http.NewServeMux()

	mux.HandleFunc(
		"/.well-known/oauth-authorization-server",
		func(w http.ResponseWriter, _ *http.Request) {
			grants := []string{"authorization_code", "refresh_token"}
			if f.supportsDevice {
				grants = append(grants, oauth.DeviceGrantType)
			}
			md := map[string]interface{}{
				"issuer":                 f.server.URL,
				"authorization_endpoint": f.server.URL + "/oauth/authorize",
				"token_endpoint":         f.server.URL + "/oauth/token",
				"registration_endpoint":  f.server.URL + "/oauth/register",
				"revocation_endpoint":    f.server.URL + "/oauth/revoke",
				"grant_types_supported":  grants,
			}
			if f.supportsDevice {
				md["device_authorization_endpoint"] = f.server.URL + "/oauth/authorize_device"
			}
			_ = json.NewEncoder(w).Encode(md)
		},
	)

	mux.HandleFunc("/oauth/register", func(w http.ResponseWriter, _ *http.Request) {
		f.registrations++
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"client_id": "registered-client"}`))
	})

	mux.HandleFunc("/oauth/authorize_device", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"device_code": "dev-1", "user_code": "AAAA-BBBB",
			"verification_uri": "https://example.com/device",
			"expires_in": 300, "interval": 1
		}`))
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		if r.Form.Get("grant_type") == "refresh_token" {
			f.refreshCalls++
		}
		_, _ = w.Write([]byte(`{
			"access_token": "oauth-access", "refresh_token": "oauth-refresh",
			"token_type": "Bearer", "scope": "read write", "expires_in": 3600
		}`))
	})

	mux.HandleFunc("/oauth/revoke", func(_ http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		f.revokedTokens = append(f.revokedTokens, r.Form.Get("token"))
	})

	mux.HandleFunc("/v2/accounts", func(w http.ResponseWriter, r *http.Request) {
		f.apiAuthHeaders = append(f.apiAuthHeaders, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"results": []}`))
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

// setupAuthTest isolates viper, flags, credentials file, and browser opener.
func setupAuthTest(t *testing.T, f *fakeOAuthServer) {
	t.Setenv(credentials.EnvVar, filepath.Join(t.TempDir(), "credentials.json"))
	viper.Reset()
	viper.Set("endpoint", f.server.URL)
	authLoginDevice = false
	authLoginWeb = false
	authLoginScopes = oauthDefaultScopes

	// Neutralize environment-based flow selection so tests behave the same
	// on developer machines, SSH sessions, and CI.
	for _, envVar := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY", "WAYLAND_DISPLAY"} {
		t.Setenv(envVar, "")
	}
	t.Setenv("DISPLAY", ":0")

	// The fake "browser" completes the authorization immediately.
	originalOpenBrowser := openBrowser
	openBrowser = func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		redirect, err := url.Parse(q.Get("redirect_uri"))
		if err != nil {
			return err
		}
		rq := redirect.Query()
		rq.Set("code", "test-code")
		rq.Set("state", q.Get("state"))
		redirect.RawQuery = rq.Encode()
		go func() {
			resp, err := http.Get(redirect.String())
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
	t.Cleanup(func() {
		openBrowser = originalOpenBrowser
		viper.Reset()
	})
}

func loadTestCredentials(t *testing.T) *credentials.File {
	path, err := credentials.Path()
	require.NoError(t, err)
	f, err := credentials.Load(path)
	require.NoError(t, err)
	return f
}

func saveTestCredentials(t *testing.T, f *fakeOAuthServer, entry *credentials.Entry) {
	host, err := oauth.CanonicalIssuer(f.server.URL)
	require.NoError(t, err)
	path, err := credentials.Path()
	require.NoError(t, err)
	require.NoError(t, credentials.Save(path, &credentials.File{
		Version:     1,
		Credentials: map[string]*credentials.Entry{host: entry},
	}))
}

func TestAuthLoginBrowserFlow(t *testing.T) {
	f := newFakeOAuthServer(t, false)
	setupAuthTest(t, f)

	var out bytes.Buffer
	authLoginCmd.SetOut(&out)
	defer authLoginCmd.SetOut(nil)

	require.NoError(t, authLoginCmd.RunE(authLoginCmd, []string{}))
	assert.Contains(t, out.String(), "Logged in")
	assert.Equal(t, 1, f.registrations, "should dynamically register a client")

	creds := loadTestCredentials(t)
	host, _ := oauth.CanonicalIssuer(f.server.URL)
	entry := creds.Credentials[host]
	require.NotNil(t, entry)
	assert.Equal(t, "registered-client", entry.ClientID)
	assert.Equal(t, "oauth-access", entry.AccessToken)
	assert.Equal(t, "oauth-refresh", entry.RefreshToken)
	assert.Equal(t, "authorization_code", entry.GrantType)
	assert.NotEmpty(t, entry.RedirectURI)
	assert.False(t, entry.ExpiresAt.IsZero())
}

func TestAuthLoginDeviceFlow(t *testing.T) {
	f := newFakeOAuthServer(t, true)
	setupAuthTest(t, f)
	authLoginDevice = true

	var out bytes.Buffer
	authLoginCmd.SetOut(&out)
	defer authLoginCmd.SetOut(nil)

	require.NoError(t, authLoginCmd.RunE(authLoginCmd, []string{}))
	assert.Contains(t, out.String(), "AAAA-BBBB", "should display the user code")
	assert.Contains(t, out.String(), "Logged in")

	creds := loadTestCredentials(t)
	host, _ := oauth.CanonicalIssuer(f.server.URL)
	entry := creds.Credentials[host]
	require.NotNil(t, entry)
	assert.Equal(t, oauth.DeviceGrantType, entry.GrantType)
	assert.Equal(t, "oauth-access", entry.AccessToken)
}

func TestAuthLoginDeviceUnsupported(t *testing.T) {
	f := newFakeOAuthServer(t, false)
	setupAuthTest(t, f)
	authLoginDevice = true

	var out bytes.Buffer
	authLoginCmd.SetOut(&out)
	defer authLoginCmd.SetOut(nil)

	err := authLoginCmd.RunE(authLoginCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support the device authorization grant")
}

func TestAuthStatus(t *testing.T) {
	t.Run("not logged in", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)

		err := authStatusCmd.RunE(authStatusCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not logged in")
	})

	t.Run("personal auth token takes precedence", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)
		viper.Set("auth_token", "personal-token")

		var out bytes.Buffer
		authStatusCmd.SetOut(&out)
		defer authStatusCmd.SetOut(nil)

		require.NoError(t, authStatusCmd.RunE(authStatusCmd, []string{}))
		assert.Contains(t, out.String(), "personal auth token")
	})

	t.Run("logged in via oauth", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)
		saveTestCredentials(t, f, &credentials.Entry{
			ClientID:    "client-1",
			AccessToken: "tok",
			Scope:       "read write",
			ExpiresAt:   time.Now().Add(time.Hour),
		})

		var out bytes.Buffer
		authStatusCmd.SetOut(&out)
		defer authStatusCmd.SetOut(nil)

		require.NoError(t, authStatusCmd.RunE(authStatusCmd, []string{}))
		assert.Contains(t, out.String(), "Logged in")
		assert.Contains(t, out.String(), "read write")
	})
}

func TestAuthLogout(t *testing.T) {
	f := newFakeOAuthServer(t, false)
	setupAuthTest(t, f)
	saveTestCredentials(t, f, &credentials.Entry{
		ClientID:           "client-1",
		AccessToken:        "access-1",
		RefreshToken:       "refresh-1",
		RevocationEndpoint: f.server.URL + "/oauth/revoke",
	})

	var out bytes.Buffer
	authLogoutCmd.SetOut(&out)
	defer authLogoutCmd.SetOut(nil)

	require.NoError(t, authLogoutCmd.RunE(authLogoutCmd, []string{}))
	assert.Contains(t, out.String(), "Logged out")
	assert.ElementsMatch(t, []string{"access-1", "refresh-1"}, f.revokedTokens,
		"should revoke both tokens")

	creds := loadTestCredentials(t)
	host, _ := oauth.CanonicalIssuer(f.server.URL)
	assert.Nil(t, creds.Credentials[host])
}

func TestNewDataAPIClient(t *testing.T) {
	listAccounts := func(t *testing.T) error {
		client, err := newDataAPIClient()
		if err != nil {
			return err
		}
		_, err = client.Accounts.List(context.Background())
		require.NoError(t, err)
		return nil
	}

	t.Run("personal auth token uses basic auth", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)
		viper.Set("auth_token", "personal-token")

		require.NoError(t, listAccounts(t))
		require.Len(t, f.apiAuthHeaders, 1)
		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("personal-token:"))
		assert.Equal(t, expected, f.apiAuthHeaders[0])
	})

	t.Run("oauth credentials use bearer auth", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)
		saveTestCredentials(t, f, &credentials.Entry{
			ClientID:    "client-1",
			AccessToken: "stored-access",
			ExpiresAt:   time.Now().Add(time.Hour),
		})

		require.NoError(t, listAccounts(t))
		require.Len(t, f.apiAuthHeaders, 1)
		assert.Equal(t, "Bearer stored-access", f.apiAuthHeaders[0])
	})

	t.Run("expired token is refreshed and persisted", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)
		saveTestCredentials(t, f, &credentials.Entry{
			ClientID:      "client-1",
			TokenEndpoint: f.server.URL + "/oauth/token",
			AccessToken:   "stale-access",
			RefreshToken:  "stale-refresh",
			ExpiresAt:     time.Now().Add(-time.Hour),
		})

		require.NoError(t, listAccounts(t))
		assert.Equal(t, 1, f.refreshCalls)
		require.Len(t, f.apiAuthHeaders, 1)
		assert.Equal(t, "Bearer oauth-access", f.apiAuthHeaders[0])

		creds := loadTestCredentials(t)
		host, _ := oauth.CanonicalIssuer(f.server.URL)
		assert.Equal(t, "oauth-access", creds.Credentials[host].AccessToken)
		assert.Equal(t, "oauth-refresh", creds.Credentials[host].RefreshToken)
	})

	t.Run("expired token without refresh token errors", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)
		saveTestCredentials(t, f, &credentials.Entry{
			ClientID:    "client-1",
			AccessToken: "stale-access",
			ExpiresAt:   time.Now().Add(-time.Hour),
		})

		_, err := newDataAPIClient()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hb auth login")
	})

	t.Run("no credentials at all errors", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)

		_, err := newDataAPIClient()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth token is required")
		assert.Contains(t, err.Error(), "hb auth login")
	})
}

func TestChooseDeviceFlow(t *testing.T) {
	withDevice := &oauth.Metadata{
		GrantTypesSupported: []string{"authorization_code", "refresh_token", oauth.DeviceGrantType},
	}
	withoutDevice := &oauth.Metadata{
		GrantTypesSupported: []string{"authorization_code", "refresh_token"},
	}
	deviceOnly := &oauth.Metadata{
		GrantTypesSupported: []string{oauth.DeviceGrantType, "refresh_token"},
	}

	reset := func(t *testing.T) {
		authLoginDevice = false
		authLoginWeb = false
		for _, envVar := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY", "WAYLAND_DISPLAY"} {
			t.Setenv(envVar, "")
		}
		t.Setenv("DISPLAY", ":0")
	}

	t.Run("conflicting flags error", func(t *testing.T) {
		reset(t)
		authLoginDevice = true
		authLoginWeb = true
		_, err := chooseDeviceFlow(withDevice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be combined")
	})

	t.Run("--device wins even without a terminal heuristic", func(t *testing.T) {
		reset(t)
		authLoginDevice = true
		useDevice, err := chooseDeviceFlow(withDevice)
		require.NoError(t, err)
		assert.True(t, useDevice)
	})

	t.Run("--web wins even over SSH", func(t *testing.T) {
		reset(t)
		authLoginWeb = true
		t.Setenv("SSH_CONNECTION", "10.0.0.1 1234 10.0.0.2 22")
		useDevice, err := chooseDeviceFlow(withDevice)
		require.NoError(t, err)
		assert.False(t, useDevice)
	})

	t.Run("SSH session prefers device flow when supported", func(t *testing.T) {
		reset(t)
		t.Setenv("SSH_CONNECTION", "10.0.0.1 1234 10.0.0.2 22")
		useDevice, err := chooseDeviceFlow(withDevice)
		require.NoError(t, err)
		assert.True(t, useDevice)
	})

	t.Run("SSH session without server device support stays on browser flow", func(t *testing.T) {
		reset(t)
		t.Setenv("SSH_TTY", "/dev/pts/0")
		useDevice, err := chooseDeviceFlow(withoutDevice)
		require.NoError(t, err)
		assert.False(t, useDevice)
	})

	t.Run("local machine prefers browser flow", func(t *testing.T) {
		reset(t)
		useDevice, err := chooseDeviceFlow(withDevice)
		require.NoError(t, err)
		assert.False(t, useDevice)
	})

	t.Run("device-only server uses device flow", func(t *testing.T) {
		reset(t)
		useDevice, err := chooseDeviceFlow(deviceOnly)
		require.NoError(t, err)
		assert.True(t, useDevice)
	})
}

func TestAuthLoginAutoSelectsDeviceOverSSH(t *testing.T) {
	f := newFakeOAuthServer(t, true)
	setupAuthTest(t, f)
	t.Setenv("SSH_CONNECTION", "10.0.0.1 1234 10.0.0.2 22")

	var out bytes.Buffer
	authLoginCmd.SetOut(&out)
	defer authLoginCmd.SetOut(nil)

	require.NoError(t, authLoginCmd.RunE(authLoginCmd, []string{}))
	assert.Contains(t, out.String(), "AAAA-BBBB", "should have auto-selected the device flow")
	assert.Contains(t, out.String(), "--web", "device flow should hint at the browser alternative")
}

func TestAuthLoginHints(t *testing.T) {
	t.Run("browser flow hints at --device when supported", func(t *testing.T) {
		f := newFakeOAuthServer(t, true)
		setupAuthTest(t, f)
		authLoginWeb = true

		var out bytes.Buffer
		authLoginCmd.SetOut(&out)
		defer authLoginCmd.SetOut(nil)

		require.NoError(t, authLoginCmd.RunE(authLoginCmd, []string{}))
		assert.Contains(t, out.String(), "--device")
	})

	t.Run("browser flow shows no hint when the server lacks the device grant", func(t *testing.T) {
		f := newFakeOAuthServer(t, false)
		setupAuthTest(t, f)

		var out bytes.Buffer
		authLoginCmd.SetOut(&out)
		defer authLoginCmd.SetOut(nil)

		require.NoError(t, authLoginCmd.RunE(authLoginCmd, []string{}))
		assert.NotContains(t, out.String(), "--device")
	})
}
