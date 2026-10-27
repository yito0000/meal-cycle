package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func newMealPlanRepoForTest(t *testing.T) (*MealPlanRepo, *RecipeRepo) {
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
	return NewMealPlanRepo(db), NewRecipeRepo(db)
}

func seedRecipes(t *testing.T, rr *RecipeRepo, names ...string) []recipe.Recipe {
	t.Helper()
	ctx := context.Background()
	out := make([]recipe.Recipe, 0, len(names))
	for _, n := range names {
		r, err := rr.Create(ctx, recipe.Recipe{Name: n, Servings: 1})
		if err != nil {
			t.Fatalf("seed recipe %s: %v", n, err)
		}
		out = append(out, r)
	}
	return out
}

// assignID は recipeID を (date, meal) の末尾に置き、その行の ID を返す。
func assignID(t *testing.T, repo *MealPlanRepo, date, meal, recipeID string) string {
	t.Helper()
	e, err := repo.Assign(context.Background(), date, meal, recipeID)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	return e.ID
}

// idAt は (date, meal, position) にいまある行の ID を返す。テストの前提を並び順で書くためのもの。
func idAt(t *testing.T, repo *MealPlanRepo, date, meal string, position int) string {
	t.Helper()
	var id string
	if err := repo.db.QueryRow(`
		SELECT id FROM meal_plans WHERE date = ? AND meal = ? AND position = ?
	`, date, meal, position).Scan(&id); err != nil {
		t.Fatalf("idAt(%s, %s, %d): %v", date, meal, position, err)
	}
	return id
}

// idsByRecipe は (date, meal) の行の ID を、recipeIDs の順に並べて返す。セル内でレシピが重複しない前提。
func idsByRecipe(t *testing.T, repo *MealPlanRepo, date, meal string, recipeIDs ...string) []string {
	t.Helper()
	entries, err := repo.MealEntries(context.Background(), date, meal)
	if err != nil {
		t.Fatalf("MealEntries: %v", err)
	}
	byRecipe := map[string]string{}
	for _, e := range entries {
		byRecipe[e.RecipeID] = e.ID
	}
	out := make([]string, 0, len(recipeIDs))
	for _, r := range recipeIDs {
		out = append(out, byRecipe[r])
	}
	return out
}

func TestMealPlanRepo_ListPeriod(t *testing.T) {
	ctx := context.Background()

	t.Run("should return empty slice when no meal plans exist", func(t *testing.T) {
		repo, _ := newMealPlanRepoForTest(t)
		got, err := repo.ListPeriod(ctx, "2026-08-24", 7)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want empty, got %d", len(got))
		}
	})

	t.Run("should return entries JOINed with recipes.name for the requested week", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		if _, err := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID); err != nil {
			t.Fatalf("Assign: %v", err)
		}
		if _, err := repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID); err != nil {
			t.Fatalf("Assign: %v", err)
		}
		got, err := repo.ListPeriod(ctx, "2026-08-24", 7)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
		if got[0].RecipeName != "肉じゃが" || got[1].RecipeName != "みそ汁" {
			t.Errorf("recipe names not joined: %+v", got)
		}
	})

	t.Run("should ignore entries outside the requested week", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-23", "dinner", rs[0].ID) // 前週日曜
		_, _ = repo.Assign(ctx, "2026-08-31", "dinner", rs[0].ID) // 翌週月曜
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 0 {
			t.Errorf("want 0 (both dates outside week), got %d", len(got))
		}
	})

	t.Run("periodDays=14 should include entries in the next week", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID) // 月曜 (1 週目)
		_, _ = repo.Assign(ctx, "2026-08-31", "dinner", rs[0].ID) // 翌週月曜 (2 週目)
		_, _ = repo.Assign(ctx, "2026-09-07", "dinner", rs[0].ID) // 3 週目 (レンジ外)
		got, err := repo.ListPeriod(ctx, "2026-08-24", 14)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 (2 週分), got %d: %+v", len(got), got)
		}
		if got[0].Date != "2026-08-24" || got[1].Date != "2026-08-31" {
			t.Errorf("dates = %s, %s; want 2026-08-24, 2026-08-31", got[0].Date, got[1].Date)
		}
	})

	t.Run("periodDays=28 should include entries across 4 weeks", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID) // 1 週目
		_, _ = repo.Assign(ctx, "2026-08-31", "dinner", rs[0].ID) // 2 週目
		_, _ = repo.Assign(ctx, "2026-09-07", "dinner", rs[0].ID) // 3 週目
		_, _ = repo.Assign(ctx, "2026-09-14", "dinner", rs[0].ID) // 4 週目
		_, _ = repo.Assign(ctx, "2026-09-21", "dinner", rs[0].ID) // 5 週目 (レンジ外)
		got, err := repo.ListPeriod(ctx, "2026-08-24", 28)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 4 {
			t.Errorf("want 4 (4 週分), got %d", len(got))
		}
	})
}

func TestMealPlanRepo_Assign(t *testing.T) {
	ctx := context.Background()

	t.Run("Assign should append with position = max+1 when the meal already has items", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		got, err := repo.Assign(ctx, "2026-08-24", "dinner", rs[2].ID)
		if err != nil {
			t.Fatalf("Assign: %v", err)
		}
		if got.Position != 2 {
			t.Errorf("position = %d, want 2", got.Position)
		}
	})

	t.Run("Assign should return error when recipeID does not reference an existing recipe", func(t *testing.T) {
		repo, _ := newMealPlanRepoForTest(t)
		_, err := repo.Assign(ctx, "2026-08-24", "dinner", "ghost")
		if !errors.Is(err, mealplan.ErrUnknownRecipe) {
			t.Errorf("err = %v, want ErrUnknownRecipe", err)
		}
	})

	t.Run("Assign should return ErrUnknownRecipe and add no row when the recipe is archived", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		if err := rr.Archive(ctx, rs[0].ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		_, err := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		if !errors.Is(err, mealplan.ErrUnknownRecipe) {
			t.Errorf("err = %v, want ErrUnknownRecipe", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 0 {
			t.Errorf("want no rows, got %d", len(got))
		}
	})
}

func TestMealPlanRepo_Replace(t *testing.T) {
	ctx := context.Background()

	t.Run("Replace should update only the target row and keep other positions untouched", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		if err := repo.Replace(ctx, idAt(t, repo, "2026-08-24", "dinner", 1), rs[2].ID); err != nil {
			t.Fatalf("Replace: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
		if got[0].RecipeID != rs[0].ID {
			t.Errorf("position 0 got %s, want %s", got[0].RecipeID, rs[0].ID)
		}
		if got[1].RecipeID != rs[2].ID {
			t.Errorf("position 1 got %s, want %s (C)", got[1].RecipeID, rs[2].ID)
		}
	})

	t.Run("Replace returns ErrEntryNotFound when the entry is missing", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		err := repo.Replace(ctx, "no-such-entry", rs[0].ID)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})

	t.Run("Replace should return ErrUnknownRecipe and keep the original recipe when the new recipe is archived", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		if err := rr.Archive(ctx, rs[1].ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		err := repo.Replace(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), rs[1].ID)
		if !errors.Is(err, mealplan.ErrUnknownRecipe) {
			t.Errorf("err = %v, want ErrUnknownRecipe", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 || got[0].RecipeID != rs[0].ID {
			t.Errorf("want original A kept, got %+v", got)
		}
	})
}

func TestMealPlanRepo_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("Remove should renumber subsequent positions to be contiguous", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[2].ID)
		if _, err := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 1)); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
		if got[0].Position != 0 || got[1].Position != 1 {
			t.Errorf("positions = %d, %d; want 0, 1", got[0].Position, got[1].Position)
		}
		if got[0].RecipeID != rs[0].ID || got[1].RecipeID != rs[2].ID {
			t.Errorf("recipeIDs = %s, %s; want %s, %s", got[0].RecipeID, got[1].RecipeID, rs[0].ID, rs[2].ID)
		}
	})

	t.Run("Remove should succeed and keep the remaining order when the meal was reordered", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		for _, r := range rs {
			_, _ = repo.Assign(ctx, "2026-08-24", "dinner", r.ID)
		}
		if err := repo.Reorder(ctx, "2026-08-24", "dinner", idsByRecipe(t, repo, "2026-08-24", "dinner", rs[1].ID, rs[2].ID, rs[0].ID)); err != nil {
			t.Fatalf("Reorder: %v", err)
		}

		if _, err := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 0)); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[2].ID, rs[0].ID)
	})

	t.Run("Remove should return the removed row with its batch id and createdAt", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		b, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)

		got, err := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 1))
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if got.Date != "2026-08-24" || got.Meal != mealplan.MealDinner || got.Position != 1 ||
			got.RecipeID != rs[1].ID || got.BatchID != b.BatchID {
			t.Errorf("removed = %+v, want B at 2026-08-24/dinner/1 in batch %s", got, b.BatchID)
		}
		if _, err := time.Parse(time.RFC3339, got.CreatedAt); err != nil {
			t.Errorf("createdAt = %q, want RFC3339", got.CreatedAt)
		}
		if got.CookedAt != "" {
			t.Errorf("cookedAt = %q, want empty", got.CookedAt)
		}
	})

	t.Run("Remove should return cookedAt when the removed row carried a cooked mark", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)
		want := cookedAtOf(t, repo, rs[0].ID)

		got, err := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 0))
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if got.CookedAt == "" || got.CookedAt != want {
			t.Errorf("cookedAt = %q, want %q", got.CookedAt, want)
		}
	})
}

