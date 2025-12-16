package usps

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/my-eq/go-usps"
	"github.com/my-eq/go-usps/models"
)

const (
	// PricesProductionBaseURL is the base URL for the USPS Prices production API
	PricesProductionBaseURL = "https://apis.usps.com/prices/v3"
	// PricesTestingBaseURL is the base URL for the USPS Prices testing API
	PricesTestingBaseURL = "https://apis-tem.usps.com/prices/v3"
)

// PricesClient is the USPS Prices API client for managing Prices 2.0 tokens.
// It supports Client Credentials, Refresh Token, and Authorization Code grant types.
type PricesClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPricesClient creates a new USPS Prices API client configured for the production environment.
// Use functional options to customize the client configuration.
//
// Example:
//
//	client := usps.NewPricesClient()
//	client := usps.NewPricesClient(usps.WithTimeout(60 * time.Second))
func NewPricesClient(opts ...usps.Option) *PricesClient {
	c := &PricesClient{
		baseURL:    PricesProductionBaseURL,
		httpClient: &http.Client{Timeout: usps.DefaultTimeout},
	}
	/*
		// Apply options using a temporary regular client
		tempClient := &usps.Client{
			baseURL:    c.baseURL,
			httpClient: c.httpClient,
		}
		for _, opt := range opts {
			opt(tempClient)
		}
		c.baseURL = tempClient.baseURL
		c.httpClient = tempClient.httpClient
	*/
	return c
}

// NewPricesTestClient creates a new USPS Prices API client configured for the testing environment.
// This is equivalent to calling NewPricesClient with WithBaseURL(PricesTestingBaseURL).
//
// Example:
//
//	client := usps.NewPricesTestClient()
func NewPricesTestClient(opts ...usps.Option) *PricesClient {
	opts = append([]usps.Option{usps.WithBaseURL(PricesTestingBaseURL)}, opts...)
	return NewPricesClient(opts...)
}

// PostToken generates Prices tokens based on the grant type.
// It supports three grant types:
//   - Client Credentials: Pass *models.ClientCredentials to get an access token
//   - Refresh Token: Pass *models.RefreshTokenCredentials to refresh an access token
//   - Authorization Code: Pass *models.AuthorizationCodeCredentials to exchange an auth code
//
// The method returns either *models.ProviderAccessTokenResponse (for client credentials)
// or *models.ProviderTokensResponse (for grants that include a refresh token).
//
// Access tokens are valid for 8 hours. Refresh tokens are valid for 7 days.
//
// Example (Client Credentials):
//
//	req := &models.ClientCredentials{
//	    GrantType:    "client_credentials",
//	    ClientID:     "your-client-id",
//	    ClientSecret: "your-client-secret",
//	    Scope:        "addresses tracking",
//	}
//	result, err := client.PostToken(ctx, req)
//	if err != nil {
//	    return err
//	}
//	accessTokenResp := result.(*models.ProviderAccessTokenResponse)
//
// Example (Refresh Token):
//
//	req := &models.RefreshTokenCredentials{
//	    GrantType:    "refresh_token",
//	    ClientID:     "your-client-id",
//	    ClientSecret: "your-client-secret",
//	    RefreshToken: "your-refresh-token",
//	}
//	result, err := client.PostToken(ctx, req)
//	if err != nil {
//	    return err
//	}
//	tokensResp := result.(*models.ProviderTokensResponse)
func (c *PricesClient) PostToken(ctx context.Context, req interface{}) (interface{}, error) {
	var contentType string
	var body io.Reader

	// Determine content type and encode body
	switch r := req.(type) {
	case *models.ClientCredentials:
		contentType = "application/x-www-form-urlencoded"
		values := url.Values{}
		values.Set("grant_type", r.GrantType)
		values.Set("client_id", r.ClientID)
		values.Set("client_secret", r.ClientSecret)
		if r.Scope != "" {
			values.Set("scope", r.Scope)
		}
		body = strings.NewReader(values.Encode())
	case *models.RefreshTokenCredentials:
		contentType = "application/json"
		jsonData, err := json.Marshal(r)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	case *models.AuthorizationCodeCredentials:
		contentType = "application/json"
		jsonData, err := json.Marshal(r)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	default:
		return nil, fmt.Errorf("unsupported request type")
	}

	// Create request
	fullURL := c.baseURL + "/token"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errResp models.StandardErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("Prices error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, &PricesError{
			StatusCode:   resp.StatusCode,
			ErrorMessage: errResp,
		}
	}

	// Try to unmarshal as ProviderTokensResponse first (has refresh_token)
	var tokensResp models.ProviderTokensResponse
	if err := json.Unmarshal(respBody, &tokensResp); err == nil && tokensResp.RefreshToken != "" {
		return &tokensResp, nil
	}

	// Otherwise unmarshal as ProviderAccessTokenResponse
	var accessTokenResp models.ProviderAccessTokenResponse
	if err := json.Unmarshal(respBody, &accessTokenResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &accessTokenResp, nil
}

// PostRevoke revokes an Prices token using HTTP Basic Authentication.
// This method is used to invalidate refresh tokens that are no longer needed
// or suspected of being compromised.
//
// The clientID and clientSecret are used for Basic Authentication as required
// by the USPS Prices API. The request specifies which token to revoke and
// optionally provides a hint about the token type.
//
// Example:
//
//	req := &models.TokenRevokeRequest{
//	    Token:         "refresh-token-to-revoke",
//	    TokenTypeHint: "refresh_token",
//	}
//	err := client.PostRevoke(ctx, "client-id", "client-secret", req)
func (c *PricesClient) PostRevoke(ctx context.Context, clientID, clientSecret string, req *models.TokenRevokeRequest) error {
	// Encode request body
	values := url.Values{}
	values.Set("token", req.Token)
	if req.TokenTypeHint != "" {
		values.Set("token_type_hint", req.TokenTypeHint)
	}

	// Create request
	fullURL := c.baseURL + "/revoke"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Basic Authentication
	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle error responses
	if resp.StatusCode >= 400 {
		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Prices error (status %d): failed to read response", resp.StatusCode)
		}

		var errResp models.StandardErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return fmt.Errorf("Prices error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return &PricesError{
			StatusCode:   resp.StatusCode,
			ErrorMessage: errResp,
		}
	}

	return nil
}

// PricesError represents an error returned by the USPS Prices API
type PricesError struct {
	StatusCode   int
	ErrorMessage models.StandardErrorResponse
}

// Error implements the error interface
func (e *PricesError) Error() string {
	if e.ErrorMessage.Error != "" {
		msg := e.ErrorMessage.Error
		if e.ErrorMessage.ErrorDescription != "" {
			msg += ": " + e.ErrorMessage.ErrorDescription
		}
		return fmt.Sprintf("Prices error (status %d): %s", e.StatusCode, msg)
	}
	return fmt.Sprintf("Prices error (status %d)", e.StatusCode)
}
