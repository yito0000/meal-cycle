package service

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
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
	undoCalls    []mealplan.SpreadResult
	restoreCalls [][]mealplan.RemovedEntry
	reorderCalls [][]string
}

type spreadCall struct {
	entryID    string
	targets    []mealplan.SpreadTarget
	mode       mealplan.CopyMode
	shareBatch mealplan.ShareBatch
}

func newFakeMealPlanRepo(recipes map[string]string) *fakeMealPlanRepo {
	return &fakeMealPlanRepo{knownRecipes: recipes}
}

func (r *fakeMealPlanRepo) indexOf(entryID string) int {
	for i, e := range r.entries {
		if e.ID == entryID {
			return i
		}
	}
	return -1
}

// idAt はテストの前提を並び順で書くための逆引き。
func (r *fakeMealPlanRepo) idAt(date, meal string, position int) string {
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			return e.ID
		}
	}
	return ""
}

func (r *fakeMealPlanRepo) closeGap(date string, meal mealplan.Meal, removed int) {
	for i, e := range r.entries {
		if e.Date == date && e.Meal == meal && e.Position > removed {
			r.entries[i].Position--
		}
	}
}

func (r *fakeMealPlanRepo) nextPosition(date, meal string) int {
	pos := 0
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal && e.Position >= pos {
			pos = e.Position + 1
		}
	}
	return pos
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
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		if out[i].Meal != out[j].Meal {
			return out[i].Meal < out[j].Meal
		}
		return out[i].Position < out[j].Position
	})
	return out, nil
}

func (r *fakeMealPlanRepo) Entry(_ context.Context, entryID string) (mealplan.Entry, error) {
	idx := r.indexOf(entryID)
	if idx < 0 {
		return mealplan.Entry{}, mealplan.ErrEntryNotFound
	}
	return r.entries[idx], nil
}

func (r *fakeMealPlanRepo) Assign(_ context.Context, date, meal, recipeID string) (mealplan.Entry, error) {
	name, ok := r.knownRecipes[recipeID]
	if !ok {
		return mealplan.Entry{}, mealplan.ErrUnknownRecipe
	}
	entry := mealplan.Entry{
		ID: mealplan.NewEntryID(), Date: date, Meal: mealplan.Meal(meal), Position: r.nextPosition(date, meal),
		RecipeID: recipeID, RecipeName: name, BatchID: mealplan.NewBatchID(),
	}
	r.entries = append(r.entries, entry)
	return entry, nil
}

func (r *fakeMealPlanRepo) Replace(_ context.Context, entryID, recipeID string) error {
	name, ok := r.knownRecipes[recipeID]
	if !ok {
		return mealplan.ErrUnknownRecipe
	}
	idx := r.indexOf(entryID)
	if idx < 0 {
		return mealplan.ErrEntryNotFound
	}
	r.entries[idx].RecipeID = recipeID
	r.entries[idx].RecipeName = name
	return nil
}

func (r *fakeMealPlanRepo) Remove(_ context.Context, entryID string) (mealplan.RemovedEntry, error) {
	idx := r.indexOf(entryID)
	if idx < 0 {
		return mealplan.RemovedEntry{}, mealplan.ErrEntryNotFound
	}
	e := r.entries[idx]
	removed := toRemovedEntry(e)
	r.entries = append(r.entries[:idx], r.entries[idx+1:]...)
	r.closeGap(e.Date, e.Meal, e.Position)
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
		ID: e.ID, Date: e.Date, Meal: e.Meal, Position: e.Position, RecipeID: e.RecipeID, BatchID: e.BatchID,
		CreatedAt: "2026-08-24T00:00:00Z",
	}
}

