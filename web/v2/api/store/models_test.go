package store

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "complete", StatusComplete)
	assert.Equal(t, "partial", StatusPartial)
}

func TestScannerResultStatusConstants(t *testing.T) {
	assert.Equal(t, "success", ResultStatusSuccess)
	assert.Equal(t, "error", ResultStatusError)
}

func TestLookupCarriesResults(t *testing.T) {
	l := Lookup{
		ID:                "id-1",
		E164:              "+33612345678",
		ScannersRequested: []string{"local", "numverify"},
		Status:            StatusPending,
		Results: []ScannerResult{
			{Scanner: "local", Status: ResultStatusSuccess, RawResponse: json.RawMessage(`{"ok":true}`)},
		},
	}

	assert.Len(t, l.ScannersRequested, 2)
	assert.Len(t, l.Results, 1)
	assert.Equal(t, "local", l.Results[0].Scanner)
	assert.JSONEq(t, `{"ok":true}`, string(l.Results[0].RawResponse))
}
