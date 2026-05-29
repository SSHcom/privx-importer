package privx

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"encoding/json"
)

var (
	// ErrTokenEmpty is returned when the access token is empty.
	ErrTokenEmpty = errors.New("token empty")
	// ErrOAuthDomainEmpty is returned when OAuth domain is empty.
	ErrOAuthDomainEmpty = errors.New("oauth domain empty")

	// ErrOAuthClientIDEmpty is returned when OAuth client ID is empty.
	ErrOAuthClientIDEmpty = errors.New("oauth client id empty")

	// ErrOAuthClientSecretEmpty is returned when OAuth client secret is empty.
	ErrOAuthClientSecretEmpty = errors.New("oauth client secret empty")
)

// ExchangeTokenRequest matches PrivX token exchange request fields.
type ExchangeTokenRequest struct {
	// Token to exchange to a PrivX access token.
	Token string `json:"token"`
	// Access token scope.
	Scope string `json:"scope,omitempty"`
	// OAuth client ID.
	ClientID string `json:"client_id,omitempty"`
}

// TokenResponse is a common OAuth-style token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ExchangeToken exchanges an external JWT to a PrivX access token.
func (c *Client) ExchangeToken(
	ctx context.Context,
	reqBody ExchangeTokenRequest,
) (TokenResponse, error) {
	var out TokenResponse

	if strings.TrimSpace(reqBody.Token) == "" {
		return out, fmt.Errorf("%w: exchange token request token is empty", ErrTokenEmpty)
	}

	err := c.doJSON(
		ctx,
		http.MethodPost,
		pathAuthTokenLogin,
		reqBody,
		&out,
	)
	if err != nil {
		return out, fmt.Errorf("exchange privx token: %w", err)
	}

	if strings.TrimSpace(out.AccessToken) == "" {
		return out, fmt.Errorf("%w: exchange response access_token is empty", ErrTokenEmpty)
	}

	return out, nil
}

// ClientCredentialsRequest matches PrivX OAuth2 client credentials request fields.
type ClientCredentialsRequest struct {
	// GrantType is the OAuth2 grant type.
	GrantType string `json:"grant_type"`

	// ClientID is the OAuth client ID.
	ClientID string `json:"client_id"`

	// ClientSecret is the OAuth client secret.
	ClientSecret string `json:"client_secret"`
}

// ClientCredentialsToken gets a PrivX access token using OAuth2 client credentials.
// This sends a JSON body to the /oauth/clientcredentials/{domain}/token endpoint.
func (c *Client) ClientCredentialsToken(
	ctx context.Context,
	reqBody ClientCredentialsRequest,
) (TokenResponse, error) {
	var out TokenResponse

	if strings.TrimSpace(reqBody.ClientID) == "" {
		return out, ErrOAuthClientIDEmpty
	}

	if strings.TrimSpace(reqBody.ClientSecret) == "" {
		return out, ErrOAuthClientSecretEmpty
	}

	if strings.TrimSpace(reqBody.GrantType) == "" {
		reqBody.GrantType = "client_credentials"
	}

	path := pathOAuthToken

	err := c.doJSON(
		ctx,
		http.MethodPost,
		path,
		reqBody,
		&out,
	)
	if err != nil {
		return out, fmt.Errorf("get privx oauth client credentials token: %w", err)
	}

	if strings.TrimSpace(out.AccessToken) == "" {
		return out, fmt.Errorf("%w: oauth response access_token is empty", ErrTokenEmpty)
	}

	return out, nil
}

// OAuthTokenRequest contains the parameters for the standard OAuth2 token endpoint
// that uses HTTP Basic Auth and form-encoded body (matching the Terraform provider pattern).
type OAuthTokenRequest struct {
	// APIClientID is the API client ID sent as "username" in the form body.
	APIClientID string

	// APIClientSecret is the API client secret sent as "password" in the form body.
	APIClientSecret string

	// OAuthClientID is used for HTTP Basic Auth username.
	OAuthClientID string

	// OAuthClientSecret is used for HTTP Basic Auth password.
	OAuthClientSecret string
}

// OAuthToken gets a PrivX access token using the OAuth2 resource owner password
// grant with HTTP Basic Auth and form-encoded body.
//
// This matches the flow documented by PrivX and used by the Terraform provider:
//   - OAuth client credentials (oauth_client_id, oauth_client_secret) go in the
//     HTTP Basic Auth header
//   - API client credentials go as username/password in the form body
//   - grant_type is "password"
//
// See: https://privx.docs.ssh.com/docs/advanced-configuration/api-client-integration/
func (c *Client) OAuthToken(
	ctx context.Context,
	req OAuthTokenRequest,
) (TokenResponse, error) {
	var out TokenResponse

	if strings.TrimSpace(req.OAuthClientID) == "" {
		return out, ErrOAuthClientIDEmpty
	}

	if strings.TrimSpace(req.OAuthClientSecret) == "" {
		return out, ErrOAuthClientSecretEmpty
	}

	if strings.TrimSpace(req.APIClientID) == "" {
		return out, fmt.Errorf("%w: api client_id is empty", ErrOAuthClientIDEmpty)
	}

	if strings.TrimSpace(req.APIClientSecret) == "" {
		return out, fmt.Errorf("%w: api client_secret is empty", ErrOAuthClientSecretEmpty)
	}

	// Build form body per PrivX docs:
	//   grant_type=password&username=<api_client_id>&password=<api_client_secret>
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", req.APIClientID)
	form.Set("password", req.APIClientSecret)

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+pathOAuthToken,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return out, fmt.Errorf("build oauth request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	// HTTP Basic Auth with OAuth client credentials.
	if strings.ContainsRune(req.OAuthClientID, ':') {
		return out, fmt.Errorf("oauth client_id must not contain ':' (RFC 7617)")
	}
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(
		[]byte(req.OAuthClientID+":"+req.OAuthClientSecret),
	))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return out, fmt.Errorf("oauth token request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return out, fmt.Errorf("read oauth response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := NewAPIError(http.MethodPost, pathOAuthToken, nil, resp.StatusCode, data)
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return out, errors.Join(ErrBadRequest, apiErr)
		case http.StatusUnauthorized:
			return out, errors.Join(ErrUnauthorized, apiErr)
		case http.StatusForbidden:
			return out, errors.Join(ErrForbidden, apiErr)
		default:
			if resp.StatusCode >= 500 {
				return out, errors.Join(ErrServerError, apiErr)
			}
			return out, apiErr
		}
	}

	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("decode oauth response: %w", err)
	}

	if strings.TrimSpace(out.AccessToken) == "" {
		return out, fmt.Errorf("%w: oauth response access_token is empty", ErrTokenEmpty)
	}

	return out, nil
}
