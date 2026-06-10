package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
)

const SearXNG = "searxng"

const defaultSearXNGAPIURL = "http://192.168.30.141:8080"
const defaultSearXNGPublicURL = "https://searxng.svc.opbta.com"

type searxngScanner struct {
	MaxResults int
	Delay      time.Duration
	httpClient *http.Client
}

type SearXNGResultItem struct {
	Title   string `json:"title,omitempty" console:"Title,omitempty"`
	URL     string `json:"url,omitempty" console:"URL,omitempty"`
	Content string `json:"content,omitempty" console:"Content,omitempty"`
	Engine  string `json:"engine,omitempty" console:"Engine,omitempty"`
}

type SearXNGQueryResult struct {
	Number      string              `json:"number" console:"-"`
	Dork        string              `json:"dork" console:"Query"`
	URL         string              `json:"url" console:"URL"`
	ResultCount int                 `json:"result_count" console:"Result count"`
	Results     []SearXNGResultItem `json:"results,omitempty" console:"Results,omitempty"`
	Error       string              `json:"error,omitempty" console:"Error,omitempty"`
}

type SearXNGScannerResponse struct {
	SocialMedia         []*SearXNGQueryResult `json:"social_media" console:"Social media,omitempty"`
	DisposableProviders []*SearXNGQueryResult `json:"disposable_providers" console:"Disposable providers,omitempty"`
	Reputation          []*SearXNGQueryResult `json:"reputation" console:"Reputation,omitempty"`
	Individuals         []*SearXNGQueryResult `json:"individuals" console:"Individuals,omitempty"`
	General             []*SearXNGQueryResult `json:"general" console:"General,omitempty"`
}

type searxngSearchResponse struct {
	NumberOfResults int `json:"number_of_results"`
	Results         []struct {
		Title   string   `json:"title"`
		URL     string   `json:"url"`
		Content string   `json:"content"`
		Engine  string   `json:"engine"`
		Engines []string `json:"engines"`
	} `json:"results"`
}

func NewSearXNGScanner(HTTPclient *http.Client) Scanner {
	maxResults := 3
	if v := os.Getenv("SEARXNG_MAX_RESULTS"); v != "" {
		val, err := strconv.Atoi(v)
		if err == nil && val >= 0 {
			maxResults = val
		}
	}

	return &searxngScanner{
		MaxResults: maxResults,
		Delay:      getSearXNGDelayFromEnv(),
		httpClient: HTTPclient,
	}
}

func (s *searxngScanner) Name() string {
	return SearXNG
}

func (s *searxngScanner) Description() string {
	return "SearXNG searches for web footprints of a given phone number and returns counts with top results."
}

func (s *searxngScanner) DryRun(_ number.Number, opts ScannerOptions) error {
	if s.apiURL(opts) == "" {
		return errors.New("SearXNG URL is not defined")
	}
	return nil
}

func (s *searxngScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	apiURL := s.apiURL(opts)
	publicURL := s.publicURL(opts)
	delay := s.delay(opts)
	if apiURL == "" {
		return nil, errors.New("SearXNG URL is not defined")
	}

	return SearXNGScannerResponse{
		SocialMedia:         s.searchDorks(apiURL, publicURL, delay, getSocialMediaDorks(n)),
		DisposableProviders: s.searchDorks(apiURL, publicURL, delay, getDisposableProvidersDorks(n)),
		Reputation:          s.searchDorks(apiURL, publicURL, delay, getReputationDorks(n)),
		Individuals:         s.searchDorks(apiURL, publicURL, delay, getIndividualsDorks(n)),
		General:             s.searchDorks(apiURL, publicURL, delay, getGeneralDorks(n)),
	}, nil
}

func (s *searxngScanner) apiURL(opts ScannerOptions) string {
	if value := strings.TrimSpace(opts.GetStringEnv("SEARXNG_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultSearXNGAPIURL
}

func (s *searxngScanner) publicURL(opts ScannerOptions) string {
	if value := strings.TrimSpace(opts.GetStringEnv("SEARXNG_PUBLIC_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return defaultSearXNGPublicURL
}

func (s *searxngScanner) delay(opts ScannerOptions) time.Duration {
	if value, ok := opts["SEARXNG_DELAY_MS"]; ok {
		switch typed := value.(type) {
		case float64:
			if typed >= 0 {
				return time.Duration(typed) * time.Millisecond
			}
		case int:
			if typed >= 0 {
				return time.Duration(typed) * time.Millisecond
			}
		case string:
			if parsed, err := strconv.Atoi(typed); err == nil && parsed >= 0 {
				return time.Duration(parsed) * time.Millisecond
			}
		}
	}

	return s.Delay
}

func (s *searxngScanner) searchDorks(apiURL string, publicURL string, delay time.Duration, dorks []*GoogleSearchDork) []*SearXNGQueryResult {
	results := make([]*SearXNGQueryResult, 0, len(dorks))
	for index, dork := range dorks {
		if index > 0 && delay > 0 {
			time.Sleep(delay)
		}

		result := &SearXNGQueryResult{
			Number: dork.Number,
			Dork:   dork.Dork,
			URL:    s.searchURL(publicURL, dork.Dork, false),
		}

		count, items, err := s.search(apiURL, dork.Dork)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.ResultCount = count
			result.Results = items
		}

		results = append(results, result)
	}

	return results
}

func getSearXNGDelayFromEnv() time.Duration {
	if v := os.Getenv("SEARXNG_DELAY_MS"); v != "" {
		val, err := strconv.Atoi(v)
		if err == nil && val >= 0 {
			return time.Duration(val) * time.Millisecond
		}
	}
	return 0
}

func (s *searxngScanner) search(baseURL string, query string) (int, []SearXNGResultItem, error) {
	client := s.httpClient
	if client == nil {
		client = &http.Client{
			Timeout: 20 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	req, err := http.NewRequest(http.MethodGet, s.searchURL(baseURL, query, true), nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
			return 0, nil, fmt.Errorf("SearXNG returned HTTP %d redirect to %s; use an internal API URL or provide scanner auth", resp.StatusCode, resp.Header.Get("Location"))
		}
		return 0, nil, fmt.Errorf("SearXNG returned HTTP %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(strings.ToLower(contentType), "json") {
		return 0, nil, errors.New("SearXNG JSON response was not returned; enable json in search.formats")
	}

	var payload searxngSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, nil, err
	}

	count := len(payload.Results)
	if payload.NumberOfResults > count {
		count = payload.NumberOfResults
	}

	items := make([]SearXNGResultItem, 0, minInt(len(payload.Results), s.MaxResults))
	for i, item := range payload.Results {
		if s.MaxResults >= 0 && i >= s.MaxResults {
			break
		}
		engine := item.Engine
		if engine == "" && len(item.Engines) > 0 {
			engine = strings.Join(item.Engines, ", ")
		}
		items = append(items, SearXNGResultItem{
			Title:   item.Title,
			URL:     item.URL,
			Content: item.Content,
			Engine:  engine,
		})
	}

	return count, items, nil
}

func (s *searxngScanner) searchURL(baseURL string, query string, jsonFormat bool) string {
	values := url.Values{}
	values.Set("q", query)
	if jsonFormat {
		values.Set("format", "json")
	}

	return fmt.Sprintf("%s/search?%s", strings.TrimRight(baseURL, "/"), values.Encode())
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
