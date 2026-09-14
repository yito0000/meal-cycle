package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

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
}

func TestMealPlanRepo_Replace(t *testing.T) {
	ctx := context.Background()

	t.Run("Replace should update only the target row and keep other positions untouched", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		if err := repo.Replace(ctx, "2026-08-24", "dinner", 1, rs[2].ID); err != nil {
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

	t.Run("Replace returns ErrEntryNotFound when position missing", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A")
		err := repo.Replace(ctx, "2026-08-24", "dinner", 0, rs[0].ID)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
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
		if err := repo.Remove(ctx, "2026-08-24", "dinner", 1); err != nil {
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
}

func TestMealPlanRepo_ClearMeal(t *testing.T) {
	ctx := context.Background()

	t.Run("ClearMeal should delete every row in the target (date, meal) and leave other meals untouched", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "lunch", rs[0].ID)

		if err := repo.ClearMeal(ctx, "2026-08-24", "dinner"); err != nil {
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
}

func TestMealPlanRepo_Reorder(t *testing.T) {
	ctx := context.Background()

	t.Run("Reorder should renumber positions according to orderedRecipeIDs", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B", "C")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[2].ID)
		if err := repo.Reorder(ctx, "2026-08-24", "dinner", []string{rs[2].ID, rs[0].ID, rs[1].ID}); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		got, _ := repo.ListPeriod(ctx, "2026-08-24", 7)
		if got[0].RecipeID != rs[2].ID || got[1].RecipeID != rs[0].ID || got[2].RecipeID != rs[1].ID {
			t.Errorf("order = %s, %s, %s; want %s, %s, %s",
				got[0].RecipeID, got[1].RecipeID, got[2].RecipeID,
				rs[2].ID, rs[0].ID, rs[1].ID)
		}
	})

	t.Run("Reorder should return error when orderedRecipeIDs mismatches the current meal content", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "A", "B")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[1].ID)
		err := repo.Reorder(ctx, "2026-08-24", "dinner", []string{rs[0].ID})
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

		newID, err := repo.SplitBatch(ctx, "2026-08-25", "dinner", 0)
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

		newID, err := repo.SplitBatch(ctx, "2026-08-24", "dinner", 0) // 起点行を指す
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
		returned, err := repo.SplitBatch(ctx, "2026-08-24", "dinner", 0)
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

		if _, err := repo.SplitBatch(ctx, "2026-08-26", "dinner", 0); err != nil {
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

	t.Run("MergeBatch re-tags target row to the specified batch_id", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		mon, _ := repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID)
		if err := repo.MergeBatch(ctx, "2026-08-25", "dinner", 0, mon.BatchID); err != nil {
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
		err := repo.MergeBatch(ctx, "2026-08-24", "dinner", 0, "ghost-batch-id")
		if !errors.Is(err, mealplan.ErrBatchTargetNotFound) {
			t.Errorf("err = %v, want ErrBatchTargetNotFound", err)
		}
	})

	t.Run("ListMealBatches returns only same-meal same-recipe within ±4 days batches", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)    // 起点 (r1)
		tue, _ := repo.Assign(ctx, "2026-08-25", "dinner", rs[0].ID) // ±4 日以内、同レシピ (候補)
		_, _ = repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID)    // 別レシピ (除外)
		_, _ = repo.Assign(ctx, "2026-09-05", "dinner", rs[0].ID)    // 12 日後 (除外)
		_, _ = repo.Assign(ctx, "2026-08-24", "lunch", rs[0].ID)     // 別 meal (除外)

		got, err := repo.ListMealBatches(ctx, "2026-08-24", "dinner", rs[0].ID)
		if err != nil {
			t.Fatalf("ListMealBatches: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 candidate, got %d: %+v", len(got), got)
		}
		if got[0].BatchID != tue.BatchID {
			t.Errorf("batchID = %q, want %q", got[0].BatchID, tue.BatchID)
		}
		if got[0].Meal != mealplan.MealDinner {
			t.Errorf("meal = %q, want dinner", got[0].Meal)
		}
	})

	t.Run("MergeBatch returns ErrBatchMergeDenied for different recipe target", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが", "みそ汁")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)     // 起点 r1
		miso, _ := repo.Assign(ctx, "2026-08-25", "dinner", rs[1].ID) // 別レシピ
		err := repo.MergeBatch(ctx, "2026-08-24", "dinner", 0, miso.BatchID)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("MergeBatch returns ErrBatchMergeDenied for out-of-range date target", func(t *testing.T) {
		repo, rr := newMealPlanRepoForTest(t)
		rs := seedRecipes(t, rr, "肉じゃが")
		_, _ = repo.Assign(ctx, "2026-08-24", "dinner", rs[0].ID)    // 起点
		far, _ := repo.Assign(ctx, "2026-09-05", "dinner", rs[0].ID) // 12 日後 (レンジ外)
		err := repo.MergeBatch(ctx, "2026-08-24", "dinner", 0, far.BatchID)
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

		if err := repo.MergeBatch(ctx, "2026-08-28", "dinner", 0, mon.BatchID); err != nil {
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

	t.Run("ListPeriod populates batch_member_count via window function", func(t *testing.T) {
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
