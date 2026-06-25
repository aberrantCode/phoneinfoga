package suppliers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// TwilioSupplierInterface describes the Twilio Lookup v2 client used by the scanner.
type TwilioSupplierInterface interface {
	Lookup(accountSID, authToken, e164 string, fields []string) (*TwilioLookupResponse, error)
}

// TwilioLineTypeIntelligence is the line_type_intelligence data package.
type TwilioLineTypeIntelligence struct {
	Type              string `json:"type"`
	CarrierName       string `json:"carrier_name"`
	MobileCountryCode string `json:"mobile_country_code"`
	MobileNetworkCode string `json:"mobile_network_code"`
	ErrorCode         *int   `json:"error_code"`
}

// TwilioCallerName is the caller_name (CNAM) data package.
type TwilioCallerName struct {
	CallerName string `json:"caller_name"`
	CallerType string `json:"caller_type"`
	ErrorCode  *int   `json:"error_code"`
}

// TwilioLastSimSwap holds the most recent SIM swap details.
type TwilioLastSimSwap struct {
	LastSimSwapDate string `json:"last_sim_swap_date"`
	SwappedPeriod   string `json:"swapped_period"`
	SwappedInPeriod bool   `json:"swapped_in_period"`
}

// TwilioSimSwap is the sim_swap data package.
type TwilioSimSwap struct {
	LastSimSwap       TwilioLastSimSwap `json:"last_sim_swap"`
	CarrierName       string            `json:"carrier_name"`
	MobileCountryCode string            `json:"mobile_country_code"`
	MobileNetworkCode string            `json:"mobile_network_code"`
	ErrorCode         *int              `json:"error_code"`
}

// TwilioCallForwarding is the call_forwarding data package.
type TwilioCallForwarding struct {
	CallForwardingStatus string `json:"call_forwarding_status"`
	CarrierName          string `json:"carrier_name"`
	ErrorCode            *int   `json:"error_code"`
}

// TwilioLookupResponse is the Twilio Lookup v2 REST API response.
type TwilioLookupResponse struct {
	CallingCountryCode   string                      `json:"calling_country_code"`
	CountryCode          string                      `json:"country_code"`
	PhoneNumber          string                      `json:"phone_number"`
	NationalFormat       string                      `json:"national_format"`
	Valid                bool                        `json:"valid"`
	ValidationErrors     []string                    `json:"validation_errors"`
	CallerName           *TwilioCallerName           `json:"caller_name"`
	SimSwap              *TwilioSimSwap              `json:"sim_swap"`
	CallForwarding       *TwilioCallForwarding       `json:"call_forwarding"`
	LineTypeIntelligence *TwilioLineTypeIntelligence `json:"line_type_intelligence"`
	URL                  string                      `json:"url"`
}

// TwilioErrorResponse is the error body Twilio returns for failed requests.
type TwilioErrorResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}

type TwilioSupplier struct {
	BaseURL string
}

const twilioDefaultBaseURL = "https://lookups.twilio.com"

func NewTwilioSupplier() *TwilioSupplier {
	return &TwilioSupplier{BaseURL: twilioDefaultBaseURL}
}

func (s *TwilioSupplier) Lookup(accountSID, authToken, e164 string, fields []string) (*TwilioLookupResponse, error) {
	logrus.
		WithField("number", e164).
		WithField("fields", strings.Join(fields, ",")).
		Debug("Running lookup operation through Twilio Lookup v2 API")

	endpoint := fmt.Sprintf("%s/v2/PhoneNumbers/%s", strings.TrimRight(s.BaseURL, "/"), url.PathEscape(e164))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if len(fields) > 0 {
		q := req.URL.Query()
		q.Set("Fields", strings.Join(fields, ","))
		req.URL.RawQuery = q.Encode()
	}

	req.SetBasicAuth(accountSID, authToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		var errResponse TwilioErrorResponse
		if err := json.NewDecoder(response.Body).Decode(&errResponse); err == nil && errResponse.Message != "" {
			return nil, fmt.Errorf("twilio lookup failed (HTTP %d): %s", response.StatusCode, errResponse.Message)
		}
		return nil, fmt.Errorf("twilio lookup returned HTTP %d", response.StatusCode)
	}

	var result TwilioLookupResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