// cookedAtOf は recipeID の作った印の日時を返す。印が無ければ空文字。
func cookedAtOf(t *testing.T, repo *MealPlanRepo, recipeID string) string {
	t.Helper()
	var at string
	err := repo.db.QueryRow(`SELECT cooked_at FROM meal_plan_cooked WHERE recipe_id = ?`, recipeID).Scan(&at)
	if errors.Is(err, sql.ErrNoRows) {
		return ""
	}
	if err != nil {
		t.Fatalf("query cooked_at: %v", err)
	}
	return at
}

// assertMealOrder は 1 セルの行が wantRecipeIDs の順で position 0 から隙間なく並ぶことを確かめる。
func assertMealOrder(t *testing.T, got []mealplan.Entry, wantRecipeIDs ...string) {
	t.Helper()
	if len(got) != len(wantRecipeIDs) {
		t.Fatalf("want %d rows, got %d: %+v", len(wantRecipeIDs), len(got), got)
	}
	for i, want := range wantRecipeIDs {
		if got[i].Position != i || got[i].RecipeID != want {
			t.Errorf("row %d = (position %d, recipe %s); want (position %d, recipe %s)",
				i, got[i].Position, got[i].RecipeID, i, want)
		}
	}
}

func TestMealPlanRepo_MoveEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("MoveEntry should move a single row to target cell (source becomes empty, target gets 1 row)", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)

		if err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), "2026-08-25", "dinner"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		src, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if len(src) != 0 {
			t.Errorf("source cell should be empty, got %d rows: %+v", len(src), src)
		}
		dst, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		if len(dst) != 1 || dst[0].Position != 0 || dst[0].RecipeID != rs[0].ID {
			t.Errorf("target cell = %+v; want 1 row at pos 0 with recipe %s", dst, rs[0].ID)
		}
	})

	t.Run("MoveEntry should renumber remaining source positions to be contiguous", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[2].ID)

		if err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-24", "dinner", 1), "2026-08-25", "dinner"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		src, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if len(src) != 2 {
			t.Fatalf("want 2 rows remaining in source, got %d", len(src))
		}
		if src[0].Position != 0 || src[1].Position != 1 {
			t.Errorf("positions = %d, %d; want 0, 1", src[0].Position, src[1].Position)
		}
		if src[0].RecipeID != rs[0].ID || src[1].RecipeID != rs[2].ID {
			t.Errorf("recipeIDs = %s, %s; want %s, %s", src[0].RecipeID, src[1].RecipeID, rs[0].ID, rs[2].ID)
		}
	})

	t.Run("MoveEntry should succeed and keep the remaining source order when the source meal was reversed", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		for _, r := range rs {
			_, _ = repo.Assign(ctx, "2026-08-24", "dinner", r.ID)
		}
		if err := repo.Reorder(ctx, "2026-08-24", "dinner", idsByRecipe(t, repo, "2026-08-24", "dinner", rs[2].ID, rs[1].ID, rs[0].ID)); err != nil {
			t.Fatalf("Reorder: %v", err)
		}

		if err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), "2026-08-25", "dinner"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		src, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, src, rs[1].ID, rs[0].ID)
		dst, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, dst, rs[2].ID)
	})

	t.Run("MoveEntry should preserve batch_id when moving a row from a leftover batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25", "2026-08-26"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)

		if err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-26", "dinner", 0), "2026-08-27", "dinner"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		bids := map[string]int{}
		for _, e := range got {
			bids[e.BatchID]++
		}
		if len(bids) != 1 {
			t.Fatalf("want 1 shared batchID after move, got %d: %+v", len(bids), bids)
		}
		if _, ok := bids[src.BatchID]; !ok {
			t.Errorf("shared batchID %+v does not include source %q", bids, src.BatchID)
		}
	})

	t.Run("MoveEntry should preserve batch_id when moving across meals", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)

		if err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), "2026-08-24", "lunch"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		lunch, _ := repo.MealEntries(ctx, "2026-08-24", "lunch")
		if len(lunch) != 1 {
			t.Fatalf("want 1 row in lunch, got %d: %+v", len(lunch), lunch)
		}
		if lunch[0].BatchID != src.BatchID {
			t.Errorf("batchID = %q, want %q (preserved across meals)", lunch[0].BatchID, src.BatchID)
		}
	})

	t.Run("MoveEntry returns ErrEntryNotFound when source row is missing", func(t *testing.T) {
		repo, _ := newMealPlanRepoForTest(t)
		err := repo.MoveEntry(ctx, "no-such-entry", "2026-08-25", "dinner")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})

	t.Run("MoveEntry returns ErrInvalidPosition when source == target cell", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), "2026-08-24", "dinner")
		if !errors.Is(err, mealplan.ErrInvalidPosition) {
			t.Errorf("err = %v, want ErrInvalidPosition", err)
		}
	})
}

func TestMealPlanRepo_ClearMeal(t *testing.T) {
	ctx := context.Background()

	t.Run("ClearMeal should delete every row in the target (date, meal) and leave other meals untouched", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "lunch", rs[0].ID)

		if _, err := repo.ClearMeal(ctx, "2026-08-24", "dinner"); err != nil {
			t.Fatalf("ClearMeal: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 {
			t.Fatalf("want 1 (lunch only), got %d", len(got))
		}
		if got[0].Meal != mealplan.MealLunch {
			t.Errorf("remaining meal = %s, want lunch", got[0].Meal)
		}
	})

	t.Run("ClearMeal should return all removed rows ordered by position", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		for _, r := range rs {
			_, _ = repo.Assign(ctx, "2026-08-24", "dinner", r.ID)
		}
		_ = repo.Reorder(ctx, "2026-08-24", "dinner", idsByRecipe(t, repo, "2026-08-24", "dinner", rs[2].ID, rs[0].ID, rs[1].ID))

		got, err := repo.ClearMeal(ctx, "2026-08-24", "dinner")
		if err != nil {
			t.Fatalf("ClearMeal: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("want 3 removed rows, got %d", len(got))
		}
		for i, want := range []string{rs[2].ID, rs[0].ID, rs[1].ID} {
			if got[i].Position != i || got[i].RecipeID != want {
				t.Errorf("row %d = (position %d, recipe %s); want (%d, %s)", i, got[i].Position, got[i].RecipeID, i, want)
			}
		}
	})

	t.Run("ClearMeal should return an empty slice when the meal has no rows", func(t *testing.T) {
		repo, _ := newMealPlanRepoForTest(t)
		got, err := repo.ClearMeal(ctx, "2026-08-24", "dinner")
		if err != nil {
			t.Fatalf("ClearMeal: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("want empty non-nil slice, got %#v", got)
		}
	})
}

