package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/config"
	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
	"github.com/yito0000/meal-cycle/internal/storage/sqlite"
)

// fridgeToday は期限判定の基準日。テストの期限日はここからの相対で書く。
var fridgeToday = time.Date(2026, 9, 23, 10, 0, 0, 0, time.FixedZone("JST", 9*60*60))

func newFridgeServiceForTest(t *testing.T) (*FridgeService, *RecipeService) {
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
	svc := NewFridgeService(db, staticConfig(config.Defaults()))
	svc.now = func() time.Time { return fridgeToday }
	return svc, NewRecipeService(db, staticConfig(config.Defaults()), nil)
}

func daysLater(n int) string {
	return fridgeToday.AddDate(0, 0, n).Format(fridge.DateLayout)
}

func validStock(name string) fridge.StockItem {
	return fridge.StockItem{Name: name, Amount: 1, Unit: "個"}
}

func TestFridgeService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should return items with urgent status when expiry is within red days", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		in := validStock("牛乳")
		in.ExpiresAt = daysLater(2)
		if _, err := svc.Add(ctx, in); err != nil {
			t.Fatalf("Add: %v", err)
		}
		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].ExpiryStatus != fridge.ExpiryUrgent || got[0].DaysLeft == nil || *got[0].DaysLeft != 2 {
			t.Errorf("want urgent with 2 days left, got %#v", got)
		}
	})

	t.Run("should return items with none status and nil daysLeft when expiry is empty", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		if _, err := svc.Add(ctx, validStock("米")); err != nil {
			t.Fatalf("Add: %v", err)
		}
		got, _ := svc.List(ctx)
		if len(got) != 1 || got[0].ExpiryStatus != fridge.ExpiryNone || got[0].DaysLeft != nil {
			t.Errorf("want none with nil daysLeft, got %#v", got)
		}
	})

	t.Run("should return expired status when expiry is in the past", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		past := validStock("豆腐")
		past.ExpiresAt = daysLater(-1)
		today := validStock("ヨーグルト")
		today.ExpiresAt = daysLater(0)
		for _, in := range []fridge.StockItem{past, today} {
			if _, err := svc.Add(ctx, in); err != nil {
				t.Fatalf("Add %s: %v", in.Name, err)
			}
		}
		got, _ := svc.List(ctx)
		status := map[string]fridge.ExpiryStatus{}
		for _, item := range got {
			status[item.Name] = item.ExpiryStatus
		}
		if status["豆腐"] != fridge.ExpiryExpired || status["ヨーグルト"] != fridge.ExpiryUrgent {
			t.Errorf("want 豆腐=expired ヨーグルト=urgent, got %v", status)
		}
	})

	t.Run("should use config thresholds when evaluating status", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		cfg := config.Defaults()
		cfg.ExpiryAlertRedDays, cfg.ExpiryAlertYellowDays = 1, 2
		svc.cfg = staticConfig(cfg)
		in := validStock("豚肉")
		in.ExpiresAt = daysLater(3)
		if _, err := svc.Add(ctx, in); err != nil {
			t.Fatalf("Add: %v", err)
		}
		got, _ := svc.List(ctx)
		if got[0].ExpiryStatus != fridge.ExpiryFresh {
			t.Errorf("want fresh with custom thresholds, got %q", got[0].ExpiryStatus)
		}
	})
}

