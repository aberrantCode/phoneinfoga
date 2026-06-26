package remote_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
	"github.com/sundowndev/phoneinfoga/v2/mocks"
)

func TestSerpAPIScanner_Metadata(t *testing.T) {
	scanner := remote.NewSerpAPIScanner(&mocks.SearchBackend{})
	assert.Equal(t, remote.SerpAPI, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestSerpAPIScanner(t *testing.T) {
	num := func() *number.Number { n, _ := number.NewNumber("15556661212"); return n }()

	t.Run("skipped without api key", func(t *testing.T) {
		m := &mocks.SearchBackend{}

		scanner := remote.NewSerpAPIScanner(m)
		lib := remote.NewLibrary(filter.NewEngine())
		lib.AddScanner(scanner)

		got, errs := lib.Scan(num, map[string]interface{}{})
		assert.Len(t, errs, 0)
		assert.Equal(t, map[string]interface{}{}, got)

		m.AssertExpectations(t)
	})

	t.Run("runs dorks across the default engine", func(t *testing.T) {
		m := &mocks.SearchBackend{}
		m.On("Search", "key", "google", mock.Anything).
			Return(2, []suppliers.SearchResultItem{
				{Title: "Example", URL: "https://example.com", Content: "snippet", Engine: "google"},
			}, nil)

		scanner := remote.NewSerpAPIScanner(m)
		lib := remote.NewLibrary(filter.NewEngine())
		lib.AddScanner(scanner)

		got, errs := lib.Scan(num, map[string]interface{}{"SERPAPI_KEY": "key"})
		assert.Len(t, errs, 0)

		raw, ok := got["serpapi"]
		assert.True(t, ok)

		resp, ok := raw.(remote.SerpAPIScannerResponse)
		assert.True(t, ok)

		// Dork generators yield a fixed number of queries per category.
		assert.Len(t, resp.SocialMedia, 8)
		assert.Len(t, resp.General, 5)

		first := resp.SocialMedia[0]
		assert.Equal(t, "google", first.Engine)
		assert.Equal(t, 2, first.ResultCount)
		assert.Len(t, first.Results, 1)
		assert.Equal(t, "Example", first.Results[0].Title)

		m.AssertExpectations(t)
	})
}
