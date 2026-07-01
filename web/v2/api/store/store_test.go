package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numberA = "+33612345678"
	numberB = "+14155552671"
)

// newTestStore opens a fresh temp-file store and migrates it.
func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "lookups.db")
	s, err := New(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	require.NoError(t, s.Migrate(context.Background()))
	return s
}

func sampleLookup(e164 string, scanners ...string) Lookup {
	return Lookup{
		E164:              e164,
		NumberInput:       e164,
		Valid:             true,
		Local:             "06 12 34 56 78",
		International:     e164,
		CountryCode:       33,
		Country:           "FR",
		Carrier:           "Orange",
		ScannersRequested: scanners,
		ClientIP:          "203.0.113.7",
		UserAgent:         "test-agent",
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	// A second (and third) Migrate call must be a no-op, not an error.
	require.NoError(t, s.Migrate(ctx))
	require.NoError(t, s.Migrate(ctx))
}

func TestCreateLookupDefaults(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local", "numverify"))
	require.NoError(t, err)

	assert.NotEmpty(t, l.ID, "store should assign an id")
	assert.Equal(t, StatusPending, l.Status)
	assert.False(t, l.CreatedAt.IsZero(), "created_at should be set")
	assert.Nil(t, l.CompletedAt)
	assert.Equal(t, []string{"local", "numverify"}, l.ScannersRequested)
}

func TestSaveAndHydrateResults(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local", "numverify"))
	require.NoError(t, err)

	start := time.Now().UTC()
	require.NoError(t, s.SaveScannerResult(ctx, ScannerResult{
		LookupID:    l.ID,
		Scanner:     "local",
		Status:      ResultStatusSuccess,
		RawResponse: json.RawMessage(`{"valid":true}`),
		StartedAt:   start,
		FinishedAt:  start.Add(12 * time.Millisecond),
		DurationMs:  12,
	}))
	require.NoError(t, s.SaveScannerResult(ctx, ScannerResult{
		LookupID:     l.ID,
		Scanner:      "numverify",
		Status:       ResultStatusError,
		ErrorMessage: "quota exceeded",
		StartedAt:    start.Add(20 * time.Millisecond),
		FinishedAt:   start.Add(360 * time.Millisecond),
		DurationMs:   340,
	}))

	got, err := s.GetLookup(ctx, l.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got.Results, 2, "results should be hydrated")

	// Ordered by started_at ascending: success (local) then error (numverify).
	assert.Equal(t, "local", got.Results[0].Scanner)
	assert.Equal(t, ResultStatusSuccess, got.Results[0].Status)
	assert.JSONEq(t, `{"valid":true}`, string(got.Results[0].RawResponse))

	assert.Equal(t, "numverify", got.Results[1].Scanner)
	assert.Equal(t, ResultStatusError, got.Results[1].Status)
	assert.Equal(t, "quota exceeded", got.Results[1].ErrorMessage)
	assert.Empty(t, got.Results[1].RawResponse, "error result has no raw payload")
}

func TestSaveScannerResultUpserts(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local"))
	require.NoError(t, err)

	now := time.Now().UTC()
	first := ScannerResult{LookupID: l.ID, Scanner: "local", Status: ResultStatusError,
		ErrorMessage: "temporary", StartedAt: now, FinishedAt: now, DurationMs: 1}
	require.NoError(t, s.SaveScannerResult(ctx, first))

	// Re-run of the same scanner within the same lookup replaces the row.
	second := ScannerResult{LookupID: l.ID, Scanner: "local", Status: ResultStatusSuccess,
		RawResponse: json.RawMessage(`{"ok":true}`), StartedAt: now, FinishedAt: now, DurationMs: 5}
	require.NoError(t, s.SaveScannerResult(ctx, second))

	got, err := s.GetLookup(ctx, l.ID)
	require.NoError(t, err)
	require.Len(t, got.Results, 1, "duplicate (lookup_id, scanner) must upsert, not duplicate")
	assert.Equal(t, ResultStatusSuccess, got.Results[0].Status)
	assert.JSONEq(t, `{"ok":true}`, string(got.Results[0].RawResponse))
}

func TestCloseLookupComplete(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local", "numverify"))
	require.NoError(t, err)

	now := time.Now().UTC()
	for _, name := range []string{"local", "numverify"} {
		require.NoError(t, s.SaveScannerResult(ctx, ScannerResult{
			LookupID: l.ID, Scanner: name, Status: ResultStatusSuccess,
			RawResponse: json.RawMessage(`{}`), StartedAt: now, FinishedAt: now, DurationMs: 1,
		}))
	}

	closed, err := s.CloseLookup(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusComplete, closed.Status)
	require.NotNil(t, closed.CompletedAt)
	assert.False(t, closed.CompletedAt.IsZero())
}

func TestCloseLookupPartial(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local", "numverify"))
	require.NoError(t, err)

	now := time.Now().UTC()
	// Only one of the two requested scanners produced a row.
	require.NoError(t, s.SaveScannerResult(ctx, ScannerResult{
		LookupID: l.ID, Scanner: "local", Status: ResultStatusSuccess,
		RawResponse: json.RawMessage(`{}`), StartedAt: now, FinishedAt: now, DurationMs: 1,
	}))

	closed, err := s.CloseLookup(ctx, l.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusPartial, closed.Status)
	require.NotNil(t, closed.CompletedAt)
}

func TestCloseLookupUnknownID(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CloseLookup(context.Background(), "does-not-exist")
	assert.Error(t, err)
}

func TestGetLookupNotFound(t *testing.T) {
	s := newTestStore(t)
	got, err := s.GetLookup(context.Background(), "missing")
	require.NoError(t, err)
	assert.Nil(t, got, "missing lookup returns (nil, nil)")
}

func TestGetLatestLookupByNumber(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	older, err := s.CreateLookup(ctx, sampleLookup(numberA, "local"))
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)
	newer, err := s.CreateLookup(ctx, sampleLookup(numberA, "local"))
	require.NoError(t, err)

	got, err := s.GetLatestLookupByNumber(ctx, numberA)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, newer.ID, got.ID, "latest should be the newest lookup")
	assert.NotEqual(t, older.ID, got.ID)
}

