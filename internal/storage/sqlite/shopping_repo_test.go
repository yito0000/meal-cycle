package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/shopping"
)

type shoppingFixture struct {
	db   *sql.DB
	repo *ShoppingRepo
	rr   *RecipeRepo
}

func newShoppingRepoForTest(t *testing.T) *shoppingFixture {
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
	return &shoppingFixture{db: db, repo: NewShoppingRepo(db), rr: NewRecipeRepo(db)}
}

func (f *shoppingFixture) recipe(t *testing.T, name string, ings ...recipe.RecipeIngredient) string {
	t.Helper()
	r, err := f.rr.Create(context.Background(), recipe.Recipe{Name: name, Servings: 1, Ingredients: ings})
	if err != nil {
		t.Fatalf("create recipe %s: %v", name, err)
	}
	return r.ID
}

// place は batch_id を明示して 1 行置く。作り置きの共有関係をテストから直接組み立てるため。
func (f *shoppingFixture) place(t *testing.T, date, meal, recipeID, batchID string) {
	t.Helper()
	ctx := context.Background()
	pos, err := nextPosition(ctx, f.db, date, meal)
	if err != nil {
		t.Fatalf("nextPosition: %v", err)
	}
	if _, err := insertMealPlan(ctx, f.db, mealplan.NewEntryID(), date, meal, pos, recipeID, batchID); err != nil {
		t.Fatalf("insertMealPlan: %v", err)
	}
}

func (f *shoppingFixture) list(t *testing.T, start string, days int) []shopping.PlannedBatch {
	t.Helper()
	got, err := f.repo.ListPlannedBatches(context.Background(), start, days)
	if err != nil {
		t.Fatalf("ListPlannedBatches: %v", err)
	}
	return got
}

func onion(n float64) recipe.RecipeIngredient {
	return recipe.RecipeIngredient{Name: "玉ねぎ", Amount: n, Unit: "個"}
}

