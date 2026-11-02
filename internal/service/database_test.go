package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

type stubFilePicker struct {
	save string
	open string
}

func (p stubFilePicker) SaveFile(_, defaultName string) (string, error) {
	if p.save == "" {
		return "", nil
	}
	return filepath.Join(p.save, defaultName), nil
}

func (p stubFilePicker) OpenFile(string) (string, error) { return p.open, nil }

func newDatabaseServiceForTest(t *testing.T, picker stubFilePicker) (*DatabaseService, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "meal-cycle.db")
	db, err := sqlite.OpenAndMigrate(dbPath)
	if err != nil {
		t.Fatalf("OpenAndMigrate: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := NewDatabaseService(db, dbPath, picker, func() {})
	svc.now = func() time.Time { return time.Date(2026, 11, 2, 21, 30, 5, 0, time.Local) }
	return svc, dir
}

func TestDatabaseService(t *testing.T) {
	ctx := context.Background()

	t.Run("should write the database to the picked path when export is called", func(t *testing.T) {
		out := t.TempDir()
		svc, _ := newDatabaseServiceForTest(t, stubFilePicker{save: out})
		got, err := svc.Export(ctx, "title")
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(out, "meal-cycle-20261102-213005.db")
		if got.Cancelled || got.Path != want {
			t.Fatalf("got %#v, want path %s", got, want)
		}
		if err := sqlite.ValidateRestoreSource(want); err != nil {
			t.Errorf("exported file is not restorable: %v", err)
		}
	})

	t.Run("should do nothing when the save dialog is closed", func(t *testing.T) {
		svc, dir := newDatabaseServiceForTest(t, stubFilePicker{})
		got, err := svc.Export(ctx, "title")
		if err != nil || !got.Cancelled {
			t.Fatalf("got %#v (%v)", got, err)
		}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "meal-cycle-2") {
				t.Errorf("unexpected file %s", e.Name())
			}
		}
	})

	t.Run("should reject a backup name that is not in the backup list", func(t *testing.T) {
		svc, _ := newDatabaseServiceForTest(t, stubFilePicker{})
		if _, err := svc.ReserveFromBackup(ctx, "../meal-cycle.db"); !errors.Is(err, ErrBackupNotFound) {
			t.Errorf("want ErrBackupNotFound, got %v", err)
		}
	})

	t.Run("should reserve the picked file when it is a valid database", func(t *testing.T) {
		out := t.TempDir()
		svc, _ := newDatabaseServiceForTest(t, stubFilePicker{save: out})
		exported, err := svc.Export(ctx, "title")
		if err != nil {
			t.Fatal(err)
		}
		svc.picker = stubFilePicker{open: exported.Path}
		got, err := svc.ReserveFromFile(ctx, "title")
		if err != nil {
			t.Fatal(err)
		}
		if got.Cancelled || !got.Pending.Pending || got.Pending.Name != "meal-cycle-20261102-213005.db" {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should return the pending restore after reserving", func(t *testing.T) {
		out := t.TempDir()
		svc, _ := newDatabaseServiceForTest(t, stubFilePicker{save: out})
		exported, _ := svc.Export(ctx, "title")
		svc.picker = stubFilePicker{open: exported.Path}
		if _, err := svc.ReserveFromFile(ctx, "title"); err != nil {
			t.Fatal(err)
		}
		got, err := svc.Pending(ctx)
		if err != nil || !got.Pending {
			t.Fatalf("got %#v (%v)", got, err)
		}
		if err := svc.CancelPending(ctx); err != nil {
			t.Fatal(err)
		}
		if got, _ := svc.Pending(ctx); got.Pending {
			t.Error("want no pending restore after cancel")
		}
	})

	t.Run("should reject a file that is not a database when reserving", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "note.db")
		if err := os.WriteFile(p, []byte("not a database file, just text"), 0o644); err != nil {
			t.Fatal(err)
		}
		svc, _ := newDatabaseServiceForTest(t, stubFilePicker{open: p})
		if _, err := svc.ReserveFromFile(ctx, "title"); !errors.Is(err, sqlite.ErrRestoreNotDatabase) {
			t.Errorf("want ErrRestoreNotDatabase, got %v", err)
		}
	})
}