func TestGetLatestLookupByNumberNone(t *testing.T) {
	s := newTestStore(t)
	got, err := s.GetLatestLookupByNumber(context.Background(), numberB)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestListLookupsByNumberOrderingLimitAndScoping(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	var ids []string
	for i := 0; i < 3; i++ {
		l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local"))
		require.NoError(t, err)
		ids = append(ids, l.ID)
		time.Sleep(2 * time.Millisecond)
	}
	// A lookup for a different number must never appear in numberA's history.
	_, err := s.CreateLookup(ctx, sampleLookup(numberB, "local"))
	require.NoError(t, err)

	list, err := s.ListLookupsByNumber(ctx, numberA, 2)
	require.NoError(t, err)
	require.Len(t, list, 2, "limit should cap results")
	// Newest first: last created id, then the one before it.
	assert.Equal(t, ids[2], list[0].ID)
	assert.Equal(t, ids[1], list[1].ID)
	// Summaries do not hydrate results.
	assert.Empty(t, list[0].Results)

	full, err := s.ListLookupsByNumber(ctx, numberA, 10)
	require.NoError(t, err)
	assert.Len(t, full, 3, "must return only numberA lookups")
	for _, l := range full {
		assert.Equal(t, numberA, l.E164)
	}
}

func TestForeignKeyCascadeDelete(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local"))
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, s.SaveScannerResult(ctx, ScannerResult{
		LookupID: l.ID, Scanner: "local", Status: ResultStatusSuccess,
		RawResponse: json.RawMessage(`{}`), StartedAt: now, FinishedAt: now, DurationMs: 1,
	}))

	// Deleting the parent lookup must cascade to its scanner_results (FK ON DELETE CASCADE
	// requires PRAGMA foreign_keys=ON on the connection).
	_, err = s.db.ExecContext(ctx, "DELETE FROM lookups WHERE id = ?", l.ID)
	require.NoError(t, err)

	var count int
	require.NoError(t, s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM scanner_results WHERE lookup_id = ?", l.ID).Scan(&count))
	assert.Equal(t, 0, count, "scanner_results should be cascade-deleted")
}
