package sqlite

import (
	"context"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/suggestion"
)

func TestSuggestionRepo(t *testing.T) {
	ctx := context.Background()
	const today = "2026-10-15"

	// seed は meal_plans の行を (id, date, recipe, batch) で直接入れ、cooked の組に作った印を付ける。
	setup := func(t *testing.T, plans [][4]string, cooked [][2]string, archived ...string) map[string]suggestion.RecipeHistory {
		t.Helper()
		_, db := newStockRepoForTest(t)
		for _, name := range []string{"r1", "r2", "r3"} {
			if _, err := db.Exec(`INSERT INTO recipes (id, name, created_at, updated_at) VALUES (?, ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
				name, name); err != nil {
				t.Fatalf("seed recipe: %v", err)
			}
		}
		for _, id := range archived {
			if _, err := db.Exec(`UPDATE recipes SET archived_at = '2026-10-01T00:00:00Z' WHERE id = ?`, id); err != nil {
				t.Fatalf("archive: %v", err)
			}
		}
		positions := map[string]int{}
		for _, p := range plans {
			pos := positions[p[1]]
			positions[p[1]]++
			if _, err := db.Exec(`INSERT INTO meal_plans (id, date, meal, position, recipe_id, created_at, batch_id)
				VALUES (?, ?, 'dinner', ?, ?, '2026-01-01T00:00:00Z', ?)`, p[0], p[1], pos, p[2], p[3]); err != nil {
				t.Fatalf("seed plan: %v", err)
			}
		}
		for _, c := range cooked {
			if _, err := db.Exec(`INSERT INTO meal_plan_cooked (batch_id, recipe_id, cooked_at) VALUES (?, ?, '2026-01-01T00:00:00Z')`,
				c[0], c[1]); err != nil {
				t.Fatalf("seed cooked: %v", err)
			}
		}
		got, err := NewSuggestionRepo(db).ListRecipeHistory(ctx, today)
		if err != nil {
			t.Fatalf("ListRecipeHistory: %v", err)
		}
		out := map[string]suggestion.RecipeHistory{}
		for _, h := range got {
			out[h.RecipeID] = h
		}
		return out
	}

	t.Run("latest date of cooked rows becomes LastCookedDate", func(t *testing.T) {
		got := setup(t, [][4]string{{"a", "2026-10-01", "r1", "b1"}, {"b", "2026-10-05", "r1", "b2"}},
			[][2]string{{"b1", "r1"}, {"b2", "r1"}})
		if got["r1"].LastCookedDate != "2026-10-05" {
			t.Errorf("got %+v", got["r1"])
		}
	})

	t.Run("rows without cooked mark are not counted", func(t *testing.T) {
		got := setup(t, [][4]string{{"a", "2026-10-01", "r1", "b1"}, {"b", "2026-10-12", "r1", "b2"}},
			[][2]string{{"b1", "r1"}})
		if got["r1"].LastCookedDate != "2026-10-01" {
			t.Errorf("got %+v", got["r1"])
		}
	})

	t.Run("cooked mark of a batch applies to every row of the batch", func(t *testing.T) {
		got := setup(t, [][4]string{{"a", "2026-10-01", "r1", "b1"}, {"b", "2026-10-03", "r1", "b1"}},
			[][2]string{{"b1", "r1"}})
		if got["r1"].LastCookedDate != "2026-10-03" {
			t.Errorf("got %+v", got["r1"])
		}
	})

	t.Run("uncooked row on or after today sets HasUpcomingPlan", func(t *testing.T) {
		got := setup(t, [][4]string{{"a", "2026-10-01", "r1", "b1"}, {"b", today, "r1", "b2"}},
			[][2]string{{"b1", "r1"}})
		if !got["r1"].HasUpcomingPlan {
			t.Errorf("got %+v", got["r1"])
		}
	})

	t.Run("cooked row after today keeps HasUpcomingPlan false", func(t *testing.T) {
		got := setup(t, [][4]string{{"a", "2026-10-20", "r1", "b1"}}, [][2]string{{"b1", "r1"}})
		if got["r1"].HasUpcomingPlan || got["r1"].LastCookedDate != "2026-10-20" {
			t.Errorf("got %+v", got["r1"])
		}
	})

	t.Run("recipe never cooked has empty LastCookedDate", func(t *testing.T) {
		got := setup(t, [][4]string{{"a", "2026-10-01", "r2", "b1"}}, nil)
		if got["r2"].LastCookedDate != "" || got["r3"].LastCookedDate != "" {
			t.Errorf("got %+v / %+v", got["r2"], got["r3"])
		}
	})

	t.Run("archived recipe is not returned", func(t *testing.T) {
		got := setup(t, nil, nil, "r2")
		if _, ok := got["r2"]; ok || len(got) != 2 {
			t.Errorf("got %+v", got)
		}
	})
}
