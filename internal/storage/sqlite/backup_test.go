package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
)

func openMigratedTo(t *testing.T, path string, version int64) *sql.DB {
	t.Helper()
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := setupGoose(); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "migrations", version); err != nil {
		t.Fatalf("UpTo(%d): %v", version, err)
	}
	return db
}

func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("ReadDir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

func TestPendingMigrations(t *testing.T) {
	t.Run("should return false when the database is at the latest version", func(t *testing.T) {
		db := openMigratedTo(t, filepath.Join(t.TempDir(), "test.db"), goose.MaxVersion)
		_, pending, err := pendingMigrations(db)
		if err != nil {
			t.Fatal(err)
		}
		if pending {
			t.Error("want no pending migrations")
		}
	})

	t.Run("should return true when migrations are applied only up to an older version", func(t *testing.T) {
		db := openMigratedTo(t, filepath.Join(t.TempDir(), "test.db"), 3)
		current, pending, err := pendingMigrations(db)
		if err != nil {
			t.Fatal(err)
		}
		if !pending {
			t.Error("want pending migrations")
		}
		if current != 3 {
			t.Errorf("current = %d, want 3", current)
		}
	})
}

func TestBackupBeforeMigrate(t *testing.T) {
	t.Run("should write a backup that holds the rows before migration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db := openMigratedTo(t, path, 2)
		if _, err := db.Exec(`INSERT INTO recipes (id, name, created_at, updated_at) VALUES ('r1', 'カレー', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
			t.Fatalf("seed: %v", err)
		}

		now := time.Date(2026, 10, 14, 9, 30, 15, 0, time.Local)
		dest, err := backupBeforeMigrate(db, path, 2, now)
		if err != nil {
			t.Fatalf("backupBeforeMigrate: %v", err)
		}
		if want := filepath.Join(filepath.Dir(path), "backups", "meal-cycle-v2-20261014-093015.db"); dest != want {
			t.Errorf("dest = %q, want %q", dest, want)
		}

		bk, err := Open(dest)
		if err != nil {
			t.Fatalf("open backup: %v", err)
		}
		defer func() { _ = bk.Close() }()
		var name string
		if err := bk.QueryRow(`SELECT name FROM recipes WHERE id = 'r1'`).Scan(&name); err != nil {
			t.Fatalf("query backup: %v", err)
		}
		if name != "カレー" {
			t.Errorf("name = %q, want カレー", name)
		}
		for _, n := range listDir(t, filepath.Dir(dest)) {
			if filepath.Ext(n) == partialExt {
				t.Errorf("partial file left behind: %s", n)
			}
		}
	})

	t.Run("should include writes still in the WAL", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db := openMigratedTo(t, path, 2)
		if _, err := db.Exec(`INSERT INTO recipes (id, name, created_at, updated_at) VALUES ('r2', '味噌汁', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if fi, err := os.Stat(path + "-wal"); err != nil || fi.Size() == 0 {
			t.Fatalf("expected a non-empty WAL file (err=%v)", err)
		}

		dest, err := backupBeforeMigrate(db, path, 2, time.Now())
		if err != nil {
			t.Fatalf("backupBeforeMigrate: %v", err)
		}
		bk, err := Open(dest)
		if err != nil {
			t.Fatalf("open backup: %v", err)
		}
		defer func() { _ = bk.Close() }()
		var n int
		if err := bk.QueryRow(`SELECT count(*) FROM recipes WHERE id = 'r2'`).Scan(&n); err != nil {
			t.Fatalf("query backup: %v", err)
		}
		if n != 1 {
			t.Errorf("want the WAL row in the backup, got %d rows", n)
		}
	})
}

func TestPruneBackups(t *testing.T) {
	t.Run("should remove the oldest backup when a sixth one is added", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db := openMigratedTo(t, path, 2)
		base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
		var created []string
		// 版の数字の桁が変わっても日時で新旧を決めることを確かめるため、版を 9 → 10 にまたがらせる。
		for i := 0; i < 6; i++ {
			dest, err := backupBeforeMigrate(db, path, int64(5+i), base.Add(time.Duration(i)*time.Hour))
			if err != nil {
				t.Fatalf("backup %d: %v", i, err)
			}
			created = append(created, filepath.Base(dest))
		}

		if err := pruneBackups(BackupDir(path), backupKeep); err != nil {
			t.Fatalf("pruneBackups: %v", err)
		}

		got := listDir(t, BackupDir(path))
		want := append([]string(nil), created[1:]...)
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got %v, want %v", got, want)
				break
			}
		}
	})

	t.Run("should remove leftover partial files", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "backups")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		partial := filepath.Join(dir, "meal-cycle-v3-20260101-000000.db.partial")
		if err := os.WriteFile(partial, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := pruneBackups(dir, backupKeep); err != nil {
			t.Fatalf("pruneBackups: %v", err)
		}
		if _, err := os.Stat(partial); !os.IsNotExist(err) {
			t.Errorf("partial file should be removed, stat err = %v", err)
		}
	})
}
