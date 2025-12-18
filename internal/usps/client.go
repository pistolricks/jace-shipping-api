package usps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	uspssdk "github.com/my-eq/go-usps"
)

// PricesBaseURL is the USPS Prices API base URL
const PricesBaseURL = "https://apis.usps.com/prices/v3"

// PricesClient is a minimal client for interacting with the USPS Prices API.
// It uses the OAuth provider from github.com/my-eq/go-usps to obtain tokens.
type PricesClient struct {
	baseURL       string
	httpClient    *http.Client
	tokenProvider *uspssdk.OAuthTokenProvider
}

// NewPricesClient creates a new PricesClient using OAuth credentials.
// By default it points to the production Prices API base URL.
func NewPricesClient(clientID, clientSecret string, opts ...uspssdk.OAuthTokenOption) *PricesClient {
	// NOTE: We alias the import as usspsdk above but refer correctly here.
	// Fix alias typo by re-declaring opts type below.
	return &PricesClient{
		baseURL:       PricesBaseURL,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		tokenProvider: uspssdk.NewOAuthTokenProvider(clientID, clientSecret, opts...),
	}
}

// WithHTTPClient sets a custom HTTP client.
func (c *PricesClient) WithHTTPClient(httpClient *http.Client) *PricesClient {
	if httpClient != nil {
		c.httpClient = httpClient
	}
	return c
}

// WithBaseURL overrides the base URL (useful for testing environments).
func (c *PricesClient) WithBaseURL(baseURL string) *PricesClient {
	if baseURL != "" {
		c.baseURL = baseURL
	}
	return c
}

// DomesticBaseRatesRequest represents the JSON body for the Domestic Prices v3
// base-rates search endpoint. Field names follow the USPS documentation.
type DomesticBaseRatesRequest struct {
	OriginZIPCode                 string  `json:"originZIPCode"`
	DestinationZIPCode            string  `json:"destinationZIPCode"`
	Weight                        float64 `json:"weight"` // ounces
	Length                        float64 `json:"length"`
	Width                         float64 `json:"width"`
	Height                        float64 `json:"height"`
	MailClass                     string  `json:"mailClass"`
	ProcessingCategory            string  `json:"processingCategory"`
	RateIndicator                 string  `json:"rateIndicator"`
	DestinationEntryFacilityType  string  `json:"destinationEntryFacilityType"`
	PriceType                     string  `json:"priceType"`
	MailingDate                   string  `json:"mailingDate"`
	AccountType                   string  `json:"accountType"`
	AccountNumber                 string  `json:"accountNumber"`
	HasNonstandardCharacteristics bool    `json:"hasNonstandardCharacteristics"`
}

// Dims represents package dimensions.
type Dims struct {
	Length float64 `json:"length,omitempty"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
	Girth  float64 `json:"girth,omitempty"`
}

// Surcharge represents a surcharge entry on a rate.
type Surcharge struct {
	Type   string  `json:"type,omitempty"`
	Amount float64 `json:"amount,omitempty"`
}

// BaseRate represents a single base rate/product returned by the API.
type BaseRate struct {
	ServiceType   string      `json:"serviceType,omitempty"`
	MailClass     string      `json:"mailClass,omitempty"`
	Zone          string      `json:"zone,omitempty"`
	BaseAmount    float64     `json:"baseAmount,omitempty"`
	TotalAmount   float64     `json:"totalAmount,omitempty"`
	Currency      string      `json:"currency,omitempty"`
	DeliveryDays  string      `json:"deliveryDays,omitempty"`
	DeliveryDate  string      `json:"deliveryDate,omitempty"`
	Surcharges    []Surcharge `json:"surcharges,omitempty"`
	AdditionalRaw interface{} `json:"additionalRaw,omitempty"` // placeholder for unknown fields per product
}

type Rate struct {
	SKU         string        `json:"SKU"`
	Description string        `json:"description"`
	PriceType   string        `json:"priceType"`
	Price       float64       `json:"price"`
	Weight      int           `json:"weight"`
	DimWeight   int           `json:"dimWeight"`
	Fees        []interface{} `json:"fees"`
	StartDate   string        `json:"startDate"`
	EndDate     string        `json:"endDate"`
	Warnings    []struct {
		WarningCode        string `json:"warningCode"`
		WarningDescription string `json:"warningDescription"`
	} `json:"warnings"`
	MailClass                    string `json:"mailClass"`
	Zone                         string `json:"zone"`
	ProcessingCategory           string `json:"processingCategory"`
	DestinationEntryFacilityType string `json:"destinationEntryFacilityType"`
	RateIndicator                string `json:"rateIndicator"`
}

// DomesticBaseRatesResponse is a typed container for common fields. It also
// preserves the original payload in Raw for forward compatibility.
type DomesticBaseRatesResponse struct {
	TotalBasePrice float64         `json:"totalBasePrice,omitempty"`
	Rates          []Rate          `json:"rates,omitempty"`
	Products       []BaseRate      `json:"products,omitempty"`
	BaseRates      []BaseRate      `json:"baseRates,omitempty"`
	Meta           map[string]any  `json:"meta,omitempty"`
	Raw            json.RawMessage `json:"-"`
}

// SearchDomesticBaseRates calls the USPS Prices API base-rates search endpoint
// using POST and returns a typed response structure with the raw payload kept
// for maximal compatibility.
func (c *PricesClient) SearchDomesticBaseRates(ctx context.Context, body DomesticBaseRatesRequest) (*DomesticBaseRatesResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("PricesClient is nil")
	}

	endpoint := c.baseURL + "/base-rates/search"

	// Acquire token
	token, err := c.tokenProvider.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth token: %w", err)
	}

	// Marshal body
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Prepare request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Execute
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read body bytes so we can both unmarshal and preserve Raw
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try decoding structured error first
		var errBody map[string]any
		_ = json.Unmarshal(data, &errBody)
		return nil, fmt.Errorf("USPS Prices API error: status=%d, body=%v", resp.StatusCode, errBody)
	}

	fmt.Println(string(data))
	out := &DomesticBaseRatesResponse{Raw: data}
	// Best-effort typed decode; unknown fields will be ignored
	_ = json.Unmarshal(data, out)
	return out, nil
}
