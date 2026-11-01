package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/pressly/goose/v3"
)

var (
	ErrDatabaseMissing = errors.New("database: file not found; start meal-cycle once to create it")
	ErrSchemaOutdated  = errors.New("database: schema is outdated; start meal-cycle once to update it")
	ErrSchemaTooNew    = errors.New("database: schema is newer than this server; update meal-cycle-mcp")
)

// goose の設定はパッケージ変数なので、MCP のツール呼び出しが並行しても版の確認が混ざらないようにする。
var gooseMu sync.Mutex

// OpenExisting は版が埋め込んだ migration と一致する既存の DB だけを開く (add-mcp-server design.md D3)。
// スキーマの更新と控えは GUI の起動 (OpenAndMigrate) だけが行う、という不変条件を保つため、ここでは何も書き換えない。
func OpenExisting(path string) (*sql.DB, error) {
	// Open は無いファイルを作ってしまうので、先に有無を確かめる。
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrDatabaseMissing
		}
		return nil, fmt.Errorf("stat db: %w", err)
	}

	db, err := Open(path)
	if err != nil {
		return nil, err
	}
	if err := checkSchemaVersion(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func checkSchemaVersion(db *sql.DB) error {
	// goose.GetDBVersion は版の表が無いと作ってしまうため、無ければ書き込まずに古いと判定する。
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'goose_db_version'`,
	).Scan(&n); err != nil {
		return fmt.Errorf("check version table: %w", err)
	}
	if n == 0 {
		return ErrSchemaOutdated
	}

	gooseMu.Lock()
	defer gooseMu.Unlock()
	latest, err := latestMigrationVersion()
	if err != nil {
		return err
	}
	current, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("get db version: %w", err)
	}
	switch {
	case current < latest:
		return ErrSchemaOutdated
	case current > latest:
		return ErrSchemaTooNew
	}
	return nil
}

func latestMigrationVersion() (int64, error) {
	if err := setupGoose(); err != nil {
		return 0, err
	}
	ms, err := goose.CollectMigrations("migrations", 0, goose.MaxVersion)
	if err != nil {
		return 0, fmt.Errorf("collect migrations: %w", err)
	}
	last, err := ms.Last()
	if err != nil {
		return 0, fmt.Errorf("last migration: %w", err)
	}
	return last.Version, nil
}
