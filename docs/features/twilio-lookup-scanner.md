# Feature Spec — Twilio Lookup Scanner

| | |
|---|---|
| **Scanner ID** | `twilio` |
| **Category** | Number intelligence (carrier / line type / mobile signals) |
| **Status** | Draft — proposed |
| **External dependency** | Twilio Lookup API v2 |
| **Auth** | Account SID + Auth Token (HTTP Basic) |
| **Default state** | Skipped unless credentials are configured |
| **Supplier** | Yes (`TwilioSupplierInterface`) |

---

## 1. Summary

Add a `twilio` scanner that queries the Twilio Lookup v2 API for live phone-number
intelligence. This is the single highest-value addition to the number-intelligence
side of the framework: it returns a richer line-type taxonomy than the existing
`numverify` scanner, current (portability-aware) carrier data, and — for accounts
with the relevant packages enabled — caller name (CNAM), SIM-swap signals, and
call-forwarding status.

The scanner sits alongside `local`, `numverify`, and `ovh` in the "what is this
number" group, and is intended to become the preferred carrier/line-type source
when an operator has Twilio credentials.

## 2. Motivation & gap filled

| Existing source | Limitation the `twilio` scanner removes |
|---|---|
| `local` (libphonenumber) | Offline only; `Carrier` is the *preferred domestic carrier code* and is frequently empty. |
| `numverify` | Coarse line type (mobile/landline); carrier is range-derived, not portability-aware. |
| `ovh` | Range-allocation only; limited to country codes 33/32/44/34/41. |

Twilio Lookup v2 provides a 12-value line-type taxonomy (e.g. `mobile`, `landline`,
`fixedVoip`, `nonFixedVoip`, `tollFree`, `premium`, `voicemail`, `pager`,
`unknown`), global VoIP detection, mobile country/network codes, and optional
mobile-network-sourced signals (SIM swap, call forwarding) that no other scanner in
the tree can supply.

## 3. Goals / Non-goals

**Goals**
- Validate and enrich a number through one authenticated Twilio Lookup v2 call.
- Expose line-type intelligence by default (cheapest, broadest-coverage package).
- Allow per-package opt-in for `caller_name`, `sim_swap`, and `call_forwarding`.
- Degrade gracefully when a requested package is unavailable for the number's
  region or the account (return what came back; surface a per-field note).

**Non-goals**
- `identity_match` / `reassigned_number` packages. These compare against
  operator-supplied PII (name, address, postal code) and are verification features,
  not recon features. They have no place in a footprinting scanner and are out of
  scope.
- Sending SMS/calls or any Verify-style possession check.

## 4. Data source

**Endpoint**
```
GET https://lookups.twilio.com/v2/PhoneNumbers/{E164}?Fields={comma_separated_packages}
Authorization: Basic base64(AccountSID:AuthToken)
```

**Packages and key fields**

| Package (Field) | Selected response fields | Coverage / notes |
|---|---|---|
| *(none — basic)* | `valid`, `validation_errors`, `national_format`, `calling_country_code` | Free; always returned. |
| `line_type_intelligence` | `type`, `carrier_name`, `mobile_country_code`, `mobile_network_code`, `error_code` | Global. Default package. |
| `caller_name` | `caller_name`, `caller_type` (`CONSUMER`/`BUSINESS`), `error_code` | US only; billed per lookup even when no data is returned. |
| `sim_swap` | `last_sim_swap.last_sim_swap_date`, `last_sim_swap.swapped_in_period`, `carrier_name`, `mobile_country_code`, `mobile_network_code` | Beta; requires carrier approval; region-limited. |
| `call_forwarding` | `call_forwarding_status` | Beta; requires carrier approval; UK availability. |

**Cost model:** the basic lookup is free; each requested data package is billed per
lookup. The scanner must therefore never request a package the operator did not
explicitly opt into.

**Rate limits:** governed by the Twilio account. The scanner makes exactly one HTTP
request per scan regardless of how many packages are selected (packages are
combined in a single `Fields` value).

## 5. User-facing behavior

```bash
# Default: line type intelligence only
export TWILIO_ACCOUNT_SID="ACxxxxxxxx"
export TWILIO_AUTH_TOKEN="xxxxxxxx"
phoneinfoga scan -n "+14159929960"

# Opt into extra (billed) packages
export TWILIO_LOOKUP_FIELDS="line_type_intelligence,caller_name"
phoneinfoga scan -n "+14159929960"
```

Console / JSON output includes validity, line type, current carrier, MCC/MNC, and
any opted-in package fields. Packages that returned an `error_code` (e.g. caller
name for a non-US number) are reported as empty with a short note rather than
failing the scan.

