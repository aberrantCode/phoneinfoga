package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"github.com/sundowndev/phoneinfoga/v2/mocks"
	"github.com/sundowndev/phoneinfoga/v2/test"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/handlers"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/server"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

type persistScanResponse struct {
	Info string `json:"info"`
}

// seedRunLookup creates a lookup the /run persistence can attach results to.
func seedRunLookup(t *testing.T, s *store.SQLiteStore) store.Lookup {
	t.Helper()
	l, err := s.CreateLookup(context.Background(), store.Lookup{
		E164: "+14152229670", NumberInput: "14152229670", Valid: true,
		ScannersRequested: []string{"fakeScanner"},
	})
	require.NoError(t, err)
	return l
}

func withFakeScanner(mock func(*mocks.Scanner)) {
	s := &mocks.Scanner{}
	s.On("Name").Return("fakeScanner")
	mock(s)
	handlers.RemoteLibrary = remote.NewLibrary(filter.NewEngine())
	handlers.RemoteLibrary.AddScanner(s)
}

func TestRunScannerPersistsSuccessWithLookupID(t *testing.T) {
	s := newLookupTestStore(t)
	l := seedRunLookup(t, s)
	withFakeScanner(func(m *mocks.Scanner) {
		m.On("Run", *test.NewFakeUSNumber(), remote.ScannerOptions{}).
			Return(persistScanResponse{Info: "test"}, nil)
	})

	w := doJSON(t, server.NewServer(), http.MethodPost, "/v2/scanners/fakeScanner/run",
		handlers.RunScannerInput{Number: "14152229670", LookupID: l.ID})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	got, err := s.GetLookup(context.Background(), l.ID)
	require.NoError(t, err)
	require.Len(t, got.Results, 1, "a scanner_results row should be persisted")
	r := got.Results[0]
	assert.Equal(t, "fakeScanner", r.Scanner)
	assert.Equal(t, store.ResultStatusSuccess, r.Status)
	assert.JSONEq(t, `{"info":"test"}`, string(r.RawResponse))
	assert.GreaterOrEqual(t, r.DurationMs, int64(0))
	assert.False(t, r.StartedAt.IsZero())
	assert.False(t, r.FinishedAt.IsZero())
}

func TestRunScannerPersistsErrorWithLookupID(t *testing.T) {
	s := newLookupTestStore(t)
	l := seedRunLookup(t, s)
	withFakeScanner(func(m *mocks.Scanner) {
		m.On("Run", *test.NewFakeUSNumber(), remote.ScannerOptions{}).
			Return(nil, errors.New("dummy error"))
	})

	w := doJSON(t, server.NewServer(), http.MethodPost, "/v2/scanners/fakeScanner/run",
		handlers.RunScannerInput{Number: "14152229670", LookupID: l.ID})
	// Response is unchanged from today (AC13): scanner error still yields 500.
	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())

	got, err := s.GetLookup(context.Background(), l.ID)
	require.NoError(t, err)
	require.Len(t, got.Results, 1, "an error scanner_results row should be persisted")
	r := got.Results[0]
	assert.Equal(t, store.ResultStatusError, r.Status)
	assert.Equal(t, "dummy error", r.ErrorMessage)
	assert.Empty(t, r.RawResponse)
}

func TestRunScannerWithoutLookupIDPersistsNothing(t *testing.T) {
	s := newLookupTestStore(t)
	l := seedRunLookup(t, s)
	withFakeScanner(func(m *mocks.Scanner) {
		m.On("Run", *test.NewFakeUSNumber(), remote.ScannerOptions{}).
			Return(persistScanResponse{Info: "test"}, nil)
	})

	// No lookupId: behaves exactly as before and writes nothing (AC13).
	w := doJSON(t, server.NewServer(), http.MethodPost, "/v2/scanners/fakeScanner/run",
		handlers.RunScannerInput{Number: "14152229670"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	got, err := s.GetLookup(context.Background(), l.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Results, "no lookupId means no scanner_results row")
}
