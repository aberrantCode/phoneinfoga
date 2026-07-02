package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/handlers"
	"github.com/sundowndev/phoneinfoga/v2/web/v2/api/store"
)

// dbPathEnv is the environment variable selecting the lookup-persistence database file.
const dbPathEnv = "PHONEINFOGA_DB_PATH"

// defaultDBPath is used when dbPathEnv is unset (spec §9).
const defaultDBPath = "./phoneinfoga.db"

// resolveDBPath returns the configured database path, falling back to the default.
func resolveDBPath() string {
	if p := os.Getenv(dbPathEnv); p != "" {
		return p
	}
	return defaultDBPath
}

// setupStore opens the SQLite store at path, runs migrations, and injects it into the
// handlers package. Persistence is always on for `serve`, so a failure here is fatal to
// the caller (PreRun exits) — callers decide how to surface the returned error.
func setupStore(path string) error {
	s, err := store.New(path)
	if err != nil {
		return fmt.Errorf("open lookup database: %w", err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		return fmt.Errorf("migrate lookup database: %w", err)
	}
	handlers.InitStore(s)
	logrus.WithField("path", path).Debug("Lookup persistence store initialized")
	return nil
}
