package oauth

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// AuthCodeFlow runs the authorization code grant with PKCE using a loopback
// redirect, following the OAuth 2.0 for Native Apps BCP (RFC 8252).
type AuthCodeFlow struct {
	HTTPClient *http.Client
	Metadata   *Metadata
	ClientID   string
	Scope      string
	// Listener is the pre-bound loopback listener the redirect will arrive
	// on. Binding before client registration lets the caller register the
	// exact redirect URI.
	Listener net.Listener
	// OpenBrowser launches the user's browser at the authorization URL. When
	// nil (or when it fails) the URL is only printed for manual use.
	OpenBrowser func(url string) error
	Out         io.Writer
	// Timeout bounds the wait for the user to complete authorization
	// (default 5 minutes).
	Timeout time.Duration
}

// ListenLoopback binds a loopback listener. Port 0 picks an ephemeral port.
func ListenLoopback(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}

// RedirectURIFor returns the loopback redirect URI for a bound listener.
func RedirectURIFor(l net.Listener) string {
	return fmt.Sprintf("http://%s/callback", l.Addr().String())
}

type callbackResult struct {
	code string
	err  error
}

// Run executes the flow: it serves the loopback redirect endpoint, directs
// the user's browser to the authorization endpoint, waits for the callback,
// and exchanges the authorization code (with the PKCE verifier) for tokens.
func (f *AuthCodeFlow) Run(ctx context.Context) (*Token, error) {
	if f.Metadata.AuthorizationEndpoint == "" {
		return nil, fmt.Errorf("authorization server does not advertise an authorization endpoint")
	}

	verifier, err := randomURLSafe(32)
	if err != nil {
		return nil, err
	}
	state, err := randomURLSafe(16)
	if err != nil {
		return nil, err
	}
	redirectURI := RedirectURIFor(f.Listener)

	authURL, err := buildAuthorizeURL(f.Metadata.AuthorizationEndpoint, url.Values{
		"response_type":         {"code"},
		"client_id":             {f.ClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {f.Scope},
		"state":                 {state},
		"code_challenge":        {s256Challenge(verifier)},
		"code_challenge_method": {"S256"},
	})
	if err != nil {
		return nil, err
	}

	results := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		res, ok := handleCallback(w, r, state)
		if !ok {
			return // not a response to this login attempt; keep waiting
		}
		select {
		case results <- res:
		default: // a result was already delivered; ignore duplicates
		}
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(f.Listener) }()
	defer func() { _ = server.Close() }()

	_, _ = fmt.Fprintf(f.Out, "Opening your browser to log in to Honeybadger.\n\n")
	// #nosec G705 -- authURL is written to the user's terminal, not an HTTP response
	_, _ = fmt.Fprintf(f.Out, "If the browser doesn't open, visit this URL:\n\n  %s\n\n", authURL)
	if f.OpenBrowser != nil {
		_ = f.OpenBrowser(authURL)
	}

	timeout := f.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var res callbackResult
	select {
	case res = <-results:
	case <-timer.C:
		return nil, fmt.Errorf("timed out waiting for authorization after %s", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if res.err != nil {
		return nil, res.err
	}

	return requestToken(ctx, f.HTTPClient, f.Metadata.TokenEndpoint, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {res.code},
		"redirect_uri":  {redirectURI},
		"client_id":     {f.ClientID},
		"code_verifier": {verifier},
	})
}

// buildAuthorizeURL appends the authorization parameters to the endpoint,
// preserving any query component the endpoint already carries.
func buildAuthorizeURL(endpoint string, params url.Values) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid authorization endpoint %q", endpoint)
	}
	existing := parsed.Query()
	for key, values := range params {
		existing[key] = values
	}
	parsed.RawQuery = existing.Encode()
	return parsed.String(), nil
}

// handleCallback processes one request to the loopback redirect endpoint.
// The second return value reports whether the request belongs to this login
// attempt: requests without a matching state (port scans, stray requests, a
// forged abort) get an error page but must not terminate the flow.
func handleCallback(w http.ResponseWriter, r *http.Request, state string) (callbackResult, bool) {
	q := r.URL.Query()

	if q.Get("state") != state {
		writeCallbackPage(
			w,
			http.StatusBadRequest,
			"Login failed",
			"The response didn't match this login attempt (state mismatch). Return to the terminal and try again.",
		)
		return callbackResult{}, false
	}
	if errCode := q.Get("error"); errCode != "" {
		writeCallbackPage(w, http.StatusBadRequest, "Login failed",
			"Authorization was not granted. You can close this tab and return to the terminal.")
		oauthErr := &Error{Code: errCode, Description: q.Get("error_description")}
		return callbackResult{err: fmt.Errorf("authorization failed: %w", oauthErr)}, true
	}
	code := q.Get("code")
	if code == "" {
		writeCallbackPage(w, http.StatusBadRequest, "Login failed",
			"The authorization response was missing a code. Return to the terminal and try again.")
		return callbackResult{err: fmt.Errorf("authorization response missing code")}, true
	}

	writeCallbackPage(
		w,
		http.StatusOK,
		"Login successful",
		"You're logged in to the Honeybadger CLI. You can close this tab and return to the terminal.",
	)
	return callbackResult{code: code}, true
}

func writeCallbackPage(w http.ResponseWriter, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>%s - Honeybadger CLI</title></head>
<body style="font-family: sans-serif; margin: 4em auto; max-width: 32em; text-align: center;">
<h1>%s</h1>
<p>%s</p>
</body>
</html>
`, title, title, message)
}