func (r *fakeMealPlanRepo) Reorder(_ context.Context, date, meal string, orderedEntryIDs []string) error {
	r.reorderCalls = append(r.reorderCalls, orderedEntryIDs)
	current := map[string]bool{}
	for _, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal {
			current[e.ID] = true
		}
	}
	if len(current) != len(orderedEntryIDs) {
		return mealplan.ErrReorderMismatch
	}
	newPos := map[string]int{}
	for pos, id := range orderedEntryIDs {
		if _, dup := newPos[id]; dup || !current[id] {
			return mealplan.ErrReorderMismatch
		}
		newPos[id] = pos
	}
	for i, e := range r.entries {
		if e.Date == date && string(e.Meal) == meal {
			r.entries[i].Position = newPos[e.ID]
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

func (r *fakeMealPlanRepo) SplitBatch(_ context.Context, entryID string) (string, error) {
	idx := r.indexOf(entryID)
	if idx < 0 {
		return "", mealplan.ErrEntryNotFound
	}
	currentBatchID := r.entries[idx].BatchID

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
		if e.BatchID == currentBatchID && e.ID != origin.ID {
			continue
		}
		kept = append(kept, e)
	}
	r.entries = kept
	for _, cr := range sameBatch[1:] {
		r.closeGap(cr.Date, cr.Meal, cr.Position)
	}

	newID := mealplan.NewBatchID()
	r.entries[r.indexOf(origin.ID)].BatchID = newID
	return newID, nil
}

func (r *fakeMealPlanRepo) MergeBatch(_ context.Context, entryID, targetBatchID string, dayRange int) error {
	idx := r.indexOf(entryID)
	if idx < 0 {
		return mealplan.ErrEntryNotFound
	}
	src := r.entries[idx]
	sameBatch := 0
	eligible := 0
	for _, e := range r.entries {
		if e.BatchID != targetBatchID || e.ID == entryID {
			continue
		}
		sameBatch++
		if e.RecipeID == src.RecipeID && absDayGap(e.Date, src.Date) <= dayRange {
			eligible++
		}
	}
	if sameBatch == 0 {
		return mealplan.ErrBatchTargetNotFound
	}
	if eligible == 0 {
		return mealplan.ErrBatchMergeDenied
	}
	r.entries[idx].BatchID = targetBatchID
	return nil
}

func (r *fakeMealPlanRepo) ListMergeCandidates(_ context.Context, entryID string, dayRange int) ([]mealplan.BatchSummary, error) {
	idx := r.indexOf(entryID)
	if idx < 0 {
		return nil, mealplan.ErrEntryNotFound
	}
	own := r.entries[idx]
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
		if e.RecipeID != own.RecipeID || e.BatchID == own.BatchID || absDayGap(e.Date, own.Date) > dayRange {
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

// Spread は呼び出しを記録し、対象セルの数だけ仮の ID を返す。反映の中身は sqlite/mealplan_repo_test.go で確かめる。
func (r *fakeMealPlanRepo) Spread(
	_ context.Context,
	entryID string,
	targets []mealplan.SpreadTarget,
	mode mealplan.CopyMode,
	shareBatch mealplan.ShareBatch,
) (mealplan.SpreadResult, error) {
	r.spreadCalls = append(r.spreadCalls, spreadCall{
		entryID: entryID, targets: targets, mode: mode, shareBatch: shareBatch,
	})
	res := mealplan.SpreadResult{AddedEntryIDs: []string{}, Removed: []mealplan.RemovedEntry{}}
	for _, tg := range targets {
		res.AddedEntryIDs = append(res.AddedEntryIDs, "added-"+tg.Date+"-"+string(tg.Meal))
	}
	return res, nil
}

func (r *fakeMealPlanRepo) UndoSpread(_ context.Context, result mealplan.SpreadResult) error {
	r.undoCalls = append(r.undoCalls, result)
	return nil
}

func (r *fakeMealPlanRepo) MoveEntry(_ context.Context, entryID, targetDate, targetMeal string) error {
	idx := r.indexOf(entryID)
	if idx < 0 {
		return mealplan.ErrEntryNotFound
	}
	moved := r.entries[idx]
	if moved.Date == targetDate && string(moved.Meal) == targetMeal {
		return mealplan.ErrInvalidPosition
	}
	r.entries = append(r.entries[:idx], r.entries[idx+1:]...)
	r.closeGap(moved.Date, moved.Meal, moved.Position)
	pos := r.nextPosition(targetDate, targetMeal)
	moved.Date = targetDate
	moved.Meal = mealplan.Meal(targetMeal)
	moved.Position = pos
	r.entries = append(r.entries, moved)
	return nil
}

func (r *fakeMealPlanRepo) SetCooked(_ context.Context, entryID string, cooked bool) error {
	idx := r.indexOf(entryID)
	if idx < 0 {
		return mealplan.ErrEntryNotFound
	}
	e := r.entries[idx]
	for i, o := range r.entries {
		if o.BatchID == e.BatchID && o.RecipeID == e.RecipeID {
			r.entries[i].Cooked = cooked
		}
	}
	return nil
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, err := svc.ListPeriod(ctx, "2026-08-24", 3)
		if !errors.Is(err, mealplan.ErrInvalidPeriodWeeks) {
			t.Errorf("err = %v, want ErrInvalidPeriodWeeks", err)
		}
	})

	t.Run("with malformed date should return ErrInvalidDate", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		if err := svc.ReplaceEntry(ctx, repo.idAt("2026-08-24", "dinner", 1), "r3"); err != nil {
			t.Fatalf("Replace: %v", err)
		}
		got, _ := svc.ListPeriod(ctx, "2026-08-24", 1)
		if got[0].RecipeID != "r1" || got[1].RecipeID != "r3" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("with an empty entryID should return ErrEntryNotFound", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		err := svc.ReplaceEntry(ctx, "", "r1")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})

	t.Run("against a non-existent entry should return ErrEntryNotFound", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		err := svc.ReplaceEntry(ctx, "no-such-entry", "r1")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})
}

