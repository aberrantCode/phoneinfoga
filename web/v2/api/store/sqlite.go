package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrLookupNotFound is returned by operations that target a lookup id that does not exist.
var ErrLookupNotFound = errors.New("store: lookup not found")

// defaultListLimit caps ListLookupsByNumber when the caller passes a non-positive limit.
const defaultListLimit = 20

// SQLiteStore is the embedded-SQLite implementation of Store. It satisfies the Store
// interface and additionally exposes Close for lifecycle management by the serve command.
type SQLiteStore struct {
	db *sql.DB
}

// compile-time assertion that *SQLiteStore implements Store.
var _ Store = (*SQLiteStore)(nil)

// New opens (creating the parent directory if needed) an SQLite database at path and
// configures per-connection PRAGMAs. It does not run migrations; call Migrate for that.
func New(path string) (*SQLiteStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("store: database path must not be empty")
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("store: create db directory %q: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open database %q: %w", path, err)
	}

	// Pin to a single connection so per-connection PRAGMAs (foreign_keys, busy_timeout)
	// always apply. SQLite serializes writers regardless, so this costs little here.
	db.SetMaxOpenConns(1)

	pragmas := []string{
		// journal_mode=DELETE (rollback journal) rather than WAL: every commit lands in
		// the single .db file, with only a transient -journal during a write. This is the
		// right choice for the mounted-volume Docker workflow (AC4): WAL leaves -wal/-shm
		// sidecars that, after an unclean container stop, go stale and prevent a fresh
		// container from reading the committed data. DELETE has no such sidecars, and with
		// SetMaxOpenConns(1) we don't need WAL's concurrent-reader capability anyway.
		"PRAGMA journal_mode=DELETE",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		// synchronous=FULL fsyncs each commit so a committed lookup is durably on disk
		// before the call returns (negligible cost for this low write-rate workload).
		"PRAGMA synchronous=FULL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("store: apply %q: %w", p, err)
		}
	}

	return &SQLiteStore{db: db}, nil
}

// Close releases the underlying database handle.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Migrate applies embedded migrations whose version exceeds the database's current
// PRAGMA user_version, in filename order, each within its own transaction. Idempotent.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("store: read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var current int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("store: read user_version: %w", err)
	}

	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		if version <= current {
			continue
		}

		body, err := fs.ReadFile(migrationFiles, "migrations/"+name)
		if err != nil {
			return fmt.Errorf("store: read migration %q: %w", name, err)
		}

		if err := s.applyMigration(ctx, version, string(body)); err != nil {
			return fmt.Errorf("store: apply migration %q: %w", name, err)
		}
	}

	return nil
}

// migrationVersion extracts the leading numeric prefix of a migration filename (e.g.
// "0001_init.sql" -> 1). This value is used only to set PRAGMA user_version.
func migrationVersion(name string) (int, error) {
	prefix := name
	if i := strings.IndexAny(name, "_."); i > 0 {
		prefix = name[:i]
	}
	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("store: migration %q has no numeric version prefix: %w", name, err)
	}
	return version, nil
}

func (s *SQLiteStore) applyMigration(ctx context.Context, version int, body string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, body); err != nil {
		_ = tx.Rollback()
		return err
	}
	// user_version cannot be parameterized; version is an int derived from a filename.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// CreateLookup inserts a new request record with a store-assigned id, created_at and
// status=pending, and returns the persisted record.
func (s *SQLiteStore) CreateLookup(ctx context.Context, l Lookup) (Lookup, error) {
	id, err := NewUUID()
	if err != nil {
		return Lookup{}, err
	}

	scanners := l.ScannersRequested
	if scanners == nil {
		scanners = []string{}
	}
	scannersJSON, err := marshalScanners(scanners)
	if err != nil {
		return Lookup{}, err
	}

	now := time.Now().UTC()

	const q = `INSERT INTO lookups (` + lookupColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := s.db.ExecContext(ctx, q,
		id, l.E164, l.NumberInput, boolToInt(l.Valid),
		nullableString(l.RawLocal), nullableString(l.Local), nullableString(l.International),
		l.CountryCode, nullableString(l.Country), nullableString(l.Carrier),
		scannersJSON, nullableString(l.ClientIP), nullableString(l.UserAgent),
		StatusPending, formatTime(now), nil,
	); err != nil {
		return Lookup{}, fmt.Errorf("store: insert lookup: %w", err)
	}

	out := l
	out.ID = id
	out.Status = StatusPending
	out.CreatedAt = now
	out.CompletedAt = nil
	out.ScannersRequested = scanners
	out.Results = nil
	return out, nil
}

