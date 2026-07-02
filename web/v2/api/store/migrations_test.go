package store

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedInitMigrationReadable(t *testing.T) {
	data, err := migrationFiles.ReadFile("migrations/0001_init.sql")
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "CREATE TABLE")
}

func TestInitMigrationAppliesAndCreatesSchema(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	sqlText, err := migrationFiles.ReadFile("migrations/0001_init.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(sqlText))
	require.NoError(t, err)

	// Both tables and the declared indexes must exist after applying the migration.
	names := map[string]string{
		"lookups":                    "table",
		"scanner_results":            "table",
		"idx_lookups_e164_created":   "index",
		"idx_results_lookup":         "index",
		"idx_results_lookup_scanner": "index",
	}
	for name, kind := range names {
		var got string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = ? AND name = ?", kind, name,
		).Scan(&got)
		require.NoErrorf(t, err, "%s %q should exist", kind, name)
		assert.Equal(t, name, got)
	}
}
