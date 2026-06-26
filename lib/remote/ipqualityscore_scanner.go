package remote

import (
	"errors"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

const IPQualityScore = "ipqualityscore"

type ipqsScanner struct {
	client suppliers.IPQSSupplierInterface
}

// IPQSScannerResponse is the IPQualityScore scanner response.
type IPQSScannerResponse struct {
	Valid       bool   `json:"valid" console:"Valid"`
	Active      bool   `json:"active" console:"Active"`
	FraudScore  int    `json:"fraud_score" console:"Fraud score"`
	RecentAbuse bool   `json:"recent_abuse" console:"Recent abuse"`
	VOIP        bool   `json:"voip" console:"VOIP"`
	Prepaid     bool   `json:"prepaid" console:"Prepaid"`
	Risky       bool   `json:"risky" console:"Risky"`
	DoNotCall   bool   `json:"do_not_call,omitempty" console:"Do not call,omitempty"`
	Leaked      bool   `json:"leaked,omitempty" console:"Leaked,omitempty"`
	Spammer     bool   `json:"spammer,omitempty" console:"Spammer,omitempty"`
	LineType    string `json:"line_type,omitempty" console:"Line type,omitempty"`
	Carrier     string `json:"carrier,omitempty" console:"Carrier,omitempty"`
	Country     string `json:"country,omitempty" console:"Country,omitempty"`
	Region      string `json:"region,omitempty" console:"Region,omitempty"`
	City        string `json:"city,omitempty" console:"City,omitempty"`
	Timezone    string `json:"timezone,omitempty" console:"Timezone,omitempty"`
}

func NewIPQualityScoreScanner(s suppliers.IPQSSupplierInterface) Scanner {
	return &ipqsScanner{client: s}
}

func (s *ipqsScanner) Name() string {
	return IPQualityScore
}

func (s *ipqsScanner) Description() string {
	return "Score a phone number for fraud and abuse signals via the IPQualityScore API."
}

func (s *ipqsScanner) DryRun(_ number.Number, opts ScannerOptions) error {
	if opts.GetStringEnv("IPQS_API_KEY") == "" {
		return errors.New("API key is not defined")
	}
	return nil
}

func (s *ipqsScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	apiKey := opts.GetStringEnv("IPQS_API_KEY")

	res, err := s.client.Validate(apiKey, n.E164)
	if err != nil {
		return nil, err
	}

	return IPQSScannerResponse{
		Valid:       res.Valid,
		Active:      res.Active,
		FraudScore:  res.FraudScore,
		RecentAbuse: res.RecentAbuse,
		VOIP:        res.VOIP,
		Prepaid:     res.Prepaid,
		Risky:       res.Risky,
		DoNotCall:   res.DoNotCall,
		Leaked:      res.Leaked,
		Spammer:     res.Spammer,
		LineType:    res.LineType,
		Carrier:     res.Carrier,
		Country:     res.Country,
		Region:      res.Region,
		City:        res.City,
		Timezone:    res.Timezone,
	}, nil
}