func TestMealPlanService_RemoveEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("should renumber positions when a middle entry is removed", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B", "r3": "C"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r3")
		removed, err := svc.RemoveEntry(ctx, repo.idAt("2026-08-24", "dinner", 1))
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

	t.Run("against a non-existent entry should return ErrEntryNotFound", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, err := svc.RemoveEntry(ctx, "no-such-entry")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})

	t.Run("should return ErrEntryNotFound when entryID is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, err := svc.RemoveEntry(ctx, "")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
		if len(repo.entries) != 1 {
			t.Errorf("meal_plans changed: %d entries", len(repo.entries))
		}
	})
}

func TestMealPlanService_ClearMeal(t *testing.T) {
	t.Run("should delete every entry in the target (date, meal)", func(t *testing.T) {
		ctx := context.Background()
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
			ID: "e1", Date: "2026-08-24", Meal: mealplan.MealDinner, Position: 1,
			RecipeID: "r1", BatchID: "b1", CreatedAt: "2026-08-20T09:00:00Z", CookedAt: "2026-08-24T19:00:00Z",
		}
	}

	t.Run("should pass valid entries to the repository", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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
		{"should return ErrInvalidRestore when id is empty", func(e *mealplan.RemovedEntry) { e.ID = "" }, mealplan.ErrInvalidRestore},
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
			svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
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

	t.Run("with entry ids missing an existing entry should return ErrReorderMismatch", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		a, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		err := svc.ReorderMeal(ctx, "2026-08-24", "dinner", []string{a.ID})
		if !errors.Is(err, mealplan.ErrReorderMismatch) {
			t.Errorf("err = %v, want ErrReorderMismatch", err)
		}
	})

	t.Run("should pass the ordered entry ids to the repository", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		a, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		b, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err := svc.ReorderMeal(ctx, "2026-08-24", "dinner", []string{b.ID, a.ID}); err != nil {
			t.Fatalf("ReorderMeal: %v", err)
		}
		if len(repo.reorderCalls) != 1 || len(repo.reorderCalls[0]) != 2 ||
			repo.reorderCalls[0][0] != b.ID || repo.reorderCalls[0][1] != a.ID {
			t.Errorf("reorderCalls = %+v, want [[%s %s]]", repo.reorderCalls, b.ID, a.ID)
		}
	})
}

func TestMealPlanService_CopyLeftover(t *testing.T) {
	ctx := context.Background()

	t.Run("with mode=append should append every source entry into each target meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")

		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, "append", false); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		// 起点日 + 2 日で合計 3 日、うち起点自身は除外され 2 日に反映
		got25, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		got26, _ := repo.MealEntries(ctx, "2026-08-26", "dinner")
		if len(got25) != 2 || len(got26) != 2 {
			t.Errorf("target counts: 25=%d, 26=%d", len(got25), len(got26))
		}
	})

	t.Run("should copy all dishes of the meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B", "r9": "既存"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r9")

		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 2, "overwrite", false); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got25, _ := repo.MealEntries(ctx, "2026-08-25", "dinner")
		if len(got25) != 2 || got25[0].RecipeID != "r1" || got25[1].RecipeID != "r2" {
			t.Errorf("got25 = %+v; want r1, r2", got25)
		}
	})

	t.Run("with days=1 should be a no-op (target list contains only the source date itself)", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		// days=1 は preset に含まれないので ErrInvalidCopyDays が返る (config 側の下敷き)
		err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 1, "append", false)
		if !errors.Is(err, mealplan.ErrInvalidCopyDays) {
			t.Errorf("err = %v, want ErrInvalidCopyDays", err)
		}
	})

	t.Run("with days=5 should return ErrInvalidCopyDays when preset is [2, 3, 4]", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 5, "append", false)
		if !errors.Is(err, mealplan.ErrInvalidCopyDays) {
			t.Errorf("err = %v, want ErrInvalidCopyDays", err)
		}
	})
}

