// Package sqlite は meal-cycle の SQLite ラッパー。
//
// ドライバは modernc.org/sqlite (Cgo なし) を使う。クロスコンパイルが
// 素直になるため (docs/design.md §5)。
package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/adrg/xdg"
	_ "modernc.org/sqlite"
)

const (
	AppDirName = "meal-cycle"
	DBFileName = "meal-cycle.db"
)

// DefaultPath は xdg 経由で解決した DB ファイルパスを返す。
//
//	Windows: %APPDATA%\meal-cycle\meal-cycle.db
//	macOS:   ~/Library/Application Support/meal-cycle/meal-cycle.db
//	Linux:   ~/.local/share/meal-cycle/meal-cycle.db
//
// 親ディレクトリは必要に応じて作られる。
func DefaultPath() (string, error) {
	p, err := xdg.DataFile(filepath.Join(AppDirName, DBFileName))
	if err != nil {
		return "", fmt.Errorf("resolve db path: %w", err)
	}
	return p, nil
}

// Open は path の SQLite DB を開く (初回は作成)。呼び出し側は返り値を Close すること。
// 単一ユーザーのデスクトップアプリ向けに WAL / foreign_keys / busy_timeout(5s) を有効化する。
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
