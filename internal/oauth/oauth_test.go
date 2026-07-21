package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func metadataHandler(md *Metadata) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(md)
	}
}

func TestDiscover(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mux := http.NewServeMux()
		server := httptest.NewServer(mux)
		defer server.Close()
		mux.HandleFunc("/.well-known/oauth-authorization-server", metadataHandler(&Metadata{
			Issuer:                server.URL,
			AuthorizationEndpoint: server.URL + "/oauth/authorize",
			TokenEndpoint:         server.URL + "/oauth/token",
			GrantTypesSupported:   []string{"authorization_code", "refresh_token"},
		}))

		md, err := Discover(context.Background(), server.Client(), server.URL)
		require.NoError(t, err)
		assert.Equal(t, server.URL+"/oauth/token", md.TokenEndpoint)
		assert.True(t, md.SupportsGrant("authorization_code"))
		assert.False(t, md.SupportsGrant(DeviceGrantType))
	})

	t.Run("not found", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()

		_, err := Discover(context.Background(), server.Client(), server.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 404")
	})

	t.Run("missing token endpoint", func(t *testing.T) {
		server := httptest.NewServer(metadataHandler(&Metadata{Issuer: "x"}))
		defer server.Close()

		_, err := Discover(context.Background(), server.Client(), server.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing token_endpoint")
	})
}

func TestSupportsGrantDefaults(t *testing.T) {
	md := &Metadata{}
	assert.True(t, md.SupportsGrant("authorization_code"))
	assert.False(t, md.SupportsGrant(DeviceGrantType))
}

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			var req RegistrationRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "none", req.TokenEndpointAuthMethod)
			assert.Equal(t, "Honeybadger CLI", req.ClientName)

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"client_id": "abc123"}`))
		}))
		defer server.Close()

		clientID, err := Register(
			context.Background(),
			server.Client(),
			server.URL,
			RegistrationRequest{
				ClientName:   "Honeybadger CLI",
				RedirectURIs: []string{"http://127.0.0.1:8000/callback"},
				GrantTypes:   []string{"authorization_code", "refresh_token"},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "abc123", clientID)
	})

	t.Run("oauth error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(
				[]byte(`{"error": "invalid_redirect_uri", "error_description": "bad uri"}`),
			)
		}))
		defer server.Close()

		_, err := Register(context.Background(), server.Client(), server.URL, RegistrationRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid_redirect_uri")
		assert.Contains(t, err.Error(), "bad uri")
	})
}

// fakeAuthServer simulates the authorization server for the auth-code flow.
// Its "browser" follows the authorize URL and invokes the loopback redirect.
type fakeAuthServer struct {
	t          *testing.T
	server     *httptest.Server
	authCode   string
	denyAccess bool
	tamperWith string // query param to overwrite in the redirect

	gotVerifier  string
	gotChallenge string
}

func newFakeAuthServer(t *testing.T) *fakeAuthServer {
	f := &fakeAuthServer{t: t, authCode: "test-auth-code"}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, f.authCode, r.Form.Get("code"))
		f.gotVerifier = r.Form.Get("code_verifier")
		if s256Challenge(f.gotVerifier) != f.gotChallenge {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(
				[]byte(
					`{"error": "invalid_grant", "error_description": "PKCE verification failed"}`,
				),
			)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "access-123",
			"refresh_token": "refresh-456",
			"token_type": "Bearer",
			"scope": "read write",
			"expires_in": 3600
		}`))
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeAuthServer) metadata() *Metadata {
	return &Metadata{
		Issuer:                f.server.URL,
		AuthorizationEndpoint: f.server.URL + "/oauth/authorize",
		TokenEndpoint:         f.server.URL + "/oauth/token",
		GrantTypesSupported:   []string{"authorization_code", "refresh_token"},
	}
}