func TestMealPlanService_CopyLeftover_ShareBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("should share the source BatchID across every copied row when shareBatch=true", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, "append", true); err != nil {
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, "append", false); err != nil {
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, "append", true); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		newID, err := svc.SplitBatch(ctx, repo.idAt("2026-08-25", "dinner", 0))
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		e, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		returned, err := svc.SplitBatch(ctx, e.ID)
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

	t.Run("returns ErrEntryNotFound for missing entry", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, err := svc.SplitBatch(ctx, "no-such-entry")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})
}

func TestMealPlanService_MergeBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("should re-tag row to the target BatchID when it exists in the same meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		monEntry, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1") // 月曜
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r1")         // 火曜 (別バッチ)

		if err := svc.MergeBatch(ctx, repo.idAt("2026-08-25", "dinner", 0), monEntry.BatchID); err != nil {
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MergeBatch(ctx, repo.idAt("2026-08-24", "dinner", 0), "ghost-batch-id")
		if !errors.Is(err, mealplan.ErrBatchTargetNotFound) {
			t.Errorf("err = %v, want ErrBatchTargetNotFound", err)
		}
	})
}

func TestMealPlanService_ListMergeCandidates(t *testing.T) {
	ctx := context.Background()

	t.Run("should return same-recipe within ±4 days batches from any meal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが", "r2": "みそ汁"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1") // 起点行 (月曜・夕、r1)
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r1") // 火曜・夕 (別バッチ、同レシピ)
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r2") // 火曜・夕 (別レシピ、除外)
		_, _ = svc.AssignRecipe(ctx, "2026-09-05", "dinner", "r1") // 12 日後 (レンジ外、除外)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "lunch", "r1")  // 別 meal、同レシピ (候補に含める)

		got, err := svc.ListMergeCandidates(ctx, repo.idAt("2026-08-24", "dinner", 0))
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

	t.Run("returns ErrEntryNotFound when entryID is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, err := svc.ListMergeCandidates(ctx, "")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})

	t.Run("should take the recipe from the entry itself", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが", "r2": "みそ汁"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		miso, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r1")
		tueMiso, _ := svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r2")

		got, err := svc.ListMergeCandidates(ctx, miso.ID)
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 1 || got[0].BatchID != tueMiso.BatchID || got[0].RecipeID != "r2" {
			t.Errorf("got %+v, want only the みそ汁 batch %q", got, tueMiso.BatchID)
		}
	})
}

func TestMealPlanService_MergeBatch_ConstraintChecks(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ErrBatchMergeDenied when target batch has different recipe", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが", "r2": "みそ汁"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")          // 起点 r1
		misoEntry, _ := svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r2") // 別レシピ (みそ汁)
		err := svc.MergeBatch(ctx, repo.idAt("2026-08-24", "dinner", 0), misoEntry.BatchID)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("returns ErrBatchMergeDenied when target batch is beyond ±4 days", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")         // 起点
		farEntry, _ := svc.AssignRecipe(ctx, "2026-09-05", "dinner", "r1") // 12 日後
		err := svc.MergeBatch(ctx, repo.idAt("2026-08-24", "dinner", 0), farEntry.BatchID)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})

	t.Run("succeeds when 1 day is skipped (Mon-Tue leftover + Fri single)", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		monEntry, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")   // 月
		_ = svc.CopyLeftover(ctx, "2026-08-24", "dinner", 2, "append", true) // 月火 leftover
		_, _ = svc.AssignRecipe(ctx, "2026-08-28", "dinner", "r1")           // 金 (4 日後、同レシピ)

		if err := svc.MergeBatch(ctx, repo.idAt("2026-08-28", "dinner", 0), monEntry.BatchID); err != nil {
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		monDinner, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1") // 月・夕
		tueLunch, _ := svc.AssignRecipe(ctx, "2026-08-25", "lunch", "r1")   // 火・昼 (別バッチ)

		if err := svc.MergeBatch(ctx, repo.idAt("2026-08-25", "lunch", 0), monDinner.BatchID); err != nil {
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")         // 月・夕 r1
		misoLunch, _ := svc.AssignRecipe(ctx, "2026-08-25", "lunch", "r2") // 火・昼 別レシピ
		err := svc.MergeBatch(ctx, repo.idAt("2026-08-24", "dinner", 0), misoLunch.BatchID)
		if !errors.Is(err, mealplan.ErrBatchMergeDenied) {
			t.Errorf("err = %v, want ErrBatchMergeDenied", err)
		}
	})
}

