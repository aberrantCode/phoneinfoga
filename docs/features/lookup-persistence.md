# Feature Spec — Lookup Persistence & History

| | |
|---|---|
| **Feature ID** | `lookup-persistence` |
| **Category** | Cross-cutting infrastructure (persistence + web UX) |
| **Status** | Draft — proposed |
| **External dependency** | None (embedded SQLite via `modernc.org/sqlite`, pure Go) |
| **Auth** | None (PhoneInfoga has no auth; see §9 privacy notes) |
| **Default state** | **Enabled.** Persistence is always on when running `serve`. |
| **Persistence** | SQLite file at `PHONEINFOGA_DB_PATH`, mountable as a Docker volume |

---

## 1. Summary

Persist **every web phone-number lookup** into an embedded SQLite database that can
be bind-mounted into the Docker container and survives service start/stop.

When a user clicks **Lookup**, the server creates a **lookup request record**
(number, scanners requested, timestamp, client IP, user agent). As each scanner
finishes, the server persists that scanner's **exact response** plus a **normalized
result** (success/error, error message, raw payload, timing).

The results page renders exactly as it does today, but the interface changes:
the metadata panel additionally surfaces the **request/results record**, and the
phone-number field, country selector, and Lookup button are **hidden** — replaced
by a **Start over** control. When a user returns later and enters a number that was
looked up before, the page shows the **most recent stored result without running a
new lookup**, while offering controls to **run a fresh request** and a
**Previous lookups** dropdown (scoped to that number) to open any past lookup in
full detail.

## 2. Motivation & gap filled

PhoneInfoga is entirely stateless today: lookups are computed and thrown away. There
is no history, no audit trail, and no way to see what a number resolved to last time
without paying for (and rate-limiting against) the upstream APIs again. This feature
adds:

- **Durability / backup** — a mountable SQLite file that outlives containers.
- **Replay without re-scanning** — return visits show the last result for free.
- **Auditability** — who (IP) looked up what (number, scanners) and when, plus the
  verbatim provider responses for later inspection.
- **Cost/quota savings** — avoid re-hitting paid scanner APIs for a repeat number.

## 3. Goals / Non-goals

**Goals**
- Persist a request record the moment a lookup starts (before any scanner runs).
- Persist each scanner's exact response + a normalized success/error result.
- Survive service restarts; store in a single mountable SQLite file.
- Change the web UI to a two-state flow (entry ↔ results) with Start over.
- On repeat number entry, replay the most recent stored result **without re-running**.
- Provide a per-number "Previous lookups" list/dropdown → full detail of any past lookup.
- Pure-Go SQLite driver — **no CGO**, Docker build stages unchanged.

**Non-goals**
- No authentication, accounts, or per-user isolation (PhoneInfoga has none).
- No cross-number "all lookups" browsing (history is **per-number only**, by decision).
- No change to the CLI `scan` command's behavior (persistence is web-only for now).
- No change to how individual scanner results are *rendered* (same components).
- No server-side scanner orchestration change — the browser still drives scans.

## 4. Architecture decisions (locked)

These were decided up front and constrain the design:

1. **Client-orchestrated + persist hooks.** The browser keeps driving lookups
   (`/v2/numbers` → per-scanner `/dryrun` + `/run`), preserving today's live
   per-scanner spinners/ETA. New `/v2/lookups` endpoints create/close the request
   record; the existing `/v2/scanners/:scanner/run` handler is **extended** to persist
   each scanner result when a `lookupId` is supplied.
2. **Per-number history only.** No global/cross-number list and no client identity.
   History endpoints **require** a number and only ever return that number's lookups.
3. **modernc pure-Go SQLite + raw `database/sql`** with an embedded migration runner
   (`embed.FS` + `PRAGMA user_version`). No ORM, no CGO — matches the repo's
   dependency-light style and keeps the alpine Docker build untouched.
