package service

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// fakeMealPlanRepo は service 層の入力バリデーションと分岐だけを検証する薄い in-memory 実装。
// SQLite 経路は sqlite/mealplan_repo_test.go でカバーする。
type fakeMealPlanRepo struct {
	entries      []mealplan.Entry
	knownRecipes map[string]string // id → name
	spreadCalls  []spreadCall
	restoreCalls [][]mealplan.RemovedEntry
}

type spreadCall struct {
	date       string
	meal       string
	position   int
	targets    []mealplan.SpreadTarget
	mode       mealplan.CopyMode
	shareBatch mealplan.ShareBatch
}

func newFakeMealPlanRepo(recipes map[string]string) *fakeMealPlanRepo {
	return &fakeMealPlanRepo{knownRecipes: recipes}
}

func (r *fakeMealPlanRepo) ListPeriod(
	_ context.Context, periodStart string, periodDays int,
) ([]mealplan.Entry, error) {
	end, _ := addDaysStr(periodStart, periodDays)
	out := make([]mealplan.Entry, 0)
	for _, e := range r.entries {
		if e.Date >= periodStart && e.Date < end {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *fakeMealPlanRepo) Assign(_ context.Context, date, meal, recipeID string) (mealplan.Entry, error) {
	name, ok := r.knownRecipes[recipeID]
	if !ok {
		return mealplan.Entry{}, mealplan.ErrUnknownRecipe
	}
	pos := 0
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position >= pos {
			pos = e.Position + 1
		}
	}
	entry := mealplan.Entry{
		Date: date, Meal: mealplan.Meal(meal), Position: pos,
		RecipeID: recipeID, RecipeName: name, BatchID: mealplan.NewBatchID(),
	}
	r.entries = append(r.entries, entry)
	return entry, nil
}

func (r *fakeMealPlanRepo) Replace(_ context.Context, date, meal string, position int, recipeID string) error {
	name, ok := r.knownRecipes[recipeID]
	if !ok {
		return mealplan.ErrUnknownRecipe
	}
	for i, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			r.entries[i].RecipeID = recipeID
			r.entries[i].RecipeName = name
			return nil
		}
	}
	return mealplan.ErrEntryNotFound
}

func (r *fakeMealPlanRepo) Remove(_ context.Context, date, meal string, position int) (mealplan.RemovedEntry, error) {
	idx := -1
	for i, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			idx = i
			break
		}
	}
	if idx < 0 {
		return mealplan.RemovedEntry{}, mealplan.ErrEntryNotFound
	}
	removed := toRemovedEntry(r.entries[idx])
	r.entries = append(r.entries[:idx], r.entries[idx+1:]...)
	for i, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position > position {
			r.entries[i].Position--
		}
	}
	return removed, nil
}

func (r *fakeMealPlanRepo) ClearMeal(_ context.Context, date, meal string) ([]mealplan.RemovedEntry, error) {
	removed := make([]mealplan.RemovedEntry, 0)
	kept := r.entries[:0]
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal {
			removed = append(removed, toRemovedEntry(e))
			continue
		}
		kept = append(kept, e)
	}
	r.entries = kept
	sort.Slice(removed, func(i, j int) bool { return removed[i].Position < removed[j].Position })
	return removed, nil
}

func (r *fakeMealPlanRepo) RestoreEntries(_ context.Context, entries []mealplan.RemovedEntry) error {
	r.restoreCalls = append(r.restoreCalls, entries)
	return nil
}

func toRemovedEntry(e mealplan.Entry) mealplan.RemovedEntry {
	return mealplan.RemovedEntry{
		Date: e.Date, Meal: e.Meal, Position: e.Position, RecipeID: e.RecipeID, BatchID: e.BatchID,
		CreatedAt: "2026-08-24T00:00:00Z",
	}
}

func (r *fakeMealPlanRepo) Reorder(_ context.Context, date, meal string, orderedRecipeIDs []string) error {
	current := map[string]int{}
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal {
			current[e.RecipeID]++
		}
	}
	requested := map[string]int{}
	for _, id := range orderedRecipeIDs {
		requested[id]++
	}
	if len(current) != len(requested) {
		return mealplan.ErrReorderMismatch
	}
	for id, n := range current {
		if requested[id] != n {
			return mealplan.ErrReorderMismatch
		}
	}
	newPos := map[string]int{}
	for pos, id := range orderedRecipeIDs {
		newPos[id] = pos
	}
	for i, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal {
			r.entries[i].Position = newPos[e.RecipeID]
		}
	}
	return nil
}

func (r *fakeMealPlanRepo) CopyLeftover(
	_ context.Context, meal string, targets []string,
	source []mealplan.Entry, mode mealplan.CopyMode, shareBatch mealplan.ShareBatch,
) error {
	prepared := prepareFakeCopySource(source, shareBatch)
	for _, target := range targets {
		if mode == mealplan.CopyModeOverwrite {
			kept := r.entries[:0]
			for _, e := range r.entries {
				if e.Date == target && string(e.Meal) == meal {
					continue
				}
				kept = append(kept, e)
			}
			r.entries = kept
		}
		existing := make([]mealplan.Entry, 0)
		for _, e := range r.entries {
			if e.Date == target && string(e.Meal) == meal {
				existing = append(existing, e)
			}
		}
		toInsert, err := mealplan.PlanCopy(target, prepared, existing, mode, shareBatch)
		if err != nil {
			return err
		}
		for i := range toInsert {
			if toInsert[i].BatchID == "" {
				toInsert[i].BatchID = mealplan.NewBatchID()
			}
		}
		r.entries = append(r.entries, toInsert...)
	}
	return nil
}

