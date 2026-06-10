package remote_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSearXNGScanner_Metadata(t *testing.T) {
	scanner := remote.NewSearXNGScanner(nil)
	assert.Equal(t, remote.SearXNG, scanner.Name())
	assert.NotEmpty(t, scanner.Description())
}

func TestSearXNGScanner_Run(t *testing.T) {
	var requestCount int
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			assert.Equal(t, "/search", req.URL.Path)
			assert.Equal(t, "json", req.URL.Query().Get("format"))
			assert.NotEmpty(t, req.URL.Query().Get("q"))

			body := `{"number_of_results":12,"results":[{"title":"Example hit","url":"https://example.com/hit","content":"Matched snippet","engine":"google"}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	n, _ := number.NewNumber("15556661212")
	scanner := remote.NewSearXNGScanner(client)
	require.NoError(t, scanner.DryRun(*n, remote.ScannerOptions{"SEARXNG_URL": "https://searxng.example"}))

	got, err := scanner.Run(*n, remote.ScannerOptions{"SEARXNG_URL": "https://searxng.example"})
	require.NoError(t, err)

	response, ok := got.(remote.SearXNGScannerResponse)
	require.True(t, ok)
	assert.Len(t, response.General, 5)
	assert.Len(t, response.SocialMedia, 8)
	assert.Len(t, response.DisposableProviders, 6)
	assert.Len(t, response.Reputation, 8)
	assert.Len(t, response.Individuals, 8)
	assert.Equal(t, 35, requestCount)
	assert.Equal(t, 12, response.General[0].ResultCount)
	assert.Equal(t, "https://example.com/hit", response.General[0].Results[0].URL)
	assert.Contains(t, response.General[0].URL, "https://searxng.svc.opbta.com/search?q=")
	assert.NotContains(t, response.General[0].URL, "format=json")
}

func TestSearXNGScanner_UsesPublicURLForOpenLinks(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "searxng-api.example:8080", req.URL.Host)

			body := `{"results":[]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	n, _ := number.NewNumber("15556661212")
	scanner := remote.NewSearXNGScanner(client)

	got, err := scanner.Run(*n, remote.ScannerOptions{
		"SEARXNG_URL":        "http://searxng-api.example:8080",
		"SEARXNG_PUBLIC_URL": "https://search.example",
	})
	require.NoError(t, err)

	response, ok := got.(remote.SearXNGScannerResponse)
	require.True(t, ok)
	assert.Contains(t, response.General[0].URL, "https://search.example/search?q=")
}

func TestSearXNGScanner_RecordsQueryErrors(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("<html></html>")),
			}, nil
		}),
	}

	n, _ := number.NewNumber("15556661212")
	scanner := remote.NewSearXNGScanner(client)

	got, err := scanner.Run(*n, remote.ScannerOptions{"SEARXNG_URL": "https://searxng.example"})
	require.NoError(t, err)

	response, ok := got.(remote.SearXNGScannerResponse)
	require.True(t, ok)
	require.NotEmpty(t, response.General)
	assert.Equal(t, "SearXNG JSON response was not returned; enable json in search.formats", response.General[0].Error)
}
