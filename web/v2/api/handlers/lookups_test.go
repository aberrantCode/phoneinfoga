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
