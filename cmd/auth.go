package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/honeybadger-io/cli/internal/credentials"
	"github.com/honeybadger-io/cli/internal/oauth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	oauthClientName    = "Honeybadger CLI"
	oauthDefaultScopes = "read write"
	tokenExpirySkew    = time.Minute
)

var (
	authLoginDevice bool
	authLoginScopes string
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate the CLI with your Honeybadger account",
	Long: `Log in to Honeybadger with OAuth instead of configuring a personal auth token.

'hb auth login' opens your browser to authorize the CLI (OAuth 2.0
authorization code flow with PKCE). On a machine without a browser, use
'hb auth login --device' to sign in from another device.

A personal auth token (--auth-token, HONEYBADGER_AUTH_TOKEN, or auth_token in
the config file) always takes precedence over OAuth credentials when set.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Honeybadger with OAuth",
	Long: `Log in to Honeybadger using OAuth 2.0.

By default this opens your browser to authorize the CLI. With --device it uses
the device authorization flow instead: the CLI shows a one-time code you enter
on another device (useful over SSH or on headless machines).

Credentials are stored in ~/.honeybadger-cli-credentials.json (override with
HONEYBADGER_CREDENTIALS_FILE) and refreshed automatically when they expire.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAuthLogin(cmd)
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and revoke stored OAuth credentials",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAuthLogout(cmd)
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current authentication status",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runAuthStatus(cmd)
	},
}

func init() {
	authLoginCmd.Flags().BoolVar(&authLoginDevice, "device", false,
		"use the device authorization flow (sign in from another device)")
	authLoginCmd.Flags().StringVar(&authLoginScopes, "scopes", oauthDefaultScopes,
		"space-separated OAuth scopes to request")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command) error {
	ctx := cmdContext(cmd)
	out := cmd.OutOrStdout()
	issuer := convertEndpointForDataAPI(viper.GetString("endpoint"))
	httpClient := &http.Client{Timeout: 30 * time.Second}

	metadata, err := oauth.Discover(ctx, httpClient, issuer)
	if err != nil {
		return err
	}

	issuerKey, err := oauth.CanonicalIssuer(issuer)
	if err != nil {
		return err
	}
	credsPath, err := credentials.Path()
	if err != nil {
		return err
	}
	credsFile, err := credentials.Load(credsPath)
	if err != nil {
		// Preserve the unreadable file instead of overwriting it on save,
		// which would silently drop credentials for other issuers.
		backup := credsPath + ".corrupt"
		if renameErr := os.Rename(credsPath, backup); renameErr != nil {
			return fmt.Errorf(
				"credentials file is unreadable (%v) and could not be moved aside: %v",
				err, renameErr,
			)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Warning: %v (moved to %s; starting fresh)\n", err, backup)
		credsFile = &credentials.File{Version: 1, Credentials: map[string]*credentials.Entry{}}
	}
	entry := credsFile.Credentials[issuerKey]
	if entry == nil {
		entry = &credentials.Entry{}
	}

	var token *oauth.Token
	if authLoginDevice {
		token, err = loginWithDeviceFlow(ctx, cmd, httpClient, metadata, entry)
	} else {
		token, err = loginWithBrowserFlow(ctx, cmd, httpClient, metadata, entry)
	}
	if err != nil {
		return err
	}

	entry.TokenEndpoint = metadata.TokenEndpoint
	entry.RevocationEndpoint = metadata.RevocationEndpoint
	entry.AccessToken = token.AccessToken
	entry.RefreshToken = token.RefreshToken
	entry.TokenType = token.TokenType
	entry.Scope = token.Scope
	entry.ExpiresAt = token.ExpiresAt
	credsFile.Credentials[issuerKey] = entry
	if err := credentials.Save(credsPath, credsFile); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "✓ Logged in to %s\n", issuerKey)
	return nil
}

