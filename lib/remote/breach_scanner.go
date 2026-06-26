package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

const Breach = "breach"

const breachDisclaimer = "Exposure indicators from a third-party breach-aggregation provider. Use only for lawful purposes with a permissible basis."

type breachScanner struct {
	client suppliers.DehashedSupplierInterface
}

// BreachEntry describes one breach record that references the number.
//
// In the default (minimal) mode Fields is nil: only the source and the list of
// field types present are reported. Fields is populated only when the operator
// has explicitly enabled BREACH_INCLUDE_FIELDS, since those values are third
// parties' personal data.
type BreachEntry struct {
	Source     string            `json:"source" console:"Source"`
	FieldTypes []string          `json:"field_types,omitempty" console:"Field types,omitempty"`
	Fields     map[string]string `json:"fields,omitempty" console:"-"`
}

// BreachScannerResponse is the breach scanner response.
type BreachScannerResponse struct {
	Found        bool          `json:"found" console:"Found"`
	TotalRecords int           `json:"total_records" console:"Total records"`
	Sources      []string      `json:"sources,omitempty" console:"Sources,omitempty"`
	Entries      []BreachEntry `json:"entries,omitempty" console:"Entries,omitempty"`
	Disclaimer   string        `json:"disclaimer,omitempty" console:"Note,omitempty"`
}

func NewBreachScanner(s suppliers.DehashedSupplierInterface) Scanner {
	return &breachScanner{client: s}
}

func (s *breachScanner) Name() string {
	return Breach
}

func (s *breachScanner) Description() string {
	return "Check whether a phone number appears in known breach corpora via an authenticated provider (disabled by default)."
}

// DryRun enforces the double gate: the scanner is skipped unless it has been
// explicitly opted into AND provider credentials are present.
func (s *breachScanner) DryRun(_ number.Number, opts ScannerOptions) error {
	if !isBreachEnabled(opts) {
		return errors.New("breach scanner is disabled (set BREACH_SCANNER_ENABLED=true to opt in)")
	}
	if opts.GetStringEnv("DEHASHED_EMAIL") == "" || opts.GetStringEnv("DEHASHED_API_KEY") == "" {
		return errors.New("breach provider credentials are not defined")
	}
	return nil
}

func (s *breachScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	email := opts.GetStringEnv("DEHASHED_EMAIL")
	apiKey := opts.GetStringEnv("DEHASHED_API_KEY")
	includeFields := isTruthy(opts.GetStringEnv("BREACH_INCLUDE_FIELDS"))

	query := fmt.Sprintf("phone:%q", breachPhoneQuery(n))

	res, err := s.client.SearchByPhone(email, apiKey, query)
	if err != nil {
		return nil, err
	}

	data := BreachScannerResponse{
		Found:        res.Total > 0 || len(res.Entries) > 0,
		TotalRecords: res.Total,
		Disclaimer:   breachDisclaimer,
	}

	sourceSet := map[string]struct{}{}
	for _, entry := range res.Entries {
		be := BreachEntry{
			Source:     entry.DatabaseName,
			FieldTypes: dehashedFieldTypes(entry),
		}
		if includeFields {
			be.Fields = dehashedFieldValues(entry)
		}
		data.Entries = append(data.Entries, be)

		if entry.DatabaseName != "" {
			sourceSet[entry.DatabaseName] = struct{}{}
		}
	}

	for source := range sourceSet {
		data.Sources = append(data.Sources, source)
	}
	sort.Strings(data.Sources)

	return data, nil
}

func isBreachEnabled(opts ScannerOptions) bool {
	return isTruthy(opts.GetStringEnv("BREACH_SCANNER_ENABLED"))
}

// isTruthy requires an explicit affirmative value, so a missing or unexpected
// value leaves a gated feature off.
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

// breachPhoneQuery reduces the number to its digits, the most canonical form
// for matching breach records (which rarely store the leading "+").
func breachPhoneQuery(n number.Number) string {
	var b strings.Builder
	for _, r := range n.E164 {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// dehashedFieldTypes reports which PII field types are present in an entry,
// without exposing any values.
func dehashedFieldTypes(e suppliers.DehashedEntry) []string {
	var types []string
	for _, c := range []struct {
		name string
		raw  json.RawMessage
	}{
		{"email", e.Email},
		{"username", e.Username},
		{"password", e.Password},
		{"hashed_password", e.HashedPassword},
		{"name", e.Name},
		{"address", e.Address},
		{"ip_address", e.IPAddress},
		{"phone", e.Phone},
		{"vin", e.VIN},
	} {
		if rawHasValue(c.raw) {
			types = append(types, c.name)
		}
	}
	return types
}

// dehashedFieldValues flattens an entry's PII values. Only called when the
// operator has explicitly opted in via BREACH_INCLUDE_FIELDS.
func dehashedFieldValues(e suppliers.DehashedEntry) map[string]string {
	fields := map[string]string{}
	add := func(name string, raw json.RawMessage) {
		if vals := rawValues(raw); len(vals) > 0 {
			fields[name] = strings.Join(vals, ", ")
		}
	}
	add("email", e.Email)
	add("username", e.Username)
	add("password", e.Password)
	add("hashed_password", e.HashedPassword)
	add("name", e.Name)
	add("address", e.Address)
	add("ip_address", e.IPAddress)
	add("phone", e.Phone)
	add("vin", e.VIN)
	return fields
}

// rawHasValue reports whether a JSON raw message holds a non-empty value,
// tolerating both the scalar-string (v1) and array (v2) API response shapes.
func rawHasValue(raw json.RawMessage) bool {
	return len(rawValues(raw)) > 0
}

// rawValues flattens a JSON raw message into non-empty strings, accepting a
// single string, an array of strings, or null/empty.
func rawValues(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	if trimmed := strings.TrimSpace(string(raw)); trimmed == "" || trimmed == "null" {
		return nil
	}

	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		var out []string
		for _, v := range arr {
			if strings.TrimSpace(v) != "" {
				out = append(out, v)
			}
		}
		return out
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		if strings.TrimSpace(single) != "" {
			return []string{single}
		}
	}
	return nil
}
