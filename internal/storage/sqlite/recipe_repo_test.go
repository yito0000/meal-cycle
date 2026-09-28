package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func newRecipeRepoForTest(t *testing.T) *RecipeRepo {
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
	return NewRecipeRepo(db)
}

func TestRecipeRepo_List(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty slice when db is fresh", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want empty, got %d", len(got))
		}
	})

	t.Run("returns three recipes ordered by updatedAt desc when three exist", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		for _, name := range []string{"first", "second", "third"} {
			if _, err := repo.Create(ctx, recipe.Recipe{Name: name, Servings: 1}); err != nil {
				t.Fatalf("Create %s: %v", name, err)
			}
		}
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("want 3, got %d", len(got))
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].UpdatedAt < got[i].UpdatedAt {
				t.Errorf("not sorted desc at %d: %q vs %q",
					i, got[i-1].UpdatedAt, got[i].UpdatedAt)
			}
		}
	})

	t.Run("returns IngredientCount matching recipe_ingredients count", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		if _, err := repo.Create(ctx, recipe.Recipe{
			Name:     "肉じゃが",
			Servings: 2,
			Ingredients: []recipe.RecipeIngredient{
				{Name: "玉ねぎ", Amount: 1, Unit: "個"},
				{Name: "じゃがいも", Amount: 3, Unit: "個"},
			},
		}); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].IngredientCount != 2 {
			t.Errorf("want IngredientCount=2, got %+v", got)
		}
	})

	t.Run("excludes archived recipes and returns only active ones", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		archived, err := repo.Create(ctx, recipe.Recipe{Name: "肉じゃが", Servings: 1})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := repo.Create(ctx, recipe.Recipe{Name: "豚汁", Servings: 1}); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := repo.Archive(ctx, archived.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].Name != "豚汁" {
			t.Errorf("want only 豚汁, got %+v", got)
		}
	})

	t.Run("returns empty slice when all recipes are archived", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{Name: "肉じゃが", Servings: 1})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := repo.Archive(ctx, created.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("want empty non-nil slice, got %#v", got)
		}
	})
}

func TestRecipeRepo_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns Ingredients Steps Notes Tags matching what was saved", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{
			Name:     "肉じゃが",
			Servings: 2,
			Tags:     []string{"和食", "煮物"},
			Ingredients: []recipe.RecipeIngredient{
				{Name: "玉ねぎ", Amount: 1, Unit: "個"},
			},
			Steps: "1. 玉ねぎを切る\n2. 煮る",
			Notes: "落し蓋を使う",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "肉じゃが" || got.Servings != 2 || got.Notes != "落し蓋を使う" {
			t.Errorf("unexpected recipe: %+v", got)
		}
		if len(got.Tags) != 2 || got.Tags[0] != "和食" {
			t.Errorf("unexpected tags: %v", got.Tags)
		}
		if got.Steps != "1. 玉ねぎを切る\n2. 煮る" {
			t.Errorf("unexpected steps: %q", got.Steps)
		}
		if len(got.Ingredients) != 1 || got.Ingredients[0].Name != "玉ねぎ" {
			t.Errorf("unexpected ingredients: %v", got.Ingredients)
		}
	})

	t.Run("returns ErrRecipeNotFound when id does not exist", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		_, err := repo.GetByID(ctx, "no-such-id")
		if !errors.Is(err, ErrRecipeNotFound) {
			t.Errorf("want ErrRecipeNotFound, got %v", err)
		}
	})

	t.Run("returns empty ArchivedAt when recipe is active", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{Name: "肉じゃが", Servings: 1})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ArchivedAt != "" {
			t.Errorf("want empty ArchivedAt, got %q", got.ArchivedAt)
		}
	})
}