func prepareFakeCopySource(source []mealplan.Entry, shareBatch mealplan.ShareBatch) []mealplan.Entry {
	if !shareBatch.IsShared() {
		return source
	}
	out := make([]mealplan.Entry, len(source))
	byRecipe := map[string]string{}
	for i, e := range source {
		bid, ok := byRecipe[e.RecipeID]
		if !ok {
			if e.BatchID != "" {
				bid = e.BatchID
			} else {
				bid = mealplan.NewBatchID()
			}
			byRecipe[e.RecipeID] = bid
		}
		out[i] = e
		out[i].BatchID = bid
	}
	return out
}

func (r *fakeMealPlanRepo) SplitBatch(_ context.Context, date, meal string, position int) (string, error) {
	var currentBatchID string
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			currentBatchID = e.BatchID
			break
		}
	}
	if currentBatchID == "" {
		return "", mealplan.ErrEntryNotFound
	}

	sameBatch := make([]mealplan.Entry, 0)
	for _, e := range r.entries {
		if e.BatchID == currentBatchID {
			sameBatch = append(sameBatch, e)
		}
	}
	if len(sameBatch) == 1 {
		return currentBatchID, nil
	}

	sort.Slice(sameBatch, func(i, j int) bool {
		if sameBatch[i].Date != sameBatch[j].Date {
			return sameBatch[i].Date < sameBatch[j].Date
		}
		return sameBatch[i].Position < sameBatch[j].Position
	})
	origin := sameBatch[0]

	kept := r.entries[:0]
	for _, e := range r.entries {
		isOrigin := e.Date == origin.Date && e.Meal == origin.Meal && e.Position == origin.Position
		if e.BatchID == currentBatchID && !isOrigin {
			continue
		}
		kept = append(kept, e)
	}
	r.entries = kept

	for _, cr := range sameBatch[1:] {
		for i := range r.entries {
			if r.entries[i].Date == cr.Date && r.entries[i].Meal == cr.Meal &&
				r.entries[i].Position > cr.Position {
				r.entries[i].Position--
			}
		}
	}

	newID := mealplan.NewBatchID()
	for i := range r.entries {
		if r.entries[i].Date == origin.Date && r.entries[i].Meal == origin.Meal &&
			r.entries[i].Position == origin.Position {
			r.entries[i].BatchID = newID
			return newID, nil
		}
	}
	return "", mealplan.ErrEntryNotFound
}

func (r *fakeMealPlanRepo) MergeBatch(
	_ context.Context, date, meal string, position int, targetBatchID string,
) error {
	var sourceRecipe string
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			sourceRecipe = e.RecipeID
			break
		}
	}
	if sourceRecipe == "" {
		return mealplan.ErrEntryNotFound
	}
	sameBatch := 0
	eligible := 0
	dayRange := r.mergeDayRange()
	for _, e := range r.entries {
		if e.BatchID != targetBatchID {
			continue
		}
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			continue
		}
		sameBatch++
		if e.RecipeID == sourceRecipe && absDayGap(e.Date, date) <= dayRange {
			eligible++
		}
	}
	if sameBatch == 0 {
		return mealplan.ErrBatchTargetNotFound
	}
	if eligible == 0 {
		return mealplan.ErrBatchMergeDenied
	}
	for i, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			r.entries[i].BatchID = targetBatchID
			return nil
		}
	}
	return mealplan.ErrEntryNotFound
}

func (r *fakeMealPlanRepo) ListMergeCandidates(
	_ context.Context, date, meal, recipeID string,
) ([]mealplan.BatchSummary, error) {
	var ownBatchID string
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.RecipeID == recipeID {
			ownBatchID = e.BatchID
			break
		}
	}
	dayRange := r.mergeDayRange()
	type rep struct {
		date     string
		meal     string
		position int
		recipe   string
		name     string
		count    int
	}
	byBatch := map[string]*rep{}
	for _, e := range r.entries {
		if e.RecipeID != recipeID {
			continue
		}
		if e.BatchID == ownBatchID {
			continue
		}
		if absDayGap(e.Date, date) > dayRange {
			continue
		}
		cur, ok := byBatch[e.BatchID]
		if !ok {
			byBatch[e.BatchID] = &rep{
				date: e.Date, meal: string(e.Meal), position: e.Position,
				recipe: e.RecipeID, name: e.RecipeName, count: 1,
			}
			continue
		}
		cur.count++
		if e.Date < cur.date || (e.Date == cur.date && e.Position < cur.position) {
			cur.date = e.Date
			cur.meal = string(e.Meal)
			cur.position = e.Position
			cur.recipe = e.RecipeID
			cur.name = e.RecipeName
		}
	}
	out := make([]mealplan.BatchSummary, 0, len(byBatch))
	for bid, r := range byBatch {
		out = append(out, mealplan.BatchSummary{
			BatchID: bid, Date: r.date, Meal: mealplan.Meal(r.meal),
			RecipeID: r.recipe, RecipeName: r.name, MemberCount: r.count,
		})
	}
	return out, nil
}

