package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func newStockRepoForTest(t *testing.T) (*StockRepo, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewStockRepo(db), db
}

// restoreAt は added_at を指定して 1 行作る。RFC3339 は秒精度で Create の連続呼び出しが
// 同着になるため、順序を検証するテストはこちらで前提を作る。
func restoreAt(t *testing.T, repo *StockRepo, id, name, unit, addedAt string) {
	t.Helper()
	if _, err := repo.Restore(context.Background(), fridge.StockItem{
		ID: id, Name: name, Amount: 1, Unit: unit, AddedAt: addedAt,
	}); err != nil {
		t.Fatalf("Restore %s: %v", name, err)
	}
}

func TestStockRepo_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty non-nil slice when db is fresh", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("want empty non-nil slice, got %#v", got)
		}
	})
}

func TestStockRepo_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("fills id and timestamps and can be read back when expiry is set", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		created, err := repo.Create(ctx, fridge.StockItem{
			Name: "牛乳", Amount: 500, Unit: "ml", ExpiresAt: "2026-09-25",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if created.ID == "" || created.AddedAt == "" || created.UpdatedAt == "" {
			t.Errorf("id / timestamps not filled: %#v", created)
		}
		list, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) != 1 || !reflect.DeepEqual(list[0], created) {
			t.Errorf("want [%#v], got %#v", created, list)
		}
	})

	t.Run("stores NULL and reads back empty string when expiry is empty", func(t *testing.T) {
		repo, db := newStockRepoForTest(t)
		created, err := repo.Create(ctx, fridge.StockItem{Name: "米", Amount: 5, Unit: "kg"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		var raw sql.NullString
		if err := db.QueryRow(`SELECT expires_at FROM stock_items WHERE id = ?`, created.ID).Scan(&raw); err != nil {
			t.Fatalf("select: %v", err)
		}
		if raw.Valid {
			t.Errorf("want NULL, got %q", raw.String)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ExpiresAt != "" {
			t.Errorf("want empty expiresAt, got %q", got.ExpiresAt)
		}
	})

	t.Run("creates two rows when the same name is created twice", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		for range 2 {
			if _, err := repo.Create(ctx, fridge.StockItem{Name: "玉ねぎ", Amount: 1, Unit: "個"}); err != nil {
				t.Fatalf("Create: %v", err)
			}
		}
		list, _ := repo.List(ctx)
		if len(list) != 2 {
			t.Errorf("want 2 rows, got %d", len(list))
		}
	})
}

func TestStockRepo_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("updates fields and keeps addedAt", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		restoreAt(t, repo, "id-1", "玉ねぎ", "個", "2026-09-01T00:00:00Z")

		got, err := repo.Update(ctx, "id-1", fridge.StockItem{
			Name: "新玉ねぎ", Amount: 2.5, Unit: "kg", ExpiresAt: "2026-10-01",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if got.Name != "新玉ねぎ" || got.Amount != 2.5 || got.Unit != "kg" || got.ExpiresAt != "2026-10-01" {
			t.Errorf("fields not updated: %#v", got)
		}
		if got.AddedAt != "2026-09-01T00:00:00Z" {
			t.Errorf("addedAt changed: %q", got.AddedAt)
		}
	})

	t.Run("clears expiry when expiresAt is empty", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		created, _ := repo.Create(ctx, fridge.StockItem{Name: "牛乳", Amount: 1, ExpiresAt: "2026-09-25"})
		got, err := repo.Update(ctx, created.ID, fridge.StockItem{Name: "牛乳", Amount: 1})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if got.ExpiresAt != "" {
			t.Errorf("want empty expiresAt, got %q", got.ExpiresAt)
		}
	})

	t.Run("returns ErrStockNotFound when id does not exist", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		_, err := repo.Update(ctx, "missing", fridge.StockItem{Name: "x", Amount: 1})
		if !errors.Is(err, fridge.ErrStockNotFound) {
			t.Errorf("want ErrStockNotFound, got %v", err)
		}
	})
}

func TestStockRepo_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("removes the row from List", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		created, _ := repo.Create(ctx, fridge.StockItem{Name: "卵", Amount: 6, Unit: "個"})
		if err := repo.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		list, _ := repo.List(ctx)
		if len(list) != 0 {
			t.Errorf("want empty, got %#v", list)
		}
	})

	t.Run("returns ErrStockNotFound when id does not exist", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		if err := repo.Delete(ctx, "missing"); !errors.Is(err, fridge.ErrStockNotFound) {
			t.Errorf("want ErrStockNotFound, got %v", err)
		}
	})
}

func TestStockRepo_Restore(t *testing.T) {
	ctx := context.Background()

	t.Run("recreates a deleted row with the same id and addedAt", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		created, _ := repo.Create(ctx, fridge.StockItem{Name: "玉ねぎ", Amount: 3, Unit: "個", ExpiresAt: "2026-09-30"})
		if err := repo.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := repo.Restore(ctx, created); err != nil {
			t.Fatalf("Restore: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != created.Name || got.Amount != created.Amount || got.Unit != created.Unit ||
			got.ExpiresAt != created.ExpiresAt || got.AddedAt != created.AddedAt {
			t.Errorf("want %#v, got %#v", created, got)
		}
	})

	t.Run("returns ErrStockAlreadyExists when id already exists", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		created, _ := repo.Create(ctx, fridge.StockItem{Name: "玉ねぎ", Amount: 3, Unit: "個"})
		if _, err := repo.Restore(ctx, created); !errors.Is(err, fridge.ErrStockAlreadyExists) {
			t.Errorf("want ErrStockAlreadyExists, got %v", err)
		}
	})
}