func TestRecipeRepo_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ErrRecipeNameEmpty and does not write when name is empty", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		_, err := repo.Create(ctx, recipe.Recipe{Name: ""})
		if !errors.Is(err, ErrRecipeNameEmpty) {
			t.Errorf("want ErrRecipeNameEmpty, got %v", err)
		}
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want no recipes written, got %d", len(got))
		}
	})

	t.Run("stores ingredients in input order and returns them in the same order via GetByID", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{
			Name: "肉じゃが",
			Ingredients: []recipe.RecipeIngredient{
				{Name: "玉ねぎ", Amount: 1, Unit: "個"},
				{Name: "にんじん", Amount: 1, Unit: "本"},
				{Name: "じゃがいも", Amount: 3, Unit: "個"},
			},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		wantOrder := []string{"玉ねぎ", "にんじん", "じゃがいも"}
		if len(got.Ingredients) != len(wantOrder) {
			t.Fatalf("want %d ingredients, got %d", len(wantOrder), len(got.Ingredients))
		}
		for i, want := range wantOrder {
			if got.Ingredients[i].Name != want {
				t.Errorf("pos %d: want %q, got %q", i, want, got.Ingredients[i].Name)
			}
		}
	})

	t.Run("preserves Markdown text in Steps across Create and GetByID", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		markdown := "## 下ごしらえ\n- 玉ねぎを切る\n- 豚肉を漬ける\n\n## 加熱\n強火で 2 分。"
		created, err := repo.Create(ctx, recipe.Recipe{Name: "x", Steps: markdown})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Steps != markdown {
			t.Errorf("Steps not preserved:\nwant %q\ngot  %q", markdown, got.Steps)
		}
	})
}

func TestRecipeRepo_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ErrRecipeNotFound when id does not exist", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		_, err := repo.Update(ctx, "no-such-id", recipe.Recipe{Name: "x"})
		if !errors.Is(err, ErrRecipeNotFound) {
			t.Errorf("want ErrRecipeNotFound, got %v", err)
		}
	})

	t.Run("returns ErrRecipeNameEmpty when name is empty", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{Name: "x"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		_, err = repo.Update(ctx, created.ID, recipe.Recipe{Name: ""})
		if !errors.Is(err, ErrRecipeNameEmpty) {
			t.Errorf("want ErrRecipeNameEmpty, got %v", err)
		}
	})

	t.Run("returns new name via GetByID after Update", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{Name: "old"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := repo.Update(ctx, created.ID, recipe.Recipe{Name: "new"}); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "new" {
			t.Errorf("want new, got %q", got.Name)
		}
	})

	t.Run("advances updatedAt after Update", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{Name: "x"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		updated, err := repo.Update(ctx, created.ID, recipe.Recipe{Name: "x"})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.UpdatedAt <= created.UpdatedAt {
			t.Errorf("updatedAt not advanced: %q <= %q", updated.UpdatedAt, created.UpdatedAt)
		}
	})

	t.Run("replaces ingredients so old rows do not linger", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{
			Name: "x",
			Ingredients: []recipe.RecipeIngredient{
				{Name: "玉ねぎ", Amount: 1, Unit: "個"},
				{Name: "にんじん", Amount: 1, Unit: "本"},
			},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := repo.Update(ctx, created.ID, recipe.Recipe{
			Name: "x",
			Ingredients: []recipe.RecipeIngredient{
				{Name: "じゃがいも", Amount: 2, Unit: "個"},
			},
		}); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if len(got.Ingredients) != 1 || got.Ingredients[0].Name != "じゃがいも" {
			t.Errorf("old ingredients lingered: %+v", got.Ingredients)
		}
	})
}

func TestRecipeRepo_Create_LongMarkdown_UsedAsSmokeSize(t *testing.T) {
	// SQLite の TEXT カラムに 4000 字クラスの Markdown が入っても壊れないことの smoke test。
	// バリデーション本体は Service 層のテストで扱う。
	repo := newRecipeRepoForTest(t)
	body := strings.Repeat("あ", 3999) + "。"
	created, err := repo.Create(context.Background(), recipe.Recipe{Name: "x", Steps: body})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len([]rune(got.Steps)) != 4000 {
		t.Errorf("Steps rune count: want 4000, got %d", len([]rune(got.Steps)))
	}
}