// Spread は呼び出しを記録するだけにする。反映の中身は sqlite/mealplan_repo_test.go で確かめる。
func (r *fakeMealPlanRepo) Spread(
	_ context.Context,
	date, meal string, position int,
	targets []mealplan.SpreadTarget,
	mode mealplan.CopyMode,
	shareBatch mealplan.ShareBatch,
) error {
	r.spreadCalls = append(r.spreadCalls, spreadCall{
		date: date, meal: meal, position: position, targets: targets, mode: mode, shareBatch: shareBatch,
	})
	return nil
}

func (r *fakeMealPlanRepo) MoveEntry(
	_ context.Context,
	sourceDate, sourceMeal string, sourcePosition int,
	targetDate, targetMeal string,
) error {
	if sourceDate == targetDate && sourceMeal == targetMeal {
		return mealplan.ErrInvalidPosition
	}
	idx := -1
	for i, e := range r.entries {
		if e.Date == sourceDate && string(e.Meal) == sourceMeal && e.Position == sourcePosition {
			idx = i
			break
		}
	}
	if idx < 0 {
		return mealplan.ErrEntryNotFound
	}
	moved := r.entries[idx]
	r.entries = append(r.entries[:idx], r.entries[idx+1:]...)
	for i, e := range r.entries {
		if e.Date == sourceDate && string(e.Meal) == sourceMeal && e.Position > sourcePosition {
			r.entries[i].Position--
		}
	}
	pos := 0
	for _, e := range r.entries {
		if e.Date == targetDate && string(e.Meal) == targetMeal && e.Position >= pos {
			pos = e.Position + 1
		}
	}
	moved.Date = targetDate
	moved.Meal = mealplan.Meal(targetMeal)
	moved.Position = pos
	r.entries = append(r.entries, moved)
	return nil
}

func (r *fakeMealPlanRepo) SetCooked(_ context.Context, date, meal string, position int, cooked bool) error {
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			for i, o := range r.entries {
				if o.BatchID == e.BatchID && o.RecipeID == e.RecipeID {
					r.entries[i].Cooked = cooked
				}
			}
			return nil
		}
	}
	return mealplan.ErrEntryNotFound
}

func (r *fakeMealPlanRepo) mergeDayRange() int {
	// service_test 独自の軽量な defaults 参照。config.Defaults() を読むと循環依存に近い形になるため、
	// 実際のプロジェクト config と同値をハードコードしておく (LeftoverCopyPresets = [2,3,4] の max)。
	return 4
}

func absDayGap(d1, d2 string) int {
	// "YYYY-MM-DD" を time.Time でパースして日数差の絶対値を返す。
	t1, err := time.Parse("2006-01-02", d1)
	if err != nil {
		return 1 << 30
	}
	t2, err := time.Parse("2006-01-02", d2)
	if err != nil {
		return 1 << 30
	}
	diff := int(t1.Sub(t2).Hours() / 24)
	if diff < 0 {
		return -diff
	}
	return diff
}

func (r *fakeMealPlanRepo) MealEntries(_ context.Context, date, meal string) ([]mealplan.Entry, error) {
	out := make([]mealplan.Entry, 0)
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal {
			out = append(out, e)
		}
	}
	return out, nil
}

func addDaysStr(s string, days int) (string, error) {
	// service_test 独自の日付ヘルパ。ListPeriod で 14/28 日先まで扱うため月跨ぎ対応する。
	if err := mealplan.ValidateDate(s); err != nil {
		return "", err
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, 0, days).Format("2006-01-02"), nil
}

func TestMealPlanService_ListPeriod(t *testing.T) {
	ctx := context.Background()

	t.Run("should return entries from the repository when periodWeeks=1", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")

		got, err := svc.ListPeriod(ctx, "2026-08-24", 1)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 1 || got[0].RecipeID != "r1" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("periodWeeks=2 should include entries in the following week", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-31", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-09-07", "dinner", "r1") // 3 週目 (レンジ外)

		got, err := svc.ListPeriod(ctx, "2026-08-24", 2)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("want 2, got %d: %+v", len(got), got)
		}
	})

	t.Run("periodWeeks=4 should include entries across 4 weeks", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-09-14", "dinner", "r1") // 4 週目
		_, _ = svc.AssignRecipe(ctx, "2026-09-21", "dinner", "r1") // 5 週目 (レンジ外)

		got, err := svc.ListPeriod(ctx, "2026-08-24", 4)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("want 2, got %d: %+v", len(got), got)
		}
	})

	t.Run("periodWeeks=3 should return ErrInvalidPeriodWeeks", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{})
		svc := newMealPlanServiceWithRepo(repo)
		_, err := svc.ListPeriod(ctx, "2026-08-24", 3)
		if !errors.Is(err, mealplan.ErrInvalidPeriodWeeks) {
			t.Errorf("err = %v, want ErrInvalidPeriodWeeks", err)
		}
	})

	t.Run("with malformed date should return ErrInvalidDate", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{})
		svc := newMealPlanServiceWithRepo(repo)
		_, err := svc.ListPeriod(ctx, "2026/08/24", 1)
		if !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("err = %v, want ErrInvalidDate", err)
		}
	})
}

