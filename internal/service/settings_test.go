package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/domain/settings"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// staticConfig は設定サービスを通さずに固定の Config を渡すテスト用の configSource。
type staticConfig config.Config

func (c staticConfig) current() config.Config { return config.Config(c) }

func openSettingsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

func seedSettingRows(t *testing.T, db *sql.DB, rows map[string]string) {
	t.Helper()
	raw := make(map[string]json.RawMessage, len(rows))
	for k, v := range rows {
		raw[k] = json.RawMessage(v)
	}
	if err := sqlite.NewSettingsRepo(db).ReplaceAll(context.Background(), raw, time.Now()); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
}

func defaultValues() settings.Values {
	return settings.FromConfig(config.Defaults())
}

func mustGet(t *testing.T, svc *SettingsService) settings.Values {
	t.Helper()
	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return got
}

func mustSave(t *testing.T, svc *SettingsService, v settings.Values) settings.Values {
	t.Helper()
	got, err := svc.Save(context.Background(), v)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	return got
}

func TestSettingsService_Get(t *testing.T) {
	t.Run("should return default values when nothing is saved", func(t *testing.T) {
		svc := NewSettingsService(openSettingsDB(t))
		if got := mustGet(t, svc); !reflect.DeepEqual(got, defaultValues()) {
			t.Errorf("got %#v, want defaults", got)
		}
	})

	t.Run("should fall back to the default only for a key whose saved value cannot be decoded", func(t *testing.T) {
		db := openSettingsDB(t)
		seedSettingRows(t, db, map[string]string{
			"expiryAlert":    `{"redDays":"abc","yellowDays":7}`,
			"measuringUnits": `["大さじ","小さじ","カップ","合"]`,
		})
		got := mustGet(t, NewSettingsService(db))
		if got.ExpiryAlertRedDays != 3 {
			t.Errorf("red days = %d, want 3", got.ExpiryAlertRedDays)
		}
		if want := []string{"大さじ", "小さじ", "カップ", "合"}; !reflect.DeepEqual(got.MeasuringUnits, want) {
			t.Errorf("measuring units = %v, want %v", got.MeasuringUnits, want)
		}
	})

	t.Run("should fall back to the default for a key whose saved value fails validation", func(t *testing.T) {
		db := openSettingsDB(t)
		seedSettingRows(t, db, map[string]string{
			"expiryAlert":                 `{"redDays":7,"yellowDays":7}`,
			"recipeImportDefaultServings": `4`,
		})
		got := mustGet(t, NewSettingsService(db))
		if got.ExpiryAlertRedDays != 3 || got.ExpiryAlertYellowDays != 7 {
			t.Errorf("expiry = %d/%d, want 3/7", got.ExpiryAlertRedDays, got.ExpiryAlertYellowDays)
		}
		if got.RecipeImportDefaultServings != 4 {
			t.Errorf("servings = %d, want 4", got.RecipeImportDefaultServings)
		}
	})

	t.Run("should ignore unknown keys", func(t *testing.T) {
		db := openSettingsDB(t)
		seedSettingRows(t, db, map[string]string{"futureKey": `{"x":1}`, "language": `"en"`})
		got := mustGet(t, NewSettingsService(db))
		want := defaultValues()
		want.Language = "en"
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})
}

