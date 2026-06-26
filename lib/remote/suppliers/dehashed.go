package suppliers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// DehashedSupplierInterface describes the breach-data provider client used by the scanner.
type DehashedSupplierInterface interface {
	SearchByPhone(email, apiKey, query string) (*DehashedResponse, error)
}

// DehashedEntry is a single record returned by the Dehashed search API.
//
// The PII fields are kept as json.RawMessage so the scanner can detect their
// presence (and, only when explicitly opted in, render them) without
// committing to the scalar-string (v1) versus array (v2) shape the API may
// return for a given field.
type DehashedEntry struct {
	ID             string          `json:"id"`
	DatabaseName   string          `json:"database_name"`
	Email          json.RawMessage `json:"email"`
	Username       json.RawMessage `json:"username"`
	Password       json.RawMessage `json:"password"`
	HashedPassword json.RawMessage `json:"hashed_password"`
	Name           json.RawMessage `json:"name"`
	Address        json.RawMessage `json:"address"`
	IPAddress      json.RawMessage `json:"ip_address"`
	Phone          json.RawMessage `json:"phone"`
	VIN            json.RawMessage `json:"vin"`
}

// DehashedResponse is the Dehashed search API response envelope.
type DehashedResponse struct {
	Success *bool           `json:"success"`
	Total   int             `json:"total"`
	Balance int             `json:"balance"`
	Entries []DehashedEntry `json:"entries"`
	Message string          `json:"message"`
}

type DehashedSupplier struct {
	BaseURL string
}

// dehashedDefaultBaseURL targets the documented v1 search endpoint, which uses
// HTTP Basic auth (email:api_key). Dehashed also offers a v2 API
// (POST https://api.dehashed.com/v2/search with a "Dehashed-Api-Key" header).
// If your account uses v2, override DEHASHED_API_URL and adjust the request
// construction in SearchByPhone accordingly — the response envelope is modeled
// to tolerate either version's field shapes.
const dehashedDefaultBaseURL = "https://api.dehashed.com/search"

func NewDehashedSupplier() *DehashedSupplier {
	return &DehashedSupplier{BaseURL: dehashedDefaultBaseURL}
}

func (s *DehashedSupplier) SearchByPhone(email, apiKey, query string) (*DehashedResponse, error) {
	// Intentionally logs only the query, never the credentials or any returned
	// records.
	logrus.
		WithField("query", query).
		Debug("Running breach lookup through Dehashed API")

	endpoint := fmt.Sprintf("%s?query=%s", s.BaseURL, url.QueryEscape(query))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(email, apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("dehashed authentication failed (HTTP 401): check DEHASHED_EMAIL and DEHASHED_API_KEY")
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("dehashed rate limit exceeded (HTTP 429)")
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("dehashed search returned HTTP %d", response.StatusCode)
	}

	var result DehashedResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Dehashed reports an explicit success:false on auth/quota failures while
	// returning HTTP 200. Treat that as an error; an absent field or a
	// legitimately empty result set is not a failure.
	if result.Success != nil && !*result.Success {
		if result.Message != "" {
			return nil, fmt.Errorf("dehashed search unsuccessful: %s", result.Message)
		}
		return nil, fmt.Errorf("dehashed search unsuccessful (authentication or quota failure)")
	}

	return &result, nil
}
