package store

import "embed"

// migrationFiles holds the SQL migrations compiled into the binary. They are applied
// in lexical filename order by the migration runner (see sqlite.go), which tracks the
// applied version via PRAGMA user_version.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS
