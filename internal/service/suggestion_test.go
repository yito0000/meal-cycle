package service

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/settings"
	"github.com/yito0000/meal-cycle/internal/domain/suggestion"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// suggestionToday は期限と経過日数の基準日。
var suggestionToday = time.Date(2026, 10, 1, 10, 0, 0, 0, time.FixedZone("JST", 9*60*60))

type suggestionDeps struct {
	svc      *SuggestionService
	settings *SettingsService
	recipe   *RecipeService
	fridge   *FridgeService
	meal     *MealPlanService
}

func newSuggestionServiceForTest(t *testing.T) suggestionDeps {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	st := NewSettingsService(db)
	svc := NewSuggestionService(db, st)
	svc.now = func() time.Time { return suggestionToday }
	fr := NewFridgeService(db, st)
	fr.now = func() time.Time { return suggestionToday }
	return suggestionDeps{
		svc:      svc,
		settings: st,
		recipe:   NewRecipeService(db, st, nil),
		fridge:   fr,
		meal:     NewMealPlanService(sqlite.NewMealPlanRepo(db), st),
	}
}

func suggestionDay(n int) string {
	return suggestionToday.AddDate(0, 0, n).Format(fridge.DateLayout)
}

func (d suggestionDeps) createRecipe(t *testing.T, name string, ingredients ...string) string {
	t.Helper()
	ings := make([]recipe.RecipeIngredient, 0, len(ingredients))
	for _, n := range ingredients {
		ings = append(ings, recipe.RecipeIngredient{Name: n, Amount: 1, Unit: "個"})
	}
	r, err := d.recipe.Create(context.Background(), recipe.Recipe{Name: name, Servings: 1, Ingredients: ings})
	if err != nil {
		t.Fatalf("create recipe: %v", err)
	}
	return r.ID
}

func (d suggestionDeps) addStock(t *testing.T, name, expiresAt string) {
	t.Helper()
	if _, err := d.fridge.Add(context.Background(), fridge.StockItem{Name: name, Amount: 1, Unit: "個", ExpiresAt: expiresAt}); err != nil {
		t.Fatalf("add stock: %v", err)
	}
}

func (d suggestionDeps) save(t *testing.T, edit func(*settings.Values)) {
	t.Helper()
	v, err := d.settings.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	edit(&v)
	if _, err := d.settings.Save(context.Background(), v); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func (d suggestionDeps) list(t *testing.T) suggestion.Result {
	t.Helper()
	got, err := d.svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return got
}

func names(cs []suggestion.Candidate) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.RecipeName)
	}
	return out
}

func TestSuggestionService_List(t *testing.T) {
	t.Run("should return empty by-stock candidates when no stock is near expiry", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		d.createRecipe(t, "卵焼き", "卵")
		d.addStock(t, "卵", suggestionDay(10))
		got := d.list(t)
		if got.ByStock == nil || len(got.ByStock) != 0 {
			t.Errorf("want empty non-nil, got %#v", got.ByStock)
		}
	})

	t.Run("should return recipes using near-expiry stock when such stock exists", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		d.createRecipe(t, "肉じゃが", "玉ねぎ")
		d.createRecipe(t, "卵焼き", "卵")
		d.addStock(t, "玉ねぎ", suggestionDay(2))
		got := d.list(t)
		if n := names(got.ByStock); len(n) != 1 || n[0] != "肉じゃが" {
			t.Errorf("got %v", n)
		}
		if s := got.ByStock[0].Stocks; len(s) != 1 || s[0].Name != "玉ねぎ" || s[0].DaysLeft != 2 {
			t.Errorf("reasons: %#v", s)
		}
	})

	t.Run("should follow saved expiry thresholds when settings change", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		d.createRecipe(t, "卵焼き", "卵")
		d.addStock(t, "卵", suggestionDay(10))
		d.save(t, func(v *settings.Values) { v.ExpiryAlertYellowDays = 10 })
		if n := names(d.list(t).ByStock); len(n) != 1 || n[0] != "卵焼き" {
			t.Errorf("got %v", n)
		}
	})
}

// cook は daysAgo 日前の夕食にレシピを置き、作った印を付ける。
func (d suggestionDeps) cook(t *testing.T, recipeID string, daysAgo int) {
	t.Helper()
	e, err := d.meal.AssignRecipe(context.Background(), suggestionDay(-daysAgo), "dinner", recipeID)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	if err := d.meal.SetCooked(context.Background(), e.ID, true); err != nil {
		t.Fatalf("set cooked: %v", err)
	}
}