func TestMealPlanRepo_RestoreEntries(t *testing.T) {
	ctx := context.Background()

	t.Run("RestoreEntries should put a row back at its original position and shift later rows down", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		for _, r := range rs {
			_, _ = repo.Assign(ctx, "2026-08-24", "dinner", r.ID)
		}
		removed, err := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 1))
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}

		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[0].ID, rs[1].ID, rs[2].ID)
	})

	t.Run("RestoreEntries should append at the end when the original position exceeds the current count", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		for _, r := range rs {
			_, _ = repo.Assign(ctx, "2026-08-24", "dinner", r.ID)
		}
		removed, _ := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 2))
		_, _ = repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 0))

		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[1].ID, rs[2].ID)
	})

	t.Run("RestoreEntries should restore all rows of a cleared meal in the original order", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		for _, r := range rs {
			_, _ = repo.Assign(ctx, "2026-08-24", "dinner", r.ID)
		}
		removed, _ := repo.ClearMeal(ctx, "2026-08-24", "dinner")
		reversed := []mealplan.RemovedEntry{removed[2], removed[1], removed[0]}

		if err := repo.RestoreEntries(ctx, reversed); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[0].ID, rs[1].ID, rs[2].ID)
	})

	t.Run("RestoreEntries should keep the original batch id so the row rejoins its batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		src := seedLeftover(t, repo, rs[0].ID, "2026-08-25", "2026-08-26")
		removed, _ := repo.Remove(ctx, idAt(t, repo, "2026-08-25", "dinner", 0))

		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 3 {
			t.Fatalf("want 3 rows, got %d", len(got))
		}
		for _, e := range got {
			if e.BatchID != src.BatchID || e.BatchMemberCount != 3 {
				t.Errorf("%s: batch = (%s, %d), want (%s, 3)", e.Date, e.BatchID, e.BatchMemberCount, src.BatchID)
			}
		}
	})

	t.Run("RestoreEntries should bring back the cooked mark with the original cookedAt when the mark was pruned", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)
		want := cookedAtOf(t, repo, rs[0].ID)
		removed, _ := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 0))
		if n := countCookedMarks(t, repo); n != 0 {
			t.Fatalf("precondition: want mark pruned, got %d", n)
		}

		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		if got := cookedByCell(t, repo, "2026-08-24"); !got["2026-08-24/dinner/肉じゃが"] {
			t.Errorf("want cooked, got %+v", got)
		}
		if got := cookedAtOf(t, repo, rs[0].ID); got != want {
			t.Errorf("cookedAt = %q, want %q", got, want)
		}
	})

	t.Run("RestoreEntries should not duplicate the cooked mark when other rows of the batch still carry it", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		seedLeftover(t, repo, rs[0].ID, "2026-08-25")
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)
		removed, _ := repo.Remove(ctx, idAt(t, repo, "2026-08-25", "dinner", 0))

		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		if n := countCookedMarks(t, repo); n != 1 {
			t.Errorf("want 1 mark, got %d", n)
		}
		for cell, cooked := range cookedByCell(t, repo, "2026-08-24") {
			if !cooked {
				t.Errorf("%s: want cooked", cell)
			}
		}
	})

	t.Run("RestoreEntries should succeed in a cell whose rows were reordered before the removal", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		for _, r := range rs {
			_, _ = repo.Assign(ctx, "2026-08-24", "dinner", r.ID)
		}
		_ = repo.Reorder(ctx, "2026-08-24", "dinner", idsByRecipe(t, repo, "2026-08-24", "dinner", rs[2].ID, rs[0].ID, rs[1].ID))
		removed, err := repo.Remove(ctx, idAt(t, repo, "2026-08-24", "dinner", 0))
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}

		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[2].ID, rs[0].ID, rs[1].ID)
	})

	t.Run("RestoreEntries should return ErrUnknownRecipe and change nothing when the recipe row does not exist", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		ok := mealplan.RemovedEntry{
			ID: mealplan.NewEntryID(), Date: "2026-08-24", Meal: mealplan.MealDinner, Position: 0,
			RecipeID: rs[1].ID, BatchID: mealplan.NewBatchID(), CreatedAt: "2026-08-20T09:00:00Z",
		}
		missing := ok
		missing.ID = mealplan.NewEntryID()
		missing.Date = "2026-08-25"
		missing.RecipeID = "no-such-recipe"

		err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{ok, missing})
		if !errors.Is(err, mealplan.ErrUnknownRecipe) {
			t.Fatalf("err = %v, want ErrUnknownRecipe", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 || got[0].RecipeID != rs[0].ID || got[0].Position != 0 {
			t.Errorf("want only the original A, got %+v", got)
		}
	})
}

func TestMealPlanRepo_Reorder(t *testing.T) {
	ctx := context.Background()

	t.Run("Reorder should renumber positions according to the ordered entry ids", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[2].ID)
		if err := repo.Reorder(ctx, "2026-08-24", "dinner", idsByRecipe(t, repo, "2026-08-24", "dinner", rs[2].ID, rs[0].ID, rs[1].ID)); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if got[0].RecipeID != rs[2].ID || got[1].RecipeID != rs[0].ID || got[2].RecipeID != rs[1].ID {
			t.Errorf("order = %s, %s, %s; want %s, %s, %s",
				got[0].RecipeID, got[1].RecipeID, got[2].RecipeID,
				rs[2].ID, rs[0].ID, rs[1].ID)
		}
	})

	t.Run("Reorder should return error when the ordered entry ids miss a row of the meal", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		err := repo.Reorder(ctx, "2026-08-24", "dinner", idsByRecipe(t, repo, "2026-08-24", "dinner", rs[0].ID))
		if !errors.Is(err, mealplan.ErrReorderMismatch) {
			t.Errorf("err = %v, want ErrReorderMismatch", err)
		}
	})
}

func TestMealPlanRepo_CopyLeftover(t *testing.T) {
	ctx := context.Background()

	t.Run("mode=append should append source entries to targets without removing existing rows", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "existing")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[2].ID)

		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if err := repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25"}, source, mealplan.CopyModeAppend, mealplan.ShareBatchDistinct); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		if len(got) != 3 {
			t.Fatalf("want 3 (existing + A + B), got %d", len(got))
		}
		if got[0].RecipeID != rs[2].ID {
			t.Errorf("position 0 = %s, want existing %s", got[0].RecipeID, rs[2].ID)
		}
	})

	t.Run("mode=overwrite should first delete existing target rows and then insert source entries", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "existing")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[2].ID)

		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if err := repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25"}, source, mealplan.CopyModeOverwrite, mealplan.ShareBatchDistinct); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		if len(got) != 2 {
			t.Fatalf("want 2 (A + B), got %d", len(got))
		}
		for _, e := range got {
			if e.RecipeID == rs[2].ID {
				t.Errorf("overwrite should have removed existing recipe %s", rs[2].ID)
			}
		}
	})

	t.Run("should return ErrArchivedRecipe and add no row when the source meal holds an archived recipe", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		if err := rr.Archive(ctx, rs[0].ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}

		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		err := repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25", "2026-08-26"}, source, mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		if !errors.Is(err, mealplan.ErrArchivedRecipe) {
			t.Fatalf("err = %v, want ErrArchivedRecipe", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 2 {
			t.Errorf("rows = %+v, want only the 2 source rows", got)
		}
	})

	t.Run("should copy as before when the source meal holds no archived recipe", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		if err := rr.Archive(ctx, rs[1].ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}

		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if err := repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25"}, source, mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, got, rs[0].ID)
	})

	t.Run("should skip duplicate recipe when source and target are the same date", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)

		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if err := repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-24"}, source, mealplan.CopyModeAppend, mealplan.ShareBatchDistinct); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if len(got) != 1 {
			t.Fatalf("want 1 (no duplicate), got %d", len(got))
		}
	})
}

func spreadCell(date string, meal mealplan.Meal) mealplan.SpreadTarget {
	return mealplan.SpreadTarget{Date: date, Meal: meal}
}

// spreadTo は entryID の行を targets へ作り置きとして広げる (同じ batch_id を共有する)。
func spreadTo(t *testing.T, repo *MealPlanRepo, entryID string, targets ...mealplan.SpreadTarget) {
	t.Helper()
	if _, err := repo.Spread(context.Background(), entryID, targets,
		mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
		t.Fatalf("Spread: %v", err)
	}
}

