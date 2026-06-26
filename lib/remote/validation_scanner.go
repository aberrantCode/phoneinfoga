package remote

import (
	"errors"
	"fmt"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

type validationScanner struct {
	provider suppliers.ValidationProvider
}

// ValidationScannerResponse is the shared response for Numverify-alternate
// validators. The Provider field disambiguates results when several validation
// scanners run together.
type ValidationScannerResponse struct {
	Provider      string `json:"provider" console:"Provider"`
	Valid         bool   `json:"valid" console:"Valid"`
	Number        string `json:"number,omitempty" console:"Number,omitempty"`
	LocalFormat   string `json:"local_format,omitempty" console:"Local format,omitempty"`
	IntlFormat    string `json:"international_format,omitempty" console:"International format,omitempty"`
	CountryName   string `json:"country_name,omitempty" console:"Country name,omitempty"`
	CountryPrefix string `json:"country_prefix,omitempty" console:"Country prefix,omitempty"`
	Location      string `json:"location,omitempty" console:"Location,omitempty"`
	Carrier       string `json:"carrier,omitempty" console:"Carrier,omitempty"`
	LineType      string `json:"line_type,omitempty" console:"Line type,omitempty"`
}

// NewValidationScanner builds a scanner for one validation provider. Register
// one per provider (Veriphone, Abstract, NumlookupAPI) in InitScanners.
func NewValidationScanner(provider suppliers.ValidationProvider) Scanner {
	return &validationScanner{provider: provider}
}

func (s *validationScanner) Name() string {
	return s.provider.Name()
}

func (s *validationScanner) Description() string {
	return fmt.Sprintf("Validate a phone number through the %s API.", s.provider.Name())
}

func (s *validationScanner) DryRun(_ number.Number, opts ScannerOptions) error {
	if opts.GetStringEnv(s.provider.KeyEnvVar()) == "" {
		return errors.New("API key is not defined")
	}
	return nil
}

func (s *validationScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	apiKey := opts.GetStringEnv(s.provider.KeyEnvVar())

	res, err := s.provider.Validate(apiKey, n.E164)
	if err != nil {
		return nil, err
	}

	return ValidationScannerResponse{
		Provider:      s.provider.Name(),
		Valid:         res.Valid,
		Number:        res.Number,
		LocalFormat:   res.LocalFormat,
		IntlFormat:    res.IntlFormat,
		CountryName:   res.CountryName,
		CountryPrefix: res.CountryPrefix,
		Location:      res.Location,
		Carrier:       res.Carrier,
		LineType:      res.LineType,
	}, nil
}
