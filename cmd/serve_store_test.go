package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/handlers"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

func TestResolveDBPathDefault(t *testing.T) {
	t.Setenv(dbPathEnv, "")
	assert.Equal(t, defaultDBPath, resolveDBPath())
}

func TestResolveDBPathFromEnv(t *testing.T) {
	t.Setenv(dbPathEnv, "custom/lookups.db")
	assert.Equal(t, "custom/lookups.db", resolveDBPath())
}

func TestSetupStoreReturnsErrorOnOpenFailure(t *testing.T) {
	// A blank path is rejected by store.New, so setupStore must surface a wrapped error.
	err := setupStore("   ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open lookup database")
}

func TestSetupStoreInitializesHandlersStore(t *testing.T) {
	// Parent dir does not exist yet — setupStore must create it (os.MkdirAll).
	path := filepath.Join(t.TempDir(), "data", "wire.db")

	require.NoError(t, setupStore(path))

	assert.NotNil(t, handlers.Store, "handlers.Store should be injected after setupStore")
	if s, ok := handlers.Store.(*store.SQLiteStore); ok {
		t.Cleanup(func() { _ = s.Close() })
	}

	info, err := os.Stat(path)
	require.NoError(t, err, "database file should be created")
	assert.False(t, info.IsDir())
}
