# Feature Spec — IPQualityScore Scanner

| | |
|---|---|
| **Scanner ID** | `ipqualityscore` |
| **Category** | Reputation / fraud scoring |
| **Status** | Draft — proposed |
| **External dependency** | IPQualityScore Phone Validation API |
| **Auth** | API key |
| **Default state** | Skipped unless an API key is configured |
| **Supplier** | Yes (`IPQSSupplierInterface`) |

---

## 1. Summary

Add an `ipqualityscore` scanner that returns a programmatic reputation verdict for a
number: a fraud score, abuse flags, and line-quality signals (VoIP, prepaid, active,
disposable). Today the framework only *generates search URLs* for reputation
(`getReputationDorks` in the Google Search scanner) — the operator still has to open
and read them. This scanner adds a machine-readable reputation result that can sit
beside those dorks and feed the web UI's urgency indicators.

## 2. Motivation & gap filled

The reputation dorks (800notes, whocallsme, tellows, truecaller, etc.) are
discovery aids, not verdicts. IPQS provides a single call that returns a 0–100 fraud
score plus boolean risk flags drawn from a carrier-backed threat network, across
150+ countries. It complements — does not replace — the dork-based reputation
discovery.

## 3. Goals / Non-goals

**Goals**
- Return fraud score, `recent_abuse`, `VOIP`, `prepaid`, `risky`, `active`, line
  type, carrier, and coarse geography for a number.
- Be cheap to adopt (IPQS offers a free tier of ~1,000 lookups/month for prototyping).
- Expose optional strictness and country hints.

**Non-goals**
- Reverse-identity / owner name. IPQS offers identity append as a separate, gated
  product; it is out of scope for this scanner (and overlaps the regulated category
  covered by other specs).
- Treating the fraud score as ground truth. It is a signal; the response should be
  presented as such.

## 4. Data source

**Endpoint**
```
GET https://www.ipqualityscore.com/api/json/phone/{API_KEY}/{number}
    ?country[]={ISO}&strictness={0-2}
```

**Selected response fields consumed:**

| Field | Meaning |
|---|---|
| `valid` | Well-formed and plausibly real. |
| `active` | Line appears active/in-service (carrier-data dependent). |
| `fraud_score` | 0–100 risk score (≥75 suspicious, ≥85 risky, ≥90 high-risk per IPQS guidance). |
| `recent_abuse` | Recent abusive activity reported across the threat network. |
| `VOIP` | Number is VoIP. |
| `prepaid` | Prepaid line. |
| `risky` | Aggregate risk indicator. |
| `line_type` | Wireless / landline / VoIP / toll-free. |
| `carrier` | Carrier name. |
| `country` / `region` / `city` / `timezone` | Geography. |
| `do_not_call` / `leaked` / `spammer` | Additional flags (provider-dependent). |

**Cost / limits:** free tier for low volume; flat per-lookup pricing above that.
One HTTP request per scan.

## 5. User-facing behavior

```bash
export IPQS_API_KEY="xxxxxxxx"
phoneinfoga scan -n "+14159929960"

# Optional tuning
export IPQS_STRICTNESS="1"          # 0 (default) .. 2 (most aggressive)
export IPQS_COUNTRY="US"            # hint for short/national inputs
```

Output shows validity, fraud score, the risk flags, line type, carrier, and
geography.

## 6. Scanner design

```go
const IPQualityScore = "ipqualityscore"
```

- **`Name()`** → `"ipqualityscore"`
- **`Description()`** → `"Score a phone number for fraud and abuse signals via the IPQualityScore API."`
- **`DryRun(n, opts)`** — return error if `IPQS_API_KEY` is empty
  (`"API key is not defined"`).
- **`Run(n, opts)`**:
  1. Resolve key, optional strictness, optional country hint.
  2. Call `supplier.Validate(n.E164, opts...)` (use `n.International` if the provider
     prefers digits without `+`).
  3. Map to `IPQSScannerResponse`.

## 7. Response schema

```go
type IPQSScannerResponse struct {
    Valid       bool   `json:"valid" console:"Valid"`
    Active      bool   `json:"active" console:"Active"`
    FraudScore  int    `json:"fraud_score" console:"Fraud score"`
    RecentAbuse bool   `json:"recent_abuse" console:"Recent abuse"`
    VOIP        bool   `json:"voip" console:"VOIP"`
    Prepaid     bool   `json:"prepaid" console:"Prepaid"`
    Risky       bool   `json:"risky" console:"Risky"`
    DoNotCall   bool   `json:"do_not_call,omitempty" console:"Do not call,omitempty"`
    Leaked      bool   `json:"leaked,omitempty" console:"Leaked,omitempty"`
    Spammer     bool   `json:"spammer,omitempty" console:"Spammer,omitempty"`
    LineType    string `json:"line_type,omitempty" console:"Line type,omitempty"`
    Carrier     string `json:"carrier,omitempty" console:"Carrier,omitempty"`
    Country     string `json:"country,omitempty" console:"Country,omitempty"`
    Region      string `json:"region,omitempty" console:"Region,omitempty"`
    City        string `json:"city,omitempty" console:"City,omitempty"`
    Timezone    string `json:"timezone,omitempty" console:"Timezone,omitempty"`
}
```

## 8. Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `IPQS_API_KEY` | yes | — | API key. |
| `IPQS_STRICTNESS` | no | `0` | 0–2; higher increases verification aggressiveness. |
| `IPQS_COUNTRY` | no | — | ISO country hint for national-format inputs. |

## 9. Supplier interface & testing

```go
type IPQSSupplierInterface interface {
    Validate(number string, opts IPQSRequestOptions) (*IPQSValidateResponse, error)
}
```

- Concrete supplier in `lib/remote/suppliers/ipqs.go`; generate `mocks/IPQSSupplier.go`.
- Tests:
  - missing key → `DryRun` error.
  - clean number (low score) happy path.
  - flagged number (high score + `recent_abuse` + `VOIP`).
  - provider/credit error surfaced.

## 10. Error handling

- IPQS returns `success: false` with a `message` on credit exhaustion or invalid
  input — map to a scan error carrying that message (mirrors how the Numverify
  supplier surfaces its error `message`).
- HTTP ≥ 400 → scan error.

## 11. Security / privacy / compliance

- The fraud score and flags are heuristic signals; present them as such in
  docs/UI, not as authoritative judgments about a person.
- `leaked` simply indicates the number appeared in IPQS data feeds — do not expand
  it into breach detail here (that is the separate, opt-in breach scanner).
- Redact resolved geography from debug logs.

## 12. Dependencies

- `net/http` only.

## 13. Open questions / future work

- Surface a normalized risk band (`low`/`medium`/`high`) derived from `fraud_score`
  for consistent UI treatment across reputation sources.
- Optionally pass the operator's IP/email when IPQS is used in a verification
  context — out of scope for recon.
