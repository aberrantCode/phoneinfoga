# Feature Spec — Breach / Leak Search Scanner

| | |
|---|---|
| **Scanner ID** | `breach` |
| **Category** | Footprint — breach/leak exposure (**sensitive**) |
| **Status** | Draft — proposed |
| **External dependency** | A vetted, authenticated breach-aggregation provider (e.g. Dehashed) |
| **Auth** | Account credential + API key |
| **Default state** | **Disabled by default.** Requires explicit opt-in **and** credentials. |
| **Supplier** | Yes (`BreachSupplierInterface`) |

---

## 1. Summary

Add a `breach` scanner that checks whether a phone number appears in known data
breaches/leaks, via a legitimate authenticated breach-aggregation provider. The
default result is **exposure presence**: which breach sources reference the number
and how many records, so an investigator or the number's owner can understand the
number's leak exposure.

This is the most sensitive scanner in the proposed set. The design treats privacy,
legality, and provider-AUP adherence as first-class requirements, not afterthoughts.
It is gated off by default and surfaces minimal data unless an operator explicitly
opts into more.

## 2. Motivation & gap filled

The framework already *points at* breach-adjacent sources through dorks (pastebin,
etc.) but cannot answer "does this number appear in known breach corpora." A
breach-aware scanner adds that signal for legitimate use cases — personal exposure
checks, security research, incident response, and fraud investigation — by querying
a provider that has lawfully aggregated and indexed breach data and that gates
access behind its own authentication and acceptable-use terms.

## 3. Goals / Non-goals

