# SearXNG Scanner

The SearXNG scanner runs the same phone-number footprint queries used by the Google Search scanner, but executes them through a configured SearXNG instance. It returns result counts and a small set of top results for each query, so the web UI can show which searches are worth opening.

## PhoneInfoga Configuration

The scanner is registered as `searxng`.

Supported environment variables:

- `SEARXNG_URL`: backend API URL used by PhoneInfoga to call SearXNG. Use an internal or unauthenticated URL when the public SearXNG hostname is behind SSO. Default in this fork: `http://192.168.30.141:8080`.
- `SEARXNG_PUBLIC_URL`: browser URL used in result links and "Open Matches" actions. Default in this fork: `https://searxng.svc.opbta.com`.
- `SEARXNG_MAX_RESULTS`: number of inline top results to keep per query. Default: `3`.
- `SEARXNG_DELAY_MS`: delay in milliseconds between SearXNG query requests. Default: `0` from the server environment. The web UI can override this per run from the browser preferences panel.

If the public SearXNG route is protected by Authentik, Authelia, or another SSO proxy, do not use that public route as `SEARXNG_URL` unless PhoneInfoga can authenticate to it. The scanner needs JSON from SearXNG, not an SSO login page. Use an internal container, bridge, macvlan, or service URL for `SEARXNG_URL`, and keep the public browser URL in `SEARXNG_PUBLIC_URL`.

## SearXNG Configuration

SearXNG must allow JSON output. In `settings.yml`, include `json` under `search.formats`:

```yaml
search:
  formats:
    - html
    - json
```

Keep `html` enabled for normal browser use. Add `json` so requests such as `/search?q=test&format=json` return JSON.

## Authenticated Public Route Pattern

For an SSO-protected deployment:

```powershell
$env:SEARXNG_URL = "http://192.168.30.141:8080"
$env:SEARXNG_PUBLIC_URL = "https://searxng.svc.opbta.com"
```

PhoneInfoga calls the internal URL for result counts. The UI opens the public URL, where the user can rely on their browser SSO session.

## Unauthenticated Route Pattern

For an unauthenticated SearXNG deployment, both values can point at the same URL:

```powershell
$env:SEARXNG_URL = "https://search.example.com"
$env:SEARXNG_PUBLIC_URL = "https://search.example.com"
```

## Runtime Behavior

For each phone number, the scanner evaluates grouped query categories:

- General footprints
- Social networks
- People and identity
- Reputation and spam
- Temporary number providers

Each row includes:

- the generated query
- the public SearXNG URL
- a result count
- up to `SEARXNG_MAX_RESULTS` inline results
- a per-query error if SearXNG could not return JSON

The scanner does not fail the whole run when one query errors. It records the error on that query so the rest of the categories can still render.

The web UI sends the configured SearXNG delay with each scanner run. This spaces out SearXNG requests and reduces pressure on the local SearXNG instance and its upstream engines. The UI can cancel the browser request while a scanner is running; cancellation stops waiting for the response in the browser, but a request already executing on the server may continue until the current scanner run returns.
