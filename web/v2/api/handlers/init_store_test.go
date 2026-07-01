package handlers_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/handlers"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

func TestInitStore(t *testing.T) {
	s, err := store.New(filepath.Join(t.TempDir(), "handlers.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	handlers.InitStore(s)
	assert.NotNil(t, handlers.Store)
	assert.Equal(t, s, handlers.Store, "InitStore should expose the injected store")
}
