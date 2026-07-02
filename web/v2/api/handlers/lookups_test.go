package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/handlers"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

// newLookupTestStore injects a fresh migrated store into the handlers package.
func newLookupTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	s, err := store.New(filepath.Join(t.TempDir(), "handlers-lookups.db"))
	require.NoError(t, err)
	require.NoError(t, s.Migrate(context.Background()))
	t.Cleanup(func() { _ = s.Close() })
	handlers.InitStore(s)
	return s
}

// newJSONContext builds a gin.Context carrying a JSON POST body for direct handler calls.
func newJSONContext(t *testing.T, body interface{}) *gin.Context {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v2/lookups", bytes.NewReader(raw))
	ctx.Request.Header.Set("User-Agent", "test-agent")
	return ctx
}

func TestCreateLookupHandler(t *testing.T) {
	s := newLookupTestStore(t)

	ctx := newJSONContext(t, handlers.CreateLookupInput{
		Number:   "14152229670",
		Scanners: []string{"local", "numverify"},
	})
	resp := handlers.CreateLookup(ctx)

	require.Equal(t, http.StatusOK, resp.Code)
	data, ok := resp.Data.(handlers.CreateLookupResponse)
	require.True(t, ok, "unexpected response type %T", resp.Data)

	assert.NotEmpty(t, data.ID)
	assert.Equal(t, store.StatusPending, data.Status)
	assert.Equal(t, []string{"local", "numverify"}, data.ScannersRequested)
	assert.Equal(t, "+14152229670", data.Number.E164)
	assert.True(t, data.Number.Valid)
	assert.NotEmpty(t, data.ClientIP)
	assert.False(t, data.CreatedAt.IsZero())

	// The record must be persisted before any scanner runs (AC1).
	got, err := s.GetLookup(context.Background(), data.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "+14152229670", got.E164)
	assert.Equal(t, "14152229670", got.NumberInput)
	assert.Equal(t, "test-agent", got.UserAgent)
	assert.Equal(t, store.StatusPending, got.Status)
}

