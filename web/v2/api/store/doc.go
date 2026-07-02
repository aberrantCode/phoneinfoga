// Package store persists PhoneInfoga web lookups and their per-scanner results
// in an embedded SQLite database.
//
// It uses the pure-Go modernc.org/sqlite driver (no CGO) with raw database/sql
// and an embedded migration runner (embed.FS + PRAGMA user_version), mirroring
// the dependency-light style of the rest of the codebase. The package exposes a
// Store interface whose sqlite implementation is wired into the serve command,
// following the same package-var pattern as web/v2/api/handlers.
package store