func TestMealPlanService_AssignRecipe(t *testing.T) {
	ctx := context.Background()

	t.Run("should return an entry with a fresh position when the meal already has items", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		got, err := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		if err != nil {
			t.Fatalf("Assign: %v", err)
		}
		if got.Position != 1 {
			t.Errorf("position = %d, want 1", got.Position)
		}
	})

	t.Run("with malformed date should return ErrInvalidDate and leave meal_plans untouched", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, err := svc.AssignRecipe(ctx, "2026/08/24", "dinner", "r1")
		if !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("err = %v, want ErrInvalidDate", err)
		}
		if len(repo.entries) != 0 {
			t.Errorf("meal_plans changed: %d entries", len(repo.entries))
		}
	})

	t.Run("with meal=snack should return ErrInvalidMeal and leave meal_plans untouched", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, err := svc.AssignRecipe(ctx, "2026-08-24", "snack", "r1")
		if !errors.Is(err, mealplan.ErrInvalidMeal) {
			t.Errorf("err = %v, want ErrInvalidMeal", err)
		}
		if len(repo.entries) != 0 {
			t.Errorf("meal_plans changed: %d entries", len(repo.entries))
		}
	})
}

func TestMealPlanService_ReplaceEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("should replace only the target position and keep others", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B", "r3": "C"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		if err := svc.ReplaceEntry(ctx, "2026-08-24", "dinner", 1, "r3"); err != nil {
			t.Fatalf("Replace: %v", err)
		}
		got, _ := svc.ListPeriod(ctx, "2026-08-24", 1)
		if got[0].RecipeID != "r1" || got[1].RecipeID != "r3" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("with position=-1 should return ErrInvalidPosition and leave meal_plans untouched", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		err := svc.ReplaceEntry(ctx, "2026-08-24", "dinner", -1, "r1")
		if !errors.Is(err, mealplan.ErrInvalidPosition) {
			t.Errorf("err = %v, want ErrInvalidPosition", err)
		}
	})

	t.Run("against a non-existent position should return ErrEntryNotFound", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		err := svc.ReplaceEntry(ctx, "2026-08-24", "dinner", 0, "r1")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})
}

func TestMealPlanService_RemoveEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("should renumber positions when a middle entry is removed", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B", "r3": "C"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r3")
		removed, err := svc.RemoveEntry(ctx, "2026-08-24", "dinner", 1)
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if removed.RecipeID != "r2" || removed.Position != 1 {
			t.Errorf("removed = %+v, want r2 at position 1", removed)
		}
		got, _ := svc.ListPeriod(ctx, "2026-08-24", 1)
		if len(got) != 2 || got[0].Position != 0 || got[1].Position != 1 {
			t.Errorf("positions after remove: %+v", got)
		}
	})

	t.Run("against a non-existent position should return ErrEntryNotFound", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, err := svc.RemoveEntry(ctx, "2026-08-24", "dinner", 0)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})
}

func TestMealPlanService_ClearMeal(t *testing.T) {
	t.Run("should delete every entry in the target (date, meal)", func(t *testing.T) {
		ctx := context.Background()
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		removed, err := svc.ClearMeal(ctx, "2026-08-24", "dinner")
		if err != nil {
			t.Fatalf("ClearMeal: %v", err)
		}
		if len(removed) != 2 || removed[0].RecipeID != "r1" || removed[1].RecipeID != "r2" {
			t.Errorf("removed = %+v, want r1, r2", removed)
		}
		got, _ := svc.ListPeriod(ctx, "2026-08-24", 1)
		if len(got) != 0 {
			t.Errorf("want 0, got %d", len(got))
		}
	})
}