// browse acts as the user's browser: it parses the authorize URL and hits the
// loopback redirect URI with the resulting code (or error).
func (f *fakeAuthServer) browse(authURL string) error {
	u, err := parseURL(authURL)
	if err != nil {
		return err
	}
	q := u.Query()
	assert.Equal(f.t, "code", q.Get("response_type"))
	assert.Equal(f.t, "S256", q.Get("code_challenge_method"))
	assert.NotEmpty(f.t, q.Get("state"))
	f.gotChallenge = q.Get("code_challenge")

	redirect, err := parseURL(q.Get("redirect_uri"))
	if err != nil {
		return err
	}
	rq := redirect.Query()
	if f.denyAccess {
		rq.Set("error", "access_denied")
		rq.Set("error_description", "user said no")
	} else {
		rq.Set("code", f.authCode)
	}
	rq.Set("state", q.Get("state"))
	if f.tamperWith == "state" {
		rq.Set("state", "wrong-state")
	}
	redirect.RawQuery = rq.Encode()

	go func() {
		resp, err := http.Get(redirect.String())
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return nil
}

func runAuthCodeFlow(t *testing.T, f *fakeAuthServer) (*Token, error) {
	listener, err := ListenLoopback(0)
	require.NoError(t, err)

	flow := &AuthCodeFlow{
		HTTPClient:  f.server.Client(),
		Metadata:    f.metadata(),
		ClientID:    "client-1",
		Scope:       "read write",
		Listener:    listener,
		OpenBrowser: f.browse,
		Out:         testWriter{t},
		Timeout:     10 * time.Second,
	}
	return flow.Run(context.Background())
}

func TestAuthCodeFlow(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newFakeAuthServer(t)
		tok, err := runAuthCodeFlow(t, f)
		require.NoError(t, err)
		assert.Equal(t, "access-123", tok.AccessToken)
		assert.Equal(t, "refresh-456", tok.RefreshToken)
		assert.Equal(t, "read write", tok.Scope)
		assert.WithinDuration(t, time.Now().Add(time.Hour), tok.ExpiresAt, time.Minute)
		assert.NotEmpty(t, f.gotVerifier)
		assert.GreaterOrEqual(
			t,
			len(f.gotVerifier),
			43,
			"PKCE verifier must be at least 43 chars (RFC 7636)",
		)
	})

	t.Run("access denied", func(t *testing.T) {
		f := newFakeAuthServer(t)
		f.denyAccess = true
		_, err := runAuthCodeFlow(t, f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access_denied")
	})

	t.Run("state mismatch", func(t *testing.T) {
		f := newFakeAuthServer(t)
		f.tamperWith = "state"
		_, err := runAuthCodeFlow(t, f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state mismatch")
	})
}

func TestDeviceFlow(t *testing.T) {
	newServer := func(t *testing.T, pollResponses []string) (*fakePollServer, *Metadata) {
		f := &fakePollServer{t: t, responses: pollResponses}
		mux := http.NewServeMux()
		mux.HandleFunc("/oauth/authorize_device", func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "client-1", r.Form.Get("client_id"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"device_code": "dev-123",
				"user_code": "ABCD-EFGH",
				"verification_uri": "https://example.com/device",
				"verification_uri_complete": "https://example.com/device?user_code=ABCD-EFGH",
				"expires_in": 300,
				"interval": 1
			}`))
		})
		mux.HandleFunc("/oauth/token", f.handleToken)
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		f.server = server
		return f, &Metadata{
			Issuer:                      server.URL,
			TokenEndpoint:               server.URL + "/oauth/token",
			DeviceAuthorizationEndpoint: server.URL + "/oauth/authorize_device",
			GrantTypesSupported:         []string{DeviceGrantType, "refresh_token"},
		}
	}

	run := func(f *fakePollServer, md *Metadata) (*Token, []time.Duration, error) {
		var sleeps []time.Duration
		flow := &DeviceFlow{
			HTTPClient: f.server.Client(),
			Metadata:   md,
			ClientID:   "client-1",
			Scope:      "read write",
			Out:        testWriter{f.t},
			sleep: func(_ context.Context, d time.Duration) error {
				sleeps = append(sleeps, d)
				return nil
			},
		}
		tok, err := flow.Run(context.Background())
		return tok, sleeps, err
	}

	t.Run("pending then slow_down then success", func(t *testing.T) {
		f, md := newServer(t, []string{"authorization_pending", "slow_down", "ok"})
		tok, sleeps, err := run(f, md)
		require.NoError(t, err)
		assert.Equal(t, "device-access-token", tok.AccessToken)
		require.Len(t, sleeps, 3)
		assert.Equal(t, 1*time.Second, sleeps[0])
		assert.Equal(t, 1*time.Second, sleeps[1])
		assert.Equal(t, 6*time.Second, sleeps[2], "slow_down must add 5 seconds to the interval")
	})

	t.Run("access denied", func(t *testing.T) {
		f, md := newServer(t, []string{"access_denied"})
		_, _, err := run(f, md)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "denied")
	})

	t.Run("expired token", func(t *testing.T) {
		f, md := newServer(t, []string{"expired_token"})
		_, _, err := run(f, md)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("unsupported by server", func(t *testing.T) {
		f, md := newServer(t, nil)
		md.GrantTypesSupported = []string{"authorization_code"}
		md.DeviceAuthorizationEndpoint = ""
		_, _, err := run(f, md)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support the device authorization grant")
	})
}

type fakePollServer struct {
	t         *testing.T
	server    *httptest.Server
	responses []string
	calls     int
}

func (f *fakePollServer) handleToken(w http.ResponseWriter, r *http.Request) {
	require.NoError(f.t, r.ParseForm())
	assert.Equal(f.t, DeviceGrantType, r.Form.Get("grant_type"))
	assert.Equal(f.t, "dev-123", r.Form.Get("device_code"))

	require.Less(f.t, f.calls, len(f.responses), "token endpoint polled more times than expected")
	response := f.responses[f.calls]
	f.calls++

	w.Header().Set("Content-Type", "application/json")
	if response == "ok" {
		_, _ = w.Write(
			[]byte(
				`{"access_token": "device-access-token", "token_type": "Bearer", "expires_in": 3600}`,
			),
		)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	_, _ = fmt.Fprintf(w, `{"error": %q}`, response)
}

func TestRefresh(t *testing.T) {
	t.Run("rotates refresh token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
			assert.Equal(t, "old-refresh", r.Form.Get("refresh_token"))
			assert.Equal(t, "client-1", r.Form.Get("client_id"))
			_, _ = w.Write(
				[]byte(
					`{"access_token": "new-access", "refresh_token": "new-refresh", "expires_in": 3600}`,
				),
			)
		}))
		defer server.Close()

		tok, err := Refresh(
			context.Background(),
			server.Client(),
			server.URL,
			"client-1",
			"old-refresh",
		)
		require.NoError(t, err)
		assert.Equal(t, "new-access", tok.AccessToken)
		assert.Equal(t, "new-refresh", tok.RefreshToken)
	})

	t.Run("keeps refresh token when not rotated", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"access_token": "new-access", "expires_in": 3600}`))
		}))
		defer server.Close()

		tok, err := Refresh(
			context.Background(),
			server.Client(),
			server.URL,
			"client-1",
			"old-refresh",
		)
		require.NoError(t, err)
		assert.Equal(t, "old-refresh", tok.RefreshToken)
	})

	t.Run("invalid grant", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
		}))
		defer server.Close()

		_, err := Refresh(
			context.Background(),
			server.Client(),
			server.URL,
			"client-1",
			"old-refresh",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid_grant")
	})
}

func TestRevoke(t *testing.T) {
	var revoked string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		revoked = r.Form.Get("token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Revoke(context.Background(), server.Client(), server.URL, "client-1", "tok-1")
	require.NoError(t, err)
	assert.Equal(t, "tok-1", revoked)
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

func parseURL(s string) (*url.URL, error) { return url.Parse(s) }
