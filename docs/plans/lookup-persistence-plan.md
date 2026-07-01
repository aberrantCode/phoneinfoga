# Implementation Plan — Lookup Persistence & History

**Spec:** [`docs/features/lookup-persistence.md`](../features/lookup-persistence.md)
**Branch:** work on the current feature branch (or `feat/lookup-persistence` off `dev`).

## How to use this checklist

- Tasks are ordered; earlier phases unblock later ones. Do them **top to bottom**.
- Each task is small, independently testable, and **must not break the build**.
- Follow repo rules: **TDD** (write the failing test first), immutability, many small
  files, comprehensive error handling, no `console.log`, no hardcoded values.
- After a task passes: tick its box `[x]`, then make **one atomic conventional commit**
  scoped to that task.
- A phase's **Gate** must be green before moving on.

Legend: `[ ]` todo · `[x]` done. Type tags: `feat` `test` `chore` `docs`.

---

## Phase 0 — Scaffolding & dependencies

- [x] **0.1** `chore` Add `modernc.org/sqlite` to `go.mod`; run `go mod tidy`. Verify it
  resolves with **no CGO** (`go build ./...` on a clean env). *(go.mod, go.sum)*
  Pinned `v1.28.0` (keeps the `go 1.20` directive; latest requires Go 1.23+). Anchored via a
  blank driver import in `web/v2/api/store/driver.go` so `go mod tidy` retains it. Pre-existing
  `examples/plugin` build error (`-buildmode=plugin` example, no `main`) is unrelated.
- [x] **0.2** `chore` Create empty package dirs: `web/v2/api/store/` and
  `web/v2/api/store/migrations/`. Add a package doc comment.
  Added `store/doc.go` (package doc comment) and `store/migrations/.gitkeep` (git can't
  track empty dirs; SQL migrations land here in task 1.3).
- **Gate:** ✅ `go build` passes for all real packages; `modernc.org/sqlite` present; no CGO
  (`CGO_ENABLED=0` build clean). Pre-existing `examples/plugin` `-buildmode=plugin` error is
  out of scope (real build target is `.`, not `./...`).

## Phase 1 — Store package (TDD)

- [x] **1.1** `feat` Add `web/v2/api/store/models.go` — `Lookup` and `ScannerResult`
  structs (fields per spec §5), `Store` interface (spec §6), status constants.
  `CompletedAt *time.Time` (nullable), `RawResponse json.RawMessage` (verbatim), status
  consts `Status*`/`ResultStatus*`. `models_test.go` asserts constant values + hydration.
- [x] **1.2** `feat` Add UUIDv4 helper `web/v2/api/store/id.go` using `crypto/rand`
  (no new dependency) + a unit test asserting format/uniqueness.
  `NewUUID()` sets RFC 4122 version/variant bits; tests assert canonical v4 regex + 1000
  unique ids. Package coverage 83.3%.
- [x] **1.3** `feat` Add migration `web/v2/api/store/migrations/0001_init.sql` (both
  tables + indexes per spec §5). Embed via `embed.FS`.
  `migrations.go` embeds `migrations/*.sql`; test applies the SQL to an in-memory DB and
  asserts both tables + all three indexes exist. Removed the now-redundant `.gitkeep`.
- [x] **1.4** `test` Write `store_test.go` (RED): open temp-file DB → `Migrate` →
  `CreateLookup` → `SaveScannerResult` (success + error rows) → `CloseLookup` →
  `GetLookup`/`GetLatestLookupByNumber`/`ListLookupsByNumber`. Assert ordering,
  hydration, cascade delete, upsert on duplicate `(lookup_id, scanner)`, and
  `Migrate` idempotency.
  RED as intended — references `New`/`*SQLiteStore` (undefined until 1.5). Contract: store
  assigns id/created_at/status=pending; results ordered by started_at; complete vs partial
  close; latest/list newest-first with limit + number scoping; FK cascade. Non-test build
  stays green; store package test compiles GREEN in 1.5.
- [x] **1.5** `feat` Implement `web/v2/api/store/sqlite.go` (`sqliteStore`) to GREEN:
  `sql.Open("sqlite", path)`, WAL/busy_timeout/foreign_keys PRAGMAs, `os.MkdirAll`
  on parent dir, embedded migration runner via `PRAGMA user_version`.
  `*SQLiteStore` (+`Close`, `New`). `SetMaxOpenConns(1)` so per-connection PRAGMAs (FK,
  busy_timeout) always apply. Parameterized SQL throughout; `user_version` set from the
  int filename prefix only. Scan/format helpers in `rows.go`. Full RED suite GREEN; store
  coverage 80.5% (≥80% gate). `ErrLookupNotFound` exported for handler 404s.
- [x] **1.6** `feat` `CloseLookup` status math: `complete` if every requested scanner
  has a result row, else `partial`. Cover both in tests.
  Membership-set logic (not row count): complete/partial + edge case where an unrequested
  extra result must not mask a missing requested scanner. All three branches covered.