// loginWithBrowserFlow runs the authorization code + PKCE flow (RFC 8252),
// registering an OAuth client for the loopback redirect when needed.
func loginWithBrowserFlow(
	ctx context.Context,
	cmd *cobra.Command,
	httpClient *http.Client,
	metadata *oauth.Metadata,
	entry *credentials.Entry,
) (*oauth.Token, error) {
	if !metadata.SupportsGrant("authorization_code") {
		return nil, fmt.Errorf("the server does not support the authorization code grant")
	}

	listener, clientID, err := browserFlowClient(ctx, httpClient, metadata, entry)
	if err != nil {
		return nil, err
	}

	flow := &oauth.AuthCodeFlow{
		HTTPClient:  httpClient,
		Metadata:    metadata,
		ClientID:    clientID,
		Scope:       authLoginScopes,
		Listener:    listener,
		OpenBrowser: openBrowser,
		Out:         cmd.OutOrStdout(),
	}
	token, err := flow.Run(ctx)
	if err != nil {
		return nil, err
	}

	entry.ClientID = clientID
	entry.RedirectURI = oauth.RedirectURIFor(listener)
	entry.GrantType = "authorization_code"
	return token, nil
}

// browserFlowClient binds the loopback listener and resolves the OAuth client
// to use with it: a configured oauth_client_id, a previously registered
// client (when its redirect port is still available), or a newly registered
// one (RFC 7591).
func browserFlowClient(
	ctx context.Context,
	httpClient *http.Client,
	metadata *oauth.Metadata,
	entry *credentials.Entry,
) (net.Listener, string, error) {
	if clientID := viper.GetString("oauth_client_id"); clientID != "" {
		listener, err := oauth.ListenLoopback(0)
		if err != nil {
			return nil, "", err
		}
		return listener, clientID, nil
	}

	// Reuse the cached registered client if we can bind its exact redirect
	// port again; loopback redirect URIs are matched exactly by the server.
	if entry.ClientID != "" && entry.GrantType == "authorization_code" && entry.RedirectURI != "" {
		if port := redirectURIPort(entry.RedirectURI); port > 0 {
			if listener, err := oauth.ListenLoopback(port); err == nil {
				return listener, entry.ClientID, nil
			}
		}
	}

	if metadata.RegistrationEndpoint == "" {
		return nil, "", fmt.Errorf(
			"the server does not support dynamic client registration; set oauth_client_id in your config or HONEYBADGER_OAUTH_CLIENT_ID",
		)
	}

	listener, err := oauth.ListenLoopback(0)
	if err != nil {
		return nil, "", err
	}
	clientID, err := oauth.Register(
		ctx,
		httpClient,
		metadata.RegistrationEndpoint,
		oauth.RegistrationRequest{
			ClientName:    oauthClientName,
			RedirectURIs:  []string{oauth.RedirectURIFor(listener)},
			GrantTypes:    grantTypesFor("authorization_code", metadata),
			ResponseTypes: []string{"code"},
			Scope:         authLoginScopes,
		},
	)
	if err != nil {
		_ = listener.Close()
		return nil, "", err
	}
	return listener, clientID, nil
}

// loginWithDeviceFlow runs the device authorization flow (RFC 8628).
func loginWithDeviceFlow(
	ctx context.Context,
	cmd *cobra.Command,
	httpClient *http.Client,
	metadata *oauth.Metadata,
	entry *credentials.Entry,
) (*oauth.Token, error) {
	clientID := viper.GetString("oauth_client_id")
	if clientID == "" && entry.ClientID != "" && entry.GrantType == oauth.DeviceGrantType {
		clientID = entry.ClientID
	}
	if clientID == "" {
		if !metadata.SupportsGrant(oauth.DeviceGrantType) {
			return nil, fmt.Errorf(
				"the server does not support the device authorization grant; run 'hb auth login' without --device to use the browser flow",
			)
		}
		if metadata.RegistrationEndpoint == "" {
			return nil, fmt.Errorf(
				"the server does not support dynamic client registration; set oauth_client_id in your config or HONEYBADGER_OAUTH_CLIENT_ID",
			)
		}
		var err error
		clientID, err = oauth.Register(
			ctx,
			httpClient,
			metadata.RegistrationEndpoint,
			oauth.RegistrationRequest{
				ClientName: oauthClientName,
				GrantTypes: grantTypesFor(oauth.DeviceGrantType, metadata),
				Scope:      authLoginScopes,
			},
		)
		if err != nil {
			return nil, err
		}
	}

	flow := &oauth.DeviceFlow{
		HTTPClient: httpClient,
		Metadata:   metadata,
		ClientID:   clientID,
		Scope:      authLoginScopes,
		Out:        cmd.OutOrStdout(),
	}
	token, err := flow.Run(ctx)
	if err != nil {
		return nil, err
	}

	entry.ClientID = clientID
	entry.RedirectURI = ""
	entry.GrantType = oauth.DeviceGrantType
	return token, nil
}