4. **Raw client IP, indefinite retention.** Full IP via Gin `ctx.ClientIP()`, kept
   forever (backup intent). PII is documented and a manual purge script is provided.

## 5. Data model

Two tables. Times are stored as RFC3339 UTC text. IDs are UUIDv4 (generated with
`crypto/rand`, no new dependency).

### 5.1 `lookups` (request record)

| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PK | UUIDv4 |
| `e164` | TEXT NOT NULL | normalized number key (indexed) |
| `number_input` | TEXT NOT NULL | raw string as submitted |
| `valid` | INTEGER | 0/1 from libphonenumber |
| `raw_local` | TEXT | number metadata snapshot… |
| `local` | TEXT | |
| `international` | TEXT | |
| `country_code` | INTEGER | |
| `country` | TEXT | |
| `carrier` | TEXT | |
| `scanners_requested` | TEXT NOT NULL | JSON array of scanner names |
| `client_ip` | TEXT | `ctx.ClientIP()` (PII) |
| `user_agent` | TEXT | request header (PII-adjacent) |
| `status` | TEXT NOT NULL | `pending` \| `complete` \| `partial` |
| `created_at` | TEXT NOT NULL | RFC3339 UTC |
| `completed_at` | TEXT | RFC3339 UTC, null until closed |

Index: `CREATE INDEX idx_lookups_e164_created ON lookups(e164, created_at DESC);`

### 5.2 `scanner_results` (one row per scanner per lookup)

| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PK | UUIDv4 |
| `lookup_id` | TEXT NOT NULL | FK → `lookups(id)` ON DELETE CASCADE (indexed) |
| `scanner` | TEXT NOT NULL | scanner `Name()` |
| `status` | TEXT NOT NULL | `success` \| `error` |
| `error_message` | TEXT | non-empty when status=error |
| `raw_response` | TEXT | **exact** scanner payload, verbatim JSON |
| `started_at` | TEXT NOT NULL | RFC3339 UTC |
| `finished_at` | TEXT NOT NULL | RFC3339 UTC |
| `duration_ms` | INTEGER NOT NULL | |

Index: `CREATE INDEX idx_results_lookup ON scanner_results(lookup_id);`
Unique: `CREATE UNIQUE INDEX idx_results_lookup_scanner ON scanner_results(lookup_id, scanner);`
(re-running a scanner within the same lookup upserts its row.)

PRAGMAs on open: `journal_mode=DELETE`, `busy_timeout=5000`, `foreign_keys=ON`,
`synchronous=FULL`. (Rollback-journal DELETE mode — not WAL — is used deliberately: with
`SetMaxOpenConns(1)` there is no need for WAL's concurrent readers, and WAL's `-wal`/`-shm`
sidecars can go stale after an unclean container stop and hide committed data from a fresh
container. DELETE keeps everything in the single `.db` file, so backup = copy that one file
and the mounted-volume Docker restart (AC4) is reliable.)

## 6. Store package (`web/v2/api/store`)

Mirrors the existing `handlers.Init` / `handlers.RemoteLibrary` package-var pattern.

```go
type Store interface {
    Migrate(context.Context) error
    CreateLookup(context.Context, Lookup) (Lookup, error)      // status=pending
    SaveScannerResult(context.Context, ScannerResult) error    // upsert on (lookup_id, scanner)
    CloseLookup(context.Context, id string) (Lookup, error)    // sets completed_at + status
    GetLookup(context.Context, id string) (*Lookup, error)     // hydrates Results
    GetLatestLookupByNumber(context.Context, e164 string) (*Lookup, error) // newest, hydrated
    ListLookupsByNumber(context.Context, e164 string, limit int) ([]Lookup, error) // summaries
}
```

- `sqliteStore` wraps `*sql.DB` (driver `"sqlite"`), migrations embedded via `embed.FS`.
- `CloseLookup` sets `status="complete"` when every requested scanner has a result row,
  else `status="partial"`.
- `raw_response` is stored as received (`json.RawMessage`), never re-shaped.

