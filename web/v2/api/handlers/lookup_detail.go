package handlers

import (
	"encoding/json"
	"time"

	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

// LookupResultDetail is a single scanner result in a lookup detail payload. Raw omits the
// `omitempty` tag so an absent payload (error results) serializes as JSON null per spec §7.
type LookupResultDetail struct {
	Scanner      string          `json:"scanner"`
	Status       string          `json:"status"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	Raw          json.RawMessage `json:"raw"`
	DurationMs   int64           `json:"durationMs"`
	StartedAt    time.Time       `json:"startedAt"`
	FinishedAt   time.Time       `json:"finishedAt"`
}

// LookupDetailResponse is the full lookup payload returned by GET /v2/lookups/:id and
// GET /v2/lookups/latest (spec §7).
type LookupDetailResponse struct {
	ID                string               `json:"id"`
	Status            string               `json:"status"`
	CreatedAt         time.Time            `json:"createdAt"`
	CompletedAt       *time.Time           `json:"completedAt"`
	ClientIP          string               `json:"clientIp"`
	UserAgent         string               `json:"userAgent"`
	Number            AddNumberResponse    `json:"number"`
	ScannersRequested []string             `json:"scannersRequested"`
	Results           []LookupResultDetail `json:"results"`
}

// lookupNumberMetadata projects the number snapshot stored on a lookup into the response
// shape shared with AddNumber.
func lookupNumberMetadata(l store.Lookup) AddNumberResponse {
	return AddNumberResponse{
		Valid:         l.Valid,
		RawLocal:      l.RawLocal,
		Local:         l.Local,
		E164:          l.E164,
		International: l.International,
		CountryCode:   l.CountryCode,
		Country:       l.Country,
		Carrier:       l.Carrier,
	}
}

// lookupDetail projects a hydrated store.Lookup into the detail response payload.
func lookupDetail(l store.Lookup) LookupDetailResponse {
	results := make([]LookupResultDetail, 0, len(l.Results))
	for _, r := range l.Results {
		results = append(results, LookupResultDetail{
			Scanner:      r.Scanner,
			Status:       r.Status,
			ErrorMessage: r.ErrorMessage,
			Raw:          r.RawResponse,
			DurationMs:   r.DurationMs,
			StartedAt:    r.StartedAt,
			FinishedAt:   r.FinishedAt,
		})
	}

	return LookupDetailResponse{
		ID:                l.ID,
		Status:            l.Status,
		CreatedAt:         l.CreatedAt,
		CompletedAt:       l.CompletedAt,
		ClientIP:          l.ClientIP,
		UserAgent:         l.UserAgent,
		Number:            lookupNumberMetadata(l),
		ScannersRequested: l.ScannersRequested,
		Results:           results,
	}
}
