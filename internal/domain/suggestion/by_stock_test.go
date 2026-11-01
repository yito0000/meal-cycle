package suggestion

import (
	"reflect"
	"testing"
	"time"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

var today = time.Date(2026, 10, 1, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60))

func day(n int) string { return today.AddDate(0, 0, n).Format(fridge.DateLayout) }

func rec(name string, ings ...string) recipe.Recipe {
	r := recipe.Recipe{ID: "id-" + name, Name: name}
	for _, n := range ings {
		r.Ingredients = append(r.Ingredients, recipe.RecipeIngredient{Name: n, Amount: 1, Unit: "個"})
	}
	return r
}

func stk(name, unit, expires string) fridge.StockItem {
	return fridge.StockItem{Name: name, Amount: 1, Unit: unit, ExpiresAt: expires}
}

func TestByStock(t *testing.T) {
	type view struct {
		Recipe  string
		Reasons []StockReason
	}
	archived := rec("削除済み", "玉ねぎ")
	archived.ArchivedAt = "2026-09-01T00:00:00Z"

	cases := []struct {
		name    string
		recipes []recipe.Recipe
		stocks  []fridge.StockItem
		limit   int
		want    []view
	}{
		{
			name:    "🔴 と 🟡 の在庫を持つレシピが候補になる",
			recipes: []recipe.Recipe{rec("肉じゃが", "玉ねぎ"), rec("サラダ", "トマト")},
			stocks:  []fridge.StockItem{stk("玉ねぎ", "個", day(2)), stk("トマト", "個", day(6))},
			want: []view{
				{"肉じゃが", []StockReason{{"玉ねぎ", 2}}},
				{"サラダ", []StockReason{{"トマト", 6}}},
			},
		},
		{
			name:    "🟢、期限切れ、期限なしの在庫だけのレシピは候補にならない",
			recipes: []recipe.Recipe{rec("卵焼き", "卵", "豆腐", "米")},
			stocks:  []fridge.StockItem{stk("卵", "個", day(10)), stk("豆腐", "丁", day(-2)), stk("米", "kg", "")},
			want:    []view{},
		},
		{
			name:    "単位が違っても食材名が同じなら候補になる",
			recipes: []recipe.Recipe{rec("生姜焼き", "豚肉")},
			stocks:  []fridge.StockItem{stk("豚肉", "パック", day(1))},
			want:    []view{{"生姜焼き", []StockReason{{"豚肉", 1}}}},
		},
		{
			name:    "カタカナとひらがなの違う食材名も一致する",
			recipes: []recipe.Recipe{rec("カレー", "たまねぎ")},
			stocks:  []fridge.StockItem{stk("タマネギ", "個", day(0))},
			want:    []view{{"カレー", []StockReason{{"タマネギ", 0}}}},
		},
		{
			name: "一致した在庫の数の降順、最も近い期限の昇順、レシピ名の昇順に並ぶ",
			recipes: []recipe.Recipe{
				rec("オニオンスープ", "玉ねぎ"),
				rec("ポトフ", "玉ねぎ", "にんじん"),
				rec("きんぴら", "にんじん"),
				rec("オニオンフライ", "玉ねぎ"),
			},
			stocks: []fridge.StockItem{stk("玉ねぎ", "個", day(2)), stk("にんじん", "本", day(5))},
			want: []view{
				{"ポトフ", []StockReason{{"玉ねぎ", 2}, {"にんじん", 5}}},
				{"オニオンスープ", []StockReason{{"玉ねぎ", 2}}},
				{"オニオンフライ", []StockReason{{"玉ねぎ", 2}}},
				{"きんぴら", []StockReason{{"にんじん", 5}}},
			},
		},
		{
			name:    "同じ食材名の在庫が 2 行あっても 1 件として数え、近いほうの期限を使う",
			recipes: []recipe.Recipe{rec("肉じゃが", "玉ねぎ")},
			stocks:  []fridge.StockItem{stk("玉ねぎ", "個", day(6)), stk("玉ねぎ", "個", day(1))},
			want:    []view{{"肉じゃが", []StockReason{{"玉ねぎ", 1}}}},
		},
		{
			name:    "削除済みのレシピは候補にならない",
			recipes: []recipe.Recipe{archived},
			stocks:  []fridge.StockItem{stk("玉ねぎ", "個", day(1))},
			want:    []view{},
		},
		{
			name:    "limit を超える候補は切られる",
			recipes: []recipe.Recipe{rec("A", "玉ねぎ"), rec("B", "玉ねぎ"), rec("C", "玉ねぎ")},
			stocks:  []fridge.StockItem{stk("玉ねぎ", "個", day(1))},
			limit:   2,
			want:    []view{{"A", []StockReason{{"玉ねぎ", 1}}}, {"B", []StockReason{{"玉ねぎ", 1}}}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			limit := tc.limit
			if limit == 0 {
				limit = 5
			}
			got := ByStock(tc.recipes, tc.stocks, today, 3, 7, limit)
			views := make([]view, 0, len(got))
			for _, c := range got {
				views = append(views, view{c.RecipeName, c.Stocks})
			}
			if !reflect.DeepEqual(views, tc.want) {
				t.Errorf("got %#v\nwant %#v", views, tc.want)
			}
		})
	}
}
