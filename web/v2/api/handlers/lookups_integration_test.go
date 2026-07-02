package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/handlers"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/server"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// doJSON issues a request through the router and returns the recorder.
func doJSON(t *testing.T, r http.Handler, method, target string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestLookupsEndToEnd walks the full lifecycle through the router: create -> close -> get
// -> latest -> list.
func TestLookupsEndToEnd(t *testing.T) {
	newLookupTestStore(t)
	r := server.NewServer()

	// Create.
	w := doJSON(t, r, http.MethodPost, "/v2/lookups", handlers.CreateLookupInput{
		Number: "14152229670", Scanners: []string{"local"},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created handlers.CreateLookupResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotEmpty(t, created.ID)
	assert.Equal(t, store.StatusPending, created.Status)

	// Close.
	w = doJSON(t, r, http.MethodPost, "/v2/lookups/"+created.ID+"/close", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var closed handlers.CloseLookupResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &closed))
	// No scanner ran, so a requested scanner is missing -> partial.
	assert.Equal(t, store.StatusPartial, closed.Status)
	assert.NotNil(t, closed.CompletedAt)

	// Get detail.
	w = doJSON(t, r, http.MethodGet, "/v2/lookups/"+created.ID, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var detail handlers.LookupDetailResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	assert.Equal(t, created.ID, detail.ID)
	assert.Equal(t, "+14152229670", detail.Number.E164)

	// Latest.
	w = doJSON(t, r, http.MethodGet, "/v2/lookups/latest?number=14152229670", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var latest handlers.LookupDetailResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &latest))
	assert.Equal(t, created.ID, latest.ID)

	// List.
	w = doJSON(t, r, http.MethodGet, "/v2/lookups?number=14152229670", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list handlers.ListLookupsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list.Lookups, 1)
	assert.Equal(t, created.ID, list.Lookups[0].ID)
}

func TestLookupsRouterBadRequests(t *testing.T) {
	newLookupTestStore(t)
	r := server.NewServer()

	// Missing number query parameter -> 400.
	assert.Equal(t, http.StatusBadRequest, doJSON(t, r, http.MethodGet, "/v2/lookups", nil).Code)
	assert.Equal(t, http.StatusBadRequest, doJSON(t, r, http.MethodGet, "/v2/lookups/latest", nil).Code)
	// Invalid number in body -> 400.
	assert.Equal(t, http.StatusBadRequest,
		doJSON(t, r, http.MethodPost, "/v2/lookups", map[string]string{"number": "abc"}).Code)
}

func TestLookupsRouterNotFound(t *testing.T) {
	newLookupTestStore(t)
	r := server.NewServer()

	assert.Equal(t, http.StatusNotFound, doJSON(t, r, http.MethodGet, "/v2/lookups/does-not-exist", nil).Code)
	assert.Equal(t, http.StatusNotFound, doJSON(t, r, http.MethodPost, "/v2/lookups/does-not-exist/close", nil).Code)
	// Valid number with no stored lookups -> 404 on latest.
	assert.Equal(t, http.StatusNotFound, doJSON(t, r, http.MethodGet, "/v2/lookups/latest?number=447700900123", nil).Code)
}

// TestLookupsNoCrossNumberLeakage covers AC10: history endpoints only ever return the
// queried number's lookups.
func TestLookupsNoCrossNumberLeakage(t *testing.T) {
	s := newLookupTestStore(t)
	r := server.NewServer()

	seedLookupFor(t, s, "+14152229670", "14152229670")
	seedLookupFor(t, s, "+14152229670", "14152229670")
	other := seedLookupFor(t, s, "+447700900123", "447700900123")

	// List for number A must not include number B's lookups.
	w := doJSON(t, r, http.MethodGet, "/v2/lookups?number=14152229670", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var list handlers.ListLookupsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list.Lookups, 2)
	for _, l := range list.Lookups {
		assert.Equal(t, "+14152229670", l.E164)
		assert.NotEqual(t, other.ID, l.ID)
	}

	// Latest for number B must be B's lookup, never A's.
	w = doJSON(t, r, http.MethodGet, "/v2/lookups/latest?number=447700900123", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var latest handlers.LookupDetailResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &latest))
	assert.Equal(t, other.ID, latest.ID)
	assert.Equal(t, "+447700900123", latest.Number.E164)
}

// TestLookupsBackwardCompatibleRoutes verifies a pre-existing endpoint still responds
// (AddNumber does not depend on the store).
func TestLookupsBackwardCompatibleRoutes(t *testing.T) {
	newLookupTestStore(t)
	r := server.NewServer()

	w := doJSON(t, r, http.MethodPost, "/v2/numbers", map[string]string{"number": "14152229670"})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
