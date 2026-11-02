package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/yito0000/meal-cycle/internal/domain/settings"
)

// フロントは文言の部分一致でメッセージを引くので、変えるときは settings の database のエラーの表も直す。
var (
	ErrRestoreNotDatabase = errors.New("database: restore source is not a meal-cycle database")
	ErrRestoreTooNew      = errors.New("database: restore source is newer than this app")
)

const (
	pendingRestoreName   = "restore-pending.db"
	pendingRestoreLabel  = "restore-pending.name"
	beforeRestoreSuffix  = "-before-restore"
	quickCheckOK         = "ok"
	gooseVersionTableSQL = `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'goose_db_version'`
)

func pendingRestorePath(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), pendingRestoreName)
}

func pendingLabelPath(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), pendingRestoreLabel)
}

// parseBackupName は meal-cycle-v<版>-<日時>[-before-restore].db を読む。
func parseBackupName(name string) (version int64, stamp string, beforeRestore bool, ok bool) {
	if !strings.HasPrefix(name, backupPrefix) || !strings.HasSuffix(name, backupExt) {
		return 0, "", false, false
	}
	base := strings.TrimSuffix(strings.TrimPrefix(name, backupPrefix), backupExt)
	if strings.HasSuffix(base, beforeRestoreSuffix) {
		beforeRestore = true
		base = strings.TrimSuffix(base, beforeRestoreSuffix)
	}
	dash := strings.IndexByte(base, '-')
	if dash <= 0 {
		return 0, "", false, false
	}
	v, err := strconv.ParseInt(base[:dash], 10, 64)
	if err != nil {
		return 0, "", false, false
	}
	stamp = base[dash+1:]
	if _, err := time.ParseInLocation(backupStampLayout, stamp, time.Local); err != nil {
		return 0, "", false, false
	}
	return v, stamp, beforeRestore, true
}

// vacuumInto は src の接続の中身を dest に 1 ファイルで書き出す。WAL にだけある変更も含めるため、ファイルコピーではなく VACUUM INTO を使い、
// 途中で落ちたときに書きかけを正規の名前で残さないよう .partial に書いてから改名する。
func vacuumInto(db *sql.DB, dest string) error {
	partial := dest + partialExt
	// VACUUM INTO は書き出し先が既にあると失敗する。
	if err := os.Remove(partial); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale partial: %w", err)
	}
	if _, err := db.Exec(`VACUUM INTO ?`, partial); err != nil {
		_ = os.Remove(partial)
		return fmt.Errorf("vacuum into: %w", err)
	}
	if err := os.Rename(partial, dest); err != nil {
		_ = os.Remove(partial)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// ExportTo は起動中の DB 全体を dest に書き出す (add-settings-database-file design.md D4)。
func ExportTo(db *sql.DB, dest string) error {
	return vacuumInto(db, dest)
}

// ListBackups は backups フォルダの控えを新しい順に返す。フォルダが無ければ空を返す。
func ListBackups(dbPath string) ([]settings.BackupFile, error) {
	entries, err := os.ReadDir(BackupDir(dbPath))
	if err != nil {
		if os.IsNotExist(err) {
			return []settings.BackupFile{}, nil
		}
		return nil, fmt.Errorf("read backup dir: %w", err)
	}
	type item struct {
		file  settings.BackupFile
		stamp string
	}
	items := make([]item, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		v, stamp, before, ok := parseBackupName(e.Name())
		if !ok {
			continue
		}
		at, _ := time.ParseInLocation(backupStampLayout, stamp, time.Local)
		items = append(items, item{
			file:  settings.BackupFile{Name: e.Name(), CreatedAt: at.Format(time.RFC3339), Version: v, BeforeRestore: before},
			stamp: stamp,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].stamp != items[j].stamp {
			return items[i].stamp > items[j].stamp
		}
		return items[i].file.Name > items[j].file.Name
	})
	out := make([]settings.BackupFile, 0, len(items))
	for _, it := range items {
		out = append(out, it.file)
	}
	return out, nil
}

func openReadOnly(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(5000)", path))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// ValidateRestoreSource は path が壊れていない meal-cycle の DB で、このアプリより新しい版でないことを確かめる (design.md D2)。
// 古い版は通す。次の起動で、いつもの更新の流れが控えを作ってから最新にする。
func ValidateRestoreSource(path string) error {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return ErrRestoreNotDatabase
	}
	db, err := openReadOnly(path)
	if err != nil {
		return ErrRestoreNotDatabase
	}
	defer db.Close()

	var check string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&check); err != nil || check != quickCheckOK {
		return ErrRestoreNotDatabase
	}
	var n int
	if err := db.QueryRow(gooseVersionTableSQL).Scan(&n); err != nil || n == 0 {
		return ErrRestoreNotDatabase
	}

	gooseMu.Lock()
	defer gooseMu.Unlock()
	latest, err := latestMigrationVersion()
	if err != nil {
		return err
	}
	current, err := goose.GetDBVersion(db)
	if err != nil {
		return ErrRestoreNotDatabase
	}
	if current > latest {
		return ErrRestoreTooNew
	}
	return nil
}

