package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RegistrationRequest is a dynamic client registration request (RFC 7591).
type RegistrationRequest struct {
	ClientName    string   `json:"client_name"`
	RedirectURIs  []string `json:"redirect_uris,omitempty"`
	GrantTypes    []string `json:"grant_types,omitempty"`
	ResponseTypes []string `json:"response_types,omitempty"`
	// TokenEndpointAuthMethod is always "none": the CLI is a public client
	// and cannot keep a client secret.
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method"`
	Scope                   string `json:"scope,omitempty"`
}

type registrationResponse struct {
	ClientID string `json:"client_id"`
	Error
}

// Register registers a new public client with the authorization server and
// returns the issued client_id.
func Register(
	ctx context.Context,
	hc *http.Client,
	registrationEndpoint string,
	reg RegistrationRequest,
) (string, error) {
	reg.TokenEndpointAuthMethod = "none"

	body, err := json.Marshal(reg)
	if err != nil {
		return "", fmt.Errorf("failed to encode registration request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, registrationEndpoint, bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req) // #nosec G704 -- URL comes from the server's OAuth metadata
	if err != nil {
		return "", fmt.Errorf("client registration failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read registration response: %w", err)
	}

	var rr registrationResponse
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if json.Unmarshal(respBody, &rr) == nil && rr.Code != "" {
			return "", fmt.Errorf("client registration failed: %w", &rr.Error)
		}
		return "", fmt.Errorf("client registration failed: HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(respBody, &rr); err != nil {
		return "", fmt.Errorf("failed to parse registration response: %w", err)
	}
	if rr.ClientID == "" {
		return "", fmt.Errorf("client registration succeeded but returned no client_id")
	}
	return rr.ClientID, nil
}
