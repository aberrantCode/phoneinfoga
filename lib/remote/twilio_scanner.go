package remote

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

const Twilio = "twilio"

const twilioDefaultFields = "line_type_intelligence"

type twilioScanner struct {
	client suppliers.TwilioSupplierInterface
}

// TwilioScannerResponse is the Twilio scanner response.
type TwilioScannerResponse struct {
	Valid             bool     `json:"valid" console:"Valid"`
	NationalFormat    string   `json:"national_format,omitempty" console:"National format,omitempty"`
	LineType          string   `json:"line_type,omitempty" console:"Line type,omitempty"`
	CarrierName       string   `json:"carrier_name,omitempty" console:"Carrier,omitempty"`
	MobileCountryCode string   `json:"mobile_country_code,omitempty" console:"MCC,omitempty"`
	MobileNetworkCode string   `json:"mobile_network_code,omitempty" console:"MNC,omitempty"`
	CallerName        string   `json:"caller_name,omitempty" console:"Caller name,omitempty"`
	CallerType        string   `json:"caller_type,omitempty" console:"Caller type,omitempty"`
	SimSwapLastDate   string   `json:"sim_swap_last_date,omitempty" console:"Last SIM swap,omitempty"`
	SimSwapInPeriod   *bool    `json:"sim_swap_in_period,omitempty" console:"SIM swapped in period,omitempty"`
	CallForwarding    string   `json:"call_forwarding_status,omitempty" console:"Call forwarding,omitempty"`
	Notes             []string `json:"notes,omitempty" console:"Notes,omitempty"`
}

func NewTwilioScanner(s suppliers.TwilioSupplierInterface) Scanner {
	return &twilioScanner{client: s}
}

func (s *twilioScanner) Name() string {
	return Twilio
}

func (s *twilioScanner) Description() string {
	return "Request live carrier, line type and mobile signals through the Twilio Lookup v2 API."
}

func (s *twilioScanner) DryRun(_ number.Number, opts ScannerOptions) error {
	if opts.GetStringEnv("TWILIO_ACCOUNT_SID") == "" || opts.GetStringEnv("TWILIO_AUTH_TOKEN") == "" {
		return errors.New("Twilio credentials are not defined")
	}
	return nil
}

func (s *twilioScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	accountSID := opts.GetStringEnv("TWILIO_ACCOUNT_SID")
	authToken := opts.GetStringEnv("TWILIO_AUTH_TOKEN")
	fields := twilioFields(opts)

	res, err := s.client.Lookup(accountSID, authToken, n.E164, fields)
	if err != nil {
		return nil, err
	}

	data := TwilioScannerResponse{
		Valid:          res.Valid,
		NationalFormat: res.NationalFormat,
	}

	if lti := res.LineTypeIntelligence; lti != nil {
		if lti.ErrorCode != nil {
			data.Notes = append(data.Notes, fmt.Sprintf("line_type_intelligence unavailable (error %d)", *lti.ErrorCode))
		} else {
			data.LineType = lti.Type
			data.CarrierName = lti.CarrierName
			data.MobileCountryCode = lti.MobileCountryCode
			data.MobileNetworkCode = lti.MobileNetworkCode
		}
	}

	if cn := res.CallerName; cn != nil {
		if cn.ErrorCode != nil {
			data.Notes = append(data.Notes, fmt.Sprintf("caller_name unavailable (error %d)", *cn.ErrorCode))
		} else {
			data.CallerName = cn.CallerName
			data.CallerType = cn.CallerType
		}
	}

	if ss := res.SimSwap; ss != nil {
		if ss.ErrorCode != nil {
			data.Notes = append(data.Notes, fmt.Sprintf("sim_swap unavailable (error %d)", *ss.ErrorCode))
		} else {
			data.SimSwapLastDate = ss.LastSimSwap.LastSimSwapDate
			swapped := ss.LastSimSwap.SwappedInPeriod
			data.SimSwapInPeriod = &swapped
			if data.CarrierName == "" {
				data.CarrierName = ss.CarrierName
			}
		}
	}

	if cf := res.CallForwarding; cf != nil {
		if cf.ErrorCode != nil {
			data.Notes = append(data.Notes, fmt.Sprintf("call_forwarding unavailable (error %d)", *cf.ErrorCode))
		} else {
			data.CallForwarding = cf.CallForwardingStatus
		}
	}

	return data, nil
}

func twilioFields(opts ScannerOptions) []string {
	raw := opts.GetStringEnv("TWILIO_LOOKUP_FIELDS")
	if raw == "" {
		raw = twilioDefaultFields
	}

	var fields []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			fields = append(fields, part)
		}
	}
	return fields
}