func TestMealPlanService_RestoreEntries(t *testing.T) {
	ctx := context.Background()
	valid := func() mealplan.RemovedEntry {
		return mealplan.RemovedEntry{
			Date: "2026-08-24", Meal: mealplan.MealDinner, Position: 1,
			RecipeID: "r1", BatchID: "b1", CreatedAt: "2026-08-20T09:00:00Z", CookedAt: "2026-08-24T19:00:00Z",
		}
	}

	t.Run("should pass valid entries to the repository", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)
		second := valid()
		second.Position = 2
		second.CookedAt = ""
		if err := svc.RestoreEntries(ctx, []mealplan.RemovedEntry{valid(), second}); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		if len(repo.restoreCalls) != 1 || len(repo.restoreCalls[0]) != 2 {
			t.Fatalf("restoreCalls = %+v, want 1 call with 2 entries", repo.restoreCalls)
		}
	})

	t.Run("should do nothing when entries is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)
		if err := svc.RestoreEntries(ctx, nil); err != nil {
			t.Fatalf("RestoreEntries: %v", err)
		}
		if len(repo.restoreCalls) != 0 {
			t.Errorf("restoreCalls = %+v, want none", repo.restoreCalls)
		}
	})

	invalid := []struct {
		name   string
		mutate func(e *mealplan.RemovedEntry)
		want   error
	}{
		{"should return ErrInvalidRestore when recipeId is empty", func(e *mealplan.RemovedEntry) { e.RecipeID = "" }, mealplan.ErrInvalidRestore},
		{"should return ErrInvalidRestore when batchId is empty", func(e *mealplan.RemovedEntry) { e.BatchID = "" }, mealplan.ErrInvalidRestore},
		{"should return ErrInvalidRestore when createdAt is not RFC3339", func(e *mealplan.RemovedEntry) { e.CreatedAt = "2026-08-20" }, mealplan.ErrInvalidRestore},
		{"should return ErrInvalidRestore when cookedAt is neither empty nor RFC3339", func(e *mealplan.RemovedEntry) { e.CookedAt = "yesterday" }, mealplan.ErrInvalidRestore},
		{"should return ErrInvalidDate when the date is invalid", func(e *mealplan.RemovedEntry) { e.Date = "2026-13-01" }, mealplan.ErrInvalidDate},
		{"should return ErrInvalidMeal when the meal is invalid", func(e *mealplan.RemovedEntry) { e.Meal = "supper" }, mealplan.ErrInvalidMeal},
		{"should return ErrInvalidPosition when the position is negative", func(e *mealplan.RemovedEntry) { e.Position = -1 }, mealplan.ErrInvalidPosition},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeMealPlanRepo(nil)
			svc := newMealPlanServiceWithRepo(repo)
			e := valid()
			tc.mutate(&e)
			err := svc.RestoreEntries(ctx, []mealplan.RemovedEntry{valid(), e})
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
			if len(repo.restoreCalls) != 0 {
				t.Errorf("restoreCalls = %+v, want none", repo.restoreCalls)
			}
		})
	}
}

func TestMealPlanService_ReorderMeal(t *testing.T) {
	ctx := context.Background()

	t.Run("with recipeIDs missing or duplicating an existing entry should return ErrReorderMismatch", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		err := svc.ReorderMeal(ctx, "2026-08-24", "dinner", []string{"r1"})
		if !errors.Is(err, mealplan.ErrReorderMismatch) {
			t.Errorf("err = %v, want ErrReorderMismatch", err)
		}
	})
}

func TestMealPlanService_CopyLeftover(t *testing.T) {
	ctx := context.Background()

	t.Run("with position=-1 and mode=append should append every source entry into each target meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")

		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, -1, "append", false); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		// 起点日 + 2 日で合計 3 日、うち起点自身は除外され 2 日に反映
		got25, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		got26, _ := repo.MealEntries(ctx, "2026-08-26", "dinner")
		if len(got25) != 2 || len(got26) != 2 {
			t.Errorf("target counts: 25=%d, 26=%d", len(got25), len(got26))
		}
	})

	t.Run("with position=0 and mode=overwrite should replace target meal with the single source item", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B", "r9": "既存"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r9")

		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 2, 0, "overwrite", false); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got25, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		if len(got25) != 1 || got25[0].RecipeID != "r1" {
			t.Errorf("got25 = %+v; want single r1", got25)
		}
	})

	t.Run("with days=1 should be a no-op (target list contains only the source date itself)", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		// days=1 は preset に含まれないので ErrInvalidCopyDays が返る (config 側の下敷き)
		err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 1, -1, "append", false)
		if !errors.Is(err, mealplan.ErrInvalidCopyDays) {
			t.Errorf("err = %v, want ErrInvalidCopyDays", err)
		}
	})

	t.Run("with days=5 should return ErrInvalidCopyDays when preset is [2, 3, 4]", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 5, -1, "append", false)
		if !errors.Is(err, mealplan.ErrInvalidCopyDays) {
			t.Errorf("err = %v, want ErrInvalidCopyDays", err)
		}
	})
}

func TestMealPlanService_CopyLeftover_ShareBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("should share the source BatchID across every copied row when shareBatch=true", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, 0, "append", true); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		bids := map[string]int{}
		for _, e := range repo.entries {
			if e.RecipeID == "r1" {
				bids[e.BatchID]++
			}
		}
		if len(bids) != 1 {
			t.Fatalf("want single shared batchID, got %d distinct: %+v", len(bids), bids)
		}
		if _, ok := bids[src.BatchID]; !ok {
			t.Errorf("shared batchID %+v does not include source %q", bids, src.BatchID)
		}
	})

	t.Run("should assign distinct BatchIDs per copied row when shareBatch=false", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, 0, "append", false); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		bids := map[string]bool{}
		for _, e := range repo.entries {
			if e.RecipeID == "r1" {
				bids[e.BatchID] = true
			}
		}
		if len(bids) != 3 {
			t.Errorf("want 3 distinct BatchIDs, got %d: %+v", len(bids), bids)
		}
	})
}

