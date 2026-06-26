package suppliers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// HLRSupplierInterface describes a configurable HLR/MNP provider client.
type HLRSupplierInterface interface {
	Lookup(baseURL, apiKey, e164 string) (*HLRLookupResponse, error)
}

// HLRLookupResponse models the common fields returned by HLR providers
// (Abstract, IPQS HLR, apilayer, etc.). Adjust the json tags to match the
// provider configured via HLR_API_URL.
type HLRLookupResponse struct {
	Status         string `json:"status"`
	Active         *bool  `json:"active"`
	CurrentCarrier string `json:"current_carrier"`
	Ported         *bool  `json:"ported"`
	Roaming        *bool  `json:"roaming"`
	LineType       string `json:"line_type"`
	MCC            string `json:"mcc"`
	MNC            string `json:"mnc"`
	Country        string `json:"country"`
	Message        string `json:"message"`
}

// HLRSupplier is stateless; the endpoint is supplied per call so it can be
// configured through scanner options.
type HLRSupplier struct{}

func NewHLRSupplier() *HLRSupplier {
	return &HLRSupplier{}
}

func (s *HLRSupplier) Lookup(baseURL, apiKey, e164 string) (*HLRLookupResponse, error) {
	logrus.WithField("number", e164).Debug("Running HLR lookup")

	values := url.Values{}
	values.Set("api_key", apiKey)
	values.Set("number", e164)
	endpoint := fmt.Sprintf("%s?%s", baseURL, values.Encode())

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HLR provider returned HTTP %d", resp.StatusCode)
	}

	var result HLRLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
