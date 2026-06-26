package remote

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/remote/suppliers"
)

const SerpAPI = "serpapi"

type serpapiScanner struct {
	client     suppliers.SearchBackendInterface
	MaxResults int
}

type SerpAPIResultItem struct {
	Title   string `json:"title,omitempty" console:"Title,omitempty"`
	URL     string `json:"url,omitempty" console:"URL,omitempty"`
	Content string `json:"content,omitempty" console:"Content,omitempty"`
	Engine  string `json:"engine,omitempty" console:"Engine,omitempty"`
}

type SerpAPIQueryResult struct {
	Number      string              `json:"number" console:"-"`
	Dork        string              `json:"dork" console:"Query"`
	Engine      string              `json:"engine,omitempty" console:"Engine,omitempty"`
	ResultCount int                 `json:"result_count" console:"Result count"`
	Results     []SerpAPIResultItem `json:"results,omitempty" console:"Results,omitempty"`
	Error       string              `json:"error,omitempty" console:"Error,omitempty"`
}

type SerpAPIScannerResponse struct {
	SocialMedia         []*SerpAPIQueryResult `json:"social_media" console:"Social media,omitempty"`
	DisposableProviders []*SerpAPIQueryResult `json:"disposable_providers" console:"Disposable providers,omitempty"`
	Reputation          []*SerpAPIQueryResult `json:"reputation" console:"Reputation,omitempty"`
	Individuals         []*SerpAPIQueryResult `json:"individuals" console:"Individuals,omitempty"`
	General             []*SerpAPIQueryResult `json:"general" console:"General,omitempty"`
}

func NewSerpAPIScanner(s suppliers.SearchBackendInterface) Scanner {
	maxResults := 3
	if v := os.Getenv("SERPAPI_MAX_RESULTS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil && val >= 0 {
			maxResults = val
		}
	}
	return &serpapiScanner{client: s, MaxResults: maxResults}
}

func (s *serpapiScanner) Name() string {
	return SerpAPI
}

func (s *serpapiScanner) Description() string {
	return "Execute phone-number footprint dorks across search engines via SerpAPI."
}

func (s *serpapiScanner) DryRun(_ number.Number, opts ScannerOptions) error {
	if opts.GetStringEnv("SERPAPI_KEY") == "" {
		return errors.New("API key is not defined")
	}
	return nil
}

func (s *serpapiScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	apiKey := opts.GetStringEnv("SERPAPI_KEY")
	engines := serpapiEngines(opts)

	return SerpAPIScannerResponse{
		SocialMedia:         s.searchDorks(apiKey, engines, getSocialMediaDorks(n)),
		DisposableProviders: s.searchDorks(apiKey, engines, getDisposableProvidersDorks(n)),
		Reputation:          s.searchDorks(apiKey, engines, getReputationDorks(n)),
		Individuals:         s.searchDorks(apiKey, engines, getIndividualsDorks(n)),
		General:             s.searchDorks(apiKey, engines, getGeneralDorks(n)),
	}, nil
}

func (s *serpapiScanner) searchDorks(apiKey string, engines []string, dorks []*GoogleSearchDork) []*SerpAPIQueryResult {
	results := make([]*SerpAPIQueryResult, 0, len(dorks)*len(engines))
	for _, dork := range dorks {
		for _, engine := range engines {
			result := &SerpAPIQueryResult{
				Number: dork.Number,
				Dork:   dork.Dork,
				Engine: engine,
			}

			count, items, err := s.client.Search(apiKey, engine, dork.Dork)
			if err != nil {
				result.Error = err.Error()
			} else {
				result.ResultCount = count
				result.Results = s.limitResults(items)
			}

			results = append(results, result)
		}
	}
	return results
}

func (s *serpapiScanner) limitResults(items []suppliers.SearchResultItem) []SerpAPIResultItem {
	out := make([]SerpAPIResultItem, 0, len(items))
	for i, item := range items {
		if s.MaxResults >= 0 && i >= s.MaxResults {
			break
		}
		out = append(out, SerpAPIResultItem{
			Title:   item.Title,
			URL:     item.URL,
			Content: item.Content,
			Engine:  item.Engine,
		})
	}
	return out
}

func serpapiEngines(opts ScannerOptions) []string {
	raw := opts.GetStringEnv("SERPAPI_ENGINES")
	if raw == "" {
		raw = "google"
	}
	var engines []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			engines = append(engines, part)
		}
	}
	if len(engines) == 0 {
		engines = []string{"google"}
	}
	return engines
}
