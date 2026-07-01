package suppliers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

// TestDehashedSupplierV2Request pins the v2 transport: a JSON POST to
// /v2/search authenticated with the DeHashed-Api-Key header, carrying the
// query in the body. The v1 endpoint (GET /search with Basic auth) now 404s.
func TestDehashedSupplierV2Request(t *testing.T) {
	defer gock.Off()

	query := `phone:"15556661212"`

	gock.New("https://api.dehashed.com").
		Post("/v2/search").
		// Go canonicalizes the header key on the wire; HTTP header names are
		// case-insensitive, so the real API accepts it either way.
		MatchHeader("Dehashed-Api-Key", "secret").
		MatchHeader("Content-Type", "application/json").
		// The query is carried in the JSON POST body (v1 put it in the URL).
		BodyString(`15556661212.*"page":1,"size":100`).
		Reply(200).
		JSON(map[string]interface{}{
			"success": true,
			"total":   1,
			"balance": 100,
			// v2 returns entry fields as arrays; DehashedEntry keeps them as
			// json.RawMessage so the array shape decodes without change.
			"entries": []map[string]interface{}{
				{
					"id":            "1",
					"database_name": "ExampleBreach",
					"email":         []string{"bob@example.com"},
					"phone":         []string{"15556661212"},
				},
			},
		})

	s := NewDehashedSupplier()
	got, err := s.SearchByPhone("a@b.com", "secret", query)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, 1, got.Total)
	assert.Len(t, got.Entries, 1)
	assert.Equal(t, "ExampleBreach", got.Entries[0].DatabaseName)
	assert.True(t, gock.IsDone(), "the v2 request should have matched the mock")
}

func TestDehashedSupplierUnauthorized(t *testing.T) {
	defer gock.Off()

	gock.New("https://api.dehashed.com").Post("/v2/search").Reply(401)

	s := NewDehashedSupplier()
	got, err := s.SearchByPhone("a@b.com", "bad-key", `phone:"1"`)

	assert.Nil(t, got)
	assert.ErrorContains(t, err, "DEHASHED_API_KEY")
}

func TestDehashedSupplierHTTPError(t *testing.T) {
	defer gock.Off()

	gock.New("https://api.dehashed.com").Post("/v2/search").Reply(404)

	s := NewDehashedSupplier()
	got, err := s.SearchByPhone("a@b.com", "secret", `phone:"1"`)

	assert.Nil(t, got)
	assert.EqualError(t, err, "dehashed search returned HTTP 404")
}

func TestDehashedSupplierUnsuccessfulEnvelope(t *testing.T) {
	defer gock.Off()

	// Dehashed reports success:false on quota/auth failures while returning 200.
	gock.New("https://api.dehashed.com").Post("/v2/search").Reply(200).
		JSON(map[string]interface{}{"success": false, "message": "invalid api key"})

	s := NewDehashedSupplier()
	got, err := s.SearchByPhone("a@b.com", "secret", `phone:"1"`)

	assert.Nil(t, got)
	assert.ErrorContains(t, err, "invalid api key")
}