// SaveScannerResult upserts a scanner's result, keyed on the unique (lookup_id, scanner).
func (s *SQLiteStore) SaveScannerResult(ctx context.Context, r ScannerResult) error {
	id := r.ID
	if id == "" {
		var err error
		if id, err = NewUUID(); err != nil {
			return err
		}
	}

	const q = `INSERT INTO scanner_results (` + resultColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(lookup_id, scanner) DO UPDATE SET
			status = excluded.status,
			error_message = excluded.error_message,
			raw_response = excluded.raw_response,
			started_at = excluded.started_at,
			finished_at = excluded.finished_at,
			duration_ms = excluded.duration_ms`
	if _, err := s.db.ExecContext(ctx, q,
		id, r.LookupID, r.Scanner, r.Status,
		nullableString(r.ErrorMessage), nullableRaw(r.RawResponse),
		formatTime(r.StartedAt), formatTime(r.FinishedAt), r.DurationMs,
	); err != nil {
		return fmt.Errorf("store: save scanner result: %w", err)
	}
	return nil
}

// CloseLookup finalizes a lookup: completed_at is set to now and status becomes complete
// when every requested scanner produced a result row, otherwise partial.
func (s *SQLiteStore) CloseLookup(ctx context.Context, id string) (Lookup, error) {
	existing, err := s.GetLookup(ctx, id)
	if err != nil {
		return Lookup{}, err
	}
	if existing == nil {
		return Lookup{}, ErrLookupNotFound
	}

	present := make(map[string]struct{}, len(existing.Results))
	for _, res := range existing.Results {
		present[res.Scanner] = struct{}{}
	}
	status := StatusComplete
	for _, name := range existing.ScannersRequested {
		if _, ok := present[name]; !ok {
			status = StatusPartial
			break
		}
	}

	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx,
		"UPDATE lookups SET status = ?, completed_at = ? WHERE id = ?",
		status, formatTime(now), id,
	); err != nil {
		return Lookup{}, fmt.Errorf("store: close lookup: %w", err)
	}

	existing.Status = status
	existing.CompletedAt = &now
	return *existing, nil
}

// GetLookup returns a lookup by id hydrated with its results, or (nil, nil) if absent.
func (s *SQLiteStore) GetLookup(ctx context.Context, id string) (*Lookup, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+lookupColumns+" FROM lookups WHERE id = ?", id)
	l, err := scanLookup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get lookup: %w", err)
	}
	if err := s.hydrateResults(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// GetLatestLookupByNumber returns the newest lookup for e164, hydrated, or (nil, nil).
func (s *SQLiteStore) GetLatestLookupByNumber(ctx context.Context, e164 string) (*Lookup, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT "+lookupColumns+" FROM lookups WHERE e164 = ? ORDER BY created_at DESC LIMIT 1", e164)
	l, err := scanLookup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get latest lookup: %w", err)
	}
	if err := s.hydrateResults(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

// ListLookupsByNumber returns per-number summaries (without results), newest first.
func (s *SQLiteStore) ListLookupsByNumber(ctx context.Context, e164 string, limit int) ([]Lookup, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}

	rows, err := s.db.QueryContext(ctx,
		"SELECT "+lookupColumns+" FROM lookups WHERE e164 = ? ORDER BY created_at DESC LIMIT ?",
		e164, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list lookups: %w", err)
	}
	defer rows.Close()

	var out []Lookup
	for rows.Next() {
		l, err := scanLookup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list lookups: %w", err)
	}
	return out, nil
}

func (s *SQLiteStore) hydrateResults(ctx context.Context, l *Lookup) error {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+resultColumns+" FROM scanner_results WHERE lookup_id = ? ORDER BY started_at ASC",
		l.ID)
	if err != nil {
		return fmt.Errorf("store: hydrate results: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		r, err := scanResult(rows)
		if err != nil {
			return err
		}
		l.Results = append(l.Results, *r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: hydrate results: %w", err)
	}
	return nil
}
