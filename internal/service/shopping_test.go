package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/shopping"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// shoppingToday は期限切れ判定の基準日。対象期間もここから組み立てる。
var shoppingToday = time.Date(2026, 9, 24, 10, 0, 0, 0, time.FixedZone("JST", 9*60*60))

const shoppingStart = "2026-09-24"

type shoppingDeps struct {
	svc    *ShoppingService
	recipe *RecipeService
	meal   *MealPlanService
	fridge *FridgeService
}

func newShoppingServiceForTest(t *testing.T) shoppingDeps {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	svc := NewShoppingService(db, staticConfig(config.Defaults()))
	svc.now = func() time.Time { return shoppingToday }
	fr := NewFridgeService(db, staticConfig(config.Defaults()))
	fr.now = func() time.Time { return shoppingToday }
	return shoppingDeps{
		svc:    svc,
		recipe: NewRecipeService(db, staticConfig(config.Defaults())),
		meal:   NewMealPlanService(sqlite.NewMealPlanRepo(db), staticConfig(config.Defaults())),
		fridge: fr,
	}
}

func (d shoppingDeps) createRecipe(t *testing.T, name string, ings ...recipe.RecipeIngredient) string {
	t.Helper()
	r, err := d.recipe.Create(context.Background(), recipe.Recipe{Name: name, Servings: 1, Ingredients: ings})
	if err != nil {
		t.Fatalf("create recipe: %v", err)
	}
	return r.ID
}

func (d shoppingDeps) assign(t *testing.T, date, recipeID string) {
	t.Helper()
	if _, err := d.meal.AssignRecipe(context.Background(), date, "dinner", recipeID); err != nil {
		t.Fatalf("assign: %v", err)
	}
}

// leftover は date の夕食の品を days 日分の作り置きとして広げる。
func (d shoppingDeps) leftover(t *testing.T, date string, days int) {
	t.Helper()
	if err := d.meal.CopyLeftover(context.Background(), date, "dinner", days,
		string(mealplan.CopyModeAppend), true); err != nil {
		t.Fatalf("copy leftover: %v", err)
	}
}

func (d shoppingDeps) addStock(t *testing.T, name string, amount float64, unit, expiresAt string) {
	t.Helper()
	if _, err := d.fridge.Add(context.Background(),
		fridge.StockItem{Name: name, Amount: amount, Unit: unit, ExpiresAt: expiresAt}); err != nil {
		t.Fatalf("add stock: %v", err)
	}
}

func (d shoppingDeps) list(t *testing.T, days int) shopping.List {
	t.Helper()
	got, err := d.svc.List(context.Background(), shoppingStart, days)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return got
}

func shoppingDay(n int) string {
	return shoppingToday.AddDate(0, 0, n).Format(fridge.DateLayout)
}

func onionIng(n float64) recipe.RecipeIngredient {
	return recipe.RecipeIngredient{Name: "玉ねぎ", Amount: n, Unit: "個"}
}

func onlyItem(t *testing.T, l shopping.List) shopping.Item {
	t.Helper()
	if len(l.Items) != 1 {
		t.Fatalf("want 1 item, got %#v", l.Items)
	}
	return l.Items[0]
}

