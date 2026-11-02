package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
)

// 利用者が変える値ではなくディスクを食い潰さないための歯止めなので config には置かない (design.md D5)。
const backupKeep = 5

const (
	backupDirName     = "backups"
	backupPrefix      = "meal-cycle-v"
	backupExt         = ".db"
	partialExt        = ".partial"
	backupStampLayout = "20060102-150405"
)

// BackupDir は dbPath の DB に対する控えの置き場所を返す。
func BackupDir(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), backupDirName)
}

func setupGoose() error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	return nil
}

func pendingMigrations(db *sql.DB) (current int64, pending bool, err error) {
	if err := setupGoose(); err != nil {
		return 0, false, err
	}
	current, err = goose.GetDBVersion(db)
	if err != nil {
		return 0, false, fmt.Errorf("get db version: %w", err)
	}
	ms, err := goose.CollectMigrations("migrations", current, goose.MaxVersion)
	// 範囲内に 1 本も無いとき goose は空ではなくエラーを返す。
	if errors.Is(err, goose.ErrNoMigrationFiles) {
		return current, false, nil
	}
	if err != nil {
		return current, false, fmt.Errorf("collect migrations: %w", err)
	}
	return current, len(ms) > 0, nil
}

// WAL に残った書き込みも含めて 1 ファイルにするため、ファイルコピーではなく VACUUM INTO を使う (design.md D3)。
// 途中で落ちたときに不完全な控えを正規の名前で残さないよう、.partial に書いてから改名する。
func backupBeforeMigrate(db *sql.DB, dbPath string, version int64, now time.Time) (string, error) {
	return writeBackup(db, dbPath, version, now, "")
}

// suffix は控えの種類の印。復元前の控え (-before-restore) を S09 の一覧で更新前の控えと見分けるために付ける。
func writeBackup(db *sql.DB, dbPath string, version int64, now time.Time, suffix string) (string, error) {
	dir := BackupDir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	name := fmt.Sprintf("%s%d-%s%s%s", backupPrefix, version, now.Format(backupStampLayout), suffix, backupExt)
	dest := filepath.Join(dir, name)
	if err := vacuumInto(db, dest); err != nil {
		return "", fmt.Errorf("write backup: %w", err)
	}
	return dest, nil
}

// 版の数字は桁数が揃わないので、ファイル名の辞書順ではなく末尾の日時で新旧を決める。
func pruneBackups(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backup dir: %w", err)
	}

	type backup struct {
		name  string
		stamp string
	}
	var backups []backup
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, backupPrefix) {
			continue
		}
		if strings.HasSuffix(name, partialExt) {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				return fmt.Errorf("remove partial backup: %w", err)
			}
			continue
		}
		// 復元前の控えも同じ数に入れる (add-settings-database-file design.md D3)。
		_, stamp, _, ok := parseBackupName(name)
		if !ok {
			continue
		}
		backups = append(backups, backup{name: name, stamp: stamp})
	}

	sort.Slice(backups, func(i, j int) bool {
		if backups[i].stamp != backups[j].stamp {
			return backups[i].stamp > backups[j].stamp
		}
		return backups[i].name > backups[j].name
	})
	for i := keep; i < len(backups); i++ {
		if err := os.Remove(filepath.Join(dir, backups[i].name)); err != nil {
			return fmt.Errorf("remove old backup: %w", err)
		}
	}
	return nil
}