- **Gate:** ✅ `go test ./web/v2/api/store/...` green; package coverage 80.5% (≥80%).

## Phase 2 — Config & serve wiring

- [x] **2.1** `feat` Add `handlers.InitStore(store)` + package var `Store` in
  `web/v2/api/handlers/init.go` (mirror `RemoteLibrary` pattern).
  `Store store.Store` (nil when unconfigured — CLI/tests); `InitStore` is a plain setter
  (no `sync.Once`) so tests can re-inject. Persistence-aware handlers must nil-check it.
- [x] **2.2** `feat` Resolve `PHONEINFOGA_DB_PATH` (default `./phoneinfoga.db`) in
  `cmd/serve.go` `PreRun`; open store, `Migrate()`, `handlers.InitStore(store)`;
  `exitWithError` on failure.
  Extracted testable `resolveDBPath`/`setupStore` (`cmd/serve_store.go`); PreRun calls
  `exitWithError` on failure. Smoke-verified: `serve` boots, creates the DB (+parent dir,
  WAL files), `user_version=1`, both tables present. Automated assertion follows in 2.3.
- [x] **2.3** `test` Add a serve-wiring test (or extend existing) asserting `handlers.Store`
  is non-nil after init with a temp path.
  `cmd/serve_store_test.go`: `setupStore(temp)` injects a non-nil `handlers.Store` and creates
  the DB file (+parent dir); default/env path resolution; open-error path. New-code coverage
  `resolveDBPath` 100%, `setupStore` 87.5%.
- **Gate:** ✅ `phoneinfoga serve` boots, creates the DB file, and migrates cleanly (live smoke
  test in 2.2 + automated `setupStore` test in 2.3).

## Phase 3 — Lookup lifecycle + read API (TDD)

- [x] **3.1** `feat` `web/v2/api/handlers/lookups.go` — `POST /v2/lookups`:
  validate number, build metadata (reuse `number.NewNumber`), capture
  `ctx.ClientIP()` + `User-Agent`, `CreateLookup(status=pending)`, return
  `{id, number, scannersRequested, clientIp, createdAt, status}`.
  `CreateLookup` handler + `CreateLookupInput`/`CreateLookupResponse`; nil-store guard (500);
  `number` built-in validator → 400 on bad input. Handler-level tests (direct gin.Context):
  happy/persisted (AC1), invalid number, nil store. Route registered in 3.6. Handler 80%,
  pkg 95.2%.
- [x] **3.2** `feat` `POST /v2/lookups/:id/close` → `CloseLookup`; return summary; `404`
  if unknown id.
  `CloseLookup` handler + `CloseLookupResponse` summary (id/status/scannersRequested/created/
  completed); `errors.Is(err, store.ErrLookupNotFound)` → 404; nil-store → 500. Tests: complete
  close, 404, nil store. Route registered in 3.6. Pkg coverage 94.4%.
- [x] **3.3** `feat` `GET /v2/lookups/:id` → full detail (spec §7); `404` if missing.
  `GetLookup` handler + shared `lookup_detail.go` projection (`LookupDetailResponse`,
  `lookupDetail`); `raw` field emits verbatim payload / JSON `null` for errors (no omitempty).
  `nil` result → 404, nil store → 500. Tests: full detail + ordering, 404, nil store. Route
  in 3.6. Pkg coverage 94.1%.
- [x] **3.4** `feat` `GET /v2/lookups/latest?number=` → newest for number, full detail;
  `400` if `number` missing, `404` if none.
  `GetLatestLookup` handler + shared `e164FromQuery` (normalizes via `number.NewNumber`, 400 on
  missing/invalid). Reuses `lookupDetail`. Tests: newest returned, missing/invalid number → 400,
  none → 404, nil store → 500. Route in 3.6. Pkg coverage 94.2%.
- [x] **3.5** `feat` `GET /v2/lookups?number=&limit=` → per-number summaries newest-first;
  `400` if `number` missing; never returns other numbers.
  `ListLookups` handler + `LookupSummary`/`ListLookupsResponse` (no results); `listLimit` parses
  optional `limit` (0 → store default). Tests: ordering+limit, AC10 number-scoping, missing
  number → 400, nil store → 500. Route in 3.6. Pkg coverage 94.1%.
- [x] **3.6** `feat` Register all five routes in `web/v2/api/server/server.go`.
  Registered via `api.WrapHandler`: POST `/v2/lookups`, POST `/v2/lookups/:id/close`, GET
  `/v2/lookups`, GET `/v2/lookups/latest`, GET `/v2/lookups/:id`. `server_test.go` asserts no
  panic on the static `latest` vs param `:id` siblings (gin 1.9 radix tree handles it) and that
  existing routes remain. Backward compat intact.
- [x] **3.7** `test` `lookups_test.go` (`httptest` + injected in-memory store): happy paths,
  `400` on missing number, `404`s, and **AC10** (no cross-number leakage).
  `lookups_integration_test.go` drives `server.NewServer()` end-to-end: create→close→get→latest
  →list lifecycle, 400 (missing/invalid number), 404s (unknown id, no-data latest), AC10 (list &
  latest scoped to queried number), and a `/v2/numbers` backward-compat check.