func TestShoppingService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should return empty dishes and empty items when no meal plans exist in period", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		got := d.list(t, 7)
		if got.Dishes == nil || got.Items == nil || len(got.Dishes) != 0 || len(got.Items) != 0 {
			t.Errorf("want empty non-nil slices, got %#v", got)
		}
	})

	t.Run("should return period end as the last day included when period days is 3", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		got := d.list(t, 3)
		if got.PeriodStart != shoppingStart || got.PeriodEnd != "2026-09-26" || got.PeriodDays != 3 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should count ingredients once when a batch spans three days", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		id := d.createRecipe(t, "肉じゃが", onionIng(2))
		d.assign(t, shoppingDay(0), id)
		d.leftover(t, shoppingDay(0), 3)
		got := d.list(t, 7)
		if it := onlyItem(t, got); it.Amount != 2 {
			t.Errorf("want 2, got %v", it.Amount)
		}
		if len(got.Dishes) != 1 || len(got.Dishes[0].Dates) != 3 || got.Dishes[0].BatchMemberCount != 3 {
			t.Errorf("dishes: %#v", got.Dishes)
		}
	})

	t.Run("should count ingredients per cooking when the same recipe is assigned separately twice", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		id := d.createRecipe(t, "肉じゃが", onionIng(2))
		d.assign(t, shoppingDay(0), id)
		d.assign(t, shoppingDay(2), id)
		if it := onlyItem(t, d.list(t, 7)); it.Amount != 4 {
			t.Errorf("want 4, got %v", it.Amount)
		}
	})

	t.Run("should count a batch once when it starts before the period and continues into it", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		id := d.createRecipe(t, "カレー", onionIng(3))
		d.assign(t, shoppingDay(-1), id)
		d.leftover(t, shoppingDay(-1), 2)
		if it := onlyItem(t, d.list(t, 7)); it.Amount != 3 {
			t.Errorf("want 3, got %v", it.Amount)
		}
	})

	t.Run("should mark started before period when a batch has a row before period start", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		id := d.createRecipe(t, "カレー", onionIng(3))
		d.assign(t, shoppingDay(-1), id)
		d.leftover(t, shoppingDay(-1), 2)
		got := d.list(t, 7)
		if len(got.Dishes) != 1 || !got.Dishes[0].StartedBeforePeriod {
			t.Errorf("dishes: %#v", got.Dishes)
		}
	})

	t.Run("should exclude ingredients of a cooked batch", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		nikujaga := d.createRecipe(t, "肉じゃが", onionIng(2))
		curry := d.createRecipe(t, "カレー", onionIng(3))
		d.assign(t, shoppingDay(0), nikujaga)
		d.assign(t, shoppingDay(2), curry)
		if err := d.meal.SetCooked(ctx, entryIDAt(t, d.meal, shoppingDay(0), "dinner", 0), true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		it := onlyItem(t, d.list(t, 7))
		if it.Amount != 3 || len(it.Recipes) != 1 || it.Recipes[0] != "カレー" {
			t.Errorf("want 3 for カレー only, got %#v", it)
		}
	})

	t.Run("should keep a cooked batch in dishes with cooked flag", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		id := d.createRecipe(t, "カレー", onionIng(3))
		d.assign(t, shoppingDay(-1), id)
		d.leftover(t, shoppingDay(-1), 2)
		if err := d.meal.SetCooked(ctx, entryIDAt(t, d.meal, shoppingDay(-1), "dinner", 0), true); err != nil {
			t.Fatalf("SetCooked: %v", err)
		}
		got := d.list(t, 7)
		if len(got.Items) != 0 {
			t.Errorf("want no items, got %#v", got.Items)
		}
		if len(got.Dishes) != 1 || !got.Dishes[0].Cooked {
			t.Errorf("dishes: %#v", got.Dishes)
		}
	})

	t.Run("should mark covered when non-expired stock with same name and unit is enough", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", onionIng(2)))
		d.addStock(t, "玉ねぎ", 2, "個", shoppingDay(3))
		if it := onlyItem(t, d.list(t, 7)); it.StockStatus != shopping.StockCovered {
			t.Errorf("want covered, got %v", it.StockStatus)
		}
	})

	t.Run("should mark none when the only stock is expired", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", onionIng(2)))
		d.addStock(t, "玉ねぎ", 5, "個", shoppingDay(-2))
		if it := onlyItem(t, d.list(t, 7)); it.StockStatus != shopping.StockNone {
			t.Errorf("want none, got %v", it.StockStatus)
		}
	})

	t.Run("should count stock that expires today", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", onionIng(2)))
		d.addStock(t, "玉ねぎ", 2, "個", shoppingDay(0))
		if it := onlyItem(t, d.list(t, 7)); it.StockStatus != shopping.StockCovered {
			t.Errorf("want covered, got %v", it.StockStatus)
		}
	})

	t.Run("should count stock that has no expiry", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", onionIng(2)))
		d.addStock(t, "玉ねぎ", 1, "個", "")
		if it := onlyItem(t, d.list(t, 7)); it.StockStatus != shopping.StockPartial {
			t.Errorf("want partial, got %v", it.StockStatus)
		}
	})

	t.Run("should merge measuring-unit ingredients into one unquantified row when the same name uses tablespoons and teaspoons", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", recipe.RecipeIngredient{Name: "醤油", Amount: 2, Unit: "大さじ"}))
		d.assign(t, shoppingDay(2), d.createRecipe(t, "生姜焼き", recipe.RecipeIngredient{Name: "醤油", Amount: 1, Unit: "小さじ"}))
		it := onlyItem(t, d.list(t, 7))
		if it.Name != "醤油" || it.Unit != "" || it.Amount != 0 || len(it.Recipes) != 2 {
			t.Errorf("got %#v", it)
		}
	})

	t.Run("should mark covered when a measuring-unit ingredient has non-expired stock with a different unit", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", recipe.RecipeIngredient{Name: "醤油", Amount: 2, Unit: "大さじ"}))
		d.addStock(t, "醤油", 1, "本", "")
		if it := onlyItem(t, d.list(t, 7)); it.StockStatus != shopping.StockCovered {
			t.Errorf("want covered, got %v", it.StockStatus)
		}
	})

	t.Run("should merge ingredients when names differ only in hiragana and katakana", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", recipe.RecipeIngredient{Name: "たまねぎ", Amount: 2, Unit: "個"}))
		d.assign(t, shoppingDay(2), d.createRecipe(t, "カレー", recipe.RecipeIngredient{Name: "タマネギ", Amount: 1, Unit: "個"}))
		it := onlyItem(t, d.list(t, 7))
		if it.Name != "たまねぎ" || it.Unit != "個" || it.Amount != 3 || len(it.Recipes) != 2 {
			t.Errorf("got %#v", it)
		}
	})

	t.Run("should count stock when stock name differs only in width or kana", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "カレー", recipe.RecipeIngredient{Name: "タマネギ", Amount: 2, Unit: "個"}))
		d.addStock(t, "ﾀﾏﾈｷﾞ", 3, "個", "")
		it := onlyItem(t, d.list(t, 7))
		if it.Name != "タマネギ" || it.StockStatus != shopping.StockCovered {
			t.Errorf("got %#v", it)
		}
	})

	t.Run("should keep separate rows when names differ in kanji and kana", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "肉じゃが", onionIng(2)))
		d.assign(t, shoppingDay(2), d.createRecipe(t, "カレー", recipe.RecipeIngredient{Name: "たまねぎ", Amount: 1, Unit: "個"}))
		if got := d.list(t, 7).Items; len(got) != 2 {
			t.Errorf("want 2 items, got %#v", got)
		}
	})

	t.Run("should return dishes and empty items when planned recipes have no ingredients", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		d.assign(t, shoppingDay(0), d.createRecipe(t, "ご飯"))
		got := d.list(t, 7)
		if len(got.Dishes) != 1 || len(got.Items) != 0 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should return ErrInvalidDate when period start is not a valid date", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		if _, err := d.svc.List(ctx, "2026-02-30", 7); !errors.Is(err, mealplan.ErrInvalidDate) {
			t.Errorf("want ErrInvalidDate, got %v", err)
		}
	})

	t.Run("should return ErrInvalidPeriodDays when period days is 5", func(t *testing.T) {
		d := newShoppingServiceForTest(t)
		if _, err := d.svc.List(ctx, "2026-09-21", 5); !errors.Is(err, shopping.ErrInvalidPeriodDays) {
			t.Errorf("want ErrInvalidPeriodDays, got %v", err)
		}
	})
}
