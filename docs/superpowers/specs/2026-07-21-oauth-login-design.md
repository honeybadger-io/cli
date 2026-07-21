# OAuth Login for the Honeybadger CLI

**Date:** 2026-07-21
**Status:** Approved (autonomous session — decisions documented for review)

## Goal

Let users authenticate the Data API commands with `hb auth login` (OAuth 2.0)
instead of pasting a personal auth token into a config file or environment
variable. Personal auth tokens remain supported and take precedence when set.

## Server facts (discovered, not assumed)

`https://app.honeybadger.io/.well-known/oauth-authorization-server` (RFC 8414)
is live in production and advertises:

- `authorization_endpoint`, `token_endpoint`, `revocation_endpoint`,
  `registration_endpoint`
- `grant_types_supported`: `authorization_code`, `refresh_token`
- `code_challenge_methods_supported`: `S256` (PKCE)
- `token_endpoint_auth_methods_supported`: `none` (public clients)
- `scopes_supported`: `read`, `write`

Notably the **device grant (`urn:ietf:params:oauth:grant-type:device_code`) is
not advertised yet**. Meanwhile `api-go` v0.8.0 added `WithBearerToken`
("sets an OAuth access token, sent as Authorization: Bearer"), confirming the
Data API accepts OAuth bearer tokens.

## Approach

Discovery-driven login supporting **both** flows, selected at runtime:

1. **Authorization code + PKCE with loopback redirect** (RFC 8252 / RFC 7636)
   — the default. Works against production today. Binds `127.0.0.1:<ephemeral>`,
   opens the browser, exchanges the code with a PKCE verifier.
2. **Device authorization grant** (RFC 8628) — used with `--device` (for SSH /
   headless machines) or when the browser flow isn't possible. Activates
   automatically once the server advertises the grant; until then it fails with
   a clear "server does not support" error.

Client identity comes from **dynamic client registration** (RFC 7591) against
the advertised `registration_endpoint` — the same mechanism MCP clients use —
with the registered `client_id` cached locally. A pre-provisioned client id can
be supplied via `oauth_client_id` config / `HONEYBADGER_OAUTH_CLIENT_ID` and
skips registration.

Alternatives considered:

- *Device flow only* (as the branch name suggests): dead on arrival — production
  doesn't support the grant yet. Kept as a first-class flow behind discovery.
- *Hard-coded client id + fixed redirect port*: simpler but requires
  out-of-band provisioning and breaks when the port is taken. Dynamic
  registration is already an open, supported path on the server.
- *OS keychain storage*: better at-rest security but adds a cgo/keyring
  dependency and cross-platform variance; deferred. File is 0600, matching
  `gh`'s default behavior.

## Components

### `internal/oauth`

Pure OAuth client, no viper/cobra dependencies:

- `Discover(ctx, httpClient, issuer)` → `Metadata` (RFC 8414; falls back to
  conventional endpoint paths under the issuer if the metadata document 404s).
- `Register(ctx, httpClient, metadata, req)` → registered client (RFC 7591).
- `AuthCodeFlow`: PKCE S256 (43-char base64url verifier from 32 random bytes),
  random `state` (validated on callback), loopback HTTP server on
  `127.0.0.1:0`, browser opener injected as a func for testability, code
  exchange at the token endpoint. 5-minute timeout.
- `DeviceFlow`: POST to `device_authorization_endpoint`; displays
  `user_code` + `verification_uri` (and `verification_uri_complete` when
  present); polls the token endpoint honoring `interval`,
  `authorization_pending`, `slow_down` (+5s per RFC 8628 §3.5),
  `access_denied`, `expired_token`.
- `Refresh`: `refresh_token` grant with the public `client_id`; returns rotated
  tokens.
- `Revoke`: RFC 7009 revocation, best-effort.
- `Token{AccessToken, RefreshToken, TokenType, Scope, ExpiresAt}`.

### `internal/credentials`

JSON file store, default `~/.honeybadger-cli-credentials.json`, written with
`0600` (and `0700` parent creation). Keyed by issuer host so US
(`app.honeybadger.io`) and EU (`eu-app.honeybadger.io`) logins coexist. Each
entry: `client_id`, `redirect_uri`, `access_token`, `refresh_token`,
`token_type`, `scope`, `expires_at`. Path overridable via
`HONEYBADGER_CREDENTIALS_FILE` (used by tests, useful for users too).

### `cmd/auth.go`

- `hb auth login` — flags: `--device` (force device flow), `--scopes`
  (default `read write`). Discovers metadata from the Data API endpoint
  (after `convertEndpointForDataAPI`), registers/reuses a client, runs the
  flow, saves credentials, prints identity-free success message.
- `hb auth logout` — revokes access + refresh tokens (best-effort), deletes
  the stored entry for the current endpoint.
- `hb auth status` — reports login state, token expiry, scopes, and whether a
  personal auth token override is in effect. Offline; no network calls.

### Token resolution (`cmd/root.go`)

New helper `newDataAPIClient() (*hbapi.Client, error)` replaces the repeated
per-command block. Precedence:

1. `auth_token` from flag / env / config → `WithAuthToken` (basic auth), as
   today.
2. Stored OAuth credentials for the endpoint's issuer host →
   `WithBearerToken`. Tokens within 60s of expiry are refreshed transparently
   and persisted (including rotated refresh tokens). Refresh failure →
   "run `hb auth login`" error.
3. Neither → error retaining the phrase "auth token is required" (tests match
   it) and now also mentioning `hb auth login`.

All ~76 Data API call sites across 13 command files switch to the helper — a
mechanical replacement of the existing 11-line block.

## Error handling

- Discovery/registration/flow errors surface the OAuth `error` +
  `error_description` fields when present.
- Browser-flow callback errors (`access_denied`) are reported in the terminal;
  the browser tab gets a small self-contained HTML page for both success and
  failure.
- Credentials file corruption → treated as absent with a warning to stderr.

## Testing

- `internal/oauth`: httptest-backed unit tests. The auth-code flow is tested
  end-to-end by injecting a fake "browser" that parses the authorize URL and
  invokes the loopback callback; the fake token endpoint validates the PKCE
  verifier and state. Device flow tests cover pending → slow_down → success,
  denial, and expiry.
- `internal/credentials`: round-trip, permissions, multi-issuer.
- `cmd`: a package-level `TestMain` points `HONEYBADGER_CREDENTIALS_FILE` at a
  temp dir so existing tests never read a developer's real credentials; new
  tests cover login/logout/status and bearer-vs-basic selection in
  `newDataAPIClient`.

## Out of scope

- OS keychain storage.
- `hb auth login --with-token` style token stdin (personal tokens already
  cover this).
- Server-side device grant work (tracked separately; CLI lights up via
  discovery when it ships).
