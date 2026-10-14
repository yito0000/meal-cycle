package service

import (
	"path/filepath"
	"testing"

	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

func TestHealthService_Ping(t *testing.T) {
	t.Run("should report dbConnected and no startupError when startup succeeded", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db, err := sqlite.OpenAndMigrate(path)
		if err != nil {
			t.Fatalf("OpenAndMigrate: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })

		got := NewHealthService(db, StartupState{DBPath: path, BackupDir: sqlite.BackupDir(path)}).Ping()
		if !got.DBConnected {
			t.Errorf("want dbConnected, got %+v", got)
		}
		if got.StartupError != "" || got.StartupDetail != "" {
			t.Errorf("want no startup error, got %+v", got)
		}
	})

	t.Run("should report the stage, detail, dbPath and backupDir when startup failed", func(t *testing.T) {
		state := StartupState{
			Stage:     string(sqlite.StageMigrate),
			Detail:    "apply migrations: boom",
			DBPath:    "/data/meal-cycle.db",
			BackupDir: "/data/backups",
		}
		got := NewHealthService(nil, state).Ping()
		want := Health{
			StartupError:  "migrate",
			StartupDetail: "apply migrations: boom",
			DBPath:        "/data/meal-cycle.db",
			BackupDir:     "/data/backups",
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("should not panic when db is nil", func(t *testing.T) {
		got := NewHealthService(nil, StartupState{}).Ping()
		if got.DBConnected {
			t.Errorf("want dbConnected false, got %+v", got)
		}
	})
}
