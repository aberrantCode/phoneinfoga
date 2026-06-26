# Feature Spec — Search-Backend Scanner (SerpAPI)

| | |
|---|---|
| **Scanner ID** | `serpapi` |
| **Category** | Web footprint (search execution) |
| **Status** | Implemented |
| **External dependency** | SerpAPI (unified Google / Bing / DuckDuckGo / Yandex) |
| **Auth** | API key |
| **Default state** | Skipped unless an API key is configured |
| **Supplier** | Yes (`SearchBackendInterface`) |

---

## 1. Summary

Add a `serpapi` scanner that executes the framework's existing phone-number dorks
against additional search indices (Bing, DuckDuckGo, Yandex, and Google) through
SerpAPI, returning result counts and top hits per category. It is the direct
sibling of the `searxng` and `googlecse` scanners — same dorks, different engines —
and reuses the dork generators already defined in `googlesearch_scanner.go` without
modification.

## 2. Motivation & gap filled

The footprint side of the tool currently executes dorks through SearXNG
(self-hosted meta-search) or Google CSE. Different engines index different corners
of the web, so a number's footprint on Bing or Yandex can differ from Google's. The
`serpapi` scanner widens coverage with minimal new code because the dork engine is
already factored out.

Crucially, this reuses the shared dork builders verbatim:

```go
getSocialMediaDorks(n)
getDisposableProvidersDorks(n)
getReputationDorks(n)
getIndividualsDorks(n)
getGeneralDorks(n)
```

The same five categories the `googlesearch` and `searxng` scanners already produce.

## 3. Goals / Non-goals

**Goals**
- Execute the existing categorized dorks against one or more engines via SerpAPI.
- Return the same response shape as the `searxng` scanner (categories → query →
  count → top results) so the UI can render it identically.
- Make the engine selectable (`google` / `bing` / `duckduckgo` / `yandex`).

**Non-goals**
- Inventing new dorks (covered by the shared generators; changes there benefit all
  search scanners).
- Scraping engines directly / bypassing their terms — SerpAPI is the sanctioned
  access layer and absorbs CAPTCHA/blocking concerns.

## 4. Data source

**Endpoint**
```
GET https://serpapi.com/search.json?engine={engine}&q={dork}&api_key={key}
```

**Fields consumed:** `search_information.total_results` (count) and the
`organic_results[]` array (`title`, `link`, `snippet`, `source`). Mapped into the
same item shape used by `searxng`.

**Cost / limits:** SerpAPI bills per search; the scanner issues one search per dork
per enabled engine. Reuse the existing inter-request delay pattern
(`SEARXNG_DELAY_MS` has an analogue here) to control spend and rate.

## 5. User-facing behavior

```bash
export SERPAPI_KEY="xxxx"
export SERPAPI_ENGINES="bing,duckduckgo"   # default: google
export SERPAPI_MAX_RESULTS="3"
export SERPAPI_DELAY_MS="200"
phoneinfoga scan -n "+14159929960"
```

Output mirrors the SearXNG scanner: each category lists its queries with a result
count and up to `SERPAPI_MAX_RESULTS` inline hits, plus a browser URL for the query.

## 6. Scanner design

```go
const SerpAPI = "serpapi"
```

- **`Name()`** → `"serpapi"`
- **`Description()`** → `"Execute phone-number footprint dorks across search engines via SerpAPI."`
- **`DryRun(n, opts)`** — return error if `SERPAPI_KEY` is empty.
- **`Run(n, opts)`** — for each enabled engine, run the five dork categories through
  `supplier.Search(engine, dork)`, applying the configured delay between requests
  (reuse the `searxng` delay/`MaxResults` handling). Aggregate into the response.

Refactor opportunity: extract the dork-execution loop currently inside
`searxng_scanner.go` (`searchDorks`) into a small shared helper that takes a
`SearchBackendInterface`, so `searxng` and `serpapi` share one execution path.

## 7. Response schema

Reuse the SearXNG response types (rename generically or alias):

```go
type SearchResultItem struct {
    Title   string `json:"title,omitempty" console:"Title,omitempty"`
    URL     string `json:"url,omitempty" console:"URL,omitempty"`
    Content string `json:"content,omitempty" console:"Content,omitempty"`
    Engine  string `json:"engine,omitempty" console:"Engine,omitempty"`
}

type SearchQueryResult struct {
    Number      string             `json:"number" console:"-"`
    Dork        string             `json:"dork" console:"Query"`
    URL         string             `json:"url" console:"URL"`
    Engine      string             `json:"engine,omitempty" console:"Engine,omitempty"`
    ResultCount int                `json:"result_count" console:"Result count"`
    Results     []SearchResultItem `json:"results,omitempty" console:"Results,omitempty"`
    Error       string             `json:"error,omitempty" console:"Error,omitempty"`
}

type SerpAPIScannerResponse struct {
    SocialMedia         []*SearchQueryResult `json:"social_media" console:"Social media,omitempty"`
    DisposableProviders []*SearchQueryResult `json:"disposable_providers" console:"Disposable providers,omitempty"`
    Reputation          []*SearchQueryResult `json:"reputation" console:"Reputation,omitempty"`
    Individuals         []*SearchQueryResult `json:"individuals" console:"Individuals,omitempty"`
    General             []*SearchQueryResult `json:"general" console:"General,omitempty"`
}
```

## 8. Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `SERPAPI_KEY` | yes | — | SerpAPI key. |
| `SERPAPI_ENGINES` | no | `google` | Comma-separated engines: `google,bing,duckduckgo,yandex`. |
| `SERPAPI_MAX_RESULTS` | no | `3` | Inline hits kept per query. |
| `SERPAPI_DELAY_MS` | no | `0` | Delay between searches to control rate/spend. |

## 9. Supplier interface & testing

```go
type SearchBackendInterface interface {
    Search(engine string, query string) (count int, results []SearchResultItem, err error)
}
```

- Concrete `SerpAPISupplier` in `lib/remote/suppliers/serpapi.go`; generate
  `mocks/SearchBackend.go`.
- Tests (reuse `searxng_scanner_test` structure):
  - missing key → `DryRun` error.
  - multi-engine happy path (stub two engines).
  - one engine erroring → recorded on the query, other categories still render.

## 10. Error handling

- Per-query/per-engine errors are recorded on the query (`Error` field), never
  failing the whole run — identical to the `searxng` behavior.
- HTTP ≥ 400 from SerpAPI → query-level error string.

## 11. Security / privacy / compliance

- This scanner only runs searches; it surfaces nothing the open web does not already
  expose. Lowest-sensitivity of the proposed additions.
- Using SerpAPI (rather than scraping) keeps engine ToS and CAPTCHA handling on the
  vendor side.

## 12. Dependencies

- `net/http` only. No new module.

## 13. Open questions / future work

- Allow direct engine APIs (Bing Web Search, etc.) as alternative
  `SearchBackendInterface` implementations for operators who already hold those keys.
- Once the shared dork-execution helper exists, fold `googlecse` onto it as well so
  all three search scanners share one path.