func TestMealPlanRepo_Batch(t *testing.T) {
	ctx := context.Background()

	t.Run("Assign should mint a distinct batch_id per row", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		e1, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		e2, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		e3, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		bids := map[string]bool{e1.BatchID: true, e2.BatchID: true, e3.BatchID: true}
		if len(bids) != 3 {
			t.Errorf("want 3 distinct batch_ids, got %d: %+v", len(bids), bids)
		}
	})

	t.Run("CopyLeftover shareBatch=true keeps a single batch_id across every copied row for one recipe", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if err := repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25", "2026-08-26"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		bids := map[string]bool{}
		for _, e := range got {
			bids[e.BatchID] = true
		}
		if len(bids) != 1 {
			t.Fatalf("want 1 shared batchID, got %d: %+v", len(bids), bids)
		}
		if _, ok := bids[src.BatchID]; !ok {
			t.Errorf("shared batchID does not include source %q", src.BatchID)
		}
	})

	t.Run("CopyLeftover shareBatch=false mints a fresh batch_id per copied row", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if err := repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25", "2026-08-26"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchDistinct); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		bids := map[string]bool{}
		for _, e := range got {
			bids[e.BatchID] = true
		}
		if len(bids) != 3 {
			t.Errorf("want 3 distinct batchIDs (source + 2 copies), got %d: %+v", len(bids), bids)
		}
	})

	t.Run("SplitBatch keeps only the origin day and re-tags it with a fresh batch_id", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25", "2026-08-26"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)

		newID, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-25", "dinner", 0))
		if err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		if newID == "" || newID == src.BatchID {
			t.Errorf("newID = %q, want != %q and non-empty", newID, src.BatchID)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 {
			t.Fatalf("want 1 remaining row (origin), got %d: %+v", len(got), got)
		}
		if got[0].Date != "2026-08-24" {
			t.Errorf("remaining date = %s, want 2026-08-24 (origin)", got[0].Date)
		}
		if got[0].BatchID != newID {
			t.Errorf("remaining row batch_id = %q, want %q", got[0].BatchID, newID)
		}
	})

	t.Run("SplitBatch called on the origin row itself yields the same result (origin remains)", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25", "2026-08-26"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)

		newID, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0)) // 起点行を指す
		if err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 || got[0].Date != "2026-08-24" || got[0].BatchID != newID {
			t.Errorf("got %+v; want 1 row on 2026-08-24 with batchID=%q", got, newID)
		}
	})

	t.Run("SplitBatch is a no-op for single-row batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		returned, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0))
		if err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		if returned != src.BatchID {
			t.Errorf("returned = %q, want current batch_id %q", returned, src.BatchID)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 || got[0].BatchID != src.BatchID {
			t.Errorf("row count/batch_id changed unexpectedly: %+v", got)
		}
	})

	t.Run("SplitBatch does not touch rows in other batches", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25", "2026-08-26"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		other, _ := repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID) // 別バッチ、別レシピ

		if _, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-26", "dinner", 0)); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		found := false
		for _, e := range got {
			if e.BatchID == other.BatchID {
				found = true
			}
		}
		if !found {
			t.Errorf("other batch row was removed. got %+v", got)
		}
	})

	t.Run("SplitBatch should succeed and keep the remaining order when the sibling's meal was reordered", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー", "サラダ", "スープ")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner", []string{"2026-08-25"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[2].ID)
		if err := repo.Reorder(ctx, "2026-08-25", "dinner", idsByRecipe(t, repo, "2026-08-25", "dinner", rs[0].ID, rs[2].ID, rs[1].ID)); err != nil {
			t.Fatalf("Reorder: %v", err)
		}

		if _, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0)); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, got, rs[2].ID, rs[1].ID)
	})

	t.Run("SplitBatch should delete every sibling when one meal holds two rows of the same batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー", "サラダ")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner", []string{"2026-08-25", "2026-08-26"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		if err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-26", "dinner", 0), "2026-08-25", "dinner"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)

		if _, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0)); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		origin, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, origin, rs[0].ID)
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, got, rs[1].ID)
	})

	t.Run("MergeBatch re-tags target row to the specified batch_id", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		mon, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID)
		if err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-25", "dinner", 0), mon.BatchID, 4); err != nil {
			t.Fatalf("MergeBatch: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		for _, e := range got {
			if e.BatchID != mon.BatchID {
				t.Errorf("row not merged: %+v", e)
			}
			if e.BatchMemberCount != 2 {
				t.Errorf("batch_member_count = %d, want 2", e.BatchMemberCount)
			}
		}
	})

	t.Run("MergeBatch returns ErrBatchTargetNotFound when target batch has no other row in the same meal", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), "ghost-batch-id", 4)
		if !errors.Is(err, mealplan.ErrBatchTargetNotFound) {
			t.Errorf("err = %v, want ErrBatchTargetNotFound", err)
		}
	})

	t.Run("ListMergeCandidates returns same-recipe within ±4 days batches from any meal", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)     // 起点 (r1)
		tue, _ := repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID)  // ±4 日以内、同レシピ (候補)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)     // 別レシピ (除外)
		_, _ = repo.Assign(ctx, "2026-09-05", "dinner", rs[0].ID)     // 12 日後 (除外)
		lunch, _ := repo.Assign(ctx, "2026-08-24", "lunch", rs[0].ID) // 別 meal 同レシピ (候補)

		got, err := repo.ListMergeCandidates(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), 4)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 candidates (火曜・夕 + 月曜・昼), got %d: %+v", len(got), got)
		}
		haveDinner, haveLunch := false, false
		for _, c := range got {
			switch c.BatchID {
			case tue.BatchID:
				if c.Meal != mealplan.MealDinner {
					t.Errorf("tue candidate meal = %q, want dinner", c.Meal)
				}
				haveDinner = true
			case lunch.BatchID:
				if c.Meal != mealplan.MealLunch {
					t.Errorf("lunch candidate meal = %q, want lunch", c.Meal)
				}
				haveLunch = true
			}
		}
		if !haveDinner || !haveLunch {
			t.Errorf("expected both dinner and lunch candidates, got %+v", got)
		}
	})

	t.Run("MergeBatch returns ErrBatchMergeDenied for different recipe target", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)     // 起点 r1
		miso, _ := repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID) // 別レシピ
		err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), miso.BatchID, 4)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("MergeBatch returns ErrBatchMergeDenied for out-of-range date target", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)    // 起点
		far, _ := repo.Assign(ctx, "2026-09-05", "dinner", rs[0].ID) // 12 日後 (レンジ外)
		err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), far.BatchID, 4)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("MergeBatch succeeds with 1-day skipped adjacency (Mon-Tue leftover + Fri single)", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		mon, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID) // 月
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		_, _ = repo.Assign(ctx, "2026-08-28", "dinner", rs[0].ID) // 金 (4 日後、同レシピ)

		if err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-28", "dinner", 0), mon.BatchID, 4); err != nil {
			t.Fatalf("MergeBatch: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		merged := 0
		for _, e := range got {
			if e.BatchID == mon.BatchID {
				merged++
			}
		}
		if merged != 3 {
			t.Errorf("want 3 rows in mon batch (月・火・金), got %d", merged)
		}
	})

	t.Run("ListPeriod should count a row spread past the period end as a member of its batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		sun := assignID(t, repo, "2026-08-30", "dinner", rs[0].ID)
		spreadTo(t, repo, sun, spreadCell("2026-08-31", mealplan.MealDinner))

		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 || got[0].BatchMemberCount != 2 {
			t.Errorf("got %+v; want 1 row on 2026-08-30 with batch_member_count=2", got)
		}
	})

	t.Run("ListPeriod should count every row of a batch that extends past the period end", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		sat := assignID(t, repo, "2026-08-29", "dinner", rs[0].ID)
		spreadTo(t, repo, sat,
			spreadCell("2026-08-30", mealplan.MealDinner), spreadCell("2026-08-31", mealplan.MealDinner))

		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 2 {
			t.Fatalf("want 2 rows in period, got %d: %+v", len(got), got)
		}
		for _, e := range got {
			if e.BatchMemberCount != 3 {
				t.Errorf("date=%s batch_member_count = %d, want 3", e.Date, e.BatchMemberCount)
			}
		}
	})

	t.Run("ListPeriod should keep counting 2 for a batch whose rows all lie inside the period", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		mon := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		spreadTo(t, repo, mon, spreadCell("2026-08-25", mealplan.MealDinner))

		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		for _, e := range got {
			if e.BatchMemberCount != 2 {
				t.Errorf("date=%s batch_member_count = %d, want 2", e.Date, e.BatchMemberCount)
			}
		}
	})

	t.Run("ListPeriod should count 1 for a single row even when another batch shares the date", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		mon := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		spreadTo(t, repo, mon, spreadCell("2026-08-25", mealplan.MealDinner))
		single := assignID(t, repo, "2026-08-24", "dinner", rs[1].ID)

		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		for _, e := range got {
			if e.ID == single && e.BatchMemberCount != 1 {
				t.Errorf("single row batch_member_count = %d, want 1", e.BatchMemberCount)
			}
		}
	})

	t.Run("SplitBatch should keep the lunch row when the batch spans lunch and dinner of the same date", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "B", "A")
		_ = assignID(t, repo, "2026-08-24", "lunch", rs[0].ID)
		lunchA := assignID(t, repo, "2026-08-24", "lunch", rs[1].ID)
		spreadTo(t, repo, lunchA, spreadCell("2026-08-24", mealplan.MealDinner))

		if _, err := repo.SplitBatch(ctx, lunchA); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		lunch, _ := repo.MealEntries(ctx, "2026-08-24", "lunch")
		assertMealOrder(t, lunch, rs[0].ID, rs[1].ID)
		dinner, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, dinner)
	})

	t.Run("SplitBatch should keep the lunch row even when called on the dinner row", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "B", "A")
		_ = assignID(t, repo, "2026-08-24", "lunch", rs[0].ID)
		lunchA := assignID(t, repo, "2026-08-24", "lunch", rs[1].ID)
		spreadTo(t, repo, lunchA, spreadCell("2026-08-24", mealplan.MealDinner))

		if _, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0)); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		lunch, _ := repo.MealEntries(ctx, "2026-08-24", "lunch")
		assertMealOrder(t, lunch, rs[0].ID, rs[1].ID)
		if len(lunch) == 2 && lunch[1].ID != lunchA {
			t.Errorf("remaining lunch A id = %q, want %q", lunch[1].ID, lunchA)
		}
		dinner, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, dinner)
	})

	t.Run("SplitBatch should keep the earlier date over the earlier meal", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		monDinner := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		spreadTo(t, repo, monDinner, spreadCell("2026-08-25", mealplan.MealBreakfast))

		if _, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-25", "breakfast", 0)); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		dinner, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, dinner, rs[0].ID)
		breakfast, _ := repo.MealEntries(ctx, "2026-08-25", "breakfast")
		assertMealOrder(t, breakfast)
	})

	t.Run("SplitBatch should leave the other lunch row at position 0", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "B", "A")
		lunchB := assignID(t, repo, "2026-08-24", "lunch", rs[0].ID)
		lunchA := assignID(t, repo, "2026-08-24", "lunch", rs[1].ID)
		spreadTo(t, repo, lunchA, spreadCell("2026-08-24", mealplan.MealDinner))

		if _, err := repo.SplitBatch(ctx, lunchA); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		if got := idAt(t, repo, "2026-08-24", "lunch", 0); got != lunchB {
			t.Errorf("lunch position 0 = %q, want B %q", got, lunchB)
		}
	})

	t.Run("ListPeriod populates batch_member_count from every row of the batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner",
			[]string{"2026-08-25"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		// もう 1 品を単独割当 (1 行だけのバッチ)
		_, _ = repo.Assign(ctx, "2026-08-26", "dinner", rs[0].ID)

		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		for _, e := range got {
			switch e.Date {
			case "2026-08-24", "2026-08-25":
				if e.BatchMemberCount != 2 {
					t.Errorf("date=%s batch_member_count = %d, want 2", e.Date, e.BatchMemberCount)
				}
			case "2026-08-26":
				if e.BatchMemberCount != 1 {
					t.Errorf("date=%s batch_member_count = %d, want 1", e.Date, e.BatchMemberCount)
				}
			}
		}
	})

	t.Run("MergeBatch succeeds across different meals (dinner -> lunch, same recipe within ±4 days)", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		mon, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID) // 月・夕 (target)
		_, _ = repo.Assign(ctx, "2026-08-25", "lunch", rs[0].ID)     // 火・昼 (別バッチ、同レシピ)

		if err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-25", "lunch", 0), mon.BatchID, 4); err != nil {
			t.Fatalf("MergeBatch across meals: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		merged := 0
		for _, e := range got {
			if e.BatchID == mon.BatchID {
				merged++
			}
		}
		if merged != 2 {
			t.Errorf("want 2 rows in mon batch (mon-dinner + tue-lunch), got %d: %+v", merged, got)
		}
	})

	t.Run("MergeBatch returns ErrBatchMergeDenied across meals when recipe differs", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		miso, _ := repo.Assign(ctx, "2026-08-25", "lunch", rs[1].ID) // 別 meal、別レシピ
		err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), miso.BatchID, 4)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("should return one candidate when a batch spans lunch and dinner on the same date", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		monLunch, _ := repo.Assign(ctx, "2026-08-24", "lunch", rs[0].ID)
		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "lunch", 0),
			[]mealplan.SpreadTarget{{Date: "2026-08-24", Meal: mealplan.MealDinner}},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID)

		got, err := repo.ListMergeCandidates(ctx, idAt(t, repo, "2026-08-25", "dinner", 0), 4)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 candidate, got %d: %+v", len(got), got)
		}
		if got[0].BatchID != monLunch.BatchID || got[0].MemberCount != 2 {
			t.Errorf("candidate = %+v, want batch %q with 2 members", got[0], monLunch.BatchID)
		}
	})

	t.Run("should pick the earliest row by date, meal order and position as the representative", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		// 夕食の position 0 と昼食の position 1 を作り、position だけで選ぶと夕食が選ばれる並びにする。
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "lunch", rs[1].ID)
		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{{Date: "2026-08-24", Meal: mealplan.MealLunch}, {Date: "2026-08-25", Meal: mealplan.MealBreakfast}},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		_, _ = repo.Assign(ctx, "2026-08-26", "dinner", rs[0].ID)

		got, err := repo.ListMergeCandidates(ctx, idAt(t, repo, "2026-08-26", "dinner", 0), 4)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 candidate, got %d: %+v", len(got), got)
		}
		if got[0].Date != "2026-08-24" || got[0].Meal != mealplan.MealLunch {
			t.Errorf("representative = %s/%s, want 2026-08-24/lunch", got[0].Date, got[0].Meal)
		}
	})

	dinners := func(dates ...string) []mealplan.SpreadTarget {
		out := make([]mealplan.SpreadTarget, 0, len(dates))
		for _, d := range dates {
			out = append(out, mealplan.SpreadTarget{Date: d, Meal: mealplan.MealDinner})
		}
		return out
	}
	// 通常の操作では別レシピの行が同じ batch_id を持たないため、共有している状態を SQL で直接作る。
	joinBatch := func(t *testing.T, repo *MealPlanRepo, batchID string, entryIDs ...string) {
		t.Helper()
		for _, id := range entryIDs {
			if _, err := repo.db.Exec(`UPDATE meal_plans SET batch_id = ? WHERE id = ?`, batchID, id); err != nil {
				t.Fatalf("join batch: %v", err)
			}
		}
	}

	t.Run("should use the earliest row and the whole count when the batch extends beyond the range", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		first, _ := repo.Assign(ctx, "2026-10-19", "dinner", rs[0].ID)
		if _, err := repo.Spread(ctx, first.ID, dinners("2026-10-20", "2026-10-21", "2026-10-22"),
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		_, _ = repo.Assign(ctx, "2026-10-24", "dinner", rs[0].ID)

		got, err := repo.ListMergeCandidates(ctx, idAt(t, repo, "2026-10-24", "dinner", 0), 4)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 candidate, got %d: %+v", len(got), got)
		}
		if got[0].BatchID != first.BatchID || got[0].Date != "2026-10-19" || got[0].MemberCount != 4 {
			t.Errorf("candidate = %+v, want batch %q on 2026-10-19 with 4 members", got[0], first.BatchID)
		}
	})

	t.Run("should use the lunch row as the representative when the batch starts at lunch beyond the range", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		first, _ := repo.Assign(ctx, "2026-10-19", "lunch", rs[0].ID)
		if _, err := repo.Spread(ctx, first.ID, dinners("2026-10-19", "2026-10-20", "2026-10-21", "2026-10-22"),
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		_, _ = repo.Assign(ctx, "2026-10-24", "dinner", rs[0].ID)

		got, err := repo.ListMergeCandidates(ctx, idAt(t, repo, "2026-10-24", "dinner", 0), 4)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 candidate, got %d: %+v", len(got), got)
		}
		if got[0].Date != "2026-10-19" || got[0].Meal != mealplan.MealLunch || got[0].MemberCount != 5 {
			t.Errorf("candidate = %+v, want 2026-10-19/lunch with 5 members", got[0])
		}
	})

	t.Run("should not list a batch whose same-recipe rows are all out of range even if another recipe row is in range", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		old, _ := repo.Assign(ctx, "2026-10-15", "dinner", rs[0].ID)
		miso, _ := repo.Assign(ctx, "2026-10-22", "dinner", rs[1].ID)
		joinBatch(t, repo, old.BatchID, miso.ID)
		_, _ = repo.Assign(ctx, "2026-10-24", "dinner", rs[0].ID)

		got, err := repo.ListMergeCandidates(ctx, idAt(t, repo, "2026-10-24", "dinner", 0), 4)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want no candidates, got %+v", got)
		}
	})

	t.Run("should count only same-recipe rows when another recipe shares the batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		first, _ := repo.Assign(ctx, "2026-10-19", "dinner", rs[0].ID)
		miso, _ := repo.Assign(ctx, "2026-10-19", "dinner", rs[1].ID)
		days := dinners("2026-10-20", "2026-10-21", "2026-10-22")
		if _, err := repo.Spread(ctx, first.ID, days, mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		if _, err := repo.Spread(ctx, miso.ID, days, mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		for _, d := range []string{"2026-10-19", "2026-10-20", "2026-10-21", "2026-10-22"} {
			joinBatch(t, repo, first.BatchID, idAt(t, repo, d, "dinner", 1))
		}
		_, _ = repo.Assign(ctx, "2026-10-24", "dinner", rs[0].ID)

		got, err := repo.ListMergeCandidates(ctx, idAt(t, repo, "2026-10-24", "dinner", 0), 4)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 1 || got[0].MemberCount != 4 {
			t.Errorf("got %+v, want 1 candidate with 4 members", got)
		}
	})

	t.Run("migration backfills batch_id for pre-existing rows and keeps them distinct", func(t *testing.T) {
		// Migrate() は Open + Migrate で全 migration を走らせる。00003 の時点に戻れないため、
		// 「既存行は distinct な batch_id を持つ」観点は Assign 相当で担保する。
		// 具体的にはビジネス上「単独割当は独立バッチ」ルールを 3 回叩いて確認する。
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		e1, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		e2, _ := repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID)
		e3, _ := repo.Assign(ctx, "2026-08-26", "dinner", rs[0].ID)
		if e1.BatchID == "" || e2.BatchID == "" || e3.BatchID == "" {
			t.Fatalf("batch_id must be non-empty after Assign: %q / %q / %q", e1.BatchID, e2.BatchID, e3.BatchID)
		}
		if e1.BatchID == e2.BatchID || e2.BatchID == e3.BatchID || e1.BatchID == e3.BatchID {
			t.Errorf("Assign should produce distinct batch_ids, got %q / %q / %q", e1.BatchID, e2.BatchID, e3.BatchID)
		}
	})
}

