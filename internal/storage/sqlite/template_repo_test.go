package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func newTemplateRepoForTest(t *testing.T) (*TemplateRepo, *RecipeRepo) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewTemplateRepo(db), NewRecipeRepo(db)
}

func TestTemplateRepo(t *testing.T) {
	ctx := context.Background()
	recipeID := func(t *testing.T, rr *RecipeRepo, name string) string {
		t.Helper()
		r, err := rr.Create(ctx, recipe.Recipe{Name: name, Servings: 1})
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}

	t.Run("should read back a saved template with its items", func(t *testing.T) {
		repo, rr := newTemplateRepoForTest(t)
		a := recipeID(t, rr, "肉じゃが")
		created, err := repo.Create(ctx, "煮物の週", []mealplan.TemplateItem{
			{DayOffset: 0, Meal: mealplan.MealDinner, Position: 0, RecipeID: a, GroupNo: 0},
			{DayOffset: 1, Meal: mealplan.MealDinner, Position: 0, RecipeID: a, GroupNo: 0},
		})
		if err != nil {
			t.Fatal(err)
		}
		got, err := repo.Get(ctx, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "煮物の週" || len(got.Items) != 2 || got.Items[1].DayOffset != 1 ||
			got.Items[0].RecipeName != "肉じゃが" || got.Items[0].Archived {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should list templates newest first with their items", func(t *testing.T) {
		repo, rr := newTemplateRepoForTest(t)
		a := recipeID(t, rr, "肉じゃが")
		item := []mealplan.TemplateItem{{DayOffset: 0, Meal: mealplan.MealDinner, RecipeID: a}}
		if _, err := repo.Create(ctx, "古い", item); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Create(ctx, "新しい", item); err != nil {
			t.Fatal(err)
		}
		got, err := repo.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Name != "新しい" || got[1].Name != "古い" || len(got[0].Items) != 1 {
			t.Errorf("got %#v", got)
		}
	})

	t.Run("should delete items together with the template", func(t *testing.T) {
		repo, rr := newTemplateRepoForTest(t)
		a := recipeID(t, rr, "肉じゃが")
		created, err := repo.Create(ctx, "煮物の週", []mealplan.TemplateItem{{DayOffset: 0, Meal: mealplan.MealDinner, RecipeID: a}})
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Delete(ctx, created.ID); err != nil {
			t.Fatal(err)
		}
		var n int
		if err := repo.db.QueryRow(`SELECT COUNT(*) FROM meal_plan_template_items`).Scan(&n); err != nil || n != 0 {
			t.Errorf("want items deleted, got %d (%v)", n, err)
		}
		if _, err := repo.Get(ctx, created.ID); !errors.Is(err, mealplan.ErrTemplateNotFound) {
			t.Errorf("want ErrTemplateNotFound, got %v", err)
		}
	})

	t.Run("should compare names without surrounding spaces in ExistsName", func(t *testing.T) {
		repo, rr := newTemplateRepoForTest(t)
		a := recipeID(t, rr, "肉じゃが")
		if _, err := repo.Create(ctx, "煮物の週", []mealplan.TemplateItem{{DayOffset: 0, Meal: mealplan.MealDinner, RecipeID: a}}); err != nil {
			t.Fatal(err)
		}
		if ok, err := repo.ExistsName(ctx, "  煮物の週 "); err != nil || !ok {
			t.Errorf("want exists, got %v (%v)", ok, err)
		}
		if ok, _ := repo.ExistsName(ctx, "カレーの週"); ok {
			t.Error("want not exists")
		}
	})
}
