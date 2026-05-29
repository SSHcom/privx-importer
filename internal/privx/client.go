package privx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/SSHcom/inventory_import_privx_dev/internal/config"
	"github.com/SSHcom/inventory_import_privx_dev/internal/output"
)

var (
	ErrUnauthorized          = errors.New("privx unauthorized")
	ErrForbidden             = errors.New("privx forbidden")
	ErrNotFound              = errors.New("privx not found")
	ErrBadRequest            = errors.New("privx bad request")
	ErrServerError           = errors.New("privx server error")
	ErrPrivXAccessTokenEmpty = errors.New("privx access token empty")
	ErrNoCredentials         = errors.New("no JWT or OAuth credentials")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	debugger   output.DebugOutput
}

// genericRawJSONResponse is used to receive JSON from PrivX API.
type genericRawJSONResponse struct {
	Count int              `json:"count"`
	Items []map[string]any `json:"items"`
}

// New establishes connection with PrivX with token exchange.
func New(
	ctx context.Context,
	cfg config.Config,
	out output.Output,
) (*Client, error) {
	var token string
	// Client without token can still do the HTTP calls needed for token exchange.
	exchangeClient := newWithToken(cfg.PrivX.BaseURL, nil, "", out)

	if cfg.PrivX.JWT != "" {
		out.Printf(output.VerbosityInfo, "Authenticating with JWT\n")
		resp, err := exchangeClient.ExchangeToken(ctx, ExchangeTokenRequest{
			Token: cfg.PrivX.JWT,
		})
		if err != nil {
			return nil, err
		}
		if resp.AccessToken == "" {
			return nil, ErrPrivXAccessTokenEmpty
		}
		token = resp.AccessToken
		if cfg.PrivX.ClientID != "" || cfg.PrivX.ClientSecret != "" {
			out.Printf(output.VerbosityNormal, "Superfluous OAuth credentials ignored\n")
		}
	} else if cfg.PrivX.ClientID != "" && cfg.PrivX.ClientSecret != "" {
		if cfg.PrivX.OAuthClientID == "" || cfg.PrivX.OAuthClientSecret == "" {
			return nil, fmt.Errorf(
				"%w: PRIVX_CLIENT_ID and PRIVX_CLIENT_SECRET are set but PRIVX_OAUTH_CLIENT_ID and PRIVX_OAUTH_CLIENT_SECRET are also required for OAuth authentication",
				ErrNoCredentials,
			)
		}

		out.Printf(output.VerbosityInfo, "Authenticating with OAuth (grant_type=password)\n")
		resp, err := exchangeClient.OAuthToken(ctx, OAuthTokenRequest{
			APIClientID:       cfg.PrivX.ClientID,
			APIClientSecret:   cfg.PrivX.ClientSecret,
			OAuthClientID:     cfg.PrivX.OAuthClientID,
			OAuthClientSecret: cfg.PrivX.OAuthClientSecret,
		})
		if err != nil {
			return nil, err
		}
		if resp.AccessToken == "" {
			return nil, ErrPrivXAccessTokenEmpty
		}
		token = resp.AccessToken
	} else {
		return nil, ErrNoCredentials
	}

	// Now create a client with proper token that can call the actual API endpoints.
	return newWithToken(cfg.PrivX.BaseURL, nil, token, out), nil
}

func newWithToken(
	baseURL string,
	httpClient *http.Client,
	token string,
	out output.Output,
) *Client {
	if httpClient == nil {
		tr := &http.Transport{
			TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}
		httpClient = &http.Client{
			Timeout:   30 * time.Second,
			Transport: tr,
		}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		token:      token,
		debugger:   out,
	}
}

func indentLines(s string, prefix string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// doJSON implements the API message exchange, sending JSON payload and receiving one.
func (c *Client) doJSON(
	ctx context.Context,
	method string,
	path string,
	reqBody any,
	respBody any,
) error {
	var bodyReader io.Reader
	var reqJSON []byte
	var err error

	if reqBody != nil {
		reqJSON, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(reqJSON)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	const MaxResponseBytes = 32 << 20 // 32 MiB - generous for largest realistic PrivX response
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to produce as informative message as possible.s
		apiErr := NewAPIError(
			method,
			path,
			reqJSON,
			resp.StatusCode,
			data,
		)

		if strings.TrimSpace(apiErr.Message) == "" {
			body := strings.TrimSpace(string(data))
			if body != "" && !strings.HasPrefix(body, "<") {
				apiErr.Message = body
			}
		}
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return errors.Join(ErrBadRequest, apiErr)
		case http.StatusUnauthorized:
			return errors.Join(ErrUnauthorized, apiErr)
		case http.StatusForbidden:
			return errors.Join(ErrForbidden, apiErr)
		case http.StatusNotFound:
			return errors.Join(ErrNotFound, apiErr)
		default:
			if resp.StatusCode >= 500 {
				return errors.Join(ErrServerError, apiErr)
			}
			return apiErr
		}
	}

	if respBody == nil || len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, respBody); err != nil {
		c.debugger.Debugf("privx debug decode error (%s %s): %v\n", method, path, err)
		c.debugger.Debugf("raw body:\n%s\n", string(data))
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

// tryDecodeAPIError will produce a detailed printout of the error.
func tryDecodeAPIError(data []byte, apiErr *APIError) error {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	var lines []string

	for _, key := range []string{"error_description", "message", "error", "detail"} {
		val, ok := obj[key].(string)
		if ok {
			v := strings.TrimSpace(val)
			if v != "" {
				lines = append(lines, key+": "+v)
			}
		}
	}

	if len(lines) > 0 {
		apiErr.Message = strings.Join(lines, "\n")
	}

	return nil
}