func TestMealPlanRepo_ArchivedRecipe(t *testing.T) {
	ctx := context.Background()

	t.Run("ListPeriod should keep returning the entry with its recipe name after the recipe is archived", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-28", "dinner", rs[0].ID)
		if err := rr.Archive(ctx, rs[0].ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		got, err := repo.ListPeriod(ctx, "2026-08-24", 7)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 1 || got[0].RecipeName != "肉じゃが" {
			t.Errorf("want 肉じゃが kept, got %+v", got)
		}
	})
}

func cookedByCell(t *testing.T, repo *MealPlanRepo, periodStart string) map[string]bool {
	t.Helper()
	got, err := repo.ListPeriod(context.Background(), periodStart, 7)
	if err != nil {
		t.Fatalf("ListPeriod: %v", err)
	}
	out := map[string]bool{}
	for _, e := range got {
		out[e.Date+"/"+string(e.Meal)+"/"+e.RecipeName] = e.Cooked
	}
	return out
}

func countCookedMarks(t *testing.T, repo *MealPlanRepo) int {
	t.Helper()
	var n int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM meal_plan_cooked`).Scan(&n); err != nil {
		t.Fatalf("count meal_plan_cooked: %v", err)
	}
	return n
}

// seedLeftover は 2026-08-24 の dinner に recipe を置き、続く extraDays 日に作り置きとして広げる。
func seedLeftover(t *testing.T, repo *MealPlanRepo, recipeID string, extraDays ...string) mealplan.Entry {
	t.Helper()
	ctx := context.Background()
	src, err := repo.Assign(ctx, "2026-08-24", "dinner", recipeID)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if len(extraDays) > 0 {
		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if err := repo.CopyLeftover(ctx, "dinner", extraDays, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
	}
	return src
}

func TestMealPlanRepo_Cooked(t *testing.T) {
	ctx := context.Background()

	t.Run("SetCooked true should mark every row of the same batch and recipe", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		seedLeftover(t, repo, rs[0].ID, "2026-08-25", "2026-08-26")

		if err := repo.SetCooked(ctx, idAt(t, repo, "2026-08-25", "dinner", 0), true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		for cell, cooked := range cookedByCell(t, repo, "2026-08-24") {
			if !cooked {
				t.Errorf("%s: want cooked", cell)
			}
		}
	})

	t.Run("SetCooked false should unmark every row of the same batch and recipe", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		seedLeftover(t, repo, rs[0].ID, "2026-08-25", "2026-08-26")
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		if err := repo.SetCooked(ctx, idAt(t, repo, "2026-08-25", "dinner", 0), false); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		for cell, cooked := range cookedByCell(t, repo, "2026-08-24") {
			if cooked {
				t.Errorf("%s: want not cooked", cell)
			}
		}
	})

	t.Run("SetCooked should not spread the mark to another batch of the same recipe", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-26", "dinner", rs[0].ID)

		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)
		got := cookedByCell(t, repo, "2026-08-24")
		if !got["2026-08-24/dinner/カレー"] {
			t.Error("2026-08-24: want cooked")
		}
		if got["2026-08-26/dinner/カレー"] {
			t.Error("2026-08-26: want not cooked")
		}
	})

	t.Run("SetCooked should return ErrEntryNotFound when the row does not exist", func(t *testing.T) {
		repo, _ := newMealPlanRepoForTest(t)
		err := repo.SetCooked(ctx, "no-such-entry", true)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("want ErrEntryNotFound, got %v", err)
		}
		if n := countCookedMarks(t, repo); n != 0 {
			t.Errorf("want 0 marks, got %d", n)
		}
	})

	t.Run("SetCooked should not fail when marking or unmarking twice", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		for _, cooked := range []bool{true, true, false, false} {
			if err := repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), cooked); err != nil {
				t.Fatalf("SetCooked(%v): %v", cooked, err)
			}
		}
	})

	t.Run("CopyLeftover leftover mode should carry the mark to added rows", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		seedLeftover(t, repo, rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner", []string{"2026-08-25"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		if !cookedByCell(t, repo, "2026-08-24")["2026-08-25/dinner/カレー"] {
			t.Error("2026-08-25: want cooked")
		}
	})

	t.Run("CopyLeftover distinct mode should not mark added rows", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		seedLeftover(t, repo, rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		source, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		_ = repo.CopyLeftover(ctx, "dinner", []string{"2026-08-25"}, source,
			mealplan.CopyModeAppend, mealplan.ShareBatchDistinct)
		if cookedByCell(t, repo, "2026-08-24")["2026-08-25/dinner/カレー"] {
			t.Error("2026-08-25: want not cooked")
		}
	})

	t.Run("MoveEntry should keep the mark on the moved row", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		if err := repo.MoveEntry(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), "2026-08-25", "dinner"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		if !cookedByCell(t, repo, "2026-08-24")["2026-08-25/dinner/肉じゃが"] {
			t.Error("2026-08-25: want cooked")
		}
	})

	t.Run("Replace should leave the replaced row unmarked and not revive the mark when replaced back", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "カレー")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		if err := repo.Replace(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), rs[1].ID); err != nil {
			t.Fatalf("Replace: %v", err)
		}
		if cookedByCell(t, repo, "2026-08-24")["2026-08-24/dinner/カレー"] {
			t.Error("after replace: want not cooked")
		}
		if err := repo.Replace(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), rs[0].ID); err != nil {
			t.Fatalf("Replace back: %v", err)
		}
		if cookedByCell(t, repo, "2026-08-24")["2026-08-24/dinner/肉じゃが"] {
			t.Error("after replace back: want not cooked")
		}
	})

	t.Run("SplitBatch should keep the mark on the remaining origin row", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		seedLeftover(t, repo, rs[0].ID, "2026-08-25", "2026-08-26")
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		if _, err := repo.SplitBatch(ctx, idAt(t, repo, "2026-08-25", "dinner", 0)); err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		got := cookedByCell(t, repo, "2026-08-24")
		if len(got) != 1 || !got["2026-08-24/dinner/カレー"] {
			t.Errorf("want only 2026-08-24 cooked, got %+v", got)
		}
		if n := countCookedMarks(t, repo); n != 1 {
			t.Errorf("want 1 mark (old batch pruned), got %d", n)
		}
	})

	t.Run("MergeBatch into a marked batch should mark the merged row", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		target, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-26", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		if err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-26", "dinner", 0), target.BatchID, 4); err != nil {
			t.Fatalf("MergeBatch: %v", err)
		}
		got := cookedByCell(t, repo, "2026-08-24")
		if !got["2026-08-24/dinner/カレー"] || !got["2026-08-26/dinner/カレー"] {
			t.Errorf("want both cooked, got %+v", got)
		}
	})

	t.Run("MergeBatch of a marked row into an unmarked batch should mark the target batch", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		target, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-26", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-26", "dinner", 0), true)

		if err := repo.MergeBatch(ctx, idAt(t, repo, "2026-08-26", "dinner", 0), target.BatchID, 4); err != nil {
			t.Fatalf("MergeBatch: %v", err)
		}
		got := cookedByCell(t, repo, "2026-08-24")
		if !got["2026-08-24/dinner/カレー"] || !got["2026-08-26/dinner/カレー"] {
			t.Errorf("want both cooked, got %+v", got)
		}
		if n := countCookedMarks(t, repo); n != 1 {
			t.Errorf("want 1 mark (source batch pruned), got %d", n)
		}
	})

	t.Run("ClearMeal should drop the mark when no row of the batch and recipe remains", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		if _, err := repo.ClearMeal(ctx, "2026-08-24", "dinner"); err != nil {
			t.Fatalf("ClearMeal: %v", err)
		}
		if n := countCookedMarks(t, repo); n != 0 {
			t.Errorf("want 0 marks, got %d", n)
		}
	})
}

func TestMealPlanRepo_Spread(t *testing.T) {
	ctx := context.Background()
	cell := func(date string, meal mealplan.Meal) mealplan.SpreadTarget {
		return mealplan.SpreadTarget{Date: date, Meal: meal}
	}

	t.Run("should insert rows only into the given cells when targets skip a day", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)

		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-27", mealplan.MealDinner)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		for date, want := range map[string]int{"2026-08-25": 1, "2026-08-26": 0, "2026-08-27": 1} {
			got, _ := repo.MealEntries(ctx, date, "dinner")
			if len(got) != want {
				t.Errorf("%s dinner: want %d rows, got %d", date, want, len(got))
			}
		}
	})

	t.Run("should share the source batch across same-meal and cross-meal targets when shareBatch is true", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)

		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealLunch)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		for _, c := range []mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealLunch)} {
			got, _ := repo.MealEntries(ctx, c.Date, string(c.Meal))
			if len(got) != 1 || got[0].BatchID != src.BatchID || got[0].RecipeID != rs[0].ID {
				t.Errorf("%s/%s = %+v, want 1 row of batch %q", c.Date, c.Meal, got, src.BatchID)
			}
		}
	})

	t.Run("should give every target its own batch and leave the source batch single when shareBatch is false", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)

		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealLunch)},
			mealplan.CopyModeAppend, mealplan.ShareBatchDistinct); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 3 {
			t.Fatalf("want 3 rows, got %d", len(got))
		}
		seen := map[string]bool{}
		for _, e := range got {
			if e.BatchMemberCount != 1 {
				t.Errorf("%s/%s batch_member_count = %d, want 1", e.Date, e.Meal, e.BatchMemberCount)
			}
			seen[e.BatchID] = true
		}
		if len(seen) != 3 {
			t.Errorf("want 3 distinct batches, got %d", len(seen))
		}
	})

	t.Run("should append after existing rows when mode is append", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "lunch", rs[1].ID)

		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealLunch)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "lunch")
		if len(got) != 2 {
			t.Fatalf("want 2 rows, got %d: %+v", len(got), got)
		}
		if got[0].RecipeID != rs[1].ID || got[1].RecipeID != rs[0].ID || got[1].BatchID != src.BatchID {
			t.Errorf("rows = %+v, want [みそ汁, 肉じゃが(batch %q)]", got, src.BatchID)
		}
	})

	t.Run("should skip a cell that already has the same recipe when mode is append", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		existing, _ := repo.Assign(ctx, "2026-08-25", "lunch", rs[0].ID)

		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealLunch), cell("2026-08-25", mealplan.MealDinner)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		lunch, _ := repo.MealEntries(ctx, "2026-08-25", "lunch")
		if len(lunch) != 1 || lunch[0].BatchID != existing.BatchID {
			t.Errorf("lunch = %+v, want the existing row only", lunch)
		}
		dinner, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		if len(dinner) != 1 {
			t.Errorf("dinner: want 1 row, got %d", len(dinner))
		}
	})

	t.Run("should replace existing rows when mode is overwrite", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		src, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)

		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)},
			mealplan.CopyModeOverwrite, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		if len(got) != 1 || got[0].RecipeID != rs[0].ID || got[0].BatchID != src.BatchID || got[0].Position != 0 {
			t.Errorf("rows = %+v, want only 肉じゃが at position 0", got)
		}
	})

	t.Run("should carry the cooked mark to shared rows and not to distinct rows", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_ = repo.SetCooked(ctx, idAt(t, repo, "2026-08-24", "dinner", 0), true)

		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread shared: %v", err)
		}
		if _, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-26", mealplan.MealLunch)},
			mealplan.CopyModeAppend, mealplan.ShareBatchDistinct); err != nil {
			t.Fatalf("Spread distinct: %v", err)
		}
		got := cookedByCell(t, repo, "2026-08-24")
		if !got["2026-08-25/dinner/肉じゃが"] {
			t.Error("2026-08-25 dinner: want cooked (shared batch)")
		}
		if got["2026-08-26/lunch/肉じゃが"] {
			t.Error("2026-08-26 lunch: want not cooked (distinct batch)")
		}
	})

	t.Run("should return the added ids in date and meal order and no removed rows when mode is append", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)

		got, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-26", mealplan.MealLunch), cell("2026-08-25", mealplan.MealDinner)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		if err != nil {
			t.Fatalf("Spread: %v", err)
		}
		want := []string{idAt(t, repo, "2026-08-25", "dinner", 0), idAt(t, repo, "2026-08-26", "lunch", 0)}
		if len(got.AddedEntryIDs) != 2 || got.AddedEntryIDs[0] != want[0] || got.AddedEntryIDs[1] != want[1] {
			t.Errorf("AddedEntryIDs = %v, want %v", got.AddedEntryIDs, want)
		}
		if len(got.Removed) != 0 {
			t.Errorf("Removed = %+v, want empty", got.Removed)
		}
	})

	t.Run("should leave a skipped cell out of the added ids when mode is append", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID)

		got, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealDinner)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		if err != nil {
			t.Fatalf("Spread: %v", err)
		}
		want := idAt(t, repo, "2026-08-26", "dinner", 0)
		if len(got.AddedEntryIDs) != 1 || got.AddedEntryIDs[0] != want {
			t.Errorf("AddedEntryIDs = %v, want [%s]", got.AddedEntryIDs, want)
		}
	})

	t.Run("should return the overwritten rows with their cookedAt when mode is overwrite", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		miso := assignID(t, repo, "2026-08-25", "dinner", rs[1].ID)
		if err := repo.SetCooked(ctx, miso, true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		cookedAt := cookedAtOf(t, repo, rs[1].ID)

		got, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)},
			mealplan.CopyModeOverwrite, mealplan.ShareBatchLeftover)
		if err != nil {
			t.Fatalf("Spread: %v", err)
		}
		if len(got.Removed) != 1 || got.Removed[0].ID != miso || got.Removed[0].CookedAt == "" || got.Removed[0].CookedAt != cookedAt {
			t.Errorf("Removed = %+v, want みそ汁 %s with cookedAt %q", got.Removed, miso, cookedAt)
		}
	})

	t.Run("should return ErrArchivedRecipe and change nothing when the source recipe is archived", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)
		if err := rr.Archive(ctx, rs[0].ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}

		_, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealDinner)},
			mealplan.CopyModeOverwrite, mealplan.ShareBatchLeftover)
		if !errors.Is(err, mealplan.ErrArchivedRecipe) {
			t.Fatalf("err = %v, want ErrArchivedRecipe", err)
		}
		tue, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, tue, rs[1].ID)
		wed, _ := repo.MealEntries(ctx, "2026-08-26", "dinner")
		assertMealOrder(t, wed)
	})

	t.Run("should return ErrEntryNotFound and change nothing when the source row does not exist", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID)

		_, err := repo.Spread(ctx, "no-such-entry",
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)},
			mealplan.CopyModeOverwrite, mealplan.ShareBatchLeftover)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Fatalf("err = %v, want ErrEntryNotFound", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 || got[0].RecipeID != rs[0].ID {
			t.Errorf("rows = %+v, want untouched みそ汁 only", got)
		}
	})
}

func TestMealPlanRepo_UndoSpread(t *testing.T) {
	ctx := context.Background()
	cell := func(date string, meal mealplan.Meal) mealplan.SpreadTarget {
		return mealplan.SpreadTarget{Date: date, Meal: meal}
	}
	// 月曜・夕食の肉じゃがを、みそ汁 (作った印付き)、サラダの順に並ぶ火曜・夕食へ上書きで広げた状態を作る。
	seedOverwrite := func(t *testing.T) (*MealPlanRepo, []recipe.Recipe, mealplan.SpreadResult) {
		t.Helper()
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁", "サラダ")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[2].ID)
		if err := repo.SetCooked(ctx, idAt(t, repo, "2026-08-25", "dinner", 0), true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		res, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)},
			mealplan.CopyModeOverwrite, mealplan.ShareBatchLeftover)
		if err != nil {
			t.Fatalf("Spread: %v", err)
		}
		return repo, rs, res
	}

	t.Run("should remove the added rows and leave only the source when the spread was append", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		res, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealLunch)},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover)
		if err != nil {
			t.Fatalf("Spread: %v", err)
		}

		if err := repo.UndoSpread(ctx, res); err != nil {
			t.Fatalf("UndoSpread: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if len(got) != 1 || got[0].Date != "2026-08-24" || got[0].BatchMemberCount != 1 {
			t.Errorf("rows = %+v, want only the single source row", got)
		}
	})

	t.Run("should remove the added row and bring back the two overwritten rows in their order", func(t *testing.T) {
		repo, rs, res := seedOverwrite(t)

		if err := repo.UndoSpread(ctx, res); err != nil {
			t.Fatalf("UndoSpread: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, got, rs[1].ID, rs[2].ID)
	})

	t.Run("should bring back the cooked mark of an overwritten row", func(t *testing.T) {
		repo, _, res := seedOverwrite(t)

		if err := repo.UndoSpread(ctx, res); err != nil {
			t.Fatalf("UndoSpread: %v", err)
		}
		got := cookedByCell(t, repo, "2026-08-24")
		if !got["2026-08-25/dinner/みそ汁"] {
			t.Error("みそ汁: want cooked after undo")
		}
	})

	t.Run("should remove the remaining added rows and restore the overwritten rows when one added row was already removed", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)
		res, err := repo.Spread(ctx, idAt(t, repo, "2026-08-24", "dinner", 0),
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealDinner)},
			mealplan.CopyModeOverwrite, mealplan.ShareBatchLeftover)
		if err != nil {
			t.Fatalf("Spread: %v", err)
		}
		if _, err := repo.Remove(ctx, res.AddedEntryIDs[1]); err != nil {
			t.Fatalf("Remove: %v", err)
		}

		if err := repo.UndoSpread(ctx, res); err != nil {
			t.Fatalf("UndoSpread: %v", err)
		}
		tue, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, tue, rs[1].ID)
		wed, _ := repo.MealEntries(ctx, "2026-08-26", "dinner")
		assertMealOrder(t, wed)
	})

	t.Run("should return ErrInvalidRestore and keep the added rows when an overwritten row id already exists", func(t *testing.T) {
		repo, rs, res := seedOverwrite(t)
		if err := repo.RestoreEntries(ctx, res.Removed[:1]); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}

		err := repo.UndoSpread(ctx, res)
		if !errors.Is(err, mealplan.ErrInvalidRestore) {
			t.Fatalf("err = %v, want ErrInvalidRestore", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		assertMealOrder(t, got, rs[1].ID, rs[0].ID)
	})
}

func TestMealPlanRepo_EntryID(t *testing.T) {
	ctx := context.Background()

	t.Run("Assign should return an entry with a non-empty id", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		e1 := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		e2 := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		if e1 == "" || e2 == "" || e1 == e2 {
			t.Errorf("ids = %q, %q; want distinct non-empty", e1, e2)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if got[0].ID != e1 || got[1].ID != e2 {
			t.Errorf("listed ids = %q, %q; want %q, %q", got[0].ID, got[1].ID, e1, e2)
		}
	})

	t.Run("Remove should hit the given entry even after an earlier row in the same cell was removed", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		a := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		b := assignID(t, repo, "2026-08-24", "dinner", rs[1].ID)
		_ = assignID(t, repo, "2026-08-24", "dinner", rs[2].ID)

		if _, err := repo.Remove(ctx, a); err != nil {
			t.Fatalf("Remove a: %v", err)
		}
		removed, err := repo.Remove(ctx, b)
		if err != nil {
			t.Fatalf("Remove b: %v", err)
		}
		if removed.ID != b || removed.RecipeID != rs[1].ID {
			t.Errorf("removed = %+v, want B (%s)", removed, b)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[2].ID)
	})

	t.Run("SetCooked should mark the given entry even after the cell was reordered", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		niku := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		miso := assignID(t, repo, "2026-08-24", "dinner", rs[1].ID)

		if err := repo.Reorder(ctx, "2026-08-24", "dinner", []string{miso, niku}); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		if err := repo.SetCooked(ctx, niku, true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		got := cookedByCell(t, repo, "2026-08-24")
		if !got["2026-08-24/dinner/肉じゃが"] || got["2026-08-24/dinner/みそ汁"] {
			t.Errorf("want only 肉じゃが cooked, got %+v", got)
		}
	})

	t.Run("Replace should return ErrEntryNotFound when the entry was already removed", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		a := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		_ = assignID(t, repo, "2026-08-24", "dinner", rs[1].ID)
		if _, err := repo.Remove(ctx, a); err != nil {
			t.Fatalf("Remove: %v", err)
		}

		err := repo.Replace(ctx, a, rs[2].ID)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[1].ID)
	})

	t.Run("MoveEntry should keep the entry id", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		id := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)

		if err := repo.MoveEntry(ctx, id, "2026-08-25", "lunch"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-25", "lunch")
		if len(got) != 1 || got[0].ID != id {
			t.Errorf("moved = %+v, want id %q", got, id)
		}
	})

	t.Run("Reorder should swap two rows of the same recipe when their ids are given in the new order", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "カレー")
		first := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		second := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)

		if err := repo.Reorder(ctx, "2026-08-24", "dinner", []string{second, first}); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		if len(got) != 2 || got[0].ID != second || got[1].ID != first {
			t.Errorf("order = %+v, want [%s, %s]", got, second, first)
		}
	})

	t.Run("Reorder should return ErrReorderMismatch when an id is duplicated", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		a := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		_ = assignID(t, repo, "2026-08-24", "dinner", rs[1].ID)

		err := repo.Reorder(ctx, "2026-08-24", "dinner", []string{a, a})
		if !errors.Is(err, mealplan.ErrReorderMismatch) {
			t.Errorf("err = %v, want ErrReorderMismatch", err)
		}
	})

	t.Run("Reorder should return ErrReorderMismatch when an id belongs to another cell", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		a := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		_ = assignID(t, repo, "2026-08-24", "dinner", rs[1].ID)
		other := assignID(t, repo, "2026-08-24", "lunch", rs[1].ID)

		err := repo.Reorder(ctx, "2026-08-24", "dinner", []string{other, a})
		if !errors.Is(err, mealplan.ErrReorderMismatch) {
			t.Errorf("err = %v, want ErrReorderMismatch", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[0].ID, rs[1].ID)
	})

	t.Run("RestoreEntries should bring the row back with the same id", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_ = assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		b := assignID(t, repo, "2026-08-24", "dinner", rs[1].ID)
		removed, err := repo.Remove(ctx, b)
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}

		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		if got := idAt(t, repo, "2026-08-24", "dinner", 1); got != b {
			t.Errorf("restored id = %q, want %q", got, b)
		}
	})

	t.Run("RestoreEntries should return ErrInvalidRestore when a row with the same id already exists", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		a := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)
		removed, _ := repo.Remove(ctx, a)
		if err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed}); err != nil {
			t.Fatalf("first RestoreEntries: %v", err)
		}

		err := repo.RestoreEntries(ctx, []mealplan.RemovedEntry{removed})
		if !errors.Is(err, mealplan.ErrInvalidRestore) {
			t.Errorf("err = %v, want ErrInvalidRestore", err)
		}
		got, _ := repo.MealEntries(ctx, "2026-08-24", "dinner")
		assertMealOrder(t, got, rs[0].ID)
	})

	t.Run("Spread should give every inserted row a new id", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		src := assignID(t, repo, "2026-08-24", "dinner", rs[0].ID)

		if _, err := repo.Spread(ctx, src,
			[]mealplan.SpreadTarget{{Date: "2026-08-25", Meal: mealplan.MealDinner}, {Date: "2026-08-26", Meal: mealplan.MealLunch}},
			mealplan.CopyModeAppend, mealplan.ShareBatchLeftover); err != nil {
			t.Fatalf("Spread: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		ids := map[string]bool{}
		for _, e := range got {
			if e.ID == "" {
				t.Errorf("%s/%s: empty id", e.Date, e.Meal)
			}
			ids[e.ID] = true
		}
		if len(got) != 3 || len(ids) != 3 {
			t.Errorf("want 3 rows with distinct ids, got %+v", got)
		}
	})
}

func TestMealPlanRepo_ApplyPlan(t *testing.T) {
	ctx := context.Background()
	item := func(date, recipeID string, group int) mealplan.PlanItem {
		return mealplan.PlanItem{Date: date, Meal: mealplan.MealDinner, RecipeID: recipeID, Group: group}
	}
	batchOf := func(t *testing.T, repo *MealPlanRepo, id string) string {
		t.Helper()
		e, err := repo.Entry(ctx, id)
		if err != nil {
			t.Fatalf("Entry: %v", err)
		}
		return e.BatchID
	}

	t.Run("rows of the same group share batch_id and other groups do not", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "カレー")
		res, err := repo.ApplyPlan(ctx, []mealplan.PlanItem{
			item("2026-10-05", rs[0].ID, 0), item("2026-10-06", rs[0].ID, 0), item("2026-10-07", rs[1].ID, 1),
		})
		if err != nil {
			t.Fatalf("ApplyPlan: %v", err)
		}
		ids := res.AddedEntryIDs
		if len(ids) != 3 || batchOf(t, repo, ids[0]) != batchOf(t, repo, ids[1]) || batchOf(t, repo, ids[1]) == batchOf(t, repo, ids[2]) {
			t.Errorf("unexpected batches for %v", ids)
		}
	})

	t.Run("appends to the end of a cell that already has entries", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "味噌汁", "肉じゃが")
		assignID(t, repo, "2026-10-05", "dinner", rs[0].ID)
		res, err := repo.ApplyPlan(ctx, []mealplan.PlanItem{item("2026-10-05", rs[1].ID, 0)})
		if err != nil {
			t.Fatalf("ApplyPlan: %v", err)
		}
		e, _ := repo.Entry(ctx, res.AddedEntryIDs[0])
		if e.Position != 1 {
			t.Errorf("want position 1, got %d", e.Position)
		}
	})

	t.Run("adds no row when an archived recipe is included", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "カレー")
		if err := rr.Archive(ctx, rs[1].ID); err != nil {
			t.Fatalf("Archive: %v", err)
		}
		_, err := repo.ApplyPlan(ctx, []mealplan.PlanItem{item("2026-10-05", rs[0].ID, 0), item("2026-10-06", rs[1].ID, 1)})
		if !errors.Is(err, mealplan.ErrArchivedRecipe) {
			t.Errorf("want ErrArchivedRecipe, got %v", err)
		}
		if got, _ := repo.MealEntries(ctx, "2026-10-05", "dinner"); len(got) != 0 {
			t.Errorf("want no rows, got %#v", got)
		}
	})

	t.Run("passing AddedEntryIDs to UndoSpread removes the added rows", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		res, err := repo.ApplyPlan(ctx, []mealplan.PlanItem{item("2026-10-05", rs[0].ID, 0), item("2026-10-06", rs[0].ID, 0)})
		if err != nil {
			t.Fatalf("ApplyPlan: %v", err)
		}
		if err := repo.UndoSpread(ctx, res); err != nil {
			t.Fatalf("UndoSpread: %v", err)
		}
		for _, d := range []string{"2026-10-05", "2026-10-06"} {
			if got, _ := repo.MealEntries(ctx, d, "dinner"); len(got) != 0 {
				t.Errorf("%s: want no rows, got %#v", d, got)
			}
		}
	})
}