func TestMealPlanService_SplitBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("should keep only the origin day and re-tag it with a fresh BatchID", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, 0, "append", true); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		newID, err := svc.SplitBatch(ctx, "2026-08-25", "dinner", 0)
		if err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		if newID == "" || newID == src.BatchID {
			t.Errorf("newID = %q, src.BatchID = %q", newID, src.BatchID)
		}
		remaining, _ := svc.ListPeriod(ctx, "2026-08-24", 1)
		if len(remaining) != 1 {
			t.Fatalf("want 1 remaining row (origin only), got %d: %+v", len(remaining), remaining)
		}
		if remaining[0].Date != "2026-08-24" || remaining[0].BatchID != newID {
			t.Errorf("remaining row = %+v, want date=2026-08-24 batchID=%q", remaining[0], newID)
		}
	})

	t.Run("is a no-op for single-row batch", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		e, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		returned, err := svc.SplitBatch(ctx, "2026-08-24", "dinner", 0)
		if err != nil {
			t.Fatalf("SplitBatch: %v", err)
		}
		if returned != e.BatchID {
			t.Errorf("returned = %q, want current batch_id %q (no-op)", returned, e.BatchID)
		}
		if len(repo.entries) != 1 {
			t.Errorf("row count changed: %d", len(repo.entries))
		}
	})

	t.Run("returns ErrEntryNotFound for missing position", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, err := svc.SplitBatch(ctx, "2026-08-24", "dinner", 0)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})
}

func TestMealPlanService_MergeBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("should re-tag row to the target BatchID when it exists in the same meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		monEntry, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1") // 月曜
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r1")         // 火曜 (別バッチ)

		if err := svc.MergeBatch(ctx, "2026-08-25", "dinner", 0, monEntry.BatchID); err != nil {
			t.Fatalf("MergeBatch: %v", err)
		}
		for _, e := range repo.entries {
			if e.Date == "2026-08-25" && e.BatchID != monEntry.BatchID {
				t.Errorf("target row not merged, got batchID=%q want=%q", e.BatchID, monEntry.BatchID)
			}
		}
	})

	t.Run("returns ErrBatchTargetNotFound when targetBatchID has no other row in the same meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MergeBatch(ctx, "2026-08-24", "dinner", 0, "ghost-batch-id")
		if !errors.Is(err, mealplan.ErrBatchTargetNotFound) {
			t.Errorf("err = %v, want ErrBatchTargetNotFound", err)
		}
	})
}

func TestMealPlanService_ListMergeCandidates(t *testing.T) {
	ctx := context.Background()

	t.Run("should return same-recipe within ±4 days batches from any meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが", "r2": "みそ汁"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1") // 起点行 (月曜・夕、r1)
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r1") // 火曜・夕 (別バッチ、同レシピ)
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r2") // 火曜・夕 (別レシピ、除外)
		_, _ = svc.AssignRecipe(ctx, "2026-09-05", "dinner", "r1") // 12 日後 (レンジ外、除外)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "lunch", "r1")  // 別 meal、同レシピ (候補に含める)

		got, err := svc.ListMergeCandidates(ctx, "2026-08-24", "dinner", "r1")
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 candidates (火曜・夕 + 月曜・昼食), got %d: %+v", len(got), got)
		}
		byMeal := map[string]bool{}
		for _, c := range got {
			byMeal[string(c.Meal)] = true
		}
		if !byMeal["dinner"] || !byMeal["lunch"] {
			t.Errorf("expected candidates from both dinner and lunch, got %+v", got)
		}
	})

	t.Run("returns ErrUnknownRecipe when recipeID is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, err := svc.ListMergeCandidates(ctx, "2026-08-24", "dinner", "")
		if !errors.Is(err, mealplan.ErrUnknownRecipe) {
			t.Errorf("err = %v, want ErrUnknownRecipe", err)
		}
	})
}

func TestMealPlanService_MergeBatch_ConstraintChecks(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ErrBatchMergeDenied when target batch has different recipe", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが", "r2": "みそ汁"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")          // 起点 r1
		misoEntry, _ := svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r2") // 別レシピ (みそ汁)
		err := svc.MergeBatch(ctx, "2026-08-24", "dinner", 0, misoEntry.BatchID)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("returns ErrBatchMergeDenied when target batch is beyond ±4 days", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")         // 起点
		farEntry, _ := svc.AssignRecipe(ctx, "2026-09-05", "dinner", "r1") // 12 日後
		err := svc.MergeBatch(ctx, "2026-08-24", "dinner", 0, farEntry.BatchID)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("succeeds when 1 day is skipped (Mon-Tue leftover + Fri single)", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		monEntry, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")      // 月
		_ = svc.CopyLeftover(ctx, "2026-08-24", "dinner", 2, 0, "append", true) // 月火 leftover
		_, _ = svc.AssignRecipe(ctx, "2026-08-28", "dinner", "r1")              // 金 (4 日後、同レシピ)

		if err := svc.MergeBatch(ctx, "2026-08-28", "dinner", 0, monEntry.BatchID); err != nil {
			t.Fatalf("MergeBatch: %v", err)
		}
		merged := 0
		for _, e := range repo.entries {
			if e.BatchID == monEntry.BatchID {
				merged++
			}
		}
		if merged != 3 {
			t.Errorf("want 3 rows in monEntry batch (mon+tue+fri), got %d", merged)
		}
	})

	t.Run("succeeds across different meals (dinner -> lunch, same recipe within ±4 days)", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		monDinner, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1") // 月・夕
		tueLunch, _ := svc.AssignRecipe(ctx, "2026-08-25", "lunch", "r1")   // 火・昼 (別バッチ)

		if err := svc.MergeBatch(ctx, "2026-08-25", "lunch", 0, monDinner.BatchID); err != nil {
			t.Fatalf("MergeBatch across meals: %v", err)
		}
		for _, e := range repo.entries {
			if e.Date == "2026-08-25" && e.Meal == mealplan.MealLunch && e.BatchID != monDinner.BatchID {
				t.Errorf("target row not merged: %+v (was %q)", e, tueLunch.BatchID)
			}
		}
	})

	t.Run("returns ErrBatchMergeDenied when target across meals is different recipe", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが", "r2": "みそ汁"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")         // 月・夕 r1
		misoLunch, _ := svc.AssignRecipe(ctx, "2026-08-25", "lunch", "r2") // 火・昼 別レシピ
		err := svc.MergeBatch(ctx, "2026-08-24", "dinner", 0, misoLunch.BatchID)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})
}