- **Gate:** ✅ `go test ./web/v2/api/...` green (store, handlers, server, api all pass).

## Phase 4 — Persist scanner results on run

- [ ] **4.1** `test` Extend `scanners_test.go` (RED): `POST /run` with `lookupId` writes a
  `scanner_results` row (success **and** error cases); **without** `lookupId` writes
  nothing (backward compat, AC13).
- [ ] **4.2** `feat` Add optional `LookupId` to `RunScannerInput`; in `RunScanner`, time the
  run and — when `lookupId` present — `SaveScannerResult` (raw verbatim, status/message,
  timing). Persist errors too.
- [ ] **4.3** `feat` Make persistence non-fatal: on store error, log warn and still return
  the scan result (spec §7).
- **Gate:** `go test ./web/v2/api/...` green; AC1–AC3, AC13 covered.

## Phase 5 — Frontend API client & state model

- [ ] **5.1** `feat` Add lookup API calls to `web/client/src/utils/index.ts`:
  `createLookup`, `closeLookup`, `getLookup`, `getLatestLookup`, `listLookups`.
- [ ] **5.2** `feat` Thread optional `lookupId` through the existing per-scanner `run`
  call (Scanner.vue / utils) without changing behavior when absent.
- [ ] **5.3** `feat` Introduce a `viewState` model in `Scan.vue`
  (`entry | results`, source `fresh | replay`, `activeLookup`).
- **Gate:** `yarn build` (in `web/client`) succeeds; existing UI still works.

## Phase 6 — Results state, metadata panel, Start over

- [ ] **6.1** `feat` In results state, **hide** the phone-number field, country selector,
  and Lookup button; render a **Start over** control that resets to entry (AC6).
- [ ] **6.2** `feat` Extend the metadata panel to show request/results record: lookup time,
  client IP, scanners requested, overall status (AC5) — alongside existing number metadata.
- [ ] **6.3** `feat` Fresh-lookup orchestration: `createLookup` → per-scanner runs with
  `lookupId` → `closeLookup` → enter `results/fresh` (AC1–AC3 end-to-end via UI).
- [ ] **6.4** `test` Component test: results state hides input + shows Start over; metadata
  panel shows the record fields.
- **Gate:** manual fresh lookup persists and renders; `yarn test:unit` green.

## Phase 7 — Replay, Run new lookup, Previous lookups dropdown

- [ ] **7.1** `feat` On Lookup click, call `getLatestLookup`; on **200** enter `results/replay`
  and render stored detail with **no** scans; on **404** run a fresh lookup (AC7).
- [ ] **7.2** `feat` Add **Run new lookup** (forces fresh scan of the same number, AC8).
- [ ] **7.3** `feat` Add **Previous lookups ▾** dropdown: `listLookups(number)`; selecting an
  entry loads `getLookup(id)` and renders in `replay` mode (AC9).
- [ ] **7.4** `feat` Replay banner ("Showing your most recent lookup from &lt;time&gt;").
- [ ] **7.5** `test` Component tests: entry→replay on 200 (**assert no `/run` calls**),
  entry→fresh on 404, Run new lookup re-scans, dropdown loads detail (AC7–AC9).
- **Gate:** `yarn test:unit` green; manual replay path shows results without re-scanning.

## Phase 8 — Docker, purge script, docs

- [ ] **8.1** `chore` `support/docker/docker-compose.yml`: add
  `PHONEINFOGA_DB_PATH=/app/data/phoneinfoga.db` env + `./data:/app/data` volume.
- [ ] **8.2** `chore` `.env.example`: document `PHONEINFOGA_DB_PATH`.
- [ ] **8.3** `feat` `support/scripts/purge-lookups.sh`: delete records older than N days
  (arg/env) and `VACUUM`. Idempotent.
- [ ] **8.4** `docs` README self-hosting section: DB path, volume mount, **backup = copy
  `./data` (incl. `-wal`/`-shm`)**, PII note, purge usage.
- **Gate:** `docker compose up` builds (no CGO), persists across `down`/`up` with the volume.

## Phase 9 — Verification & close-out

- [ ] **9.1** `test` Full Go suite green: `go test ./...`; confirm ≥80% on new packages.
- [ ] **9.2** `test` Frontend suite green: `yarn lint && yarn test:unit && yarn build`.
- [ ] **9.3** `test` Manual E2E in Docker: fresh lookup → restart container → replay from
  disk (AC4); verify AC1–AC13 checklist in the spec.
- [ ] **9.4** `chore` Self-review for CRITICAL/HIGH issues (security-reviewer on IP/PII
  handling + SQL); ensure parameterized queries only.
- [ ] **9.5** `docs` Flip spec **Status** to `Implemented`; open a PR to `dev` with Summary
  + Test Plan (mind the 400/800-line size rule — split the PR if needed).
- **Gate:** all boxes ticked; PR opened.
