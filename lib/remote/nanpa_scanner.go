package remote

import (
	"errors"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

const NANPA = "nanpa"

type nanpaScanner struct {
	client suppliers.NumberingPlanSupplierInterface
}

// NANPAScannerResponse is the NANPA numbering-plan scanner response.
type NANPAScannerResponse struct {
	Found       bool   `json:"found" console:"Found"`
	NPA         string `json:"npa,omitempty" console:"Area code,omitempty"`
	NXX         string `json:"nxx,omitempty" console:"Exchange,omitempty"`
	RateCenter  string `json:"rate_center,omitempty" console:"Rate center,omitempty"`
	State       string `json:"state,omitempty" console:"State,omitempty"`
	OCN         string `json:"ocn,omitempty" console:"OCN,omitempty"`
	Carrier     string `json:"carrier,omitempty" console:"Carrier of record,omitempty"`
	LATA        string `json:"lata,omitempty" console:"LATA,omitempty"`
	BlockHolder string `json:"block_holder,omitempty" console:"Block holder,omitempty"`
}

func NewNANPAScanner(s suppliers.NumberingPlanSupplierInterface) Scanner {
	return &nanpaScanner{client: s}
}

func (s *nanpaScanner) Name() string {
	return NANPA
}

func (s *nanpaScanner) Description() string {
	return "Resolve the NPA-NXX of a North American number to rate center, state, and carrier of record."
}

func (s *nanpaScanner) DryRun(n number.Number, opts ScannerOptions) error {
	if n.CountryCode != 1 {
		return errors.New("NANPA lookup applies to +1 (North American) numbers only")
	}
	if opts.GetStringEnv("NANPA_API_URL") == "" {
		return errors.New("NANPA_API_URL is not defined")
	}
	return nil
}

func (s *nanpaScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	baseURL := opts.GetStringEnv("NANPA_API_URL")
	apiKey := opts.GetStringEnv("NANPA_API_KEY")

	npa, nxx, ok := nanpaSplit(n)
	if !ok {
		return nil, errors.New("could not derive NPA-NXX from number")
	}

	res, err := s.client.Lookup(baseURL, apiKey, npa, nxx)
	if err != nil {
		return nil, err
	}

	return NANPAScannerResponse{
		Found:       res.RateCenter != "" || res.OCN != "" || res.State != "",
		NPA:         npa,
		NXX:         nxx,
		RateCenter:  res.RateCenter,
		State:       res.State,
		OCN:         res.OCN,
		Carrier:     res.Carrier,
		LATA:        res.LATA,
		BlockHolder: res.BlockHolder,
	}, nil
}

// nanpaSplit derives the area code (NPA) and exchange (NXX) from the number's
// E.164 digits, dropping the country code.
func nanpaSplit(n number.Number) (string, string, bool) {
	var digits []rune
	for _, r := range n.E164 {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) == 11 && digits[0] == '1' {
		digits = digits[1:]
	}
	if len(digits) < 6 {
		return "", "", false
	}
	return string(digits[0:3]), string(digits[3:6]), true
}