## 7. API contract (new + extended)

Registered in `web/v2/api/server/server.go` `registerRoutes()` via `api.WrapHandler`.

| Method & path | Purpose |
|---|---|
| `POST /v2/lookups` | Create request record; capture IP/UA/metadata; return `{id, number, status:"pending"}` |
| `POST /v2/scanners/:scanner/run` *(extended)* | Accepts optional `lookupId`; persists that scanner's result (success or error) |
| `POST /v2/lookups/:id/close` | Finalize: set `completed_at`, compute `complete`/`partial` |
| `GET /v2/lookups?number=<n>&limit=20` | **Per-number** summaries, newest first (powers dropdown) |
| `GET /v2/lookups/latest?number=<n>` | Most recent lookup for a number, full detail (replay); 404 if none |
| `GET /v2/lookups/:id` | Full detail (metadata + all scanner results) |

Rules:
- `number` query params are normalized through `number.NewNumber` to E164 before query.
- History endpoints **require** `number`; missing → `400`. They never return other numbers.
- Persisting a scanner result must **never** fail the `/run` response — on store error,
  log a warning and still return the scan result (availability > durability).
- `/run` **without** `lookupId`, `/v2/numbers`, `/dryrun`, `/scanners` are unchanged
  (full backward compatibility).

Detail payload (for `GET /v2/lookups/:id` and `/latest`):

```json
{
  "id": "…", "status": "complete",
  "createdAt": "…", "completedAt": "…",
  "clientIp": "203.0.113.7", "userAgent": "…",
  "number": { "valid": true, "e164": "+33612345678", "country": "FR", "carrier": "…", "...": "…" },
  "scannersRequested": ["local", "numverify"],
  "results": [
    { "scanner": "local", "status": "success", "raw": { /* verbatim */ }, "durationMs": 12, "startedAt": "…", "finishedAt": "…" },
    { "scanner": "numverify", "status": "error", "errorMessage": "quota exceeded", "raw": null, "durationMs": 340, "startedAt": "…", "finishedAt": "…" }
  ]
}
```

## 8. Web UI flow (`web/client/src/views/Scan.vue`)

Two states: **entry** and **results**. Results has two sources: `fresh` (just ran)
or `replay` (loaded from history). The **per-scanner display components are unchanged**
— persisted `raw` payloads are byte-identical to live `/run` results, so
`ScannerSummary` / `SearchActionGroups` / `JsonViewer` are fed the stored payloads
exactly as they are fed live ones.

**On Lookup click:**
1. Validate → E164.
2. `GET /v2/lookups/latest?number=E164`.
   - **200** → enter `results/replay`: render stored metadata + results, run **no** scans.
     Banner: "Showing your most recent lookup from &lt;time&gt;." Controls:
     **[Run new lookup]** · **[Start over]** · **[Previous lookups ▾]**.
   - **404** → proceed to fresh lookup (step 3).
3. **Fresh lookup:**
   1. `POST /v2/lookups {number, scanners}` → `lookupId` + metadata.
   2. Existing per-scanner `dryrun`/`run` flow, adding `lookupId` to each `run` body.
   3. When all scanners settle → `POST /v2/lookups/:id/close`.
   4. Enter `results/fresh`.

**Results state:** phone-number field, country selector, and Lookup button are
**hidden**; a **Start over** link resets to entry. Metadata panel additionally shows:
lookup time, client IP, scanners requested, overall status.

- **Run new lookup** — forces step 3 for the same number (skips the `/latest` check).
- **Previous lookups ▾** — on open, `GET /v2/lookups?number=E164`; selecting an entry →
  `GET /v2/lookups/:id` → render its detail in `replay` mode (no re-run).

## 9. Configuration, Docker & privacy

- **Env** `PHONEINFOGA_DB_PATH` — default `./phoneinfoga.db`; Docker default
  `/app/data/phoneinfoga.db`. Parent dir created via `os.MkdirAll` on startup.
