package oauth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DeviceFlow runs the device authorization grant (RFC 8628): it requests a
// user code, tells the user where to enter it, and polls the token endpoint
// until authorization completes.
type DeviceFlow struct {
	HTTPClient *http.Client
	Metadata   *Metadata
	ClientID   string
	Scope      string
	Out        io.Writer

	// sleep is injectable for tests; nil means a context-aware time.Sleep.
	sleep func(ctx context.Context, d time.Duration) error
}

type deviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

// Run executes the device flow and returns the issued token.
func (f *DeviceFlow) Run(ctx context.Context) (*Token, error) {
	if f.Metadata.DeviceAuthorizationEndpoint == "" || !f.Metadata.SupportsGrant(DeviceGrantType) {
		return nil, fmt.Errorf(
			"the server does not support the device authorization grant; run 'hb auth login' without --device to use the browser flow",
		)
	}

	var auth deviceAuthorization
	form := url.Values{"client_id": {f.ClientID}}
	if f.Scope != "" {
		form.Set("scope", f.Scope)
	}
	if err := postForm(
		ctx,
		f.HTTPClient,
		f.Metadata.DeviceAuthorizationEndpoint,
		form,
		&auth,
	); err != nil {
		return nil, fmt.Errorf("device authorization request failed: %w", err)
	}
	if auth.DeviceCode == "" || auth.UserCode == "" {
		return nil, fmt.Errorf("device authorization response was missing device_code or user_code")
	}

	_, _ = fmt.Fprintf(f.Out, "First copy your one-time code: %s\n\n", auth.UserCode)
	if auth.VerificationURIComplete != "" {
		_, _ = fmt.Fprintf(
			f.Out, "Then visit (code pre-filled): %s\n", auth.VerificationURIComplete,
		)
	}
	if auth.VerificationURI != "" {
		_, _ = fmt.Fprintf(f.Out, "Or enter the code at: %s\n", auth.VerificationURI)
	}
	_, _ = fmt.Fprintf(f.Out, "\nWaiting for authorization...\n")

	interval := time.Duration(auth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second // RFC 8628 §3.2 default
	}
	expiresIn := time.Duration(auth.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 15 * time.Minute
	}
	deadline := time.Now().Add(expiresIn)

	sleep := f.sleep
	if sleep == nil {
		sleep = sleepContext
	}

	for {
		if err := sleep(ctx, interval); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf(
				"the device code expired before authorization completed; run 'hb auth login' again",
			)
		}

		tok, err := requestToken(ctx, f.HTTPClient, f.Metadata.TokenEndpoint, url.Values{
			"grant_type":  {DeviceGrantType},
			"device_code": {auth.DeviceCode},
			"client_id":   {f.ClientID},
		})
		if err == nil {
			return tok, nil
		}

		oauthErr, ok := err.(*Error)
		if !ok {
			return nil, err
		}
		switch oauthErr.Code {
		case "authorization_pending":
			// keep polling
		case "slow_down":
			interval += 5 * time.Second // RFC 8628 §3.5
		case "access_denied":
			return nil, fmt.Errorf("authorization was denied")
		case "expired_token":
			return nil, fmt.Errorf(
				"the device code expired before authorization completed; run 'hb auth login' again",
			)
		default:
			return nil, err
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