func runAuthLogout(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	issuer := convertEndpointForDataAPI(viper.GetString("endpoint"))
	issuerKey, err := oauth.CanonicalIssuer(issuer)
	if err != nil {
		return err
	}
	credsPath, err := credentials.Path()
	if err != nil {
		return err
	}
	credsFile, err := credentials.Load(credsPath)
	if err != nil {
		return err
	}
	entry := credsFile.Credentials[issuerKey]
	if entry == nil {
		_, _ = fmt.Fprintf(out, "Not logged in to %s\n", issuerKey)
		return nil
	}

	// Revoke tokens best-effort (RFC 7009); log out locally regardless.
	if entry.RevocationEndpoint != "" && entry.ClientID != "" {
		ctx, cancel := context.WithTimeout(cmdContext(cmd), 30*time.Second)
		defer cancel()
		httpClient := &http.Client{Timeout: 30 * time.Second}
		for _, token := range []string{entry.RefreshToken, entry.AccessToken} {
			if token == "" {
				continue
			}
			if err := oauth.Revoke(
				ctx,
				httpClient,
				entry.RevocationEndpoint,
				entry.ClientID,
				token,
			); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to revoke token: %v\n", err)
			}
		}
	}

	delete(credsFile.Credentials, issuerKey)
	if err := credentials.Save(credsPath, credsFile); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "✓ Logged out of %s\n", issuerKey)
	return nil
}

func runAuthStatus(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	issuer := convertEndpointForDataAPI(viper.GetString("endpoint"))
	issuerKey, err := oauth.CanonicalIssuer(issuer)
	if err != nil {
		return err
	}

	if viper.GetString("auth_token") != "" {
		_, _ = fmt.Fprintln(
			out,
			"Using a personal auth token (--auth-token, HONEYBADGER_AUTH_TOKEN, or config file).",
		)
		_, _ = fmt.Fprintln(out, "Personal auth tokens take precedence over OAuth credentials.")
		return nil
	}

	credsPath, err := credentials.Path()
	if err != nil {
		return err
	}
	credsFile, err := credentials.Load(credsPath)
	if err != nil {
		return err
	}
	entry := credsFile.Credentials[issuerKey]
	if entry == nil || entry.AccessToken == "" {
		return fmt.Errorf("not logged in to %s. Run 'hb auth login' to authenticate", issuerKey)
	}

	_, _ = fmt.Fprintf(out, "Logged in to %s via OAuth\n", issuerKey)
	if entry.Scope != "" {
		_, _ = fmt.Fprintf(out, "Scopes: %s\n", entry.Scope)
	}
	switch {
	case entry.ExpiresAt.IsZero():
		_, _ = fmt.Fprintln(out, "Token expiry: none reported")
	case entry.Expired(time.Now(), 0):
		if entry.RefreshToken != "" {
			_, _ = fmt.Fprintf(out, "Token expired %s (will refresh automatically on next use)\n",
				entry.ExpiresAt.Local().Format(time.RFC3339))
		} else {
			_, _ = fmt.Fprintf(out, "Token expired %s (run 'hb auth login' to sign in again)\n",
				entry.ExpiresAt.Local().Format(time.RFC3339))
		}
	default:
		_, _ = fmt.Fprintf(out, "Token expires: %s\n", entry.ExpiresAt.Local().Format(time.RFC3339))
	}
	return nil
}