func TestFridgeService_Add(t *testing.T) {
	ctx := context.Background()

	t.Run("should trim name and unit when input has surrounding spaces", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		got, err := svc.Add(ctx, fridge.StockItem{Name: "  玉ねぎ ", Amount: 3, Unit: " 個　"})
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		if got.Name != "玉ねぎ" || got.Unit != "個" {
			t.Errorf("want trimmed, got name=%q unit=%q", got.Name, got.Unit)
		}
	})

	t.Run("should return ErrStockNameEmpty when name is blank", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		_, err := svc.Add(ctx, fridge.StockItem{Name: "   ", Amount: 1})
		if !errors.Is(err, fridge.ErrStockNameEmpty) {
			t.Errorf("want ErrStockNameEmpty, got %v", err)
		}
	})

	t.Run("should return ErrStockNameTooLong when name exceeds 100 runes", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		if _, err := svc.Add(ctx, validStock(strings.Repeat("あ", 100))); err != nil {
			t.Fatalf("100 runes should pass: %v", err)
		}
		_, err := svc.Add(ctx, validStock(strings.Repeat("あ", 101)))
		if !errors.Is(err, fridge.ErrStockNameTooLong) {
			t.Errorf("want ErrStockNameTooLong, got %v", err)
		}
	})

	t.Run("should return ErrStockAmountOutOfRange when amount is zero, negative, or above 99999", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		for _, amount := range []float64{0, -1, 99999.5} {
			in := validStock("卵")
			in.Amount = amount
			if _, err := svc.Add(ctx, in); !errors.Is(err, fridge.ErrStockAmountOutOfRange) {
				t.Errorf("amount %v: want ErrStockAmountOutOfRange, got %v", amount, err)
			}
		}
		for _, amount := range []float64{0.5, 99999} {
			in := validStock("卵")
			in.Amount = amount
			if _, err := svc.Add(ctx, in); err != nil {
				t.Errorf("amount %v should pass: %v", amount, err)
			}
		}
	})

	t.Run("should return ErrStockUnitTooLong when unit exceeds 20 runes", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		in := validStock("卵")
		in.Unit = strings.Repeat("個", 21)
		if _, err := svc.Add(ctx, in); !errors.Is(err, fridge.ErrStockUnitTooLong) {
			t.Errorf("want ErrStockUnitTooLong, got %v", err)
		}
	})

	t.Run("should return ErrStockInvalidExpiry when expiry is not a valid date", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		for _, exp := range []string{"2026-02-30", "2026/09/30", "tomorrow"} {
			in := validStock("卵")
			in.ExpiresAt = exp
			if _, err := svc.Add(ctx, in); !errors.Is(err, fridge.ErrStockInvalidExpiry) {
				t.Errorf("expiry %q: want ErrStockInvalidExpiry, got %v", exp, err)
			}
		}
	})

	t.Run("should accept past expiry date", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		in := validStock("豆腐")
		in.ExpiresAt = daysLater(-2)
		got, err := svc.Add(ctx, in)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		if got.ExpiryStatus != fridge.ExpiryExpired || got.DaysLeft == nil || *got.DaysLeft != -2 {
			t.Errorf("want expired with -2 days, got %#v", got)
		}
	})
}

func TestFridgeService_AddMany(t *testing.T) {
	ctx := context.Background()

	t.Run("should add all items with trimmed name and unit when every input is valid", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		got, err := svc.AddMany(ctx, []fridge.StockItem{
			{Name: " 玉ねぎ ", Amount: 3, Unit: " 個 "},
			{Name: "豚ロース", Amount: 200, Unit: "g"},
		})
		if err != nil {
			t.Fatalf("AddMany: %v", err)
		}
		if len(got) != 2 || got[0].Name != "玉ねぎ" || got[0].Unit != "個" {
			t.Errorf("want trimmed 2 items, got %#v", got)
		}
		list, _ := svc.List(ctx)
		if len(list) != 2 {
			t.Errorf("want 2 rows, got %d", len(list))
		}
	})

	t.Run("should return added items with expiry status evaluated when expiry is set", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		got, err := svc.AddMany(ctx, []fridge.StockItem{
			{Name: "牛乳", Amount: 1, Unit: "本", ExpiresAt: daysLater(1)},
			{Name: "米", Amount: 5, Unit: "kg"},
		})
		if err != nil {
			t.Fatalf("AddMany: %v", err)
		}
		if got[0].ExpiryStatus != fridge.ExpiryUrgent || got[0].DaysLeft == nil || *got[0].DaysLeft != 1 {
			t.Errorf("want urgent with 1 day left, got %#v", got[0])
		}
		if got[1].ExpiryStatus != fridge.ExpiryNone || got[1].DaysLeft != nil {
			t.Errorf("want none with nil daysLeft, got %#v", got[1])
		}
	})

	t.Run("should not add any item when one of the inputs has an empty name", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		_, err := svc.AddMany(ctx, []fridge.StockItem{validStock("玉ねぎ"), {Name: "  ", Amount: 1}})
		if !errors.Is(err, fridge.ErrStockNameEmpty) {
			t.Errorf("want ErrStockNameEmpty, got %v", err)
		}
		list, _ := svc.List(ctx)
		if len(list) != 0 {
			t.Errorf("want 0 rows, got %d", len(list))
		}
	})

	t.Run("should wrap ErrStockAmountOutOfRange with the row index when the second input has zero amount", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		bad := validStock("塩")
		bad.Amount = 0
		_, err := svc.AddMany(ctx, []fridge.StockItem{validStock("玉ねぎ"), bad})
		if !errors.Is(err, fridge.ErrStockAmountOutOfRange) {
			t.Fatalf("want ErrStockAmountOutOfRange, got %v", err)
		}
		if !strings.Contains(err.Error(), "row 1") {
			t.Errorf("want row index in message, got %q", err.Error())
		}
	})

	t.Run("should not merge into existing stock when the same name and unit already exist", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		if _, err := svc.Add(ctx, fridge.StockItem{Name: "玉ねぎ", Amount: 1, Unit: "個"}); err != nil {
			t.Fatalf("Add: %v", err)
		}
		if _, err := svc.AddMany(ctx, []fridge.StockItem{{Name: "玉ねぎ", Amount: 2, Unit: "個"}}); err != nil {
			t.Fatalf("AddMany: %v", err)
		}
		list, _ := svc.List(ctx)
		if len(list) != 2 {
			t.Fatalf("want 2 rows, got %d", len(list))
		}
		amounts := map[float64]bool{}
		for _, it := range list {
			amounts[it.Amount] = true
		}
		if !amounts[1] || !amounts[2] {
			t.Errorf("want amounts 1 and 2, got %#v", list)
		}
	})

	t.Run("should return empty slice without error when inputs are empty", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		got, err := svc.AddMany(ctx, []fridge.StockItem{})
		if err != nil {
			t.Fatalf("AddMany: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("want empty non-nil slice, got %#v", got)
		}
	})
}

