package sqlite

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestOpenExisting(t *testing.T) {
	t.Run("should return ErrDatabaseMissing and not create the file when the file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")

		_, err := OpenExisting(path)
		if !errors.Is(err, ErrDatabaseMissing) {
			t.Fatalf("err = %v, want ErrDatabaseMissing", err)
		}
		if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("file should not be created, stat err = %v", statErr)
		}
	})

	t.Run("should return ErrSchemaOutdated and leave the version and backups unchanged when the schema is old", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		db := openMigratedTo(t, path, 3)
		_ = db.Close()

		_, err := OpenExisting(path)
		if !errors.Is(err, ErrSchemaOutdated) {
			t.Fatalf("err = %v, want ErrSchemaOutdated", err)
		}

		check, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = check.Close() }()
		if v, err := goose.GetDBVersion(check); err != nil || v != 3 {
			t.Errorf("version = %d (err=%v), want 3", v, err)
		}
		if got := listDir(t, BackupDir(path)); len(got) != 0 {
			t.Errorf("want no backups, got %v", got)
		}
	})

	t.Run("should return ErrSchemaTooNew when the version is newer than the embedded migrations", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		db := openMigratedTo(t, path, goose.MaxVersion)
		if _, err := db.Exec(
			`INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1)`, latestVersion(t)+1,
		); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()

		_, err := OpenExisting(path)
		if !errors.Is(err, ErrSchemaTooNew) {
			t.Fatalf("err = %v, want ErrSchemaTooNew", err)
		}
	})

	t.Run("should open the database when the schema is at the latest version", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "meal-cycle.db")
		db := openMigratedTo(t, path, goose.MaxVersion)
		_ = db.Close()

		got, err := OpenExisting(path)
		if err != nil {
			t.Fatalf("OpenExisting: %v", err)
		}
		defer func() { _ = got.Close() }()
		if err := got.Ping(); err != nil {
			t.Errorf("Ping: %v", err)
		}
	})
}