func TestMealPlanService_MergeRange(t *testing.T) {
	ctx := context.Background()
	setup := func(t *testing.T, presets []int) (*MealPlanService, *fakeMealPlanRepo) {
		t.Helper()
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		cfg := config.Defaults()
		cfg.LeftoverCopyPresets = presets
		return newMealPlanServiceWithRepo(repo, staticConfig(cfg)), repo
	}

	t.Run("should list a batch 6 days away as a merge candidate when leftover presets are 2 and 7", func(t *testing.T) {
		svc, repo := setup(t, []int{2, 7})
		mon, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-30", "dinner", "r1")

		got, err := svc.ListMergeCandidates(ctx, repo.idAt("2026-08-30", "dinner", 0))
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 1 || got[0].BatchID != mon.BatchID {
			t.Errorf("got %+v, want only the Monday batch %q", got, mon.BatchID)
		}
	})

	t.Run("should not list a batch 3 days away as a merge candidate when leftover presets are only 2", func(t *testing.T) {
		svc, repo := setup(t, []int{2})
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-27", "dinner", "r1")

		got, err := svc.ListMergeCandidates(ctx, repo.idAt("2026-08-27", "dinner", 0))
		if err != nil {
			t.Fatalf("ListMergeCandidates: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want no candidates, got %+v", got)
		}
	})

	t.Run("should merge into a batch 6 days away when leftover presets are 2 and 7", func(t *testing.T) {
		svc, repo := setup(t, []int{2, 7})
		mon, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-30", "dinner", "r1")

		if err := svc.MergeBatch(ctx, repo.idAt("2026-08-30", "dinner", 0), mon.BatchID); err != nil {
			t.Fatalf("MergeBatch: %v", err)
		}
		if got := repo.entries[repo.indexOf(repo.idAt("2026-08-30", "dinner", 0))].BatchID; got != mon.BatchID {
			t.Errorf("batchID = %q, want %q", got, mon.BatchID)
		}
	})

	t.Run("should deny merging into a batch 5 days away with ErrBatchMergeDenied when leftover presets are 2, 3 and 4", func(t *testing.T) {
		svc, repo := setup(t, []int{2, 3, 4})
		mon, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-29", "dinner", "r1")

		err := svc.MergeBatch(ctx, repo.idAt("2026-08-29", "dinner", 0), mon.BatchID)
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
	setup := func(t *testing.T) (*MealPlanService, *fakeMealPlanRepo, string) {
		t.Helper()
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		src, err := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		if err != nil {
			t.Fatalf("AssignRecipe: %v", err)
		}
		return svc, repo, src.ID
	}

	t.Run("should pass the given targets to the repository unchanged when all targets are valid", func(t *testing.T) {
		svc, repo, src := setup(t)
		targets := []mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-27", mealplan.MealLunch)}

		if _, err := svc.SpreadEntry(ctx, src, targets, "overwrite", false); err != nil {
			t.Fatalf("SpreadEntry: %v", err)
		}
		if len(repo.spreadCalls) != 1 {
			t.Fatalf("want 1 repo call, got %d", len(repo.spreadCalls))
		}
		got := repo.spreadCalls[0]
		if got.entryID != src || got.mode != mealplan.CopyModeOverwrite || got.shareBatch.IsShared() {
			t.Errorf("call = %+v", got)
		}
		if len(got.targets) != 2 || got.targets[0] != targets[0] || got.targets[1] != targets[1] {
			t.Errorf("targets = %+v, want %+v", got.targets, targets)
		}
	})

	t.Run("should return ErrInvalidSpreadTarget and not reach the repository when a target is out of range", func(t *testing.T) {
		svc, repo, src := setup(t)
		targets := []mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-31", mealplan.MealDinner)}

		_, err := svc.SpreadEntry(ctx, src, targets, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidSpreadTarget) {
			t.Errorf("err = %v, want ErrInvalidSpreadTarget", err)
		}
		if len(repo.spreadCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.spreadCalls))
		}
	})

	t.Run("should return ErrInvalidSpreadTarget and not reach the repository when targets is empty", func(t *testing.T) {
		svc, repo, src := setup(t)

		_, err := svc.SpreadEntry(ctx, src, nil, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidSpreadTarget) {
			t.Errorf("err = %v, want ErrInvalidSpreadTarget", err)
		}
		if len(repo.spreadCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.spreadCalls))
		}
	})

	t.Run("should return ErrInvalidCopyMode when mode is neither append nor overwrite", func(t *testing.T) {
		svc, _, src := setup(t)

		_, err := svc.SpreadEntry(ctx, src,
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)}, "unknown", true)
		if !errors.Is(err, mealplan.ErrInvalidCopyMode) {
			t.Errorf("err = %v, want ErrInvalidCopyMode", err)
		}
	})

	t.Run("should return ErrEntryNotFound when entryID is empty or unknown", func(t *testing.T) {
		svc, repo, _ := setup(t)

		for _, id := range []string{"", "no-such-entry"} {
			_, err := svc.SpreadEntry(ctx, id,
				[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner)}, "append", true)
			if !errors.Is(err, mealplan.ErrEntryNotFound) {
				t.Errorf("id %q: err = %v, want ErrEntryNotFound", id, err)
			}
		}
		if len(repo.spreadCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.spreadCalls))
		}
	})

	t.Run("should return the added entry ids when spreading succeeds", func(t *testing.T) {
		svc, _, src := setup(t)

		got, err := svc.SpreadEntry(ctx, src,
			[]mealplan.SpreadTarget{cell("2026-08-25", mealplan.MealDinner), cell("2026-08-26", mealplan.MealLunch)}, "append", true)
		if err != nil {
			t.Fatalf("SpreadEntry: %v", err)
		}
		want := []string{"added-2026-08-25-dinner", "added-2026-08-26-lunch"}
		if len(got.AddedEntryIDs) != 2 || got.AddedEntryIDs[0] != want[0] || got.AddedEntryIDs[1] != want[1] {
			t.Errorf("AddedEntryIDs = %v, want %v", got.AddedEntryIDs, want)
		}
	})

	t.Run("should reject a target in an earlier meal of the same day", func(t *testing.T) {
		svc, repo, src := setup(t)

		_, err := svc.SpreadEntry(ctx, src,
			[]mealplan.SpreadTarget{cell("2026-08-24", mealplan.MealLunch)}, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidSpreadTarget) {
			t.Errorf("err = %v, want ErrInvalidSpreadTarget", err)
		}
		if len(repo.spreadCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.spreadCalls))
		}
	})

	t.Run("should derive the range from LeftoverCopyPresets when presets change", func(t *testing.T) {
		svc, _, src := setup(t)
		cfg := config.Defaults()
		cfg.LeftoverCopyPresets = []int{2, 6}
		svc.cfg = staticConfig(cfg)
		far := []mealplan.SpreadTarget{cell("2026-08-29", mealplan.MealDinner)}

		if _, err := svc.SpreadEntry(ctx, src, far, "append", true); err != nil {
			t.Errorf("5 days later with presets {2, 6}: err = %v, want nil", err)
		}
		cfg.LeftoverCopyPresets = []int{2}
		svc.cfg = staticConfig(cfg)
		_, err := svc.SpreadEntry(ctx, src,
			[]mealplan.SpreadTarget{cell("2026-08-26", mealplan.MealDinner)}, "append", true)
		if !errors.Is(err, mealplan.ErrInvalidSpreadTarget) {
			t.Errorf("2 days later with presets {2}: err = %v, want ErrInvalidSpreadTarget", err)
		}
	})
}