func TestCreateLookupHandlerInvalidNumber(t *testing.T) {
	newLookupTestStore(t)

	ctx := newJSONContext(t, map[string]string{"number": "not-a-number"})
	resp := handlers.CreateLookup(ctx)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCreateLookupHandlerWithoutStore(t *testing.T) {
	handlers.InitStore(nil)

	ctx := newJSONContext(t, handlers.CreateLookupInput{Number: "14152229670"})
	resp := handlers.CreateLookup(ctx)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

// newParamContext builds a gin.Context carrying route params (no request body).
func newParamContext(t *testing.T, params gin.Params) *gin.Context {
	t.Helper()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	ctx.Params = params
	return ctx
}

func seedLookup(t *testing.T, s *store.SQLiteStore, scanners ...string) store.Lookup {
	t.Helper()
	l, err := s.CreateLookup(context.Background(), store.Lookup{
		E164: "+14152229670", NumberInput: "14152229670", Valid: true,
		ScannersRequested: scanners,
	})
	require.NoError(t, err)
	return l
}

func TestCloseLookupHandlerComplete(t *testing.T) {
	s := newLookupTestStore(t)
	l := seedLookup(t, s, "local")
	now := time.Now().UTC()
	require.NoError(t, s.SaveScannerResult(context.Background(), store.ScannerResult{
		LookupID: l.ID, Scanner: "local", Status: store.ResultStatusSuccess,
		StartedAt: now, FinishedAt: now, DurationMs: 1,
	}))

	ctx := newParamContext(t, gin.Params{{Key: "id", Value: l.ID}})
	resp := handlers.CloseLookup(ctx)

	require.Equal(t, http.StatusOK, resp.Code)
	data, ok := resp.Data.(handlers.CloseLookupResponse)
	require.True(t, ok, "unexpected response type %T", resp.Data)
	assert.Equal(t, l.ID, data.ID)
	assert.Equal(t, store.StatusComplete, data.Status)
	require.NotNil(t, data.CompletedAt)
}

func TestCloseLookupHandlerNotFound(t *testing.T) {
	newLookupTestStore(t)

	ctx := newParamContext(t, gin.Params{{Key: "id", Value: "no-such-id"}})
	resp := handlers.CloseLookup(ctx)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestCloseLookupHandlerWithoutStore(t *testing.T) {
	handlers.InitStore(nil)

	ctx := newParamContext(t, gin.Params{{Key: "id", Value: "x"}})
	resp := handlers.CloseLookup(ctx)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

// seedLookupWithResults seeds a lookup plus one success and one error scanner result.
func seedLookupWithResults(t *testing.T, s *store.SQLiteStore) store.Lookup {
	t.Helper()
	l := seedLookup(t, s, "local", "numverify")
	now := time.Now().UTC()
	require.NoError(t, s.SaveScannerResult(context.Background(), store.ScannerResult{
		LookupID: l.ID, Scanner: "local", Status: store.ResultStatusSuccess,
		RawResponse: []byte(`{"valid":true}`), StartedAt: now, FinishedAt: now.Add(time.Millisecond), DurationMs: 1,
	}))
	require.NoError(t, s.SaveScannerResult(context.Background(), store.ScannerResult{
		LookupID: l.ID, Scanner: "numverify", Status: store.ResultStatusError,
		ErrorMessage: "quota exceeded", StartedAt: now.Add(2 * time.Millisecond), FinishedAt: now.Add(3 * time.Millisecond), DurationMs: 1,
	}))
	return l
}

func TestGetLookupHandler(t *testing.T) {
	s := newLookupTestStore(t)
	l := seedLookupWithResults(t, s)

	ctx := newParamContext(t, gin.Params{{Key: "id", Value: l.ID}})
	resp := handlers.GetLookup(ctx)

	require.Equal(t, http.StatusOK, resp.Code)
	data, ok := resp.Data.(handlers.LookupDetailResponse)
	require.True(t, ok, "unexpected response type %T", resp.Data)

	assert.Equal(t, l.ID, data.ID)
	assert.Equal(t, "+14152229670", data.Number.E164)
	assert.Equal(t, []string{"local", "numverify"}, data.ScannersRequested)
	require.Len(t, data.Results, 2)

	// Ordered by started_at: success (verbatim raw) then error (raw null + message).
	assert.Equal(t, "local", data.Results[0].Scanner)
	assert.JSONEq(t, `{"valid":true}`, string(data.Results[0].Raw))
	assert.Equal(t, "numverify", data.Results[1].Scanner)
	assert.Equal(t, store.ResultStatusError, data.Results[1].Status)
	assert.Equal(t, "quota exceeded", data.Results[1].ErrorMessage)
	assert.Nil(t, data.Results[1].Raw)
}

func TestGetLookupHandlerNotFound(t *testing.T) {
	newLookupTestStore(t)

	ctx := newParamContext(t, gin.Params{{Key: "id", Value: "missing"}})
	resp := handlers.GetLookup(ctx)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestGetLookupHandlerWithoutStore(t *testing.T) {
	handlers.InitStore(nil)

	ctx := newParamContext(t, gin.Params{{Key: "id", Value: "x"}})
	resp := handlers.GetLookup(ctx)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

// newGetContext builds a gin.Context for a GET request at target (may include a query).
func newGetContext(t *testing.T, target string) *gin.Context {
	t.Helper()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, target, nil)
	return ctx
}

func TestGetLatestLookupHandler(t *testing.T) {
	s := newLookupTestStore(t)
	_ = seedLookup(t, s, "local")
	time.Sleep(2 * time.Millisecond)
	newer := seedLookup(t, s, "local")

	ctx := newGetContext(t, "/v2/lookups/latest?number=14152229670")
	resp := handlers.GetLatestLookup(ctx)

	require.Equal(t, http.StatusOK, resp.Code)
	data, ok := resp.Data.(handlers.LookupDetailResponse)
	require.True(t, ok, "unexpected response type %T", resp.Data)
	assert.Equal(t, newer.ID, data.ID, "latest should be the newest lookup")
	assert.Equal(t, "+14152229670", data.Number.E164)
}

func TestGetLatestLookupHandlerMissingNumber(t *testing.T) {
	newLookupTestStore(t)
	resp := handlers.GetLatestLookup(newGetContext(t, "/v2/lookups/latest"))
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGetLatestLookupHandlerInvalidNumber(t *testing.T) {
	newLookupTestStore(t)
	resp := handlers.GetLatestLookup(newGetContext(t, "/v2/lookups/latest?number=abc"))
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGetLatestLookupHandlerNone(t *testing.T) {
	newLookupTestStore(t)
	// Valid number, but no lookups stored for it.
	resp := handlers.GetLatestLookup(newGetContext(t, "/v2/lookups/latest?number=447700900123"))
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestGetLatestLookupHandlerWithoutStore(t *testing.T) {
	handlers.InitStore(nil)
	resp := handlers.GetLatestLookup(newGetContext(t, "/v2/lookups/latest?number=14152229670"))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

// seedLookupFor seeds a lookup for an explicit e164/input pair.
func seedLookupFor(t *testing.T, s *store.SQLiteStore, e164, input string) store.Lookup {
	t.Helper()
	l, err := s.CreateLookup(context.Background(), store.Lookup{
		E164: e164, NumberInput: input, Valid: true, ScannersRequested: []string{"local"},
	})
	require.NoError(t, err)
	return l
}

func TestListLookupsHandlerOrderingAndLimit(t *testing.T) {
	s := newLookupTestStore(t)
	var ids []string
	for i := 0; i < 3; i++ {
		ids = append(ids, seedLookup(t, s, "local").ID)
		time.Sleep(2 * time.Millisecond)
	}

	ctx := newGetContext(t, "/v2/lookups?number=14152229670&limit=2")
	resp := handlers.ListLookups(ctx)

	require.Equal(t, http.StatusOK, resp.Code)
	data, ok := resp.Data.(handlers.ListLookupsResponse)
	require.True(t, ok, "unexpected response type %T", resp.Data)
	require.Len(t, data.Lookups, 2, "limit should cap results")
	// Newest first.
	assert.Equal(t, ids[2], data.Lookups[0].ID)
	assert.Equal(t, ids[1], data.Lookups[1].ID)
	assert.Equal(t, store.StatusPending, data.Lookups[0].Status)
}

func TestListLookupsHandlerScopedToNumber(t *testing.T) {
	s := newLookupTestStore(t)
	seedLookup(t, s, "local")                            // numberA (+14152229670)
	seedLookup(t, s, "local")                            // numberA
	seedLookupFor(t, s, "+447700900123", "447700900123") // a different number

	ctx := newGetContext(t, "/v2/lookups?number=14152229670")
	resp := handlers.ListLookups(ctx)

	require.Equal(t, http.StatusOK, resp.Code)
	data := resp.Data.(handlers.ListLookupsResponse)
	require.Len(t, data.Lookups, 2, "must return only the queried number's lookups (AC10)")
	for _, l := range data.Lookups {
		assert.Equal(t, "+14152229670", l.E164)
	}
}

func TestListLookupsHandlerMissingNumber(t *testing.T) {
	newLookupTestStore(t)
	resp := handlers.ListLookups(newGetContext(t, "/v2/lookups"))
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestListLookupsHandlerWithoutStore(t *testing.T) {
	handlers.InitStore(nil)
	resp := handlers.ListLookups(newGetContext(t, "/v2/lookups?number=14152229670"))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}
