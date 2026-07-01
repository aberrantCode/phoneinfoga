package suppliers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

// dehashedDefaultBaseURL targets the Dehashed v2 search endpoint. v2 replaced
// the retired v1 "GET /search" endpoint (which used HTTP Basic email:api_key
// auth) — v1 now returns HTTP 404. v2 takes a JSON POST body and authenticates
// with the "DeHashed-Api-Key" header. DehashedEntry fields are modeled as
// json.RawMessage so v2's array-shaped values decode without change.
const dehashedDefaultBaseURL = "https://api.dehashed.com/v2/search"

// dehashedDefaultPageSize bounds a single lookup. v2 accepts 1..10000; a phone
// number rarely maps to many records, so a modest first page is sufficient.
const dehashedDefaultPageSize = 100

// dehashedSearchRequest is the v2 POST body. The boolean toggles are sent
// explicitly (all false) to match the documented request shape.
type dehashedSearchRequest struct {
	Query    string `json:"query"`
	Page     int    `json:"page"`
	Size     int    `json:"size"`
	Wildcard bool   `json:"wildcard"`
	Regex    bool   `json:"regex"`
	DeDupe   bool   `json:"de_dupe"`
}

func NewDehashedSupplier() *DehashedSupplier {
	return &DehashedSupplier{BaseURL: dehashedDefaultBaseURL}
}

func (s *DehashedSupplier) SearchByPhone(email, apiKey, query string) (*DehashedResponse, error) {
	// v2 authenticates with the DeHashed-Api-Key header alone. email is retained
	// in the signature (and the scanner's credential gate) for backward-
	// compatible configuration but is no longer part of the request.
	_ = email

	// Intentionally logs only the query, never the credentials or any returned
	// records.
	logrus.
		WithField("query", query).
		Debug("Running breach lookup through Dehashed v2 API")

	payload, err := json.Marshal(dehashedSearchRequest{
		Query: query,
		Page:  1,
		Size:  dehashedDefaultPageSize,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, s.BaseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("DeHashed-Api-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("dehashed authentication failed (HTTP 401): check DEHASHED_API_KEY")
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