func TestMealPlanService_UndoSpread(t *testing.T) {
	ctx := context.Background()
	removed := func() mealplan.RemovedEntry {
		return mealplan.RemovedEntry{
			ID: "e1", Date: "2026-08-25", Meal: mealplan.MealDinner, Position: 0,
			RecipeID: "r1", BatchID: "b1", CreatedAt: "2026-08-20T09:00:00Z", CookedAt: "2026-08-24T19:00:00Z",
		}
	}

	t.Run("should reject an undo whose removed entry has an invalid date", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		bad := removed()
		bad.Date = "2026/08/25"

		err := svc.UndoSpread(ctx, mealplan.SpreadResult{AddedEntryIDs: []string{"a1"}, Removed: []mealplan.RemovedEntry{bad}})
		if !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("err = %v, want ErrInvalidDate", err)
		}
		if len(repo.undoCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.undoCalls))
		}
	})

	t.Run("should reject an undo whose removed entry has an empty id", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		bad := removed()
		bad.ID = ""

		err := svc.UndoSpread(ctx, mealplan.SpreadResult{AddedEntryIDs: []string{"a1"}, Removed: []mealplan.RemovedEntry{bad}})
		if !errors.Is(err, mealplan.ErrInvalidRestore) {
			t.Errorf("err = %v, want ErrInvalidRestore", err)
		}
		if len(repo.undoCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.undoCalls))
		}
	})

	t.Run("should reject an undo whose added entry id is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))

		err := svc.UndoSpread(ctx, mealplan.SpreadResult{AddedEntryIDs: []string{""}})
		if !errors.Is(err, mealplan.ErrInvalidRestore) {
			t.Errorf("err = %v, want ErrInvalidRestore", err)
		}
	})

	t.Run("should pass the result to the repository when the undo input is valid", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		in := mealplan.SpreadResult{AddedEntryIDs: []string{"a1", "a2"}, Removed: []mealplan.RemovedEntry{removed()}}

		if err := svc.UndoSpread(ctx, in); err != nil {
			t.Fatalf("UndoSpread: %v", err)
		}
		if len(repo.undoCalls) != 1 || len(repo.undoCalls[0].AddedEntryIDs) != 2 || len(repo.undoCalls[0].Removed) != 1 {
			t.Errorf("undoCalls = %+v, want 1 call with the given result", repo.undoCalls)
		}
	})

	t.Run("should do nothing when the undo result is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(nil)
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))

		if err := svc.UndoSpread(ctx, mealplan.SpreadResult{}); err != nil {
			t.Fatalf("UndoSpread: %v", err)
		}
		if len(repo.undoCalls) != 0 {
			t.Errorf("repo should not be called, got %d calls", len(repo.undoCalls))
		}
	})
}

