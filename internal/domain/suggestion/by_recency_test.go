package suggestion

import (
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func TestByRecency(t *testing.T) {
	type view struct {
		Recipe string
		Days   int
	}
	hist := func(name string, daysAgo int, upcoming bool) RecipeHistory {
		return RecipeHistory{RecipeID: "id-" + name, LastCookedDate: day(-daysAgo), HasUpcomingPlan: upcoming}
	}
	cases := []struct {
		name    string
		recipes []recipe.Recipe
		history []RecipeHistory
		limit   int
		want    []view
	}{
		{
			name:    "経過日数がちょうど閾値のレシピは候補になる",
			recipes: []recipe.Recipe{rec("麻婆豆腐")},
			history: []RecipeHistory{hist("麻婆豆腐", 10, false)},
			want:    []view{{"麻婆豆腐", 10}},
		},
		{
			name:    "閾値に 1 日足りないレシピは候補にならない",
			recipes: []recipe.Recipe{rec("麻婆豆腐")},
			history: []RecipeHistory{hist("麻婆豆腐", 9, false)},
			want:    []view{},
		},
		{
			name:    "最終調理日が未来 (経過日数が負) のレシピは候補にならない",
			recipes: []recipe.Recipe{rec("カレー")},
			history: []RecipeHistory{hist("カレー", -2, false)},
			want:    []view{},
		},
		{
			name:    "一度も作っていないレシピは候補にならない",
			recipes: []recipe.Recipe{rec("鮭のホイル焼き"), rec("煮物")},
			history: []RecipeHistory{{RecipeID: "id-鮭のホイル焼き"}},
			want:    []view{},
		},
		{
			name:    "HasUpcomingPlan のレシピは候補にならない",
			recipes: []recipe.Recipe{rec("麻婆豆腐")},
			history: []RecipeHistory{hist("麻婆豆腐", 14, true)},
			want:    []view{},
		},
		{
			name:    "経過日数の降順、レシピ名の昇順に並ぶ",
			recipes: []recipe.Recipe{rec("麻婆豆腐"), rec("鮭のホイル焼き"), rec("カレー"), rec("オムレツ")},
			history: []RecipeHistory{
				hist("麻婆豆腐", 14, false), hist("鮭のホイル焼き", 21, false),
				hist("カレー", 14, false), hist("オムレツ", 30, false),
			},
			want: []view{{"オムレツ", 30}, {"鮭のホイル焼き", 21}, {"カレー", 14}, {"麻婆豆腐", 14}},
		},
		{
			name:    "limit を超える候補は切られる",
			recipes: []recipe.Recipe{rec("A"), rec("B"), rec("C")},
			history: []RecipeHistory{hist("A", 30, false), hist("B", 20, false), hist("C", 15, false)},
			limit:   2,
			want:    []view{{"A", 30}, {"B", 20}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			limit := tc.limit
			if limit == 0 {
				limit = 5
			}
			got := ByRecency(tc.recipes, tc.history, today, 10, limit)
			views := make([]view, 0, len(got))
			for _, c := range got {
				views = append(views, view{c.RecipeName, c.DaysSince})
			}
			if !reflect.DeepEqual(views, tc.want) {
				t.Errorf("got %#v\nwant %#v", views, tc.want)
			}
		})
	}
}
