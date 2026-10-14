package sqlite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func latestVersion(t *testing.T) int64 {
	t.Helper()
	if err := setupGoose(); err != nil {
		t.Fatal(err)
	}
	ms, err := goose.CollectMigrations("migrations", 0, goose.MaxVersion)
	if err != nil {
		t.Fatal(err)
	}
	last, err := ms.Last()
	if err != nil {
		t.Fatal(err)
	}
	return last.Version
}

func TestStartup_OpenAndMigrate(t *testing.T) {
	t.Run("should create no backup when the database file does not exist yet", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		db, err := OpenAndMigrate(path)
		if err != nil {
			t.Fatalf("OpenAndMigrate: %v", err)
		}
		defer func() { _ = db.Close() }()

		v, err := goose.GetDBVersion(db)
		if err != nil {
			t.Fatal(err)
		}
		if want := latestVersion(t); v != want {
			t.Errorf("version = %d, want %d", v, want)
		}
		if got := listDir(t, BackupDir(path)); len(got) != 0 {
			t.Errorf("want no backups, got %v", got)
		}
	})

	t.Run("should create no backup when no migration is pending", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		db := openMigratedTo(t, path, goose.MaxVersion)
		_ = db.Close()

		db2, err := OpenAndMigrate(path)
		if err != nil {
			t.Fatalf("OpenAndMigrate: %v", err)
		}
		defer func() { _ = db2.Close() }()

		if got := listDir(t, BackupDir(path)); len(got) != 0 {
			t.Errorf("want no backups, got %v", got)
		}
	})

	t.Run("should create one backup and migrate when migrations are pending", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		db := openMigratedTo(t, path, 3)
		_ = db.Close()

		db2, err := OpenAndMigrate(path)
		if err != nil {
			t.Fatalf("OpenAndMigrate: %v", err)
		}
		defer func() { _ = db2.Close() }()

		got := listDir(t, BackupDir(path))
		if len(got) != 1 {
			t.Fatalf("want 1 backup, got %v", got)
		}
		bk, err := Open(filepath.Join(BackupDir(path), got[0]))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = bk.Close() }()
		if v, err := goose.GetDBVersion(bk); err != nil || v != 3 {
			t.Errorf("backup version = %d (err=%v), want 3", v, err)
		}
		if v, err := goose.GetDBVersion(db2); err != nil || v != latestVersion(t) {
			t.Errorf("db version = %d (err=%v), want latest", v, err)
		}
	})

	t.Run("should return a backup-stage error and leave the schema version unchanged when the backups directory is not writable", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		db := openMigratedTo(t, path, 3)
		_ = db.Close()
		// ディレクトリの権限は root 実行だと効かないため、同名のファイルを置いて作れなくする。
		if err := os.WriteFile(BackupDir(path), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := OpenAndMigrate(path)
		var se *StartupError
		if !errors.As(err, &se) {
			t.Fatalf("want *StartupError, got %v", err)
		}
		if se.Stage != StageBackup {
			t.Errorf("stage = %q, want %q", se.Stage, StageBackup)
		}

		check, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = check.Close() }()
		if v, err := goose.GetDBVersion(check); err != nil || v != 3 {
			t.Errorf("version = %d (err=%v), want 3", v, err)
		}
	})

	t.Run("should return an open-stage error when the path is a directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}

		_, err := OpenAndMigrate(path)
		var se *StartupError
		if !errors.As(err, &se) {
			t.Fatalf("want *StartupError, got %v", err)
		}
		if se.Stage != StageOpen {
			t.Errorf("stage = %q, want %q", se.Stage, StageOpen)
		}
	})
}
