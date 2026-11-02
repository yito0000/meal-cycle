package suggestion

import (
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func TestByTime(t *testing.T) {
	r := func(name string, minutes int) recipe.Recipe {
		return recipe.Recipe{ID: "id-" + name, Name: name, CookMinutes: minutes}
	}
	archived := r("削除済み", 10)
	archived.ArchivedAt = "2026-10-01T00:00:00Z"

	cases := []struct {
		name    string
		recipes []recipe.Recipe
		max     int
		limit   int
		want    []string
	}{
		{"maxMinutes 以下のレシピだけを返す", []recipe.Recipe{r("豚キムチ", 15), r("肉じゃが", 40), r("冷奴", 20)}, 20, 5, []string{"豚キムチ", "冷奴"}},
		{"所要時間 0 のレシピを返さない", []recipe.Recipe{r("豚汁", 0), r("冷奴", 5)}, 20, 5, []string{"冷奴"}},
		{"削除済みのレシピを返さない", []recipe.Recipe{archived, r("冷奴", 5)}, 20, 5, []string{"冷奴"}},
		{"所要時間の昇順、同じならレシピ名の昇順に並べる", []recipe.Recipe{r("豚キムチ", 15), r("b", 10), r("a", 10)}, 20, 5, []string{"a", "b", "豚キムチ"}},
		{"limit 件で切る", []recipe.Recipe{r("a", 1), r("b", 2), r("c", 3)}, 20, 2, []string{"a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ByTime(tc.recipes, tc.max, tc.limit)
			names := make([]string, 0, len(got))
			for _, c := range got {
				names = append(names, c.RecipeName)
			}
			if !reflect.DeepEqual(names, tc.want) {
				t.Errorf("got %v, want %v", names, tc.want)
			}
		})
	}

	t.Run("根拠として所要時間を持つ", func(t *testing.T) {
		got := ByTime([]recipe.Recipe{r("冷奴", 5)}, 20, 5)
		if len(got) != 1 || got[0].CookMinutes != 5 {
			t.Errorf("got %#v", got)
		}
	})
}