func TestMealPlanService_SpreadEntry(t *testing.T) {
	ctx := context.Background()
	cell := func(date string, meal mealplan.Meal) mealplan.SpreadTarget {
		return mealplan.SpreadTarget{Date: date, Meal: meal}
	}

	t.Run("should pass the given targets to the repository unchanged when all targets are valid", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)
		targets := []mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-27", mealplan.MealLunch)}

		if err := svc.SpreadEntry(ctx, "2026-08-24", "dinner", 1, targets, "overwrite", false); err != nil {
			t.Fatalf("SpreadEntry: %v", err)
		}
		if len(repo.spreadCalls) != 1 {
			t.Fatalf("want 1 repo call, got %d", len(repo.spreadCalls))
		}
		got := repo.spreadCalls[0]
		if got.date != "2026-08-24" || got.meal != "dinner" || got.position != 1 ||
			got.mode != mealplan.CopyModeOverwrite || got.shareBatch.IsShared() {
			t.Errorf("call = %+v", got)
		}
		if len(got.targets) != 2 || got.targets[0] != targets[0] || got.targets[1] != targets[1] {
			t.Errorf("targets = %+v, want %+v", got.targets, targets)
		}
	})

	t.Run("should return ErrInvalidSpreadTarget and not reach the repository when a target is out of range", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)
		targets := []mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-31", mealplan.MealDinner)}

		err := svc.SpreadEntry(ctx, "2026-08-24", "dinner", 0, targets, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidSpreadTarget) {
			t.Errorf("err = %v, want ErrInvalidSpreadTarget", err)
		}
		if len(repo.spreadCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.spreadCalls))
		}
	})

	t.Run("should return ErrInvalidSpreadTarget and not reach the repository when targets is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)

		err := svc.SpreadEntry(ctx, "2026-08-24", "dinner", 0, nil, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidSpreadTarget) {
			t.Errorf("err = %v, want ErrInvalidSpreadTarget", err)
		}
		if len(repo.spreadCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.spreadCalls))
		}
	})

	t.Run("should return ErrInvalidCopyMode when mode is neither append nor overwrite", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)

		err := svc.SpreadEntry(ctx, "2026-08-24", "dinner", 0,
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)}, "unknown", true)
		if !errors.Is(err, mealplan.ErrInvalidCopyMode) {
			t.Errorf("err = %v, want ErrInvalidCopyMode", err)
		}
	})

	t.Run("should return ErrInvalidPosition when the source position is negative", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)

		err := svc.SpreadEntry(ctx, "2026-08-24", "dinner", -1,
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)}, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidPosition) {
			t.Errorf("err = %v, want ErrInvalidPosition", err)
		}
	})

	t.Run("should derive the range from LeftoverCopyPresets when presets change", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo)
		svc.cfg.LeftoverCopyPresets = []int{2, 6}
		far := []mealplan.SpreadTarget{cell("2026-08-29", mealplan.MealDinner)}

		if err := svc.SpreadEntry(ctx, "2026-08-24", "dinner", 0, far, "append", true); err != nil {
			t.Errorf("5 days later with presets {2, 6}: err = %v, want nil", err)
		}
		svc.cfg.LeftoverCopyPresets = []int{2}
		err := svc.SpreadEntry(ctx, "2026-08-24", "dinner", 0,
			[]mealplan.SpreadTarget{cell("2026-08-26", mealplan.MealDinner)}, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidSpreadTarget) {
			t.Errorf("2 days later with presets {2}: err = %v, want ErrInvalidSpreadTarget", err)
		}
	})
}