func TestFridgeService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("should keep addedAt when item is updated", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		created, _ := svc.Add(ctx, validStock("玉ねぎ"))
		in := created
		in.Amount = 2
		got, err := svc.Update(ctx, created.ID, in)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if got.Amount != 2 || got.AddedAt != created.AddedAt {
			t.Errorf("want amount 2 and same addedAt, got %#v", got)
		}
	})

	t.Run("should return ErrStockNotFound when id does not exist", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		if _, err := svc.Update(ctx, "missing", validStock("x")); !errors.Is(err, fridge.ErrStockNotFound) {
			t.Errorf("want ErrStockNotFound, got %v", err)
		}
	})
}

func TestFridgeService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove item when id exists", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		created, _ := svc.Add(ctx, validStock("卵"))
		if err := svc.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		got, _ := svc.List(ctx)
		if len(got) != 0 {
			t.Errorf("want empty, got %#v", got)
		}
	})
}

func TestFridgeService_Restore(t *testing.T) {
	ctx := context.Background()

	t.Run("should recreate item with same id and addedAt when restoring deleted item", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		in := validStock("玉ねぎ")
		in.ExpiresAt = daysLater(7)
		created, _ := svc.Add(ctx, in)
		if err := svc.Delete(ctx, created.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		got, err := svc.Restore(ctx, created)
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		if got.ID != created.ID || got.AddedAt != created.AddedAt || got.ExpiryStatus != fridge.ExpirySoon {
			t.Errorf("want same id / addedAt with soon status, got %#v", got)
		}
	})

	t.Run("should return ErrStockInvalidRestore when id is empty", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		in := validStock("玉ねぎ")
		in.AddedAt = "2026-09-01T00:00:00Z"
		if _, err := svc.Restore(ctx, in); !errors.Is(err, fridge.ErrStockInvalidRestore) {
			t.Errorf("want ErrStockInvalidRestore, got %v", err)
		}
	})

	t.Run("should return ErrStockInvalidRestore when addedAt is not RFC3339", func(t *testing.T) {
		svc, _ := newFridgeServiceForTest(t)
		in := validStock("玉ねぎ")
		in.ID = "id-1"
		in.AddedAt = "2026-09-01"
		if _, err := svc.Restore(ctx, in); !errors.Is(err, fridge.ErrStockInvalidRestore) {
			t.Errorf("want ErrStockInvalidRestore, got %v", err)
		}
	})
}

func TestFridgeService_ListSuggestions(t *testing.T) {
	ctx := context.Background()

	t.Run("should include recipe ingredient names when stock is empty", func(t *testing.T) {
		svc, recipes := newFridgeServiceForTest(t)
		in := validRecipe("きのこパスタ")
		in.Ingredients = []recipe.RecipeIngredient{{Name: "しめじ", Amount: 1, Unit: "パック"}}
		if _, err := recipes.Create(ctx, in); err != nil {
			t.Fatalf("create recipe: %v", err)
		}
		got, err := svc.ListSuggestions(ctx)
		if err != nil {
			t.Fatalf("ListSuggestions: %v", err)
		}
		if len(got.Names) != 1 || got.Names[0] != (fridge.NameSuggestion{Name: "しめじ", Unit: "パック"}) {
			t.Errorf("want [しめじ/パック], got %#v", got.Names)
		}
	})
}
