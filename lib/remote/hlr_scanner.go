package remote

import (
	"errors"

	"github.com/nyaruka/phonenumbers"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

const HLR = "hlr"

// hlrDefaultURL is an explicit placeholder, not a working endpoint. It is
// treated as "unconfigured": when HLR_API_URL is empty or still set to this
// value, DryRun skips the scanner cleanly instead of issuing a doomed request
// to a non-existent host. Set HLR_API_URL to your chosen HLR provider.
const hlrDefaultURL = "https://api.example-hlr-provider.com/v1/lookup"

type hlrScanner struct {
	client suppliers.HLRSupplierInterface
}

// HLRScannerResponse is the HLR scanner response.
type HLRScannerResponse struct {
	Active         *bool  `json:"active,omitempty" console:"Active,omitempty"`
	Status         string `json:"status,omitempty" console:"Status,omitempty"`
	CurrentCarrier string `json:"current_carrier,omitempty" console:"Current carrier,omitempty"`
	Ported         *bool  `json:"ported,omitempty" console:"Ported,omitempty"`
	Roaming        *bool  `json:"roaming,omitempty" console:"Roaming,omitempty"`
	LineType       string `json:"line_type,omitempty" console:"Line type,omitempty"`
	MCC            string `json:"mcc,omitempty" console:"MCC,omitempty"`
	MNC            string `json:"mnc,omitempty" console:"MNC,omitempty"`
	Country        string `json:"country,omitempty" console:"Country,omitempty"`
}

func NewHLRScanner(s suppliers.HLRSupplierInterface) Scanner {
	return &hlrScanner{client: s}
}

func (s *hlrScanner) Name() string {
	return HLR
}

func (s *hlrScanner) Description() string {
	return "Check live network status and current carrier of a mobile number via an HLR lookup."
}

func (s *hlrScanner) DryRun(n number.Number, opts ScannerOptions) error {
	if opts.GetStringEnv("HLR_API_KEY") == "" {
		return errors.New("API key is not defined")
	}
	if u := opts.GetStringEnv("HLR_API_URL"); u == "" || u == hlrDefaultURL {
		return errors.New("HLR_API_URL is not configured with a provider endpoint")
	}
	if !hlrLineIsMobile(n) {
		return errors.New("HLR lookup applies to mobile numbers only")
	}
	return nil
}

func (s *hlrScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	// DryRun guarantees HLR_API_KEY and a real (non-placeholder) HLR_API_URL.
	apiKey := opts.GetStringEnv("HLR_API_KEY")
	baseURL := opts.GetStringEnv("HLR_API_URL")

	res, err := s.client.Lookup(baseURL, apiKey, n.E164)
	if err != nil {
		return nil, err
	}

	return HLRScannerResponse{
		Active:         res.Active,
		Status:         res.Status,
		CurrentCarrier: res.CurrentCarrier,
		Ported:         res.Ported,
		Roaming:        res.Roaming,
		LineType:       res.LineType,
		MCC:            res.MCC,
		MNC:            res.MNC,
		Country:        res.Country,
	}, nil
}

// hlrLineIsMobile recomputes the number type, since the Number struct does not
// carry it. HLR data applies to mobile numbers only.
func hlrLineIsMobile(n number.Number) bool {
	num, err := phonenumbers.Parse(n.E164, "")
	if err != nil {
		return false
	}
	switch phonenumbers.GetNumberType(num) {
	case phonenumbers.MOBILE, phonenumbers.FIXED_LINE_OR_MOBILE:
		return true
	default:
		return false
	}
}