- **`serve` wiring** — open store + `Migrate()` in `NewServeCmd` `PreRun`, then
  `handlers.InitStore(store)`; fail fast (`exitWithError`) if the DB can't open.
- **docker-compose.yml** — add `PHONEINFOGA_DB_PATH` env and a `./data:/app/data`
  bind mount. **Backup = copy the `./data` directory** (a single `.db` file, no `-wal`/`-shm`).
- **`.env.example`** — document `PHONEINFOGA_DB_PATH`.
- **PII** — `client_ip`, `user_agent`, and phone numbers are personal data, stored
  raw and retained indefinitely by design. A purge script
  `support/scripts/purge-lookups.sh` deletes records older than N days and `VACUUM`s.
  This is documented in the README's self-hosting section.

## 10. Acceptance criteria

- **AC1 — Request record on start.** Given a valid number + selected scanners, when
  Lookup runs a fresh lookup, then a `lookups` row (e164, input, scanners_requested,
  created_at, client_ip, user_agent, status=`pending`) exists **before** any scanner runs.
- **AC2 — Per-scanner persistence.** For each scanner that runs, a `scanner_results`
  row is written with correct `status` (`success`/`error`), `error_message` when failing,
  `raw_response` verbatim, and timing. A success scanner and an error scanner each
  produce a correctly-typed row.
- **AC3 — Finalize.** After all requested scanners settle and `close` is called,
  `completed_at` is set and `status` is `complete` (all requested have rows) or `partial`.
- **AC4 — Survives restart / mountable.** After writing records, stopping and restarting
  the service (or container with `./data` mounted) leaves all lookups + results queryable.
- **AC5 — Results render unchanged.** A completed fresh lookup renders per-scanner results
  identically to today; the metadata panel additionally shows time, IP, scanners requested,
  and overall status.
- **AC6 — Input hidden in results.** In the results state the phone-number field, country
  selector, and Lookup button are hidden and replaced by a **Start over** control.
- **AC7 — Replay without re-running.** Given a prior lookup for a number, when the same
  number is entered and Lookup clicked, the most recent stored result is shown with **no**
  `/run` calls and **no** new `lookups`/`scanner_results` rows created.
- **AC8 — Force new request.** From replay, **Run new lookup** executes a full fresh
  lookup, persists a **new** `lookups` row + results, and shows those new results.
- **AC9 — Per-number history.** With multiple prior lookups, the **Previous lookups**
  dropdown lists that number's lookups newest-first; selecting one shows full detail
  (metadata + all scanner results) without re-running.
- **AC10 — No cross-number leakage.** History endpoints require `number` and never return
  lookups for any other number.
- **AC11 — PII & retention.** Client IP is stored raw; PII is documented; the purge script
  exists and removes old records + `VACUUM`s.
- **AC12 — No-CGO build.** `make build` and the Docker image build succeed with the
  modernc driver and **no** added C toolchain / `CGO_ENABLED`.
- **AC13 — Backward compatibility.** `/v2/numbers`, `/dryrun`, `/scanners`, and `/run`
  **without** `lookupId` behave exactly as before; CLI `scan` is unaffected.

## 11. Testing strategy

- **Store unit tests** (temp-file SQLite): create → save (success + error) → close →
  get/latest/list; cascade delete; ordering; upsert on re-run; migration idempotency.
- **Handler tests** (`httptest`, reuse the fake-scanner + `RemoteLibrary` pattern with an
  injected in-memory store): create, run-with-`lookupId` persists, close status math,
  list/latest/get, `400` on missing number, `404`s.
- **Frontend tests** (jest + vue-test-utils): entry→replay when `/latest` 200; entry→fresh
  when 404; results hides input; Start over resets; Run new lookup re-scans; dropdown loads
  detail. Assert **no** `/run` calls in replay.
- **Coverage** — ≥80% on new Go code (repo standard).
- **Manual/E2E** — full round trip in Docker with a mounted volume, then restart and confirm
  replay from disk.