// ReserveRestore は src の内容を DB と同じフォルダの restore-pending.db に写して、次の起動で当てる予約にする (design.md D1)。
// 場所を覚えるだけにしないのは、起動までに元のファイルが動いたり消えたりしても予約が壊れないようにするため。
func ReserveRestore(dbPath, src, label string) error {
	if err := ValidateRestoreSource(src); err != nil {
		return err
	}
	db, err := openReadOnly(src)
	if err != nil {
		return ErrRestoreNotDatabase
	}
	defer db.Close()
	if err := vacuumInto(db, pendingRestorePath(dbPath)); err != nil {
		return fmt.Errorf("copy restore source: %w", err)
	}
	if err := os.WriteFile(pendingLabelPath(dbPath), []byte(label), 0o644); err != nil {
		return fmt.Errorf("write restore label: %w", err)
	}
	return nil
}

// PendingRestore は予約があればその名前を返す。
func PendingRestore(dbPath string) (settings.PendingRestore, error) {
	if _, err := os.Stat(pendingRestorePath(dbPath)); err != nil {
		if os.IsNotExist(err) {
			return settings.PendingRestore{}, nil
		}
		return settings.PendingRestore{}, fmt.Errorf("stat pending restore: %w", err)
	}
	label, err := os.ReadFile(pendingLabelPath(dbPath))
	if err != nil && !os.IsNotExist(err) {
		return settings.PendingRestore{}, fmt.Errorf("read restore label: %w", err)
	}
	return settings.PendingRestore{Pending: true, Name: string(label)}, nil
}

func CancelRestore(dbPath string) error {
	for _, p := range []string{pendingRestorePath(dbPath), pendingLabelPath(dbPath)} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove pending restore: %w", err)
		}
	}
	return nil
}

// applyPendingRestore は予約があれば、今の DB を復元前の控えに書き出してから予約の内容で置き換える (design.md D3)。
// 失敗したら予約は消さずに残し、次の起動でもう一度試す。控えを書く前に失敗したなら今の DB は変わっていない。
func applyPendingRestore(dbPath string, now time.Time) error {
	pending := pendingRestorePath(dbPath)
	if _, err := os.Stat(pending); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat pending restore: %w", err)
	}

	if _, err := os.Stat(dbPath); err == nil {
		if err := backupBeforeRestore(dbPath, now); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat db: %w", err)
	}

	for _, side := range []string{dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(side); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", filepath.Base(side), err)
		}
	}
	// 同じフォルダなので改名で済み、途中で止まっても予約か今の DB のどちらかは必ず残る。
	if err := os.Rename(pending, dbPath); err != nil {
		return fmt.Errorf("replace db: %w", err)
	}
	if err := os.Remove(pendingLabelPath(dbPath)); err != nil && !os.IsNotExist(err) {
		log.Printf("meal-cycle: remove restore label: %v", err)
	}
	if err := pruneBackups(BackupDir(dbPath), backupKeep); err != nil {
		log.Printf("meal-cycle: prune backups: %v", err)
	}
	return nil
}

func backupBeforeRestore(dbPath string, now time.Time) error {
	db, err := Open(dbPath)
	if err != nil {
		return fmt.Errorf("open current db: %w", err)
	}
	defer db.Close()
	var version int64
	var n int
	if err := db.QueryRow(gooseVersionTableSQL).Scan(&n); err != nil {
		return fmt.Errorf("check version table: %w", err)
	}
	if n > 0 {
		gooseMu.Lock()
		if err := setupGoose(); err == nil {
			version, _ = goose.GetDBVersion(db)
		}
		gooseMu.Unlock()
	}
	if _, err := writeBackup(db, dbPath, version, now, beforeRestoreSuffix); err != nil {
		return fmt.Errorf("backup before restore: %w", err)
	}
	return nil
}
