package sqlite

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

// migratedDB は最新まで更新した DB を path に作り、印の行を 1 つ入れて開いたまま返す。
func migratedDB(t *testing.T, path, mark string) *sql.DB {
	t.Helper()
	db := openMigratedTo(t, path, goose.MaxVersion)
	if _, err := db.Exec(`INSERT INTO settings (key, value, updated_at) VALUES ('mark', ?, '2026-11-02T00:00:00Z')`, `"`+mark+`"`); err != nil {
		t.Fatalf("insert mark: %v", err)
	}
	return db
}

func readMark(t *testing.T, path string) string {
	t.Helper()
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var v string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = 'mark'`).Scan(&v); err != nil {
		t.Fatalf("read mark: %v", err)
	}
	return strings.Trim(v, `"`)
}

func TestExportTo(t *testing.T) {
	t.Run("should include changes only in the WAL when exporting", func(t *testing.T) {
		dir := t.TempDir()
		db := migratedDB(t, filepath.Join(dir, "meal-cycle.db"), "肉じゃが")
		dest := filepath.Join(dir, "export.db")
		if err := ExportTo(db, dest); err != nil {
			t.Fatal(err)
		}
		if got := readMark(t, dest); got != "肉じゃが" {
			t.Errorf("mark = %q", got)
		}
		if _, err := os.Stat(dest + partialExt); !os.IsNotExist(err) {
			t.Errorf("want no partial file, got %v", err)
		}
	})

	t.Run("should return an error without leaving a partial file when the destination is not writable", func(t *testing.T) {
		dir := t.TempDir()
		db := migratedDB(t, filepath.Join(dir, "meal-cycle.db"), "x")
		dest := filepath.Join(dir, "missing", "export.db")
		if err := ExportTo(db, dest); err == nil {
			t.Fatal("want error")
		}
		if _, err := os.Stat(dest + partialExt); !os.IsNotExist(err) {
			t.Errorf("want no partial file, got %v", err)
		}
	})
}

func TestListBackups(t *testing.T) {
	t.Run("should list migration and restore backups newest first", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "meal-cycle.db")
		dir := BackupDir(dbPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for _, n := range []string{
			"meal-cycle-v11-20261101-090000.db",
			"meal-cycle-v12-20261102-090000-before-restore.db",
			"meal-cycle-v3-20261030-090000.db",
			"unrelated.txt",
		} {
			if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		got, err := ListBackups(dbPath)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 || got[0].Name != "meal-cycle-v12-20261102-090000-before-restore.db" || !got[0].BeforeRestore ||
			got[0].Version != 12 || got[1].Version != 11 || got[1].BeforeRestore || got[2].Version != 3 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should return empty when the backups folder does not exist", func(t *testing.T) {
		got, err := ListBackups(filepath.Join(t.TempDir(), "meal-cycle.db"))
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("got %#v (%v)", got, err)
		}
	})
}

