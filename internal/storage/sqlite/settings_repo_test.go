package sqlite

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func newSettingsRepoForTest(t *testing.T) *SettingsRepo {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewSettingsRepo(db)
}

func TestSettingsRepo(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)

	t.Run("returns empty map when db is fresh", func(t *testing.T) {
		repo := newSettingsRepoForTest(t)
		got, err := repo.Load(ctx)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want empty, got %v", got)
		}
	})

	t.Run("reads back rows written by ReplaceAll", func(t *testing.T) {
		repo := newSettingsRepoForTest(t)
		rows := map[string]json.RawMessage{
			"language":    json.RawMessage(`"en"`),
			"expiryAlert": json.RawMessage(`{"redDays":2,"yellowDays":7}`),
		}
		if err := repo.ReplaceAll(ctx, rows, now); err != nil {
			t.Fatalf("ReplaceAll: %v", err)
		}
		got, err := repo.Load(ctx)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(got) != 2 || string(got["language"]) != `"en"` ||
			string(got["expiryAlert"]) != `{"redDays":2,"yellowDays":7}` {
			t.Errorf("got %v", got)
		}
	})

	t.Run("drops keys missing from the second ReplaceAll", func(t *testing.T) {
		repo := newSettingsRepoForTest(t)
		if err := repo.ReplaceAll(ctx, map[string]json.RawMessage{
			"language": json.RawMessage(`"en"`), "recipeImportDefaultServings": json.RawMessage(`4`),
		}, now); err != nil {
			t.Fatalf("first ReplaceAll: %v", err)
		}
		if err := repo.ReplaceAll(ctx, map[string]json.RawMessage{
			"recipeImportDefaultServings": json.RawMessage(`3`),
		}, now); err != nil {
			t.Fatalf("second ReplaceAll: %v", err)
		}
		got, err := repo.Load(ctx)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(got) != 1 || string(got["recipeImportDefaultServings"]) != `3` {
			t.Errorf("got %v", got)
		}
	})

	t.Run("returns empty map after DeleteAll", func(t *testing.T) {
		repo := newSettingsRepoForTest(t)
		if err := repo.ReplaceAll(ctx, map[string]json.RawMessage{"language": json.RawMessage(`"en"`)}, now); err != nil {
			t.Fatalf("ReplaceAll: %v", err)
		}
		if err := repo.DeleteAll(ctx); err != nil {
			t.Fatalf("DeleteAll: %v", err)
		}
		got, err := repo.Load(ctx)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want empty, got %v", got)
		}
	})
}
