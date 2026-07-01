-- 0001_init: lookup persistence & history schema.
-- Times are stored as RFC3339 UTC text. IDs are UUIDv4 strings.

CREATE TABLE IF NOT EXISTS lookups (
    id                 TEXT PRIMARY KEY,
    e164               TEXT NOT NULL,
    number_input       TEXT NOT NULL,
    valid              INTEGER,
    raw_local          TEXT,
    local              TEXT,
    international       TEXT,
    country_code       INTEGER,
    country            TEXT,
    carrier            TEXT,
    scanners_requested TEXT NOT NULL,
    client_ip          TEXT,
    user_agent         TEXT,
    status             TEXT NOT NULL,
    created_at         TEXT NOT NULL,
    completed_at       TEXT
);

CREATE INDEX IF NOT EXISTS idx_lookups_e164_created ON lookups(e164, created_at DESC);

CREATE TABLE IF NOT EXISTS scanner_results (
    id            TEXT PRIMARY KEY,
    lookup_id     TEXT NOT NULL REFERENCES lookups(id) ON DELETE CASCADE,
    scanner       TEXT NOT NULL,
    status        TEXT NOT NULL,
    error_message TEXT,
    raw_response  TEXT,
    started_at    TEXT NOT NULL,
    finished_at   TEXT NOT NULL,
    duration_ms   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_results_lookup ON scanner_results(lookup_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_results_lookup_scanner ON scanner_results(lookup_id, scanner);
