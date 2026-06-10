package remote_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/sundowndev/phoneinfoga/v2/lib/filter"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
	"testing"
)

func TestGoogleSearchScanner_Metadata(t *testing.T) {
	scanner := remote.NewGoogleSearchScanner()
	assert.Equal(t, remote.Googlesearch, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestGoogleSearchScanner(t *testing.T) {
	n, _ := number.NewNumber("15556661212")
	scanner := remote.NewGoogleSearchScanner()
	lib := remote.NewLibrary(filter.NewEngine())
	lib.AddScanner(scanner)

	assert.NoError(t, scanner.DryRun(*n, remote.ScannerOptions{}))

	got, errs := lib.Scan(n, remote.ScannerOptions{})
	assert.Len(t, errs, 0)

	response, ok := got["googlesearch"].(remote.GoogleSearchResponse)
	assert.True(t, ok)
	assert.Len(t, response.General, 5)
	assert.Len(t, response.SocialMedia, 8)
	assert.Len(t, response.DisposableProviders, 6)
	assert.Len(t, response.Reputation, 8)
	assert.Len(t, response.Individuals, 8)

	assert.Contains(t, response.General[0].Dork, `"+15556661212"`)
	assert.Contains(t, response.General[0].Dork, `"15556661212"`)
	assert.Contains(t, response.General[0].Dork, `"5556661212"`)
	assert.Contains(t, response.General[0].URL, "https://www.google.com/search?q=")
	assert.Contains(t, response.SocialMedia[0].Dork, "site:facebook.com")
	assert.Contains(t, response.Reputation[1].Dork, `"spam" OR "scam"`)
}
