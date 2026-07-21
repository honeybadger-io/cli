// Package oauth implements the OAuth 2.0 flows used by `hb auth login`:
// authorization server metadata discovery (RFC 8414), dynamic client
// registration (RFC 7591), the authorization code grant with PKCE for native
// apps (RFC 8252, RFC 7636), the device authorization grant (RFC 8628),
// refresh token grant, and token revocation (RFC 7009).
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DeviceGrantType is the RFC 8628 device authorization grant type identifier.
const DeviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// Metadata is the authorization server metadata document (RFC 8414).
type Metadata struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	DeviceAuthorizationEndpoint   string   `json:"device_authorization_endpoint"`
	RegistrationEndpoint          string   `json:"registration_endpoint"`
	RevocationEndpoint            string   `json:"revocation_endpoint"`
	GrantTypesSupported           []string `json:"grant_types_supported"`
	ScopesSupported               []string `json:"scopes_supported"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

// SupportsGrant reports whether the server advertises the given grant type.
// Per RFC 8414, an absent grant_types_supported implies the default of
// authorization_code and implicit.
func (m *Metadata) SupportsGrant(grant string) bool {
	if len(m.GrantTypesSupported) == 0 {
		return grant == "authorization_code"
	}
	for _, g := range m.GrantTypesSupported {
		if g == grant {
			return true
		}
	}
	return false
}

// CanonicalIssuer normalizes an issuer URL for comparison and storage:
// lowercased scheme and host, no trailing slash, no query or fragment. It
// also enforces the RFC 8414 transport rules: HTTPS is required except for
// loopback hosts (which keep local development and tests working).
func CanonicalIssuer(issuer string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid issuer URL %q", issuer)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if err := checkEndpointScheme(parsed); err != nil {
		return "", fmt.Errorf("invalid issuer URL %q: %w", issuer, err)
	}
	return parsed.String(), nil
}

// checkEndpointScheme requires https, allowing http only for loopback hosts.
func checkEndpointScheme(u *url.URL) error {
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("http is only allowed for loopback hosts; use https")
	default:
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
}

// wellKnownURL builds the RFC 8414 §3.1 metadata URL: the well-known path is
// inserted between the authority and any issuer path component.
func wellKnownURL(issuer string) (string, error) {
	parsed, err := url.Parse(issuer)
	if err != nil {
		return "", fmt.Errorf("invalid issuer URL %q", issuer)
	}
	path := parsed.Path
	parsed.Path = "/.well-known/oauth-authorization-server" + path
	return parsed.String(), nil
}

// validateEndpoints checks that every advertised endpoint the CLI will send
// credentials to is a well-formed HTTPS (or loopback) URL.
func (m *Metadata) validateEndpoints() error {
	endpoints := map[string]string{
		"authorization_endpoint":        m.AuthorizationEndpoint,
		"token_endpoint":                m.TokenEndpoint,
		"device_authorization_endpoint": m.DeviceAuthorizationEndpoint,
		"registration_endpoint":         m.RegistrationEndpoint,
		"revocation_endpoint":           m.RevocationEndpoint,
	}
	for name, endpoint := range endpoints {
		if endpoint == "" {
			continue
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("invalid OAuth server metadata: malformed %s %q", name, endpoint)
		}
		if err := checkEndpointScheme(parsed); err != nil {
			return fmt.Errorf("invalid OAuth server metadata: %s: %w", name, err)
		}
	}
	return nil
}

// Discover fetches the authorization server metadata for the given issuer
// from the RFC 8414 well-known location. The returned metadata is validated:
// the advertised issuer must match the requested one (mix-up protection,
// RFC 8414 §3.3) and all endpoints must be HTTPS (or loopback).
func Discover(ctx context.Context, hc *http.Client, issuer string) (*Metadata, error) {
	canonical, err := CanonicalIssuer(issuer)
	if err != nil {
		return nil, err
	}
	wellKnown, err := wellKnownURL(canonical)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req) // #nosec G704 -- URL is the user-configured endpoint's well-known path
	if err != nil {
		return nil, fmt.Errorf("OAuth discovery failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"OAuth discovery failed: %s returned HTTP %d (the server may not support OAuth login)",
			wellKnown, resp.StatusCode,
		)
	}

	var md Metadata
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&md); err != nil {
		return nil, fmt.Errorf("failed to parse OAuth server metadata: %w", err)
	}

	advertised, err := CanonicalIssuer(md.Issuer)
	if err != nil || advertised != canonical {
		return nil, fmt.Errorf(
			"OAuth server metadata issuer %q does not match the requested issuer %q",
			md.Issuer, canonical,
		)
	}
	if md.TokenEndpoint == "" {
		return nil, fmt.Errorf("invalid OAuth server metadata: missing token_endpoint")
	}
	if err := md.validateEndpoints(); err != nil {
		return nil, err
	}
	return &md, nil
}

// Token is a set of issued OAuth credentials.
type Token struct {
	AccessToken  string // #nosec G117 -- this type's purpose is to carry tokens
	RefreshToken string // #nosec G117
	TokenType    string
	Scope        string
	// ExpiresAt is the absolute expiry of the access token; zero when the
	// server did not report expires_in.
	ExpiresAt time.Time
}

// Error is an OAuth 2.0 error response (RFC 6749 §5.2).
type Error struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *Error) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Description)
	}
	return e.Code
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"` // #nosec G117 -- OAuth token endpoint response
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"` // #nosec G117
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
}