func TestMealPlanService_MoveEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("should move a single row from source cell to target cell", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")

		if err := svc.MoveEntry(ctx, "2026-08-24", "dinner", 0, "2026-08-25", "dinner"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		if len(repo.entries) != 1 {
			t.Fatalf("want 1 entry after move, got %d", len(repo.entries))
		}
		got := repo.entries[0]
		if got.Date != "2026-08-25" || got.Meal != mealplan.MealDinner || got.Position != 0 {
			t.Errorf("moved entry = %+v; want date=2026-08-25 meal=dinner pos=0", got)
		}
	})

	t.Run("should preserve batch_id when moving across meals", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")

		if err := svc.MoveEntry(ctx, "2026-08-24", "dinner", 0, "2026-08-24", "lunch"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		if repo.entries[0].BatchID != src.BatchID {
			t.Errorf("batchID after cross-meal move = %q, want %q", repo.entries[0].BatchID, src.BatchID)
		}
	})

	t.Run("with malformed source date should return ErrInvalidDate and leave meal_plans untouched", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MoveEntry(ctx, "2026/08/24", "dinner", 0, "2026-08-25", "dinner")
		if !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("err = %v, want ErrInvalidDate", err)
		}
		if len(repo.entries) != 1 || repo.entries[0].Date != "2026-08-24" {
			t.Errorf("meal_plans changed: %+v", repo.entries)
		}
	})

	t.Run("with source meal=snack should return ErrInvalidMeal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MoveEntry(ctx, "2026-08-24", "snack", 0, "2026-08-25", "dinner")
		if !errors.Is(err, mealplan.ErrInvalidMeal) {
			t.Errorf("err = %v, want ErrInvalidMeal", err)
		}
	})

	t.Run("with negative source position should return ErrInvalidPosition", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		err := svc.MoveEntry(ctx, "2026-08-24", "dinner", -1, "2026-08-25", "dinner")
		if !errors.Is(err, mealplan.ErrInvalidPosition) {
			t.Errorf("err = %v, want ErrInvalidPosition", err)
		}
	})

	t.Run("with source == target cell should return ErrInvalidPosition", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MoveEntry(ctx, "2026-08-24", "dinner", 0, "2026-08-24", "dinner")
		if !errors.Is(err, mealplan.ErrInvalidPosition) {
			t.Errorf("err = %v, want ErrInvalidPosition", err)
		}
	})
}

// openMealPlanDB は実 SQLite を開く。作った印は組単位で広がり再起動後も残ることを確かめるため、fake ではなく実 DB を使う。
func openMealPlanDB(t *testing.T, path string) (*MealPlanService, *RecipeService) {
	t.Helper()
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewMealPlanService(sqlite.NewMealPlanRepo(db)), NewRecipeService(db)
}

func cookedDates(t *testing.T, svc *MealPlanService) map[string]bool {
	t.Helper()
	got, err := svc.ListPeriod(context.Background(), "2026-08-24", 1)
	if err != nil {
		t.Fatalf("ListPeriod: %v", err)
	}
	out := map[string]bool{}
	for _, e := range got {
		out[e.Date] = e.Cooked
	}
	return out
}

func TestMealPlanService_SetCooked(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) (*MealPlanService, string, string) {
		t.Helper()
		path := filepath.Join(t.TempDir(), "test.db")
		svc, rs := openMealPlanDB(t, path)
		r, err := rs.Create(ctx, recipe.Recipe{Name: "カレー", Servings: 1})
		if err != nil {
			t.Fatalf("create recipe: %v", err)
		}
		return svc, r.ID, path
	}

	t.Run("should mark all rows of the same batch and recipe when marking one row", func(t *testing.T) {
		svc, rid, _ := setup(t)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", rid)
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, 0, string(mealplan.CopyModeAppend), true); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		if err := svc.SetCooked(ctx, "2026-08-25", "dinner", 0, true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		got := cookedDates(t, svc)
		if len(got) != 3 {
			t.Fatalf("want 3 rows, got %+v", got)
		}
		for d, c := range got {
			if !c {
				t.Errorf("%s: want cooked", d)
			}
		}
	})

	t.Run("should not spread the mark to a separate batch of the same recipe", func(t *testing.T) {
		svc, rid, _ := setup(t)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", rid)
		_, _ = svc.AssignRecipe(ctx, "2026-08-26", "dinner", rid)
		_ = svc.SetCooked(ctx, "2026-08-24", "dinner", 0, true)
		got := cookedDates(t, svc)
		if !got["2026-08-24"] || got["2026-08-26"] {
			t.Errorf("want only 2026-08-24 cooked, got %+v", got)
		}
	})

	t.Run("should unmark all rows of the same batch and recipe when unmarking one row", func(t *testing.T) {
		svc, rid, _ := setup(t)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", rid)
		_ = svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, 0, string(mealplan.CopyModeAppend), true)
		_ = svc.SetCooked(ctx, "2026-08-24", "dinner", 0, true)
		if err := svc.SetCooked(ctx, "2026-08-25", "dinner", 0, false); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		for d, c := range cookedDates(t, svc) {
			if c {
				t.Errorf("%s: want not cooked", d)
			}
		}
	})

	t.Run("should return ErrEntryNotFound when the row does not exist", func(t *testing.T) {
		svc, _, _ := setup(t)
		err := svc.SetCooked(ctx, "2026-08-24", "dinner", 0, true)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("want ErrEntryNotFound, got %v", err)
		}
	})

	t.Run("should return ErrInvalidDate when the date is malformed", func(t *testing.T) {
		svc, _, _ := setup(t)
		err := svc.SetCooked(ctx, "2026/08/24", "dinner", 0, true)
		if !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("want ErrInvalidDate, got %v", err)
		}
	})

	t.Run("should keep the mark after reopening the database", func(t *testing.T) {
		svc, rid, path := setup(t)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", rid)
		if err := svc.SetCooked(ctx, "2026-08-24", "dinner", 0, true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		reopened, _ := openMealPlanDB(t, path)
		if !cookedDates(t, reopened)["2026-08-24"] {
			t.Error("want cooked after reopen")
		}
	})
}