func TestMealPlanService_MoveEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("should move a single row from source cell to target cell", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")

		if err := svc.MoveEntry(ctx, repo.idAt("2026-08-24", "dinner", 0), "2026-08-25", "dinner"); err != nil {
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
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")

		if err := svc.MoveEntry(ctx, src.ID, "2026-08-24", "lunch"); err != nil {
			t.Fatalf("MoveEntry: %v", err)
		}
		if repo.entries[0].BatchID != src.BatchID {
			t.Errorf("batchID after cross-meal move = %q, want %q", repo.entries[0].BatchID, src.BatchID)
		}
	})

	t.Run("with malformed target date should return ErrInvalidDate and leave meal_plans untouched", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MoveEntry(ctx, src.ID, "2026/08/25", "dinner")
		if !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("err = %v, want ErrInvalidDate", err)
		}
		if len(repo.entries) != 1 || repo.entries[0].Date != "2026-08-24" {
			t.Errorf("meal_plans changed: %+v", repo.entries)
		}
	})

	t.Run("with target meal=snack should return ErrInvalidMeal", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MoveEntry(ctx, src.ID, "2026-08-25", "snack")
		if !errors.Is(err, mealplan.ErrInvalidMeal) {
			t.Errorf("err = %v, want ErrInvalidMeal", err)
		}
	})

	t.Run("with an empty entryID should return ErrEntryNotFound", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		err := svc.MoveEntry(ctx, "", "2026-08-25", "dinner")
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})

	t.Run("with source == target cell should return ErrInvalidPosition", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo, staticConfig(config.Defaults()))
		src, _ := svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MoveEntry(ctx, src.ID, "2026-08-24", "dinner")
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
	return NewMealPlanService(sqlite.NewMealPlanRepo(db), staticConfig(config.Defaults())), NewRecipeService(db, staticConfig(config.Defaults()), nil)
}

// entryIDAt は (date, meal, position) にいまある行の ID を返す。テストの前提を並び順で書くためのもの。
func entryIDAt(t *testing.T, svc *MealPlanService, date, meal string, position int) string {
	t.Helper()
	got, err := svc.ListPeriod(context.Background(), date, 1)
	if err != nil {
		t.Fatalf("ListPeriod: %v", err)
	}
	for _, e := range got {
		if e.Date == date && string(e.Meal) == meal && e.Position == position {
			return e.ID
		}
	}
	t.Fatalf("no entry at %s/%s/%d", date, meal, position)
	return ""
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
		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, string(mealplan.CopyModeAppend), true); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		if err := svc.SetCooked(ctx, entryIDAt(t, svc, "2026-08-25", "dinner", 0), true); err != nil {
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
		_ = svc.SetCooked(ctx, entryIDAt(t, svc, "2026-08-24", "dinner", 0), true)
		got := cookedDates(t, svc)
		if !got["2026-08-24"] || got["2026-08-26"] {
			t.Errorf("want only 2026-08-24 cooked, got %+v", got)
		}
	})

	t.Run("should unmark all rows of the same batch and recipe when unmarking one row", func(t *testing.T) {
		svc, rid, _ := setup(t)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", rid)
		_ = svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, string(mealplan.CopyModeAppend), true)
		_ = svc.SetCooked(ctx, entryIDAt(t, svc, "2026-08-24", "dinner", 0), true)
		if err := svc.SetCooked(ctx, entryIDAt(t, svc, "2026-08-25", "dinner", 0), false); err != nil {
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
		err := svc.SetCooked(ctx, "no-such-entry", true)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("want ErrEntryNotFound, got %v", err)
		}
	})

	t.Run("should return ErrEntryNotFound when entryID is empty", func(t *testing.T) {
		svc, _, _ := setup(t)
		err := svc.SetCooked(ctx, "", true)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("want ErrEntryNotFound, got %v", err)
		}
	})

	t.Run("should keep the mark after reopening the database", func(t *testing.T) {
		svc, rid, path := setup(t)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", rid)
		if err := svc.SetCooked(ctx, entryIDAt(t, svc, "2026-08-24", "dinner", 0), true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		reopened, _ := openMealPlanDB(t, path)
		if !cookedDates(t, reopened)["2026-08-24"] {
			t.Error("want cooked after reopen")
		}
	})
}

