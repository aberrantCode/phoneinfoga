package store

// Blank-import the pure-Go SQLite driver so it registers itself under the
// database/sql driver name "sqlite". This keeps the build CGO-free: modernc.org/sqlite
// is a transpiled port of SQLite that needs no C toolchain.
import _ "modernc.org/sqlite"
