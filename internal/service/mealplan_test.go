package service

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

// fakeMealPlanRepo は service 層の入力バリデーションと分岐だけを検証する薄い in-memory 実装。
// SQLite 経路は sqlite/mealplan_repo_test.go でカバーする。
type fakeMealPlanRepo struct {
	entries      []mealplan.Entry
	knownRecipes map[string]string // id → name
}

func newFakeMealPlanRepo(recipes map[string]string) *fakeMealPlanRepo {
	return &fakeMealPlanRepo{knownRecipes: recipes}
}

func (r *fakeMealPlanRepo) ListWeek(_ context.Context, weekStart string) ([]mealplan.Entry, error) {
	end, _ := addDaysStr(weekStart, 7)
	out := make([]mealplan.Entry, 0)
	for _, e := range r.entries {
		if e.Date >= weekStart && e.Date < end {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *fakeMealPlanRepo) Assign(_ context.Context, date, slot, recipeID string) (mealplan.Entry, error) {
	name, ok := r.knownRecipes[recipeID]
	if !ok {
		return mealplan.Entry{}, mealplan.ErrUnknownRecipe
	}
	pos := 0
	for _, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot && e.Position >= pos {
			pos = e.Position + 1
		}
	}
	entry := mealplan.Entry{
		Date: date, Slot: mealplan.Slot(slot), Position: pos,
		RecipeID: recipeID, RecipeName: name, BatchID: mealplan.NewBatchID(),
	}
	r.entries = append(r.entries, entry)
	return entry, nil
}

func (r *fakeMealPlanRepo) Replace(_ context.Context, date, slot string, position int, recipeID string) error {
	name, ok := r.knownRecipes[recipeID]
	if !ok {
		return mealplan.ErrUnknownRecipe
	}
	for i, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot && e.Position == position {
			r.entries[i].RecipeID = recipeID
			r.entries[i].RecipeName = name
			return nil
		}
	}
	return mealplan.ErrEntryNotFound
}

func (r *fakeMealPlanRepo) Remove(_ context.Context, date, slot string, position int) error {
	idx := -1
	for i, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot && e.Position == position {
			idx = i
			break
		}
	}
	if idx < 0 {
		return mealplan.ErrEntryNotFound
	}
	r.entries = append(r.entries[:idx], r.entries[idx+1:]...)
	for i, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot && e.Position > position {
			r.entries[i].Position--
		}
	}
	return nil
}

func (r *fakeMealPlanRepo) ClearSlot(_ context.Context, date, slot string) error {
	kept := r.entries[:0]
	for _, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot {
			continue
		}
		kept = append(kept, e)
	}
	r.entries = kept
	return nil
}

func (r *fakeMealPlanRepo) Reorder(_ context.Context, date, slot string, orderedRecipeIDs []string) error {
	current := map[string]int{}
	for _, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot {
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
		if e.Date == date && string(e.Slot) == slot {
			r.entries[i].Position = newPos[e.RecipeID]
		}
	}
	return nil
}

