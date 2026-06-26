package suppliers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ValidationProvider is the shared interface for Numverify-alternate validators.
// Each implementation declares its own key environment variable so a single
// generic scanner can wrap any provider.
type ValidationProvider interface {
	Name() string
	KeyEnvVar() string
	Validate(apiKey, e164 string) (*ValidationResult, error)
}

// ValidationResult is the normalized output shared by all validation providers.
type ValidationResult struct {
	Valid         bool
	Number        string
	LocalFormat   string
	IntlFormat    string
	CountryName   string
	CountryPrefix string
	Location      string
	Carrier       string
	LineType      string
}

// fetchJSON performs a GET request and decodes a JSON body. Optional headers
// let providers that authenticate via a request header (rather than a query
// parameter) pass their key without placing it in the URL. It deliberately
// does not log the endpoint, which for some providers carries the API key as a
// query parameter.
func fetchJSON(endpoint string, headers map[string]string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("validation provider returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// --- Veriphone ---

type VeriphoneProvider struct {
	BaseURL string
}

func NewVeriphoneProvider() *VeriphoneProvider {
	return &VeriphoneProvider{BaseURL: "https://api.veriphone.io"}
}

func (p *VeriphoneProvider) Name() string      { return "veriphone" }
func (p *VeriphoneProvider) KeyEnvVar() string { return "VERIPHONE_API_KEY" }

type veriphoneResponse struct {
	Status              string `json:"status"`
	Phone               string `json:"phone"`
	PhoneValid          bool   `json:"phone_valid"`
	PhoneType           string `json:"phone_type"`
	PhoneRegion         string `json:"phone_region"`
	Country             string `json:"country"`
	CountryCode         string `json:"country_code"`
	CountryPrefix       string `json:"country_prefix"`
	InternationalNumber string `json:"international_number"`
	LocalNumber         string `json:"local_number"`
	E164                string `json:"e164"`
	Carrier             string `json:"carrier"`
}

func (p *VeriphoneProvider) Validate(apiKey, e164 string) (*ValidationResult, error) {
	values := url.Values{}
	values.Set("phone", e164)
	values.Set("key", apiKey)
	endpoint := fmt.Sprintf("%s/v2/verify?%s", p.BaseURL, values.Encode())

	var body veriphoneResponse
	if err := fetchJSON(endpoint, nil, &body); err != nil {
		return nil, err
	}
	return &ValidationResult{
		Valid:         body.PhoneValid,
		Number:        body.E164,
		LocalFormat:   body.LocalNumber,
		IntlFormat:    body.InternationalNumber,
		CountryName:   body.Country,
		CountryPrefix: body.CountryPrefix,
		Location:      body.PhoneRegion,
		Carrier:       body.Carrier,
		LineType:      body.PhoneType,
	}, nil
}

// --- Abstract ---

type AbstractProvider struct {
	BaseURL string
}

func NewAbstractProvider() *AbstractProvider {
	return &AbstractProvider{BaseURL: "https://phonevalidation.abstractapi.com"}
}

func (p *AbstractProvider) Name() string      { return "abstract" }
func (p *AbstractProvider) KeyEnvVar() string { return "ABSTRACT_PHONE_API_KEY" }

type abstractResponse struct {
	Phone    string `json:"phone"`
	Valid    bool   `json:"valid"`
	Type     string `json:"type"`
	Carrier  string `json:"carrier"`
	Location string `json:"location"`
	Country  struct {
		Code   string `json:"code"`
		Name   string `json:"name"`
		Prefix string `json:"prefix"`
	} `json:"country"`
	Format struct {
		International string `json:"international"`
		Local         string `json:"local"`
	} `json:"format"`
}

func (p *AbstractProvider) Validate(apiKey, e164 string) (*ValidationResult, error) {
	values := url.Values{}
	values.Set("api_key", apiKey)
	values.Set("phone", e164)
	endpoint := fmt.Sprintf("%s/v1/?%s", p.BaseURL, values.Encode())

	var body abstractResponse
	if err := fetchJSON(endpoint, nil, &body); err != nil {
		return nil, err
	}
	return &ValidationResult{
		Valid:         body.Valid,
		Number:        body.Phone,
		LocalFormat:   body.Format.Local,
		IntlFormat:    body.Format.International,
		CountryName:   body.Country.Name,
		CountryPrefix: body.Country.Prefix,
		Location:      body.Location,
		Carrier:       body.Carrier,
		LineType:      body.Type,
	}, nil
}

// --- NumlookupAPI ---

type NumlookupAPIProvider struct {
	BaseURL string
}

func NewNumlookupAPIProvider() *NumlookupAPIProvider {
	return &NumlookupAPIProvider{BaseURL: "https://api.numlookupapi.com"}
}

func (p *NumlookupAPIProvider) Name() string      { return "numlookupapi" }
func (p *NumlookupAPIProvider) KeyEnvVar() string { return "NUMLOOKUPAPI_API_KEY" }

type numlookupResponse struct {
	Valid               bool   `json:"valid"`
	Number              string `json:"number"`
	LocalFormat         string `json:"local_format"`
	InternationalFormat string `json:"international_format"`
	CountryPrefix       string `json:"country_prefix"`
	CountryCode         string `json:"country_code"`
	CountryName         string `json:"country_name"`
	Location            string `json:"location"`
	Carrier             string `json:"carrier"`
	LineType            string `json:"line_type"`
}

func (p *NumlookupAPIProvider) Validate(apiKey, e164 string) (*ValidationResult, error) {
	// numlookupapi authenticates with an "apikey" request header, keeping the
	// key out of the URL. See https://numlookupapi.com/docs/validate.
	endpoint := fmt.Sprintf("%s/v1/validate/%s", p.BaseURL, url.PathEscape(e164))

	var body numlookupResponse
	if err := fetchJSON(endpoint, map[string]string{"apikey": apiKey}, &body); err != nil {
		return nil, err
	}
	return &ValidationResult{
		Valid:         body.Valid,
		Number:        body.Number,
		LocalFormat:   body.LocalFormat,
		IntlFormat:    body.InternationalFormat,
		CountryName:   body.CountryName,
		CountryPrefix: body.CountryPrefix,
		Location:      body.Location,
		Carrier:       body.Carrier,
		LineType:      body.LineType,
	}, nil
}