## 6. Scanner design

```go
const Twilio = "twilio"
```

- **`Name()`** → `"twilio"`
- **`Description()`** → `"Request live carrier, line type and mobile signals through the Twilio Lookup v2 API."`
- **`DryRun(n, opts)`** — skip the scanner unless usable:
  - return error if `TWILIO_ACCOUNT_SID` or `TWILIO_AUTH_TOKEN` is empty
    (`"Twilio credentials are not defined"`).
  - optionally return error if `n.Valid == false`, to avoid spending a lookup on a
    number libphonenumber already rejected.
- **`Run(n, opts)`**:
  1. Resolve credentials and the package list (`TWILIO_LOOKUP_FIELDS`, default
     `line_type_intelligence`).
  2. Call `supplier.Lookup(n.E164, fields)`.
  3. Map the provider payload into `TwilioScannerResponse`, copying any per-package
     `error_code` into the corresponding `*_note` field.
  4. Return the struct.

## 7. Response schema

```go
type TwilioScannerResponse struct {
    Valid               bool   `json:"valid" console:"Valid"`
    NationalFormat      string `json:"national_format,omitempty" console:"National format,omitempty"`
    LineType            string `json:"line_type,omitempty" console:"Line type,omitempty"`
    CarrierName         string `json:"carrier_name,omitempty" console:"Carrier,omitempty"`
    MobileCountryCode   string `json:"mobile_country_code,omitempty" console:"MCC,omitempty"`
    MobileNetworkCode   string `json:"mobile_network_code,omitempty" console:"MNC,omitempty"`
    CallerName          string `json:"caller_name,omitempty" console:"Caller name,omitempty"`
    CallerType          string `json:"caller_type,omitempty" console:"Caller type,omitempty"`
    SimSwapLastDate     string `json:"sim_swap_last_date,omitempty" console:"Last SIM swap,omitempty"`
    SimSwapInPeriod     *bool  `json:"sim_swap_in_period,omitempty" console:"SIM swapped in period,omitempty"`
    CallForwarding      string `json:"call_forwarding_status,omitempty" console:"Call forwarding,omitempty"`
    Notes               []string `json:"notes,omitempty" console:"Notes,omitempty"`
}
```

## 8. Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `TWILIO_ACCOUNT_SID` | yes | — | Account SID for HTTP Basic auth. |
| `TWILIO_AUTH_TOKEN` | yes | — | Auth token for HTTP Basic auth. |
| `TWILIO_LOOKUP_FIELDS` | no | `line_type_intelligence` | Comma-separated package list. Extra packages are billed. |

All read through `ScannerOptions.GetStringEnv`, so they are settable per request or
via the environment.

## 9. Supplier interface & testing

```go
type TwilioSupplierInterface interface {
    Lookup(e164 string, fields []string) (*TwilioLookupResponse, error)
}
```

- Concrete `TwilioSupplier` lives in `lib/remote/suppliers/twilio.go` and owns the
  HTTP/Basic-auth client.
- Generate `mocks/TwilioSupplier.go` (mockery) following the existing
  `NumverifySupplier` mock.
- Table-driven tests in `twilio_scanner_test.go`:
  - missing credentials → `DryRun` returns error.
  - line-type-only happy path.
  - caller-name requested on a non-US number → empty + note, scan still succeeds.
  - supplier error → surfaced via the errors map.
- Add a golden-file case mirroring `test/goldenfile` for the default field set.

## 10. Error handling

- Treat HTTP ≥ 400 as a scan error (return `nil, err`); the framework records it in
  the per-scanner errors map without aborting the run.
- Per-package `error_code` values are *not* scan errors — they are expected (e.g.
  region mismatch) and become `Notes` entries.

## 11. Security / privacy / compliance

- Caller name and SIM-swap data are personal data; redact the response from debug
  logs (log the number being scanned, not the resolved identity fields).
- SIM-swap and call-forwarding packages require carrier approval and are intended
  by Twilio for fraud/identity use cases; document that operators are responsible
  for using them within Twilio's acceptable-use terms.
- Never log the auth token.

## 12. Dependencies

- No new Go module required; a thin `net/http` client with Basic auth is sufficient
  (the official Twilio SDK is optional and heavier than needed here).

## 13. Open questions / future work

- Should `sms_pumping_risk` (part of the Lookup fraud signals) be added as a fourth
  opt-in package once an operator requests it?
- Add a single combined "score" derived from line type + SIM-swap recency for the
  web UI urgency indicator? (Out of scope for v1.)
