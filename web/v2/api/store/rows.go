package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

// timeLayout is the on-disk time format. RFC3339Nano preserves sub-second precision so
// created_at DESC ordering is stable even for lookups created within the same second.
const timeLayout = time.RFC3339Nano

const lookupColumns = "id, e164, number_input, valid, raw_local, local, international, " +
	"country_code, country, carrier, scanners_requested, client_ip, user_agent, status, " +
	"created_at, completed_at"

const resultColumns = "id, lookup_id, scanner, status, error_message, raw_response, " +
	"started_at, finished_at, duration_ms"

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// nullableString returns nil for empty strings so they persist as SQL NULL.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nullableRaw returns nil for empty payloads so error results store raw_response as NULL.
func nullableRaw(r json.RawMessage) interface{} {
	if len(r) == 0 {
		return nil
	}
	return string(r)
}

// marshalScanners encodes the requested scanner list as JSON text for storage.
func marshalScanners(scanners []string) (string, error) {
	b, err := json.Marshal(scanners)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(timeLayout, s)
}

// scanLookup materializes a lookups row (without its results).
func scanLookup(sc rowScanner) (*Lookup, error) {
	var (
		l           Lookup
		valid       sql.NullInt64
		rawLocal    sql.NullString
		local       sql.NullString
		intl        sql.NullString
		countryCode sql.NullInt64
		country     sql.NullString
		carrier     sql.NullString
		scanners    string
		clientIP    sql.NullString
		userAgent   sql.NullString
		createdAt   string
		completedAt sql.NullString
	)

	if err := sc.Scan(
		&l.ID, &l.E164, &l.NumberInput, &valid, &rawLocal, &local, &intl,
		&countryCode, &country, &carrier, &scanners, &clientIP, &userAgent,
		&l.Status, &createdAt, &completedAt,
	); err != nil {
		return nil, err
	}

	l.Valid = valid.Int64 == 1
	l.RawLocal = rawLocal.String
	l.Local = local.String
	l.International = intl.String
	l.CountryCode = int32(countryCode.Int64)
	l.Country = country.String
	l.Carrier = carrier.String
	l.ClientIP = clientIP.String
	l.UserAgent = userAgent.String

	if err := json.Unmarshal([]byte(scanners), &l.ScannersRequested); err != nil {
		return nil, err
	}

	created, err := parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	l.CreatedAt = created

	if completedAt.Valid {
		completed, err := parseTime(completedAt.String)
		if err != nil {
			return nil, err
		}
		l.CompletedAt = &completed
	}

	return &l, nil
}

// scanResult materializes a scanner_results row.
func scanResult(sc rowScanner) (*ScannerResult, error) {
	var (
		r          ScannerResult
		errMessage sql.NullString
		raw        sql.NullString
		startedAt  string
		finishedAt string
	)

	if err := sc.Scan(
		&r.ID, &r.LookupID, &r.Scanner, &r.Status, &errMessage, &raw,
		&startedAt, &finishedAt, &r.DurationMs,
	); err != nil {
		return nil, err
	}

	r.ErrorMessage = errMessage.String
	if raw.Valid && raw.String != "" {
		r.RawResponse = json.RawMessage(raw.String)
	}

	started, err := parseTime(startedAt)
	if err != nil {
		return nil, err
	}
	r.StartedAt = started

	finished, err := parseTime(finishedAt)
	if err != nil {
		return nil, err
	}
	r.FinishedAt = finished

	return &r, nil
}
