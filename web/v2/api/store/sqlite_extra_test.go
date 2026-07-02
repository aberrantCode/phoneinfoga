package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRejectsEmptyPath(t *testing.T) {
	_, err := New("   ")
	assert.Error(t, err)
}

func TestCreateLookupStoresInvalidNumber(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	in := sampleLookup(numberA, "local")
	in.Valid = false
	created, err := s.CreateLookup(ctx, in)
	require.NoError(t, err)

	got, err := s.GetLookup(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.Valid, "valid=0 should read back as false")
}

func TestListLookupsByNumberDefaultsLimit(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.CreateLookup(ctx, sampleLookup(numberA, "local"))
	require.NoError(t, err)

	// A non-positive limit falls back to the default rather than returning nothing.
	list, err := s.ListLookupsByNumber(ctx, numberA, 0)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestMigrationVersion(t *testing.T) {
	cases := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{"0001_init.sql", 1, false},
		{"0002_add_column.sql", 2, false},
		{"12.sql", 12, false},
		{"init.sql", 0, true},
		{"_x.sql", 0, true},
	}
	for _, tc := range cases {
		got, err := migrationVersion(tc.name)
		if tc.wantErr {
			assert.Errorf(t, err, "expected error for %q", tc.name)
			continue
		}
		require.NoErrorf(t, err, "unexpected error for %q", tc.name)
		assert.Equal(t, tc.want, got)
	}
}

func TestSaveScannerResultWithExplicitID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	l, err := s.CreateLookup(ctx, sampleLookup(numberA, "local"))
	require.NoError(t, err)

	now := time.Now().UTC()
	require.NoError(t, s.SaveScannerResult(ctx, ScannerResult{
		ID: "explicit-result-id", LookupID: l.ID, Scanner: "local",
		Status: ResultStatusSuccess, StartedAt: now, FinishedAt: now, DurationMs: 1,
	}))

	got, err := s.GetLookup(ctx, l.ID)
	require.NoError(t, err)
	require.Len(t, got.Results, 1)
	assert.Equal(t, "explicit-result-id", got.Results[0].ID)
}

func TestApplyMigrationRollsBackOnError(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Invalid SQL must roll back and leave user_version untouched.
	err := s.applyMigration(ctx, 999, "THIS IS NOT VALID SQL;")
	require.Error(t, err)

	var version int
	require.NoError(t, s.db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, 1, version, "failed migration must not advance user_version")
}

func TestConnectionPragmas(t *testing.T) {
	s := newTestStore(t)

	var journal string
	require.NoError(t, s.db.QueryRow("PRAGMA journal_mode").Scan(&journal))
	assert.Equal(t, "delete", journal)

	var fk int
	require.NoError(t, s.db.QueryRow("PRAGMA foreign_keys").Scan(&fk))
	assert.Equal(t, 1, fk)

	// synchronous FULL == 2: durability for the mounted-volume Docker workflow (AC4).
	var sync int
	require.NoError(t, s.db.QueryRow("PRAGMA synchronous").Scan(&sync))
	assert.Equal(t, 2, sync)
}

func TestMigrateSetsUserVersion(t *testing.T) {
	s := newTestStore(t)
	var version int
	require.NoError(t, s.db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, 1, version, "user_version should track the applied migration")
}
