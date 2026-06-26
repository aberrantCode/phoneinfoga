package suppliers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// SearchResultItem is one organic search hit returned by a search backend.
type SearchResultItem struct {
	Title   string
	URL     string
	Content string
	Engine  string
}

// SearchBackendInterface executes a single query against a named engine and
// returns a result count and the organic hits.
type SearchBackendInterface interface {
	Search(apiKey, engine, query string) (int, []SearchResultItem, error)
}

type SerpAPISupplier struct {
	BaseURL string
}

const serpAPIDefaultBaseURL = "https://serpapi.com/search.json"

func NewSerpAPISupplier() *SerpAPISupplier {
	return &SerpAPISupplier{BaseURL: serpAPIDefaultBaseURL}
}

type serpAPIResponse struct {
	SearchInformation struct {
		TotalResults int `json:"total_results"`
	} `json:"search_information"`
	OrganicResults []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
		Source  string `json:"source"`
	} `json:"organic_results"`
	Error string `json:"error"`
}

func (s *SerpAPISupplier) Search(apiKey, engine, query string) (int, []SearchResultItem, error) {
	logrus.WithField("engine", engine).WithField("query", query).Debug("Running search through SerpAPI")

	values := url.Values{}
	values.Set("engine", engine)
	values.Set("q", query)
	values.Set("api_key", apiKey)
	endpoint := fmt.Sprintf("%s?%s", s.BaseURL, values.Encode())

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, nil, fmt.Errorf("SerpAPI returned HTTP %d", resp.StatusCode)
	}

	var payload serpAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, nil, err
	}
	if payload.Error != "" {
		return 0, nil, fmt.Errorf("SerpAPI error: %s", payload.Error)
	}

	items := make([]SearchResultItem, 0, len(payload.OrganicResults))
	for _, r := range payload.OrganicResults {
		items = append(items, SearchResultItem{
			Title:   r.Title,
			URL:     r.Link,
			Content: r.Snippet,
			Engine:  engine,
		})
	}

	count := payload.SearchInformation.TotalResults
	if count == 0 {
		count = len(items)
	}
	return count, items, nil
}