func TestRecipeRepo_Update_Archived(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ErrRecipeArchived and keeps name and UpdatedAt when recipe is archived", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created, err := repo.Create(ctx, recipe.Recipe{Name: "肉じゃが", Servings: 1})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := repo.Archive(ctx, created.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		_, err = repo.Update(ctx, created.ID, recipe.Recipe{Name: "豚汁", Servings: 1})
		if !errors.Is(err, ErrRecipeArchived) {
			t.Fatalf("want ErrRecipeArchived, got %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "肉じゃが" || got.UpdatedAt != created.UpdatedAt {
			t.Errorf("recipe changed: %+v", got)
		}
	})
}

func TestRecipeRepo_Archive(t *testing.T) {
	ctx := context.Background()

	create := func(t *testing.T, repo *RecipeRepo) recipe.Recipe {
		t.Helper()
		created, err := repo.Create(ctx, recipe.Recipe{
			Name:        "肉じゃが",
			Servings:    2,
			Ingredients: []recipe.RecipeIngredient{{Name: "玉ねぎ", Amount: 1, Unit: "個"}},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		return created
	}

	t.Run("sets ArchivedAt in RFC3339 when recipe is active", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created := create(t, repo)
		if err := repo.Archive(ctx, created.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if _, err := time.Parse(time.RFC3339, got.ArchivedAt); err != nil {
			t.Errorf("ArchivedAt not RFC3339: %q (%v)", got.ArchivedAt, err)
		}
	})

	t.Run("keeps UpdatedAt unchanged", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created := create(t, repo)
		if err := repo.Archive(ctx, created.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.UpdatedAt != created.UpdatedAt {
			t.Errorf("UpdatedAt changed: %q -> %q", created.UpdatedAt, got.UpdatedAt)
		}
	})

	t.Run("returns nil and keeps first ArchivedAt when already archived", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created := create(t, repo)
		const first = "2026-01-01T00:00:00Z"
		if _, err := repo.db.ExecContext(ctx,
			`UPDATE recipes SET archived_at = ? WHERE id = ?`, first, created.ID); err != nil {
			t.Fatalf("seed archived_at: %v", err)
		}
		if err := repo.Archive(ctx, created.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ArchivedAt != first {
			t.Errorf("want %q, got %q", first, got.ArchivedAt)
		}
	})

	t.Run("returns ErrRecipeNotFound when id does not exist", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		if err := repo.Archive(ctx, "no-such-id"); !errors.Is(err, ErrRecipeNotFound) {
			t.Errorf("want ErrRecipeNotFound, got %v", err)
		}
	})

	t.Run("keeps ingredients returned by GetByID", func(t *testing.T) {
		repo := newRecipeRepoForTest(t)
		created := create(t, repo)
		if err := repo.Archive(ctx, created.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if len(got.Ingredients) != 1 || got.Ingredients[0].Name != "玉ねぎ" {
			t.Errorf("unexpected ingredients: %+v", got.Ingredients)
		}
	})

	t.Run("stays archived after reopening the database", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.db")
		db, err := Open(path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if err := Migrate(db); err != nil {
			t.Fatalf("Migrate: %v", err)
		}
		repo := NewRecipeRepo(db)
		created := create(t, repo)
		if err := repo.Archive(ctx, created.ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}

		reopened, err := Open(path)
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}
		t.Cleanup(func() { _ = reopened.Close() })
		if err := Migrate(reopened); err != nil {
			t.Fatalf("Migrate: %v", err)
		}
		got, err := NewRecipeRepo(reopened).List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want archived recipe hidden after reopen, got %+v", got)
		}
	})
}