func TestStockRepo_ListSuggestions(t *testing.T) {
	ctx := context.Background()

	createRecipe := func(t *testing.T, db *sql.DB, name, updatedAt string, ings ...recipe.RecipeIngredient) {
		t.Helper()
		rec, err := NewRecipeRepo(db).Create(ctx, recipe.Recipe{Name: name, Servings: 1, Ingredients: ings})
		if err != nil {
			t.Fatalf("create recipe: %v", err)
		}
		if _, err := db.Exec(`UPDATE recipes SET updated_at = ? WHERE id = ?`, updatedAt, rec.ID); err != nil {
			t.Fatalf("set updated_at: %v", err)
		}
	}

	t.Run("returns stock names and recipe ingredient names without duplicates", func(t *testing.T) {
		repo, db := newStockRepoForTest(t)
		restoreAt(t, repo, "s1", "玉ねぎ", "個", "2026-09-01T00:00:00Z")
		restoreAt(t, repo, "s2", "玉ねぎ", "個", "2026-09-02T00:00:00Z")
		createRecipe(t, db, "肉じゃが", "2026-09-01T00:00:00Z",
			recipe.RecipeIngredient{Name: "玉ねぎ", Amount: 1, Unit: "個"},
			recipe.RecipeIngredient{Name: "しめじ", Amount: 1, Unit: "パック"},
		)
		got, err := repo.ListSuggestions(ctx)
		if err != nil {
			t.Fatalf("ListSuggestions: %v", err)
		}
		names := make([]string, 0, len(got.Names))
		for _, n := range got.Names {
			names = append(names, n.Name)
		}
		if !reflect.DeepEqual(names, []string{"しめじ", "玉ねぎ"}) {
			t.Errorf("want [しめじ 玉ねぎ], got %v", names)
		}
	})

	t.Run("uses the unit of the newest stock row when the same name has several rows", func(t *testing.T) {
		repo, _ := newStockRepoForTest(t)
		restoreAt(t, repo, "s1", "豚肉", "パック", "2026-09-01T00:00:00Z")
		restoreAt(t, repo, "s2", "豚肉", "g", "2026-09-05T00:00:00Z")
		got, _ := repo.ListSuggestions(ctx)
		want := []fridge.NameSuggestion{{Name: "豚肉", Unit: "g"}}
		if !reflect.DeepEqual(got.Names, want) {
			t.Errorf("want %v, got %v", want, got.Names)
		}
	})

	t.Run("uses the unit of the newest recipe when the name is not in stock", func(t *testing.T) {
		repo, db := newStockRepoForTest(t)
		createRecipe(t, db, "old", "2026-09-01T00:00:00Z", recipe.RecipeIngredient{Name: "しめじ", Amount: 100, Unit: "g"})
		createRecipe(t, db, "new", "2026-09-10T00:00:00Z", recipe.RecipeIngredient{Name: "しめじ", Amount: 1, Unit: "パック"})
		got, _ := repo.ListSuggestions(ctx)
		want := []fridge.NameSuggestion{{Name: "しめじ", Unit: "パック"}}
		if !reflect.DeepEqual(got.Names, want) {
			t.Errorf("want %v, got %v", want, got.Names)
		}
	})

	t.Run("prefers the stock unit over the recipe unit when the name is in both", func(t *testing.T) {
		repo, db := newStockRepoForTest(t)
		restoreAt(t, repo, "s1", "玉ねぎ", "個", "2026-09-01T00:00:00Z")
		createRecipe(t, db, "newer recipe", "2026-09-20T00:00:00Z", recipe.RecipeIngredient{Name: "玉ねぎ", Amount: 200, Unit: "g"})
		got, _ := repo.ListSuggestions(ctx)
		want := []fridge.NameSuggestion{{Name: "玉ねぎ", Unit: "個"}}
		if !reflect.DeepEqual(got.Names, want) {
			t.Errorf("want %v, got %v", want, got.Names)
		}
	})

	t.Run("returns distinct non-empty units", func(t *testing.T) {
		repo, db := newStockRepoForTest(t)
		restoreAt(t, repo, "s1", "玉ねぎ", "個", "2026-09-01T00:00:00Z")
		restoreAt(t, repo, "s2", "卵", "個", "2026-09-01T00:00:00Z")
		restoreAt(t, repo, "s3", "塩", "", "2026-09-01T00:00:00Z")
		createRecipe(t, db, "r", "2026-09-01T00:00:00Z",
			recipe.RecipeIngredient{Name: "醤油", Amount: 1, Unit: "ml"},
			recipe.RecipeIngredient{Name: "胡椒", Amount: 1, Unit: ""},
		)
		got, _ := repo.ListSuggestions(ctx)
		if !reflect.DeepEqual(got.Units, []string{"ml", "個"}) {
			t.Errorf("want [ml 個], got %v", got.Units)
		}
	})
}
