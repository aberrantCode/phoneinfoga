# Feature Spec — Validation-Alternate Scanners

| | |
|---|---|
| **Scanner IDs** | `veriphone`, `abstract`, `numlookupapi` |
| **Category** | Number validation (Numverify alternatives / redundancy) |
| **Status** | Implemented |
| **External dependency** | Veriphone, Abstract Phone Validation, NumlookupAPI |
| **Auth** | API key (per provider) |
| **Default state** | Each skipped unless its own API key is configured |
| **Supplier** | Yes — one shared interface, three implementations |

---

## 1. Summary

Add a small family of validation scanners that mirror the existing `numverify`
scanner against alternative providers. They return the same shape of data
(validity, carrier, region, line type) and exist for **redundancy and
cross-checking**: when the Numverify quota is exhausted, when an operator wants a
second opinion, or when one provider has better coverage for a given country.

> **Scope note.** Item 4 was proposed as a *category* (Veriphone / Abstract /
> NumlookupAPI). Rather than three near-identical specs, this single spec defines
> one shared `ValidationProvider` supplier interface and three thin scanners that
> implement it. This matches the repo convention (one scanner = one source =
> one `Name()`) while keeping the design DRY.

## 2. Motivation & gap filled

`numverify` is the only validation source today and is a single point of failure: if
its key is missing or its monthly quota is spent, the framework loses validation
entirely. These alternates provide:

- **Failover** — keep validating when Numverify is unavailable.
- **Corroboration** — agreement across providers raises confidence; disagreement
  flags an ambiguous number.
- **Coverage spread** — providers differ by region/carrier data quality.

Two of the three have free tiers suitable for prototyping (Abstract offers ~100
requests/month with no card; Veriphone and NumlookupAPI offer free developer tiers).

## 3. Goals / Non-goals

**Goals**
- Reuse the exact `numverify` design pattern for each provider.
- Share one supplier interface so adding a fourth provider is trivial.
- Keep each scanner independently gated by its own key.

**Non-goals**
- Live HLR/network status (see the `hlr` spec) — these are static/range validators.
- Reputation scoring (see the `ipqualityscore` spec).
- Automatically reconciling disagreements into a single verdict (left to the
  operator / UI for v1).

## 4. Data sources

| Provider | Endpoint (shape) | Auth | Notable fields |
|---|---|---|---|
| Veriphone | `GET https://api.veriphone.io/v2/verify?phone={E164}&key={key}` | query key | `phone_valid`, `phone_type`, `carrier`, `country`, `phone_region` |
| Abstract | `GET https://phonevalidation.abstractapi.com/v1/?api_key={key}&phone={E164}` | query key | `valid`, `type`, `carrier`, `country.name`, `location` |
| NumlookupAPI | `GET https://api.numlookupapi.com/v1/validate/{E164}` (key sent as `apikey` **header**) | header key | `valid`, `line_type`, `carrier`, `country_name`, `location` |

All three are single-request, JSON, and conceptually identical to the Numverify
validate call.

## 5. User-facing behavior

```bash
# Enable whichever providers you have keys for; each runs independently
export VERIPHONE_API_KEY="xxxx"
export ABSTRACT_PHONE_API_KEY="xxxx"
export NUMLOOKUPAPI_API_KEY="xxxx"
phoneinfoga scan -n "+14159929960"
```

Each enabled provider contributes its own result block. With several enabled, the
operator can eyeball agreement on validity / line type / carrier.

## 6. Scanner design

```go
const Veriphone    = "veriphone"
const Abstract     = "abstract"
const NumlookupAPI = "numlookupapi"
```

Each scanner is a thin wrapper over the shared supplier interface:

- **`Name()`** → provider id.
- **`Description()`** → e.g. `"Validate a phone number through the Veriphone API."`
- **`DryRun(n, opts)`** — return error if the provider's own key env var is empty.
- **`Run(n, opts)`** — call `provider.Validate(n.International)`, map to the shared
  `ValidationScannerResponse`.

Registration in `InitScanners` adds all three, each constructed with its concrete
provider implementation.

## 7. Response schema (shared)

```go
type ValidationScannerResponse struct {
    Provider      string `json:"provider" console:"Provider"`
    Valid         bool   `json:"valid" console:"Valid"`
    Number        string `json:"number,omitempty" console:"Number,omitempty"`
    LocalFormat   string `json:"local_format,omitempty" console:"Local format,omitempty"`
    IntlFormat    string `json:"international_format,omitempty" console:"International format,omitempty"`
    CountryName   string `json:"country_name,omitempty" console:"Country name,omitempty"`
    CountryPrefix string `json:"country_prefix,omitempty" console:"Country prefix,omitempty"`
    Location      string `json:"location,omitempty" console:"Location,omitempty"`
    Carrier       string `json:"carrier,omitempty" console:"Carrier,omitempty"`
    LineType      string `json:"line_type,omitempty" console:"Line type,omitempty"`
}
```

The `Provider` field lets the UI/JSON consumer distinguish blocks when several
alternates run together.

## 8. Configuration

| Variable | Provider | Required for that provider |
|---|---|---|
| `VERIPHONE_API_KEY` | Veriphone | yes |
| `ABSTRACT_PHONE_API_KEY` | Abstract | yes |
| `NUMLOOKUPAPI_API_KEY` | NumlookupAPI | yes |

## 9. Supplier interface & testing

```go
type ValidationProvider interface {
    Name() string
    Validate(internationalNumber string) (*ValidationResult, error)
}
```

- Implementations: `lib/remote/suppliers/veriphone.go`, `abstract.go`,
  `numlookupapi.go`. Each normalizes its provider payload into `ValidationResult`.
- Generate one mock per implementation, or a single `mocks/ValidationProvider.go`
  parameterized by provider name.
- Tests (per provider): missing key → `DryRun` error; valid-number happy path;
  invalid-number path; provider error surfaced. Reuse the `numverify_scanner_test`
  structure.

## 10. Error handling

- Each provider maps HTTP ≥ 400 / `success:false` to a scan error carrying the
  provider message, exactly as the Numverify supplier does.
- A failure in one provider never affects the others (independent goroutines via the
  framework's fan-out).

## 11. Security / privacy / compliance

- Standard handling: never log keys; redact resolved location/carrier from debug
  logs.
- These are low-sensitivity validators (no identity, no reputation), so no special
  gating beyond key presence.

## 12. Dependencies

- `net/http` only.

## 13. Open questions / future work

- A future `validation-consensus` post-processor could aggregate all enabled
  validators into a single agreement summary (valid: 3/3, line type: 2 mobile / 1
  voip). Deliberately out of scope here to keep each scanner independent.