func TestShoppingRepo_ListPlannedBatches(t *testing.T) {
	const start = "2026-09-24"

	t.Run("should return empty slice when no meal plans exist", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		got := f.list(t, start, 7)
		if got == nil || len(got) != 0 {
			t.Errorf("want empty non-nil slice, got %#v", got)
		}
	})

	t.Run("should return single assignment with ingredients in position order", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "肉じゃが", onion(2), recipe.RecipeIngredient{Name: "豚肉", Amount: 200, Unit: "g"})
		f.place(t, "2026-09-25", "dinner", id, "b1")
		got := f.list(t, start, 7)
		if len(got) != 1 {
			t.Fatalf("want 1, got %d", len(got))
		}
		b := got[0]
		if b.FirstDate != "2026-09-25" || b.BatchID != "b1" || b.RecipeID != id || b.RecipeName != "肉じゃが" {
			t.Errorf("unexpected batch: %#v", b)
		}
		wantIngs := []recipe.RecipeIngredient{onion(2), {Name: "豚肉", Amount: 200, Unit: "g"}}
		if !reflect.DeepEqual(b.Ingredients, wantIngs) {
			t.Errorf("ingredients: got %#v, want %#v", b.Ingredients, wantIngs)
		}
		if b.BatchMemberCount != 1 || b.StartedBeforePeriod {
			t.Errorf("single assignment: got count=%d before=%v", b.BatchMemberCount, b.StartedBeforePeriod)
		}
	})

	t.Run("should keep counting ingredients of an archived recipe that remains in the plan", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "肉じゃが", onion(1))
		f.place(t, "2026-09-25", "dinner", id, "b1")
		if err := f.rr.Archive(context.Background(), id); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got := f.list(t, start, 7)
		if len(got) != 1 || got[0].RecipeName != "肉じゃが" {
			t.Fatalf("want 肉じゃが kept, got %#v", got)
		}
		if !reflect.DeepEqual(got[0].Ingredients, []recipe.RecipeIngredient{onion(1)}) {
			t.Errorf("ingredients: got %#v", got[0].Ingredients)
		}
	})

	t.Run("should merge three rows of one batch into one with earliest first date", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "肉じゃが", onion(2))
		for _, d := range []string{"2026-09-26", "2026-09-24", "2026-09-25"} {
			f.place(t, d, "dinner", id, "b1")
		}
		got := f.list(t, start, 7)
		if len(got) != 1 {
			t.Fatalf("want 1, got %d", len(got))
		}
		want := []string{"2026-09-24", "2026-09-25", "2026-09-26"}
		if got[0].FirstDate != "2026-09-24" || !reflect.DeepEqual(got[0].Dates, want) || got[0].BatchMemberCount != 3 {
			t.Errorf("got %#v", got[0])
		}
	})

	t.Run("should list meals in breakfast lunch dinner order without duplicates", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "カレー", onion(3))
		f.place(t, "2026-09-24", "dinner", id, "b1")
		f.place(t, "2026-09-25", "breakfast", id, "b1")
		f.place(t, "2026-09-25", "dinner", id, "b1")
		got := f.list(t, start, 7)
		if want := []string{"breakfast", "dinner"}; !reflect.DeepEqual(got[0].Meals, want) {
			t.Errorf("meals: got %v, want %v", got[0].Meals, want)
		}
	})

	t.Run("should set first meal and cell position of the first row in the period", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		a := f.recipe(t, "肉じゃが", onion(1))
		b := f.recipe(t, "みそ汁", onion(0.5))
		f.place(t, "2026-09-24", "dinner", a, "b1")
		f.place(t, "2026-09-24", "dinner", b, "b2")
		got := f.list(t, start, 7)
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
		byName := map[string]shopping.PlannedBatch{got[0].RecipeName: got[0], got[1].RecipeName: got[1]}
		if byName["肉じゃが"].FirstMeal != "dinner" || byName["肉じゃが"].FirstPosition != 0 ||
			byName["みそ汁"].FirstMeal != "dinner" || byName["みそ汁"].FirstPosition != 1 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should return two batches when the same recipe has different batch ids", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "肉じゃが", onion(2))
		f.place(t, "2026-09-24", "dinner", id, "b1")
		f.place(t, "2026-09-26", "dinner", id, "b2")
		if got := f.list(t, start, 7); len(got) != 2 {
			t.Errorf("want 2, got %d", len(got))
		}
	})

	t.Run("should split one batch id per recipe when recipes are mixed", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		a := f.recipe(t, "肉じゃが", onion(2))
		b := f.recipe(t, "カレー", onion(3))
		f.place(t, "2026-09-24", "dinner", a, "b1")
		f.place(t, "2026-09-25", "dinner", b, "b1")
		got := f.list(t, start, 7)
		if len(got) != 2 || got[0].RecipeID != a || got[1].RecipeID != b {
			t.Fatalf("want 2 batches per recipe, got %#v", got)
		}
		if got[0].BatchMemberCount != 2 {
			t.Errorf("member count counts whole batch id: got %d", got[0].BatchMemberCount)
		}
	})

	t.Run("should mark started before period when batch continues from the previous day", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "カレー", onion(3))
		f.place(t, "2026-09-23", "dinner", id, "b1")
		f.place(t, "2026-09-24", "dinner", id, "b1")
		got := f.list(t, start, 7)
		if len(got) != 1 {
			t.Fatalf("want 1, got %d", len(got))
		}
		b := got[0]
		if b.FirstDate != start || !reflect.DeepEqual(b.Dates, []string{start}) ||
			!b.StartedBeforePeriod || b.BatchMemberCount != 2 {
			t.Errorf("got %#v", b)
		}
	})

	t.Run("should not mark started before period when batch continues after the period", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "カレー", onion(3))
		f.place(t, "2026-09-30", "dinner", id, "b1")
		f.place(t, "2026-10-01", "dinner", id, "b1")
		got := f.list(t, start, 7)
		if len(got) != 1 || got[0].StartedBeforePeriod || got[0].BatchMemberCount != 2 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should exclude batches that only have rows outside the period", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "カレー", onion(3))
		f.place(t, "2026-09-23", "dinner", id, "b1")
		f.place(t, "2026-10-01", "dinner", id, "b2")
		if got := f.list(t, start, 7); len(got) != 0 {
			t.Errorf("want 0, got %#v", got)
		}
	})

	t.Run("should only include the first three days when period days is 3", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "カレー", onion(3))
		f.place(t, "2026-09-26", "dinner", id, "b1")
		f.place(t, "2026-09-27", "dinner", id, "b2")
		got := f.list(t, start, 3)
		if len(got) != 1 || got[0].FirstDate != "2026-09-26" {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should return recipes without ingredients as empty ingredients", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		id := f.recipe(t, "ご飯")
		f.place(t, "2026-09-24", "dinner", id, "b1")
		got := f.list(t, start, 7)
		if len(got) != 1 || got[0].Ingredients == nil || len(got[0].Ingredients) != 0 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should include breakfast lunch and dinner", func(t *testing.T) {
		f := newShoppingRepoForTest(t)
		a := f.recipe(t, "トースト")
		b := f.recipe(t, "サンドイッチ")
		c := f.recipe(t, "カレー")
		f.place(t, "2026-09-24", "breakfast", a, "b1")
		f.place(t, "2026-09-24", "lunch", b, "b2")
		f.place(t, "2026-09-24", "dinner", c, "b3")
		if got := f.list(t, start, 7); len(got) != 3 {
			t.Errorf("want 3, got %d", len(got))
		}
	})
}