**Goals**
- Report breach/leak **presence** for a number: source/database names, record
  counts, and the *types* of fields present (e.g. "email present", "username
  present"), drawn from a vetted provider.
- Ship disabled by default; require an explicit enable flag plus credentials.
- Minimize data exposure by default; gate any retrieval of associated record fields
  behind a second explicit opt-in with a warning.

**Non-goals**
- **Scraping, circumventing, or aggregating leaked data outside a sanctioned
  provider.** The scanner only calls an authenticated provider that the operator is
  licensed to use; it never sources breach data by any other means.
- Dumping other people's credentials by default. Associated fields are other
  individuals' personal data and are not returned unless explicitly enabled.
- Bulk/automated enumeration of numbers (rate-limited, single-number scope).
- Any use for surveillance, stalking, doxxing, or harassment — explicitly out of
  scope and contrary to the project's stated non-features.

## 4. Data source

Designed around a provider that exposes an authenticated search API keyed by phone
number (Dehashed is the reference implementation):

```
POST https://api.dehashed.com/v2/search
DeHashed-Api-Key: {api_key}
Content-Type: application/json

{"query": "phone:{E164}", "page": 1, "size": 100}
```

> The retired v1 endpoint (`GET /search` with HTTP Basic `email:api_key` auth)
> now returns HTTP 404. v2 authenticates with the `DeHashed-Api-Key` header
> alone and returns entry field values as arrays.

**Default fields consumed (minimal mode):**

| Field | Use |
|---|---|
| `entries[].database_name` | Which breach/source references the number. |
| `total` | Count of matching records. |
| presence of `email` / `username` / `name` / `address` per entry | Reported as **field-type flags**, not values, in default mode. |

**Extended mode (opt-in, see §11):** the actual associated field values an entry
contains. Off unless `BREACH_INCLUDE_FIELDS=true`.

**Provider terms:** access, rate, and permitted use are governed by the provider's
AUP and the operator's subscription. The scanner must not exceed the provider's
documented rate limits and must pass through the operator's own credentials only.

## 5. User-facing behavior

```bash
# BOTH are required; absence of either disables the scanner
export BREACH_SCANNER_ENABLED="true"
export DEHASHED_EMAIL="account@example.com"
export DEHASHED_API_KEY="xxxx"

# Default (minimal): presence + source names + field-type flags only
phoneinfoga scan -n "+14159929960"

# Extended (explicit opt-in; surfaces leaked field VALUES — handle lawfully)
export BREACH_INCLUDE_FIELDS="true"
phoneinfoga scan -n "+14159929960"
```

Default output: a list of breach sources that reference the number, a record count,
and which field types appear — e.g. `HaveBeenSeen (3 records): email, username`.
No leaked values are shown in default mode.

## 6. Scanner design

```go
const Breach = "breach"
```

- **`Name()`** → `"breach"`
- **`Description()`** → `"Check whether a phone number appears in known breach corpora via an authenticated provider (disabled by default)."`
- **`DryRun(n, opts)`** — **double gate**, skip unless all hold:
  ```go
  if !truthy(opts.GetStringEnv("BREACH_SCANNER_ENABLED")) {
      return errors.New("breach scanner is disabled (set BREACH_SCANNER_ENABLED=true to opt in)")
  }
  if opts.GetStringEnv("DEHASHED_EMAIL") == "" || opts.GetStringEnv("DEHASHED_API_KEY") == "" {
      return errors.New("breach provider credentials are not defined")
  }
  ```
- **`Run(n, opts)`**:
  1. Query the provider for `phone:{n.E164}` (and, if the provider supports it, the
     national variant) via the supplier.
  2. Build the **minimal** response: source names, counts, field-type flags.
  3. Only if `BREACH_INCLUDE_FIELDS=true`, attach associated field values.
  4. Return the struct.

## 7. Response schema

```go
type BreachEntry struct {
    Source     string   `json:"source" console:"Source"`
    RecordCount int      `json:"record_count" console:"Records"`
    FieldTypes []string `json:"field_types,omitempty" console:"Field types,omitempty"`
    // Populated ONLY when BREACH_INCLUDE_FIELDS=true:
    Fields     map[string]string `json:"fields,omitempty" console:"-"`
}

type BreachScannerResponse struct {
    Found       bool          `json:"found" console:"Found"`
    TotalRecords int           `json:"total_records" console:"Total records"`
    Sources     []string      `json:"sources,omitempty" console:"Sources,omitempty"`
    Entries     []BreachEntry `json:"entries,omitempty" console:"Entries,omitempty"`
    Disclaimer  string        `json:"disclaimer,omitempty" console:"Note,omitempty"`
}
```

`Disclaimer` carries a short, always-present note that results are exposure
indicators from a third-party provider and must be used lawfully.

## 8. Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `BREACH_SCANNER_ENABLED` | yes (to run at all) | `false` | Master opt-in switch. |
| `DEHASHED_EMAIL` | yes | — | Provider account identifier (Basic auth). |
| `DEHASHED_API_KEY` | yes | — | Provider API key (Basic auth). |
| `BREACH_INCLUDE_FIELDS` | no | `false` | Opt-in to retrieve associated field **values** (third parties' PII). |
| `BREACH_API_URL` | no | provider default | Endpoint override. |

## 9. Supplier interface & testing

```go
type BreachSupplierInterface interface {
    SearchByPhone(e164 string, includeFields bool) (*BreachProviderResponse, error)
}
```

- Concrete supplier in `lib/remote/suppliers/breach.go`; generate
  `mocks/BreachSupplier.go`.
- Tests:
  - scanner disabled (`BREACH_SCANNER_ENABLED` unset) → `DryRun` error, even when
    credentials are present.
  - enabled but missing credentials → `DryRun` error.
  - enabled + credentials, no hits → `Found: false`.
  - enabled + credentials, hits, **minimal mode** → assert `Fields` is empty and
    only field-type flags are present (regression guard against leaking values by
    default).
  - extended mode only attaches values when explicitly enabled.
  - provider auth/rate error surfaced.

## 10. Error handling

- Provider HTTP ≥ 400 (auth failure, rate limit) → scan error carrying the provider
  message; the run continues for other scanners.
- No hits is a normal `Found: false`, not an error.

## 11. Security / privacy / compliance — **first-class requirements**

This section is normative, not advisory. A reviewer should block the feature if any
item is unmet.

- **Off by default.** The scanner must not run without the explicit
  `BREACH_SCANNER_ENABLED` opt-in *and* credentials (the double gate in `DryRun`).
- **Sanctioned source only.** Data comes solely from an authenticated provider the
  operator is licensed to use. The scanner must never scrape, mirror, or otherwise
  obtain breach data outside that provider, and must pass through only the
  operator's own credentials.
- **Data minimization by default.** Default mode returns presence, counts, and
  field *types* — never leaked values. Retrieving associated values requires the
  separate `BREACH_INCLUDE_FIELDS` opt-in and emits a warning; those values are
  third parties' personal data.
- **Log hygiene.** Never log credentials. Never write breach results (sources,
  counts, or values) to debug logs. Do not persist results to disk as part of normal
  operation.
- **Legal posture.** Querying breach corpora can constitute processing of personal
  and, in some records, sensitive data. Operators are responsible for a lawful basis
  (e.g. GDPR Art. 6/9 considerations, CCPA, and equivalent local law) and a
  permissible purpose. Document this in the scanner's user docs and surface the
  `Disclaimer` field on every result.
- **Aligned with project scope.** The project explicitly does not support tracking,
  surveilling, or de-anonymizing individuals for harassment. This scanner inherits
  that boundary: legitimate exposure assessment / security research only.
- **Rate discipline.** Respect the provider's documented limits; single-number
  scope; no bulk enumeration helpers shipped with the scanner.

## 12. Dependencies

- `net/http` with Basic auth only. No new module.

## 13. Open questions / future work

- Should the default provider be HaveIBeenPwned-style presence-only (which is even
  more minimal but historically email-centric) versus a phone-indexed provider?
  Evaluate coverage vs. minimization.
- Consider an audit hook (operator-supplied) recording that a breach query was run,
  by whom, and for what stated purpose — useful for teams that need an access trail.
- Consider redacting partial values even in extended mode (e.g. masking local-part
  of emails) as a middle setting between minimal and full.
