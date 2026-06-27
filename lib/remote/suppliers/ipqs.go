package suppliers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// IPQSSupplierInterface describes the IPQualityScore phone validation client.
type IPQSSupplierInterface interface {
	Validate(apiKey, phone string) (*IPQSValidateResponse, error)
}

// IPQSValidateResponse is the IPQualityScore phone API response.
type IPQSValidateResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Valid       bool   `json:"valid"`
	Active      bool   `json:"active"`
	FraudScore  int    `json:"fraud_score"`
	RecentAbuse bool   `json:"recent_abuse"`
	VOIP        bool   `json:"VOIP"`
	Prepaid     bool   `json:"prepaid"`
	Risky       bool   `json:"risky"`
	DoNotCall   bool   `json:"do_not_call"`
	Leaked      bool   `json:"leaked"`
	Spammer     bool   `json:"spammer"`
	LineType    string `json:"line_type"`
	Carrier     string `json:"carrier"`
	Country     string `json:"country"`
	Region      string `json:"region"`
	City        string `json:"city"`
	Timezone    string `json:"timezone"`
}

type IPQSSupplier struct {
	BaseURL string
}

const ipqsDefaultBaseURL = "https://www.ipqualityscore.com/api/json/phone"

func NewIPQSSupplier() *IPQSSupplier {
	return &IPQSSupplier{BaseURL: ipqsDefaultBaseURL}
}

func (s *IPQSSupplier) Validate(apiKey, phone string) (*IPQSValidateResponse, error) {
	logrus.WithField("number", phone).Debug("Running reputation lookup through IPQualityScore API")

	endpoint := fmt.Sprintf("%s/%s/%s", s.BaseURL, url.PathEscape(apiKey), url.PathEscape(phone))

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
		return nil, fmt.Errorf("IPQualityScore returned HTTP %d", resp.StatusCode)
	}

	var result IPQSValidateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Success && result.Message != "" {
		return nil, fmt.Errorf("IPQualityScore request failed: %s", result.Message)
	}

	return &result, nil
}
