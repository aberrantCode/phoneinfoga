<p align="center">
  <img src="./docs/images/banner.png" width=500  alt="project logo"/>
</p>

<h4 align="center">Information gathering framework for phone numbers</h4>

<p align="center">
  <a href="https://sundowndev.github.io/phoneinfoga/">Upstream docs</a> •
  <a href="https://petstore.swagger.io/?url=https://raw.githubusercontent.com/sundowndev/phoneinfoga/master/web/docs/swagger.yaml">API documentation</a> •
  <a href="./docs/features/">Scanner feature specs</a>
</p>

> **This is a fork.** It extends [sundowndev/phoneinfoga](https://github.com/sundowndev/phoneinfoga)
> — which is stable but unmaintained upstream — with **11 additional scanners**, a
> browser-based credentials/config plane, and a redesigned web client. See
> [What this fork adds](#what-this-fork-adds) for the full delta. All original
> credit belongs to [@sundowndev](https://github.com/sundowndev); this fork keeps
> the same GPL-3.0 license and scanner architecture.

## About

PhoneInfoga is one of the most advanced tools to scan international phone numbers. It gathers basic information such as country, area, carrier and line type, then uses a collection of pluggable **scanners** to enrich a number with OSINT — footprinting, reputation, breach exposure, live network status and more. It doesn't automate everything; it's an investigator's assistant, not a magic "trace a phone" button (see [Anti-features](#anti-features)).

Each scanner is a thin adapter over an external data source, registered in `lib/remote/init.go`. Because scanners compose rather than modify the core engine, this fork adds capabilities purely by registering new ones — and each stays **dormant and skip-clean until you configure its credentials**, so nothing new runs (or fails) unless you opt in.

## What this fork adds

Upstream ships **4** scanners. This fork registers **15**. Everything new is credential-gated and off by default.

### New scanners

| Scanner ID | Category | What it adds | Requires |
|---|---|---|---|
| `twilio` | Number intelligence | Twilio Lookup v2 — richer line-type taxonomy, portability-aware carrier, optional CNAM / SIM-swap / call-forwarding | `TWILIO_ACCOUNT_SID` + `TWILIO_AUTH_TOKEN` |
| `hlr` | Live network status | Home Location Register lookup — is the mobile number active/reachable *right now*, serving carrier after porting, roaming (no call/SMS sent). Mobile-only | `HLR_API_KEY` + `HLR_API_URL` |
| `nanpa` | Range allocation | Resolves +1 (North American) numbers to NPA-NXX rate center, geography and carrier-of-record (OCN) — the NANP equivalent of what `ovh` does for a few EU countries | `NANPA_API_URL` |
| `ipqualityscore` | Reputation / fraud | Machine-readable fraud score + abuse flags and line signals (VoIP, prepaid, disposable, active) — beyond just generating dork URLs | `IPQS_API_KEY` |
| `breach` | Breach / leak exposure **(sensitive)** | Whether the number appears in known breaches/leaks and how many records, via an authenticated aggregator (Dehashed). **Double-gated:** must be explicitly enabled *and* have credentials | `BREACH_SCANNER_ENABLED=true` + `DEHASHED_EMAIL` + `DEHASHED_API_KEY` |
| `serpapi` | Web footprint | Runs the existing phone-number dorks through SerpAPI (Google/Bing/DuckDuckGo/Yandex), returning result counts and top hits | `SERPAPI_KEY` |
| `googlecse` | Web footprint | Executes dorks via Google Programmable Search (Custom Search Engine) | `GOOGLECSE_CX` + `GOOGLE_API_KEY` |
| `searxng` | Web footprint | Executes dorks against a self-hosted/instance SearXNG meta-search | `SEARXNG_URL` |
| `veriphone` | Validation | Numverify-alternative validation for redundancy / cross-checking | `VERIPHONE_API_KEY` |
| `abstract` | Validation | Abstract Phone Validation as an alternate provider | `ABSTRACT_PHONE_API_KEY` |
| `numlookupapi` | Validation | NumlookupAPI as an alternate validation provider | `NUMLOOKUPAPI_API_KEY` |

The three validation scanners share one `ValidationProvider` supplier interface — configure only the providers you actually use, and use them to cross-check or fail over when a Numverify quota is exhausted.

### Web client changes

- **Runtime credentials plane** (`web/config_controller.go`): a `GET`/`POST /api/config` endpoint lets you inspect and set scanner credentials in the running process **without a restart or editing `.env`**. It's hardened — **loopback-only**, requires a JSON content-type (defeats simple-request CSRF), writes are restricted to an allowlist, and secret values are returned masked (last 4 chars) and never in full.
- **Scanner credentials modal** (`ScannerCredentials.vue`) to drive that endpoint from the browser.
- **Scanner panel gating**: the panel stays hidden until a valid number is entered, and scanners that can't run surface as **disabled toggles with a failure reason**, so it's obvious *why* a scanner is unavailable.
- **"PCB phosphor console" theme** — a redesigned masthead, scanner toggles and footer.

## Features

- Check if a phone number exists and gather basics: country, line type, carrier
- OSINT footprinting via external APIs, phone books and multiple search backends (SerpAPI, Google CSE, SearXNG, Google search dorks)
- Live network intelligence: Twilio Lookup v2 and HLR status
- Reputation / fraud scoring and **breach-exposure** checks
- Range/rate-center allocation for +1 (NANPA) and select EU countries (OVH)
- Redundant validation across Numverify, Veriphone, Abstract and NumlookupAPI
- Run scans from the browser GUI, or programmatically via the [REST API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/sundowndev/phoneinfoga/master/web/docs/swagger.yaml) and [Go modules](https://pkg.go.dev/github.com/sundowndev/phoneinfoga/v2)
- **Lookup persistence & history**: every web lookup is saved to an embedded SQLite database so returning to a number replays the last result without re-scanning (see below)

## Quick start

```bash
# 1. Copy the credentials template and fill in the scanners you want.
cp .env.example .env

# 2. Serve the web client + API (reads .env automatically).
phoneinfoga serve
```

Every scanner is skipped cleanly when its credentials are absent, so you can start with none configured and enable providers incrementally. See [`.env.example`](./.env.example) for the full, annotated list of environment variables and [`docs/features/`](./docs/features/) for the design spec behind each scanner.

## Self-hosting: lookup persistence

When you run `phoneinfoga serve`, every lookup made from the web UI is persisted to an
embedded [SQLite](https://sqlite.org) database (pure-Go driver — no CGO, no extra services).
Returning to a number you've looked up before **replays the most recent stored result without
re-running the scanners**, and a per-number **Previous lookups** dropdown opens any past lookup
in full detail. The CLI `scan` command is unaffected — persistence is web-only.

**Database path.** Set `PHONEINFOGA_DB_PATH` to choose the file (default `./phoneinfoga.db`).
The parent directory is created automatically on startup. Persistence is always on for `serve`;
if the database can't be opened or migrated, `serve` exits with an error.

**Docker.** The provided [`support/docker/docker-compose.yml`](./support/docker/docker-compose.yml)
sets `PHONEINFOGA_DB_PATH=/app/data/phoneinfoga.db` and bind-mounts `./data:/app/data`, so lookups
survive `docker compose down`/`up`:

```bash
cd support/docker
docker compose up -d      # creates ./data/phoneinfoga.db, persisted across restarts
```

**Backup.** To back up, copy the **entire `./data` directory** while the service is stopped (or
at least copy the `.db` file **together with its `-wal` and `-shm` sidecar files** — the WAL may
hold committed data not yet folded into the main file):

```bash
cp -a support/docker/data /path/to/backup/   # or back up $PHONEINFOGA_DB_PATH and its -wal/-shm
```

**Privacy / PII.** Records include the **client IP**, **User-Agent** and the **phone numbers**
looked up, and are retained **indefinitely** by design. Treat the database as personal data. To
enforce a retention window, run the purge script (deletes lookups older than N days — 30 by
default — and their scanner results, then `VACUUM`s; it's idempotent):

```bash
# delete records older than 30 days
PHONEINFOGA_DB_PATH=./phoneinfoga.db support/scripts/purge-lookups.sh

# or a custom window (e.g. 7 days) — needs the sqlite3 CLI
support/scripts/purge-lookups.sh 7
```

Schedule it (e.g. via cron) if you want automatic pruning.

## Anti-features

- Does not claim to provide relevant or verified data — it's just a tool!
- Does not "track" a phone or its owner in real time
- Does not get precise phone location
- Does not hack a phone

## License

This tool is licensed under the GNU General Public License v3.0, following upstream [sundowndev/phoneinfoga](https://github.com/sundowndev/phoneinfoga).

[Icon](https://www.flaticon.com/free-icon/fingerprint-search-symbol-of-secret-service-investigation_48838) made by <a href="https://www.freepik.com/" title="Freepik">Freepik</a> from <a href="https://www.flaticon.com/" title="Flaticon">flaticon.com</a> is licensed by <a href="http://creativecommons.org/licenses/by/3.0/" title="Creative Commons BY 3.0" target="_blank">CC 3.0 BY</a>.