func TestSettingsService_Save(t *testing.T) {
	t.Run("should persist values so that a new service on the same database returns them", func(t *testing.T) {
		db := openSettingsDB(t)
		v := defaultValues()
		v.ExpiryAlertRedDays = 2
		v.Language = "en"
		mustSave(t, NewSettingsService(db), v)

		got := mustGet(t, NewSettingsService(db))
		if got.ExpiryAlertRedDays != 2 || got.Language != "en" {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should store only keys that differ from the defaults", func(t *testing.T) {
		db := openSettingsDB(t)
		v := defaultValues()
		v.ExpiryAlertRedDays = 2
		mustSave(t, NewSettingsService(db), v)

		rows, err := sqlite.NewSettingsRepo(db).Load(context.Background())
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(rows) != 1 || string(rows["expiryAlert"]) != `{"redDays":2,"yellowDays":7}` {
			t.Errorf("rows = %v", rows)
		}
	})

	t.Run("should reject empty leftover copy presets and keep the previous values", func(t *testing.T) {
		svc := NewSettingsService(openSettingsDB(t))
		v := defaultValues()
		v.LeftoverCopyPresets = []int{}
		if _, err := svc.Save(context.Background(), v); !errors.Is(err, settings.ErrLeftoverPresetsCount) {
			t.Fatalf("err = %v, want ErrLeftoverPresetsCount", err)
		}
		if got := mustGet(t, svc); !reflect.DeepEqual(got, defaultValues()) {
			t.Errorf("values changed: %#v", got)
		}
	})

	t.Run("should reject red days not below yellow days and keep the previous values", func(t *testing.T) {
		svc := NewSettingsService(openSettingsDB(t))
		v := defaultValues()
		v.ExpiryAlertRedDays, v.ExpiryAlertYellowDays = 7, 7
		if _, err := svc.Save(context.Background(), v); !errors.Is(err, settings.ErrRedNotBelowYellow) {
			t.Fatalf("err = %v, want ErrRedNotBelowYellow", err)
		}
		if got := svc.current(); got.ExpiryAlertRedDays != 3 || got.ExpiryAlertYellowDays != 7 {
			t.Errorf("current expiry = %d/%d, want 3/7", got.ExpiryAlertRedDays, got.ExpiryAlertYellowDays)
		}
	})

	t.Run("should return normalized values with sorted presets and trimmed units", func(t *testing.T) {
		svc := NewSettingsService(openSettingsDB(t))
		v := defaultValues()
		v.LeftoverCopyPresets = []int{5, 2}
		v.MeasuringUnits = []string{" 大さじ ", "合 "}
		got := mustSave(t, svc, v)
		if want := []int{2, 5}; !reflect.DeepEqual(got.LeftoverCopyPresets, want) {
			t.Errorf("presets = %v, want %v", got.LeftoverCopyPresets, want)
		}
		if want := []string{"大さじ", "合"}; !reflect.DeepEqual(got.MeasuringUnits, want) {
			t.Errorf("units = %v, want %v", got.MeasuringUnits, want)
		}
	})
}

func TestSettingsService_Reset(t *testing.T) {
	t.Run("should return default values and remove all saved rows", func(t *testing.T) {
		db := openSettingsDB(t)
		svc := NewSettingsService(db)
		v := defaultValues()
		v.ExpiryAlertRedDays = 2
		v.MeasuringUnits = append(v.MeasuringUnits, "合")
		mustSave(t, svc, v)

		got, err := svc.Reset(context.Background())
		if err != nil {
			t.Fatalf("Reset: %v", err)
		}
		if !reflect.DeepEqual(got, defaultValues()) {
			t.Errorf("got %#v, want defaults", got)
		}
		rows, err := sqlite.NewSettingsRepo(db).Load(context.Background())
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if len(rows) != 0 {
			t.Errorf("rows = %v, want none", rows)
		}
	})
}

type propagationDeps struct {
	settings *SettingsService
	fridge   *FridgeService
	meal     *MealPlanService
	recipe   *RecipeService
	shopping shoppingDeps
}

func newPropagationDeps(t *testing.T) propagationDeps {
	t.Helper()
	db := openSettingsDB(t)
	st := NewSettingsService(db)
	fr := NewFridgeService(db, st)
	fr.now = func() time.Time { return fridgeToday }
	sh := NewShoppingService(db, st)
	sh.now = func() time.Time { return shoppingToday }
	shFridge := NewFridgeService(db, st)
	shFridge.now = func() time.Time { return shoppingToday }
	rc := NewRecipeService(db, st)
	meal := NewMealPlanService(sqlite.NewMealPlanRepo(db), st)
	return propagationDeps{
		settings: st,
		fridge:   fr,
		meal:     meal,
		recipe:   rc,
		shopping: shoppingDeps{svc: sh, recipe: rc, meal: meal, fridge: shFridge},
	}
}

func (d propagationDeps) save(t *testing.T, modify func(v *settings.Values)) {
	t.Helper()
	v := mustGet(t, d.settings)
	modify(&v)
	mustSave(t, d.settings, v)
}

func (d propagationDeps) previewOne(t *testing.T, content string) recipe.ImportCandidate {
	t.Helper()
	got, err := d.recipe.PreviewImport(context.Background(), []recipe.ImportFile{{Name: "a.md", Content: content}})
	if err != nil {
		t.Fatalf("PreviewImport: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 candidate, got %d", len(got))
	}
	return got[0]
}

func TestSettingsService_Propagation(t *testing.T) {
	ctx := context.Background()

	t.Run("should evaluate fridge expiry with the saved red days without restarting", func(t *testing.T) {
		d := newPropagationDeps(t)
		if _, err := d.fridge.Add(ctx, fridge.StockItem{Name: "A", Amount: 1, Unit: "個", ExpiresAt: daysLater(5)}); err != nil {
			t.Fatalf("Add: %v", err)
		}
		d.save(t, func(v *settings.Values) { v.ExpiryAlertRedDays = 5 })

		items, err := d.fridge.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(items) != 1 || items[0].ExpiryStatus != fridge.ExpiryUrgent {
			t.Errorf("got %#v, want red", items)
		}
	})

	t.Run("should accept a leftover copy of days added to the presets and reject removed days", func(t *testing.T) {
		d := newPropagationDeps(t)
		r, err := d.recipe.Create(ctx, recipe.Recipe{Name: "カレー", Servings: 1})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := d.meal.AssignRecipe(ctx, "2026-08-24", "dinner", r.ID); err != nil {
			t.Fatalf("AssignRecipe: %v", err)
		}
		d.save(t, func(v *settings.Values) { v.LeftoverCopyPresets = []int{2, 5} })

		if err := d.meal.CopyLeftover(ctx, "2026-08-24", "dinner", 5, string(mealplan.CopyModeAppend), true); err != nil {
			t.Errorf("copy 5 days: %v", err)
		}
		err = d.meal.CopyLeftover(ctx, "2026-08-24", "dinner", 3, string(mealplan.CopyModeAppend), true)
		if !errors.Is(err, mealplan.ErrInvalidCopyDays) {
			t.Errorf("copy 3 days: err = %v, want ErrInvalidCopyDays", err)
		}
	})

	t.Run("should accept a shopping period of days added to the presets", func(t *testing.T) {
		d := newPropagationDeps(t)
		d.save(t, func(v *settings.Values) { v.ShoppingPeriodDayPresets = append(v.ShoppingPeriodDayPresets, 10) })

		got, err := d.shopping.svc.List(ctx, shoppingStart, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if got.PeriodDays != 10 {
			t.Errorf("period days = %d, want 10", got.PeriodDays)
		}
	})

	t.Run("should import an unquantified word added to the settings as amount 0 with the word as unit", func(t *testing.T) {
		d := newPropagationDeps(t)
		d.save(t, func(v *settings.Values) { v.UnquantifiedWords = append(v.UnquantifiedWords, "ひとかけ") })

		c := d.previewOne(t, "---\nname: 生姜焼き\nservings: 1\ningredients:\n  - name: 生姜\n    amount: ひとかけ\n---\n")
		if c.Issue != recipe.ImportIssueNone {
			t.Fatalf("issue = %q (%s)", c.Issue, c.IssueArg)
		}
		ings := c.Recipe.Ingredients
		if len(ings) != 1 || ings[0].Name != "生姜" || ings[0].Amount != 0 || ings[0].Unit != "ひとかけ" {
			t.Errorf("ingredients = %#v", ings)
		}
	})

	t.Run("should import a recipe without servings using the saved default servings", func(t *testing.T) {
		d := newPropagationDeps(t)
		d.save(t, func(v *settings.Values) { v.RecipeImportDefaultServings = 4 })

		c := d.previewOne(t, "---\nname: カレー\n---\n")
		if c.Issue != recipe.ImportIssueNone {
			t.Fatalf("issue = %q (%s)", c.Issue, c.IssueArg)
		}
		if c.Recipe.Servings != 4 {
			t.Errorf("servings = %d, want 4", c.Recipe.Servings)
		}
	})

	t.Run("should treat a measuring unit added to the settings as unquantified in the shopping list", func(t *testing.T) {
		d := newPropagationDeps(t)
		d.save(t, func(v *settings.Values) { v.MeasuringUnits = append(v.MeasuringUnits, "合") })
		sd := d.shopping
		sd.assign(t, shoppingDay(0), sd.createRecipe(t, "炊き込みご飯", recipe.RecipeIngredient{Name: "米", Amount: 2, Unit: "合"}))

		it := onlyItem(t, sd.list(t, 7))
		if it.Name != "米" || it.Unit != "" || it.Amount != 0 {
			t.Errorf("got %#v", it)
		}
		if !slices.Contains(d.settings.current().MeasuringUnits, "合") {
			t.Errorf("current measuring units missing 合")
		}
	})
}
