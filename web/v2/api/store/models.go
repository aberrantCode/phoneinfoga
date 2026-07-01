package store

import (
	"context"
	"encoding/json"
	"time"
)

// Lookup status values (lookups.status). A lookup starts pending, then closes as
// complete (every requested scanner produced a result row) or partial (some did not).
const (
	StatusPending  = "pending"
	StatusComplete = "complete"
	StatusPartial  = "partial"
)

// Scanner result status values (scanner_results.status).
const (
	ResultStatusSuccess = "success"
	ResultStatusError   = "error"
)

// Lookup is a persisted web lookup request record plus a snapshot of the number
// metadata it was created with. When hydrated by GetLookup / GetLatestLookupByNumber
// it also carries its per-scanner Results, newest scanner rows excluded from summaries.
type Lookup struct {
	ID                string          `json:"id"`
	E164              string          `json:"e164"`
	NumberInput       string          `json:"numberInput"`
	Valid             bool            `json:"valid"`
	RawLocal          string          `json:"rawLocal"`
	Local             string          `json:"local"`
	International     string          `json:"international"`
	CountryCode       int32           `json:"countryCode"`
	Country           string          `json:"country"`
	Carrier           string          `json:"carrier"`
	ScannersRequested []string        `json:"scannersRequested"`
	ClientIP          string          `json:"clientIp"`
	UserAgent         string          `json:"userAgent"`
	Status            string          `json:"status"`
	CreatedAt         time.Time       `json:"createdAt"`
	CompletedAt       *time.Time      `json:"completedAt,omitempty"`
	Results           []ScannerResult `json:"results,omitempty"`
}

// ScannerResult is a single scanner's outcome within a lookup. RawResponse holds the
// scanner's exact payload verbatim (never re-shaped) so replay is byte-identical to a
// live run; it is nil for error results.
type ScannerResult struct {
	ID           string          `json:"id"`
	LookupID     string          `json:"lookupId"`
	Scanner      string          `json:"scanner"`
	Status       string          `json:"status"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	RawResponse  json.RawMessage `json:"raw,omitempty"`
	StartedAt    time.Time       `json:"startedAt"`
	FinishedAt   time.Time       `json:"finishedAt"`
	DurationMs   int64           `json:"durationMs"`
}

// Store persists lookups and their scanner results. It mirrors the package-var
// wiring pattern used by web/v2/api/handlers; the concrete implementation is backed
// by embedded SQLite (see sqlite.go).
type Store interface {
	// Migrate applies pending schema migrations. It is safe to call repeatedly.
	Migrate(ctx context.Context) error
	// CreateLookup inserts a new request record with status=pending and returns it.
	CreateLookup(ctx context.Context, l Lookup) (Lookup, error)
	// SaveScannerResult upserts a scanner's result, keyed on (lookup_id, scanner).
	SaveScannerResult(ctx context.Context, r ScannerResult) error
	// CloseLookup sets completed_at and computes complete/partial status.
	CloseLookup(ctx context.Context, id string) (Lookup, error)
	// GetLookup returns a lookup by id, hydrated with its scanner results.
	GetLookup(ctx context.Context, id string) (*Lookup, error)
	// GetLatestLookupByNumber returns the newest lookup for an e164, hydrated.
	GetLatestLookupByNumber(ctx context.Context, e164 string) (*Lookup, error)
	// ListLookupsByNumber returns per-number summaries (no results), newest first.
	ListLookupsByNumber(ctx context.Context, e164 string, limit int) ([]Lookup, error)
}
