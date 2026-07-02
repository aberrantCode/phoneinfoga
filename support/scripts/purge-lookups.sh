#!/usr/bin/env bash
#
# purge-lookups.sh — delete persisted web lookups (and their scanner results, via
# the ON DELETE CASCADE foreign key) older than N days, then VACUUM to reclaim
# space. Safe to re-run (idempotent): re-running with no newly-aged rows is a no-op.
#
# Usage:
#   support/scripts/purge-lookups.sh [DAYS]
#
# DAYS is taken from the first argument, else $PHONEINFOGA_PURGE_DAYS, else 30.
# The database file is $PHONEINFOGA_DB_PATH (default ./phoneinfoga.db) — the same
# variable the `serve` command uses.
#
# Requires the sqlite3 CLI.

set -euo pipefail

DB_PATH="${PHONEINFOGA_DB_PATH:-./phoneinfoga.db}"
DAYS="${1:-${PHONEINFOGA_PURGE_DAYS:-30}}"

if ! [[ "${DAYS}" =~ ^[0-9]+$ ]]; then
  echo "purge-lookups: DAYS must be a non-negative integer (got '${DAYS}')" >&2
  exit 1
fi

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "purge-lookups: the sqlite3 CLI is required but was not found" >&2
  exit 1
fi

if [[ ! -f "${DB_PATH}" ]]; then
  echo "purge-lookups: no database at '${DB_PATH}'; nothing to purge."
  exit 0
fi

has_table="$(sqlite3 "${DB_PATH}" \
  "SELECT name FROM sqlite_master WHERE type='table' AND name='lookups';")"
if [[ -z "${has_table}" ]]; then
  echo "purge-lookups: no 'lookups' table in '${DB_PATH}'; nothing to purge."
  exit 0
fi

before="$(sqlite3 "${DB_PATH}" "SELECT COUNT(*) FROM lookups;")"

# julianday() parses the stored RFC3339 timestamps; unparseable rows yield NULL and
# are never matched. foreign_keys=ON is required for the cascade to scanner_results.
sqlite3 "${DB_PATH}" <<SQL
PRAGMA foreign_keys=ON;
DELETE FROM lookups WHERE julianday(created_at) < julianday('now', '-${DAYS} days');
VACUUM;
SQL

after="$(sqlite3 "${DB_PATH}" "SELECT COUNT(*) FROM lookups;")"

echo "purge-lookups: removed $((before - after)) lookup(s) older than ${DAYS} day(s) from '${DB_PATH}' (${after} remaining)."
