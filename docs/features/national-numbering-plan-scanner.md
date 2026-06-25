# Feature Spec — National Numbering-Plan Scanner

| | |
|---|---|
| **Scanner ID** | `nanpa` |
| **Category** | Number intelligence (range / rate-center allocation) |
| **Status** | Draft — proposed |
| **External dependency** | NANPA / NPA-NXX data (public dataset or commercial API) |
| **Auth** | None (public dataset) or API key (commercial provider) |
| **Default state** | Skipped unless the number is in the North American Numbering Plan (country code 1) |
| **Supplier** | Yes (`NumberingPlanSupplierInterface`) |

---

## 1. Summary

Add a `nanpa` scanner that resolves North American (country code 1) numbers to their
NPA-NXX allocation: rate center, geography (state/province), and carrier-of-record
(OCN). This generalizes the concept the `ovh` scanner already implements for a
handful of European countries (FR/BE/GB/ES/CH range data) to the much larger NANP
footprint (US, Canada, and other +1 territories).

## 2. Motivation & gap filled

The `ovh` scanner proves the pattern is valuable — map a number to its allocated
geography/carrier from authoritative numbering data — but it is limited to five
country codes. NANP coverage is conspicuously absent despite being one of the
highest-volume regions. The `nanpa` scanner adds:

- **Rate center & geography** for +1 numbers (finer than country/area).
- **Carrier-of-record (OCN)** — the operating company the NXX block was assigned to.
- A clean extension point: the same supplier interface can later host other national
  plans (Ofcom for UK landline blocks, etc.), turning `ovh`/`nanpa` into a family.

> This is range/allocation data, not live status. It tells you who a block was
> assigned to, not who serves the number today (that is the `hlr` scanner). The two
> are complementary.

## 3. Goals / Non-goals

**Goals**
- Resolve NPA-NXX → rate center, state/province, OCN/carrier-of-record, LATA.
- Gate strictly to country code 1, mirroring how `ovh` gates to its supported codes.
- Support either a bundled public dataset or a commercial NPA-NXX API behind one
  interface.

**Non-goals**
- Porting-aware/current carrier (covered by `hlr` via LRN/MNP).
- Subscriber identity or location beyond rate-center geography.
- Non-NANP countries in v1 (designed to extend, not delivered here).

## 4. Data source

Two viable backends behind the same interface:

**A. Public dataset (no auth).** NANPA publishes central-office-code (NXX) assignment
data; community mirrors (e.g. localcallingguide / telcodata exports) provide
NPA-NXX → rate center / OCN / LATA. Bundle or periodically refresh a local lookup
table; resolution is then offline and free.

**B. Commercial API (API key).** A hosted NPA-NXX/rate-center API for operators who
prefer not to maintain the dataset.

**Fields consumed:**

| Field | Meaning |
|---|---|
| `rate_center` | Telephone rate center for the NPA-NXX. |
| `state` / `province` | Geography. |
| `ocn` | Operating Company Number (carrier-of-record). |
| `carrier` | Carrier name for the OCN. |
| `lata` | Local Access and Transport Area. |
| `block_holder` | Thousands-block holder, when available. |

## 5. User-facing behavior

```bash
# Public-dataset mode: works offline, no key
phoneinfoga scan -n "+12025551234"

# Commercial-API mode
export NANPA_API_KEY="xxxx"
export NANPA_API_URL="https://npa-nxx.example.com/v1"
phoneinfoga scan -n "+12025551234"
```

For a +1 number, output shows rate center, state, carrier-of-record, and LATA. For
any non-+1 number the scanner is silently skipped.

## 6. Scanner design

```go
const NANPA = "nanpa"
```

- **`Name()`** → `"nanpa"`
- **`Description()`** → `"Resolve North American (NANP) numbers to rate center and carrier-of-record from NPA-NXX data."`
- **`DryRun(n, opts)`** — gate exactly like `ovh`:
  ```go
  if n.CountryCode != 1 {
      return fmt.Errorf("country code %d is not in the NANP", n.CountryCode)
  }
  ```
  In commercial-API mode, additionally require `NANPA_API_KEY`. In public-dataset
  mode, no key is required.
- **`Run(n, opts)`** — derive NPA (area code) and NXX (next 3 digits) from
  `n.RawLocal`/`n.E164`, call `supplier.Lookup(npa, nxx)`, map to response.

## 7. Response schema

```go
type NANPAScannerResponse struct {
    Found       bool   `json:"found" console:"Found"`
    NPA         string `json:"npa,omitempty" console:"Area code,omitempty"`
    NXX         string `json:"nxx,omitempty" console:"Exchange,omitempty"`
    RateCenter  string `json:"rate_center,omitempty" console:"Rate center,omitempty"`
    State       string `json:"state,omitempty" console:"State,omitempty"`
    OCN         string `json:"ocn,omitempty" console:"OCN,omitempty"`
    Carrier     string `json:"carrier,omitempty" console:"Carrier of record,omitempty"`
    LATA        string `json:"lata,omitempty" console:"LATA,omitempty"`
    BlockHolder string `json:"block_holder,omitempty" console:"Block holder,omitempty"`
}
```

## 8. Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `NANPA_API_KEY` | only in commercial-API mode | — | API key. |
| `NANPA_API_URL` | no | provider default | Commercial endpoint override. |
| `NANPA_DATASET_PATH` | only in dataset mode | bundled path | Location of the local NPA-NXX table. |

## 9. Supplier interface & testing

```go
type NumberingPlanSupplierInterface interface {
    Lookup(npa string, nxx string) (*NumberingPlanResult, error)
}
```

- Two implementations: `suppliers/nanpa_dataset.go` (offline table) and
  `suppliers/nanpa_api.go` (HTTP). Generate `mocks/NumberingPlanSupplier.go`.
- Tests:
  - non-+1 number → `DryRun` error (gate).
  - known NPA-NXX → resolved rate center / OCN (dataset mode, deterministic).
  - unknown NPA-NXX → `Found: false`.
  - API mode error surfaced.

## 10. Error handling

- Dataset miss is a normal `Found: false` result, not an error.
- API HTTP ≥ 400 → scan error.

## 11. Security / privacy / compliance

- NPA-NXX allocation data is public regulatory information; no personal data is
  involved, so no special gating beyond the NANP country-code check.
- If bundling the dataset, respect the source's redistribution terms and document
  refresh cadence (assignments change over time).

## 12. Dependencies

- Dataset mode: a CSV/SQLite reader for the bundled table (standard library or a
  light dependency).
- API mode: `net/http`.

## 13. Open questions / future work

- Generalize into a `numbering` family keyed by country code, with `ovh`, `nanpa`,
  and future national plans implementing `NumberingPlanSupplierInterface`.
- Optional ENUM/NAPTR (`e164.arpa`) DNS check as a zero-cost add-on — noted as
  low-yield in practice because the public ENUM tree is sparsely populated.
