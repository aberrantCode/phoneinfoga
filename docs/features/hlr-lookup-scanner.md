# Feature Spec — HLR Lookup Scanner

| | |
|---|---|
| **Scanner ID** | `hlr` |
| **Category** | Number intelligence (live network status) |
| **Status** | Draft — proposed |
| **External dependency** | A configurable HLR/MNP provider (e.g. Abstract, apilayer, IPQS HLR) |
| **Auth** | API key |
| **Default state** | Skipped unless an API key is configured **and** the number is mobile |
| **Supplier** | Yes (`HLRSupplierInterface`) |

---

## 1. Summary

Add an `hlr` scanner that performs a Home Location Register lookup against a
configurable provider. Unlike range-allocation lookups (`local`, `ovh`) and
validation APIs (`numverify`), an HLR query reflects the *current live state* of a
mobile subscriber: whether the number is active/reachable right now, which carrier
is serving it after any porting, and roaming status. The lookup does not place a
call or send an SMS.

## 2. Motivation & gap filled

The framework can currently tell you the carrier a number's *range* was allocated
to. It cannot tell you whether the number is live today or whether it has been
ported to a different carrier. Thousands of numbers are ported daily, so the
allocated carrier and the serving carrier routinely disagree. The `hlr` scanner
closes that gap with portability-aware, real-time status.

This distinguishes three increasingly authoritative layers the tool can now offer:

1. **Range allocation** — `local`, `ovh` (who the block was issued to).
2. **Validation / static carrier** — `numverify` (well-formed, coarse carrier).
3. **Live network state** — `hlr` (active now, current serving carrier, roaming).

## 3. Goals / Non-goals

**Goals**
- Return live reachability, current serving carrier, ported flag, and line type for
  mobile numbers.
- Be provider-agnostic behind a supplier interface, with one default provider.
- Skip cleanly for number types the HLR cannot answer (landline, VoIP, toll-free).

**Non-goals**
- Locating the subscriber or returning identity. HLR returns network status, not a
  person.
- Acting as a generic validation fallback for non-mobile numbers (that is the job
  of the `validation-alternates` scanner spec).

## 4. Data source

HLR providers expose a similar REST shape. Designed around a generic provider with
an overridable base URL:

```
GET {HLR_API_URL}/lookup?api_key={key}&number={E164}
```

**Representative response fields consumed:**

| Field | Meaning |
|---|---|
| `active` / `status` | Whether the subscriber is currently reachable (`active` / `inactive` / `unknown`). |
| `current_carrier` | Serving carrier after porting (MNP-aware). |
| `ported` | Whether the number has been ported away from its original block. |
| `roaming` | Whether the subscriber is currently roaming (provider-dependent). |
| `line_type` | `mobile` / (others not applicable). |
| `mcc` / `mnc` | Mobile country/network codes. |
| `country` | Country of the serving network. |

**Cost:** HLR queries are billed per lookup and are typically more expensive than
range/validation lookups, because they touch carrier signaling data. Document this;
it reinforces the mobile-only gate.

## 5. User-facing behavior

```bash
export HLR_API_KEY="xxxxxxxx"
# Optional override if self-hosting or switching providers
export HLR_API_URL="https://phonevalidation.abstractapi.com/v1"
phoneinfoga scan -n "+14159929960"
```

For a mobile number, output shows active status, current carrier, ported flag,
roaming, and country. For a landline/VoIP number the scanner is silently skipped
(visible in debug logs as "scanner was ignored because it should not run").

## 6. Scanner design

```go
const HLR = "hlr"
```

- **`Name()`** → `"hlr"`
- **`Description()`** → `"Check live network status and current carrier of a mobile number via an HLR lookup."`
- **`DryRun(n, opts)`** — the distinctive part is the **mobile-only gate**:
  - return error if `HLR_API_KEY` is empty.
  - re-derive the number type with libphonenumber and return error if it is not a
    mobile (or fixed-or-mobile) number:
    ```go
    num, _ := phonenumbers.Parse(n.E164, "")
    t := phonenumbers.GetNumberType(num)
    if t != phonenumbers.MOBILE && t != phonenumbers.FIXED_LINE_OR_MOBILE {
        return fmt.Errorf("HLR lookup applies to mobile numbers only")
    }
    ```
    (The `number.Number` struct does not carry the line type, so it is recomputed
    here rather than threaded through.)
- **`Run(n, opts)`** — call `supplier.Lookup(n.E164)`, map to `HLRScannerResponse`,
  return.

## 7. Response schema

```go
type HLRScannerResponse struct {
    Active         *bool  `json:"active,omitempty" console:"Active,omitempty"`
    Status         string `json:"status,omitempty" console:"Status,omitempty"`
    CurrentCarrier string `json:"current_carrier,omitempty" console:"Current carrier,omitempty"`
    Ported         *bool  `json:"ported,omitempty" console:"Ported,omitempty"`
    Roaming        *bool  `json:"roaming,omitempty" console:"Roaming,omitempty"`
    LineType       string `json:"line_type,omitempty" console:"Line type,omitempty"`
    MCC            string `json:"mcc,omitempty" console:"MCC,omitempty"`
    MNC            string `json:"mnc,omitempty" console:"MNC,omitempty"`
    Country        string `json:"country,omitempty" console:"Country,omitempty"`
}
```

## 8. Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `HLR_API_KEY` | yes | — | Provider API key. |
| `HLR_API_URL` | no | provider default | Override for self-hosting or provider swap. |

## 9. Supplier interface & testing

```go
type HLRSupplierInterface interface {
    Lookup(e164 string) (*HLRLookupResponse, error)
}
```

- Concrete supplier in `lib/remote/suppliers/hlr.go`; generate `mocks/HLRSupplier.go`.
- Tests:
  - missing key → `DryRun` error.
  - landline / VoIP / toll-free numbers → `DryRun` error (mobile gate). Use known
    test numbers per type.
  - mobile happy path (active + ported).
  - supplier error surfaced.

## 10. Error handling

- HTTP ≥ 400 → scan error in the per-scanner errors map.
- `unknown`/empty status from the provider is a valid result, not an error.

## 11. Security / privacy / compliance

- HLR data is network metadata, not identity, but reachability/roaming is still
  sensitive; redact resolved fields from debug logs.
- Some jurisdictions and provider AUPs restrict HLR querying to legitimate
  routing/fraud purposes; document operator responsibility.

## 12. Dependencies

- `nyaruka/phonenumbers` (already a dependency) for the mobile-type gate.
- `net/http` for the provider client.

## 13. Open questions / future work

- Support multiple registered providers simultaneously (`hlr-abstract`, `hlr-ipqs`)
  for cross-checking, sharing `HLRSupplierInterface`?
- Cache results briefly to avoid duplicate billed lookups within a single multi-scan
  session.