func TestSuggestionService_List_ByRecency(t *testing.T) {
	t.Run("should return recipes cooked long ago when threshold has passed", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		d.cook(t, d.createRecipe(t, "麻婆豆腐"), 14)
		d.cook(t, d.createRecipe(t, "カレー"), 3)
		got := d.list(t).ByRecency
		if len(got) != 1 || got[0].RecipeName != "麻婆豆腐" || got[0].DaysSince != 14 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should follow saved recency threshold when settings change", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		d.cook(t, d.createRecipe(t, "肉じゃが"), 7)
		if got := d.list(t).ByRecency; len(got) != 0 {
			t.Fatalf("want none before saving, got %#v", got)
		}
		d.save(t, func(v *settings.Values) { v.RecencyThresholdDays = 5 })
		if n := names(d.list(t).ByRecency); len(n) != 1 || n[0] != "肉じゃが" {
			t.Errorf("got %v", n)
		}
	})
}

func TestSuggestionService_WeekPlans(t *testing.T) {
	ctx := context.Background()
	// suggestionToday (木曜) の翌週の月曜。
	const weekStart = "2026-10-05"
	weekDay := func(offset int) string {
		d, _ := time.Parse(fridge.DateLayout, weekStart)
		return d.AddDate(0, 0, offset).Format(fridge.DateLayout)
	}
	slotDates := func(r suggestion.WeekPlanResult) []string {
		out := make([]string, 0, len(r.Slots))
		for _, s := range r.Slots {
			out = append(out, s.Date)
		}
		return out
	}
	plan := func(t *testing.T, d suggestionDeps) suggestion.WeekPlanResult {
		t.Helper()
		got, err := d.svc.WeekPlans(ctx, weekStart)
		if err != nil {
			t.Fatalf("WeekPlans: %v", err)
		}
		return got
	}

	t.Run("should return empty slots when every target cell has entries", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		id := d.createRecipe(t, "カレー")
		for i := 0; i < 5; i++ {
			if _, err := d.meal.AssignRecipe(ctx, weekDay(i), "dinner", id); err != nil {
				t.Fatalf("assign: %v", err)
			}
		}
		got := plan(t, d)
		if len(got.Slots) != 0 || got.Plans == nil || len(got.Plans) != 0 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should skip cells that already have entries when building slots", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		d.createRecipe(t, "肉じゃが")
		id := d.createRecipe(t, "カレー")
		if _, err := d.meal.AssignRecipe(ctx, weekDay(2), "dinner", id); err != nil {
			t.Fatalf("assign: %v", err)
		}
		got := plan(t, d)
		want := []string{weekDay(0), weekDay(1), weekDay(3), weekDay(4)}
		if !reflect.DeepEqual(slotDates(got), want) {
			t.Errorf("slots: got %v, want %v", slotDates(got), want)
		}
	})

	t.Run("should follow saved fill days and meal when settings change", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		d.createRecipe(t, "カレー")
		d.save(t, func(v *settings.Values) {
			v.WeeklyFillDays = []int{6, 7}
			v.WeeklyFillMeal = "lunch"
		})
		got := plan(t, d)
		if !reflect.DeepEqual(slotDates(got), []string{weekDay(5), weekDay(6)}) || got.Slots[0].Meal != "lunch" {
			t.Errorf("slots: %#v", got.Slots)
		}
	})

	t.Run("should return ErrInvalidDate when weekStart is not a date", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		if _, err := d.svc.WeekPlans(ctx, "2026-02-30"); !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("want ErrInvalidDate, got %v", err)
		}
	})

	t.Run("should return the same plans when called twice with unchanged data", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		for _, n := range []string{"カレー", "肉じゃが", "麻婆豆腐", "鮭のホイル焼き"} {
			d.createRecipe(t, n, "玉ねぎ")
		}
		d.addStock(t, "玉ねぎ", suggestionDay(2))
		d.cook(t, d.createRecipe(t, "オムレツ"), 20)
		first, second := plan(t, d), plan(t, d)
		if !reflect.DeepEqual(first, second) || len(first.Plans) == 0 {
			t.Errorf("first %#v\nsecond %#v", first, second)
		}
	})

	t.Run("should not use recipes that have an upcoming plan", func(t *testing.T) {
		d := newSuggestionServiceForTest(t)
		planned := d.createRecipe(t, "カレー", "玉ねぎ")
		d.createRecipe(t, "肉じゃが")
		d.addStock(t, "玉ねぎ", suggestionDay(2))
		if _, err := d.meal.AssignRecipe(ctx, weekDay(6), "lunch", planned); err != nil {
			t.Fatalf("assign: %v", err)
		}
		for _, p := range plan(t, d).Plans {
			for _, it := range p.Items {
				if it.RecipeID == planned {
					t.Errorf("planned recipe used in %s", p.Kind)
				}
			}
		}
	})
}