// postForm sends a form-encoded POST and decodes the JSON response into out.
// Non-2xx responses are returned as *Error when the body carries an OAuth
// error document.
func postForm(
	ctx context.Context,
	hc *http.Client,
	endpoint string,
	form url.Values,
	out interface{},
) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req) // #nosec G704 -- URL comes from the server's OAuth metadata
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var oauthErr Error
		if json.Unmarshal(body, &oauthErr) == nil && oauthErr.Code != "" {
			return &oauthErr
		}
		return fmt.Errorf("%s returned HTTP %d", endpoint, resp.StatusCode)
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("failed to parse response from %s: %w", endpoint, err)
		}
	}
	return nil
}

// requestToken posts to the token endpoint and converts the response into a
// Token with an absolute expiry.
func requestToken(
	ctx context.Context,
	hc *http.Client,
	tokenEndpoint string,
	form url.Values,
) (*Token, error) {
	var tr tokenResponse
	if err := postForm(ctx, hc, tokenEndpoint, form, &tr); err != nil {
		return nil, err
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned no access token")
	}
	tok := &Token{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tr.TokenType,
		Scope:        tr.Scope,
	}
	if tr.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return tok, nil
}

// Refresh exchanges a refresh token for a new token pair (RFC 6749 §6).
func Refresh(
	ctx context.Context,
	hc *http.Client,
	tokenEndpoint, clientID, refreshToken string,
) (*Token, error) {
	tok, err := requestToken(ctx, hc, tokenEndpoint, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	})
	if err != nil {
		return nil, err
	}
	// Servers that do not rotate refresh tokens omit them from the response;
	// keep using the current one.
	if tok.RefreshToken == "" {
		tok.RefreshToken = refreshToken
	}
	return tok, nil
}

// Revoke revokes a token at the revocation endpoint (RFC 7009).
func Revoke(
	ctx context.Context,
	hc *http.Client,
	revocationEndpoint, clientID, token string,
) error {
	return postForm(ctx, hc, revocationEndpoint, url.Values{
		"token":     {token},
		"client_id": {clientID},
	}, nil)
}

// randomURLSafe returns n random bytes base64url-encoded without padding.
func randomURLSafe(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// s256Challenge computes the S256 PKCE code challenge for a verifier
// (RFC 7636 §4.2).
func s256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