// spec「品の指定は並び順に左右されない」のシナリオ。画面が一覧を取り直す前の連続操作を、古い一覧の ID で再現する。
func TestMealPlanService_EntryAddressing(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T, names ...string) (*MealPlanService, []string) {
		t.Helper()
		svc, rs := openMealPlanDB(t, filepath.Join(t.TempDir(), "test.db"))
		ids := make([]string, 0, len(names))
		for _, n := range names {
			r, err := rs.Create(ctx, recipe.Recipe{Name: n, Servings: 1})
			if err != nil {
				t.Fatalf("create recipe: %v", err)
			}
			e, err := svc.AssignRecipe(ctx, "2026-08-24", "dinner", r.ID)
			if err != nil {
				t.Fatalf("AssignRecipe: %v", err)
			}
			ids = append(ids, e.ID)
		}
		return svc, ids
	}
	dinnerNames := func(t *testing.T, svc *MealPlanService) []string {
		t.Helper()
		got, err := svc.ListPeriod(ctx, "2026-08-24", 1)
		if err != nil {
			t.Fatalf("ListPeriod: %v", err)
		}
		out := make([]string, 0)
		for _, e := range got {
			if e.Date == "2026-08-24" && e.Meal == mealplan.MealDinner {
				out = append(out, e.RecipeName)
			}
		}
		return out
	}

	t.Run("should remove only the two given dishes when they are removed in a row", func(t *testing.T) {
		svc, ids := setup(t, "肉じゃが", "みそ汁", "ほうれん草の胡麻和え")

		if _, err := svc.RemoveEntry(ctx, ids[0]); err != nil {
			t.Fatalf("RemoveEntry 肉じゃが: %v", err)
		}
		if _, err := svc.RemoveEntry(ctx, ids[1]); err != nil {
			t.Fatalf("RemoveEntry みそ汁: %v", err)
		}
		if got := dinnerNames(t, svc); len(got) != 1 || got[0] != "ほうれん草の胡麻和え" {
			t.Errorf("dinner = %v, want [ほうれん草の胡麻和え]", got)
		}
	})

	t.Run("should mark the given dish when it is marked right after a reorder", func(t *testing.T) {
		svc, ids := setup(t, "肉じゃが", "みそ汁")

		if err := svc.ReorderMeal(ctx, "2026-08-24", "dinner", []string{ids[1], ids[0]}); err != nil {
			t.Fatalf("ReorderMeal: %v", err)
		}
		if err := svc.SetCooked(ctx, ids[0], true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		got, _ := svc.ListPeriod(ctx, "2026-08-24", 1)
		cooked := map[string]bool{}
		for _, e := range got {
			cooked[e.RecipeName] = e.Cooked
		}
		if !cooked["肉じゃが"] || cooked["みそ汁"] {
			t.Errorf("cooked = %v, want only 肉じゃが", cooked)
		}
	})

	t.Run("should reject an operation on a dish that was already removed and keep the other dishes", func(t *testing.T) {
		svc, ids := setup(t, "肉じゃが", "みそ汁")
		removed, err := svc.RemoveEntry(ctx, ids[0])
		if err != nil {
			t.Fatalf("RemoveEntry: %v", err)
		}

		err = svc.ReplaceEntry(ctx, ids[0], removed.RecipeID)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
		if got := dinnerNames(t, svc); len(got) != 1 || got[0] != "みそ汁" {
			t.Errorf("dinner = %v, want [みそ汁]", got)
		}
	})
}