func TestValidateRestoreSource(t *testing.T) {
	t.Run("should reject a file that is not SQLite", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.db")
		if err := os.WriteFile(p, []byte("hello, this is not a database at all"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ValidateRestoreSource(p); !errors.Is(err, ErrRestoreNotDatabase) {
			t.Errorf("want ErrRestoreNotDatabase, got %v", err)
		}
	})

	t.Run("should reject SQLite without the goose version table", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "other.db")
		db, err := Open(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`CREATE TABLE t (x INTEGER)`); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		if err := ValidateRestoreSource(p); !errors.Is(err, ErrRestoreNotDatabase) {
			t.Errorf("want ErrRestoreNotDatabase, got %v", err)
		}
	})

	t.Run("should reject a version newer than the embedded maximum", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "new.db")
		db := openMigratedTo(t, p, goose.MaxVersion)
		if _, err := db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1)`, latestVersion(t)+1); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		if err := ValidateRestoreSource(p); !errors.Is(err, ErrRestoreTooNew) {
			t.Errorf("want ErrRestoreTooNew, got %v", err)
		}
	})

	t.Run("should accept an older version", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "old.db")
		db := openMigratedTo(t, p, 3)
		_ = db.Close()
		if err := ValidateRestoreSource(p); err != nil {
			t.Errorf("want nil, got %v", err)
		}
	})
}

func TestReserveRestore(t *testing.T) {
	t.Run("should keep the reservation after the source file is removed", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "meal-cycle.db")
		src := filepath.Join(dir, "src", "export.db")
		if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
			t.Fatal(err)
		}
		db := migratedDB(t, src, "肉じゃが")
		_ = db.Close()
		if err := ReserveRestore(dbPath, src, "export.db"); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(filepath.Dir(src)); err != nil {
			t.Fatal(err)
		}
		got, err := PendingRestore(dbPath)
		if err != nil || !got.Pending || got.Name != "export.db" {
			t.Errorf("got %#v (%v)", got, err)
		}
		if mark := readMark(t, pendingRestorePath(dbPath)); mark != "肉じゃが" {
			t.Errorf("mark = %q", mark)
		}
	})

	t.Run("should replace the reservation when reserving twice", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "meal-cycle.db")
		a, b := filepath.Join(dir, "a.db"), filepath.Join(dir, "b.db")
		_ = migratedDB(t, a, "A").Close()
		_ = migratedDB(t, b, "B").Close()
		if err := ReserveRestore(dbPath, a, "a.db"); err != nil {
			t.Fatal(err)
		}
		if err := ReserveRestore(dbPath, b, "b.db"); err != nil {
			t.Fatal(err)
		}
		got, _ := PendingRestore(dbPath)
		if got.Name != "b.db" || readMark(t, pendingRestorePath(dbPath)) != "B" {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should clear the pending restore when cancelled", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "meal-cycle.db")
		src := filepath.Join(dir, "a.db")
		_ = migratedDB(t, src, "A").Close()
		if err := ReserveRestore(dbPath, src, "a.db"); err != nil {
			t.Fatal(err)
		}
		if err := CancelRestore(dbPath); err != nil {
			t.Fatal(err)
		}
		if got, err := PendingRestore(dbPath); err != nil || got.Pending {
			t.Errorf("got %#v (%v)", got, err)
		}
	})
}

func TestStartup_OpenAndMigrate_Restore(t *testing.T) {
	countBackups := func(t *testing.T, dbPath string) (migrate, restore int) {
		t.Helper()
		list, err := ListBackups(dbPath)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range list {
			if b.BeforeRestore {
				restore++
			} else {
				migrate++
			}
		}
		return migrate, restore
	}

	t.Run("should back up the current database before replacing it and clear the reservation", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "meal-cycle.db")
		_ = migratedDB(t, dbPath, "カレー").Close()
		src := filepath.Join(dir, "export.db")
		_ = migratedDB(t, src, "肉じゃが").Close()
		if err := ReserveRestore(dbPath, src, "export.db"); err != nil {
			t.Fatal(err)
		}

		db, err := OpenAndMigrate(dbPath)
		if err != nil {
			t.Fatalf("OpenAndMigrate: %v", err)
		}
		_ = db.Close()

		if got := readMark(t, dbPath); got != "肉じゃが" {
			t.Errorf("mark = %q, want restored", got)
		}
		list, _ := ListBackups(dbPath)
		if len(list) != 1 || !list[0].BeforeRestore || readMark(t, filepath.Join(BackupDir(dbPath), list[0].Name)) != "カレー" {
			t.Errorf("backups: %#v", list)
		}
		if p, _ := PendingRestore(dbPath); p.Pending {
			t.Error("want reservation cleared")
		}
	})

	t.Run("should create both restore and migration backups when restoring an older version", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "meal-cycle.db")
		_ = migratedDB(t, dbPath, "now").Close()
		src := filepath.Join(dir, "old.db")
		_ = openMigratedTo(t, src, 3).Close()
		if err := ReserveRestore(dbPath, src, "old.db"); err != nil {
			t.Fatal(err)
		}

		db, err := OpenAndMigrate(dbPath)
		if err != nil {
			t.Fatalf("OpenAndMigrate: %v", err)
		}
		defer func() { _ = db.Close() }()
		if v, err := goose.GetDBVersion(db); err != nil || v != latestVersion(t) {
			t.Errorf("version = %d (%v)", v, err)
		}
		if m, r := countBackups(t, dbPath); m != 1 || r != 1 {
			t.Errorf("backups migrate=%d restore=%d, want 1 and 1", m, r)
		}
	})

	t.Run("should return a restore-stage error and keep the database and reservation when backups is not writable", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "meal-cycle.db")
		_ = migratedDB(t, dbPath, "カレー").Close()
		src := filepath.Join(dir, "export.db")
		_ = migratedDB(t, src, "肉じゃが").Close()
		if err := ReserveRestore(dbPath, src, "export.db"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(BackupDir(dbPath), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := OpenAndMigrate(dbPath)
		var se *StartupError
		if !errors.As(err, &se) || se.Stage != StageRestore {
			t.Fatalf("want restore-stage error, got %v", err)
		}
		if got := readMark(t, dbPath); got != "カレー" {
			t.Errorf("mark = %q, want unchanged", got)
		}
		if p, _ := PendingRestore(dbPath); !p.Pending {
			t.Error("want reservation kept")
		}
	})

	t.Run("should behave as before when there is no reservation", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "meal-cycle.db")
		_ = migratedDB(t, dbPath, "x").Close()
		db, err := OpenAndMigrate(dbPath)
		if err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		if m, r := countBackups(t, dbPath); m != 0 || r != 0 {
			t.Errorf("want no backups, got %d / %d", m, r)
		}
	})
}