// newDataAPIClient builds a Data API client authenticated with the personal
// auth token when configured, or stored OAuth credentials otherwise
// (refreshing the access token transparently when it has expired).
func newDataAPIClient() (*hbapi.Client, error) {
	endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))
	client := hbapi.NewClient().WithBaseURL(endpoint)

	if authToken := viper.GetString("auth_token"); authToken != "" {
		return client.WithAuthToken(authToken), nil
	}

	accessToken, err := storedAccessToken(endpoint)
	if err != nil {
		return nil, err
	}
	if accessToken == "" {
		return nil, fmt.Errorf(
			"auth token is required. Run 'hb auth login' to sign in with your browser, or set a personal auth token using the --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
		)
	}
	return client.WithBearerToken(accessToken), nil
}

// storedAccessToken returns a valid stored OAuth access token for the issuer,
// refreshing and persisting it when expired. It returns "" (no error) when
// the user has never logged in.
func storedAccessToken(issuer string) (string, error) {
	issuerKey, err := oauth.CanonicalIssuer(issuer)
	if err != nil {
		return "", nil //nolint:nilerr // an unparsable endpoint just means no stored OAuth creds
	}
	credsPath, err := credentials.Path()
	if err != nil {
		return "", nil //nolint:nilerr // no resolvable home dir means no stored OAuth creds
	}
	credsFile, err := credentials.Load(credsPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: ignoring credentials file: %v\n", err)
		return "", nil
	}
	entry := credsFile.Credentials[issuerKey]
	if entry == nil || entry.AccessToken == "" {
		return "", nil
	}

	if !entry.Expired(time.Now(), tokenExpirySkew) {
		return entry.AccessToken, nil
	}
	if entry.RefreshToken == "" || entry.TokenEndpoint == "" || entry.ClientID == "" {
		return "", fmt.Errorf(
			"your Honeybadger session has expired. Run 'hb auth login' to sign in again",
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	token, err := oauth.Refresh(
		ctx,
		httpClient,
		entry.TokenEndpoint,
		entry.ClientID,
		entry.RefreshToken,
	)
	if err != nil {
		return "", fmt.Errorf(
			"failed to refresh your Honeybadger session (%v). Run 'hb auth login' to sign in again",
			err,
		)
	}

	rotated := token.RefreshToken != entry.RefreshToken
	entry.AccessToken = token.AccessToken
	entry.RefreshToken = token.RefreshToken
	entry.TokenType = token.TokenType
	if token.Scope != "" {
		entry.Scope = token.Scope
	}
	entry.ExpiresAt = token.ExpiresAt
	if err := credentials.Save(credsPath, credsFile); err != nil {
		if rotated {
			// The server invalidated the old refresh token; losing the new
			// one would silently break every later command.
			return "", fmt.Errorf(
				"failed to save refreshed credentials (%v); run 'hb auth login' to sign in again",
				err,
			)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to save refreshed credentials: %v\n", err)
	}
	return token.AccessToken, nil
}

// grantTypesFor returns the grant types to register for a flow, including
// refresh_token when the server supports it.
func grantTypesFor(primary string, metadata *oauth.Metadata) []string {
	grants := []string{primary}
	if metadata.SupportsGrant("refresh_token") {
		grants = append(grants, "refresh_token")
	}
	return grants
}

// cmdContext returns the command's context, falling back to Background when
// the command is run outside Execute (as in tests).
func cmdContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// redirectURIPort extracts the port from a stored loopback redirect URI,
// returning 0 when it cannot be determined.
func redirectURIPort(redirectURI string) int {
	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		return 0
	}
	return port
}

// openBrowser launches the default browser at the given URL. It is a
// variable so tests can substitute a fake browser.
var openBrowser = func(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start() // #nosec G204 -- u is our authorize URL
	case "windows":
		// #nosec G204 -- u is our authorize URL
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return exec.Command("xdg-open", u).
			Start()
		// #nosec G204 - u is the OAuth authorize URL we built
	}
}
