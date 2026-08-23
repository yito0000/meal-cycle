// Package sqlite wraps the SQLite database used by meal-cycle.
//
// Uses modernc.org/sqlite (Cgo-free) so cross-compilation stays simple
// across Windows / macOS / Linux. See docs/design.md #5.
package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/adrg/xdg"
	_ "modernc.org/sqlite"
)

const (
	// AppDirName is the sub-directory under the XDG data dir where meal-cycle
	// stores its SQLite database.
	AppDirName = "meal-cycle"
	// DBFileName is the SQLite database file name.
	DBFileName = "meal-cycle.db"
)

// DefaultPath returns the DB file path resolved via xdg:
//
//	Windows: %APPDATA%\meal-cycle\meal-cycle.db
//	macOS:   ~/Library/Application Support/meal-cycle/meal-cycle.db
//	Linux:   ~/.local/share/meal-cycle/meal-cycle.db
//
// Parent directories are created as needed.
func DefaultPath() (string, error) {
	p, err := xdg.DataFile(filepath.Join(AppDirName, DBFileName))
	if err != nil {
		return "", fmt.Errorf("resolve db path: %w", err)
	}
	return p, nil
}

// Open opens (creating on first run) the SQLite database at path.
// Callers must Close the returned handle.
//
// Enables WAL, foreign keys, and a 5 s busy timeout — sensible defaults for
// a single-user desktop app.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}