func (r *fakeMealPlanRepo) CopyLeftover(
	_ context.Context, slot string, targets []string,
	source []mealplan.Entry, mode mealplan.CopyMode, shareBatch mealplan.ShareBatch,
) error {
	prepared := prepareFakeCopySource(source, shareBatch)
	for _, target := range targets {
		if mode == mealplan.CopyModeOverwrite {
			kept := r.entries[:0]
			for _, e := range r.entries {
				if e.Date == target && string(e.Slot) == slot {
					continue
				}
				kept = append(kept, e)
			}
			r.entries = kept
		}
		existing := make([]mealplan.Entry, 0)
		for _, e := range r.entries {
			if e.Date == target && string(e.Slot) == slot {
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

func (r *fakeMealPlanRepo) SplitBatch(_ context.Context, date, slot string, position int) (string, error) {
	var currentBatchID string
	for _, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot && e.Position == position {
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
		isOrigin := e.Date == origin.Date && e.Slot == origin.Slot && e.Position == origin.Position
		if e.BatchID == currentBatchID && !isOrigin {
			continue
		}
		kept = append(kept, e)
	}
	r.entries = kept

	for _, cr := range sameBatch[1:] {
		for i := range r.entries {
			if r.entries[i].Date == cr.Date && r.entries[i].Slot == cr.Slot &&
				r.entries[i].Position > cr.Position {
				r.entries[i].Position--
			}
		}
	}

	newID := mealplan.NewBatchID()
	for i := range r.entries {
		if r.entries[i].Date == origin.Date && r.entries[i].Slot == origin.Slot &&
			r.entries[i].Position == origin.Position {
			r.entries[i].BatchID = newID
			return newID, nil
		}
	}
	return "", mealplan.ErrEntryNotFound
}

func (r *fakeMealPlanRepo) MergeBatch(
	_ context.Context, date, slot string, position int, targetBatchID string,
) error {
	var sourceRecipe string
	for _, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot && e.Position == position {
			sourceRecipe = e.RecipeID
			break
		}
	}
	if sourceRecipe == "" {
		return mealplan.ErrEntryNotFound
	}
	sameSlotAndBatch := 0
	eligible := 0
	dayRange := r.mergeDayRange()
	for _, e := range r.entries {
		if string(e.Slot) != slot || e.BatchID != targetBatchID {
			continue
		}
		if e.Date == date && e.Position == position {
			continue
		}
		sameSlotAndBatch++
		if e.RecipeID == sourceRecipe && absDayGap(e.Date, date) <= dayRange {
			eligible++
		}
	}
	if sameSlotAndBatch == 0 {
		return mealplan.ErrBatchTargetNotFound
	}
	if eligible == 0 {
		return mealplan.ErrBatchMergeDenied
	}
	for i, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot && e.Position == position {
			r.entries[i].BatchID = targetBatchID
			return nil
		}
	}
	return mealplan.ErrEntryNotFound
}

func (r *fakeMealPlanRepo) ListSlotBatches(
	_ context.Context, date, slot, recipeID string,
) ([]mealplan.BatchSummary, error) {
	var ownBatchID string
	for _, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot {
			ownBatchID = e.BatchID
			break
		}
	}
	dayRange := r.mergeDayRange()
	type rep struct {
		date     string
		position int
		recipe   string
		name     string
		count    int
	}
	byBatch := map[string]*rep{}
	for _, e := range r.entries {
		if string(e.Slot) != slot || e.RecipeID != recipeID {
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
			byBatch[e.BatchID] = &rep{date: e.Date, position: e.Position, recipe: e.RecipeID, name: e.RecipeName, count: 1}
			continue
		}
		cur.count++
		if e.Date < cur.date || (e.Date == cur.date && e.Position < cur.position) {
			cur.date = e.Date
			cur.position = e.Position
			cur.recipe = e.RecipeID
			cur.name = e.RecipeName
		}
	}
	out := make([]mealplan.BatchSummary, 0, len(byBatch))
	for bid, r := range byBatch {
		out = append(out, mealplan.BatchSummary{
			BatchID: bid, Date: r.date, Slot: mealplan.Slot(slot),
			RecipeID: r.recipe, RecipeName: r.name, MemberCount: r.count,
		})
	}
	return out, nil
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

func (r *fakeMealPlanRepo) SlotEntries(_ context.Context, date, slot string) ([]mealplan.Entry, error) {
	out := make([]mealplan.Entry, 0)
	for _, e := range r.entries {
		if e.Date == date && string(e.Slot) == slot {
			out = append(out, e)
		}
	}
	return out, nil
}

func addDaysStr(s string, days int) (string, error) {
	// service_test 独自の軽量ヘルパ。domain 側の ValidateDate 経由で厳密性は担保される。
	if err := mealplan.ValidateDate(s); err != nil {
		return "", err
	}
	// 7 日先を返せば十分。date の year-month を跨ぐ計算は sqlite 実装側で保証されるので、
	// このテストでは同月内の範囲だけ扱う。
	return s[:8] + advanceDay(s[8:], days), nil
}

func advanceDay(dd string, days int) string {
	n := (int(dd[0]-'0')*10 + int(dd[1]-'0')) + days
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestMealPlanService_ListWeek_returnsRepoEntries(t *testing.T) {
	t.Run("should return entries from the repository when the week is requested", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(context.Background(), "2026-08-24", "dinner", "r1")

		got, err := svc.ListWeek(context.Background(), "2026-08-24")
		if err != nil {
			t.Fatalf("ListWeek: %v", err)
		}
		if len(got) != 1 || got[0].RecipeID != "r1" {
			t.Errorf("got %+v", got)
		}
	})
}

func TestMealPlanService_AssignRecipe(t *testing.T) {
	ctx := context.Background()

	t.Run("should return an entry with a fresh position when the slot already has items", func(t *testing.T) {
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

	t.Run("with slot=snack should return ErrInvalidSlot and leave meal_plans untouched", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, err := svc.AssignRecipe(ctx, "2026-08-24", "snack", "r1")
		if !errors.Is(err, mealplan.ErrInvalidSlot) {
			t.Errorf("err = %v, want ErrInvalidSlot", err)
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
		got, _ := svc.ListWeek(ctx, "2026-08-24")
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
		if err := svc.RemoveEntry(ctx, "2026-08-24", "dinner", 1); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		got, _ := svc.ListWeek(ctx, "2026-08-24")
		if len(got) != 2 || got[0].Position != 0 || got[1].Position != 1 {
			t.Errorf("positions after remove: %+v", got)
		}
	})

	t.Run("against a non-existent position should return ErrEntryNotFound", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		err := svc.RemoveEntry(ctx, "2026-08-24", "dinner", 0)
		if !errors.Is(err, mealplan.ErrEntryNotFound) {
			t.Errorf("err = %v, want ErrEntryNotFound", err)
		}
	})
}

func TestMealPlanService_ClearSlot(t *testing.T) {
	t.Run("should delete every entry in the target (date, slot)", func(t *testing.T) {
		ctx := context.Background()
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		if err := svc.ClearSlot(ctx, "2026-08-24", "dinner"); err != nil {
			t.Fatalf("ClearSlot: %v", err)
		}
		got, _ := svc.ListWeek(ctx, "2026-08-24")
		if len(got) != 0 {
			t.Errorf("want 0, got %d", len(got))
		}
	})
}

func TestMealPlanService_ReorderSlot(t *testing.T) {
	ctx := context.Background()

	t.Run("with recipeIDs missing or duplicating an existing entry should return ErrReorderMismatch", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		err := svc.ReorderSlot(ctx, "2026-08-24", "dinner", []string{"r1"})
		if !errors.Is(err, mealplan.ErrReorderMismatch) {
			t.Errorf("err = %v, want ErrReorderMismatch", err)
		}
	})
}

func TestMealPlanService_CopyLeftover(t *testing.T) {
	ctx := context.Background()

	t.Run("with position=-1 and mode=append should append every source entry into each target slot", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")

		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 3, -1, "append", false); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		// 起点日 + 2 日で合計 3 日、うち起点自身は除外され 2 日に反映
		got25, _ := repo.SlotEntries(ctx, "2026-08-25", "dinner")
		got26, _ := repo.SlotEntries(ctx, "2026-08-26", "dinner")
		if len(got25) != 2 || len(got26) != 2 {
			t.Errorf("target counts: 25=%d, 26=%d", len(got25), len(got26))
		}
	})

	t.Run("with position=0 and mode=overwrite should replace target slot with the single source item", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A", "r2": "B", "r9": "既存"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r2")
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r9")

		if err := svc.CopyLeftover(ctx, "2026-08-24", "dinner", 2, 0, "overwrite", false); err != nil {
			t.Fatalf("CopyLeftover: %v", err)
		}
		got25, _ := repo.SlotEntries(ctx, "2026-08-25", "dinner")
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
		remaining, _ := svc.ListWeek(ctx, "2026-08-24")
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

	t.Run("should re-tag row to the target BatchID when it exists in the same slot", func(t *testing.T) {
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

	t.Run("returns ErrBatchTargetNotFound when targetBatchID has no other row in the same slot", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		err := svc.MergeBatch(ctx, "2026-08-24", "dinner", 0, "ghost-batch-id")
		if !errors.Is(err, mealplan.ErrBatchTargetNotFound) {
			t.Errorf("err = %v, want ErrBatchTargetNotFound", err)
		}
	})
}

func TestMealPlanService_ListSlotBatches(t *testing.T) {
	ctx := context.Background()

	t.Run("should return only same-slot same-recipe within ±4 days batches", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "肉じゃが", "r2": "みそ汁"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1") // 起点行 (月曜・夕、r1)
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r1") // 火曜・夕 (別バッチ、同レシピ)
		_, _ = svc.AssignRecipe(ctx, "2026-08-25", "dinner", "r2") // 火曜・夕 (別レシピ、除外)
		_, _ = svc.AssignRecipe(ctx, "2026-09-05", "dinner", "r1") // 12 日後 (レンジ外、除外)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "lunch", "r1")  // 別 slot、除外

		got, err := svc.ListSlotBatches(ctx, "2026-08-24", "dinner", "r1")
		if err != nil {
			t.Fatalf("ListSlotBatches: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("want 1 candidate, got %d: %+v", len(got), got)
		}
		if got[0].Date != "2026-08-25" || got[0].RecipeID != "r1" {
			t.Errorf("candidate = %+v, want date=2026-08-25 recipe=r1", got[0])
		}
	})

	t.Run("returns ErrUnknownRecipe when recipeID is empty", func(t *testing.T) {
		repo := newFakeMealPlanRepo(map[string]string{"r1": "A"})
		svc := newMealPlanServiceWithRepo(repo)
		_, _ = svc.AssignRecipe(ctx, "2026-08-24", "dinner", "r1")
		_, err := svc.ListSlotBatches(ctx, "2026-08-24", "dinner", "")
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
}
