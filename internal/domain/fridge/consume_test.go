package fridge

import (
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func TestPlanConsumption(t *testing.T) {
	ing := func(name string, amount float64, unit string) recipe.RecipeIngredient {
		return recipe.RecipeIngredient{Name: name, Amount: amount, Unit: unit}
	}
	stock := func(id, name string, amount float64, unit, expires, added string) StockItem {
		return StockItem{ID: id, Name: name, Amount: amount, Unit: unit, ExpiresAt: expires, AddedAt: added}
	}
	type alloc struct {
		StockID string
		Amount  float64
	}
	type lineView struct {
		Name     string
		Required float64
		Allocs   []alloc
	}
	units := []string{"大さじ", "小さじ", "カップ"}

	cases := []struct {
		name        string
		ingredients []recipe.RecipeIngredient
		stocks      []StockItem
		staples     []string
		want        []lineView
		wantMissing []string
	}{
		{
			name:        "賞味期限の近い在庫から割り当て、期限なしの在庫は最後になる",
			ingredients: []recipe.RecipeIngredient{ing("玉ねぎ", 4, "個")},
			stocks: []StockItem{
				stock("none", "玉ねぎ", 5, "個", "", "2026-09-01T00:00:00Z"),
				stock("late", "玉ねぎ", 3, "個", "2026-10-10", "2026-09-01T00:00:00Z"),
				stock("soon", "玉ねぎ", 1, "個", "2026-10-02", "2026-09-01T00:00:00Z"),
			},
			want: []lineView{{"玉ねぎ", 4, []alloc{{"soon", 1}, {"late", 3}, {"none", 0}}}},
		},
		{
			name:        "同じ期限の在庫は追加日時の古い順に割り当てる",
			ingredients: []recipe.RecipeIngredient{ing("卵", 2, "個")},
			stocks: []StockItem{
				stock("new", "卵", 6, "個", "2026-10-05", "2026-09-10T00:00:00Z"),
				stock("old", "卵", 1, "個", "2026-10-05", "2026-09-01T00:00:00Z"),
			},
			want: []lineView{{"卵", 2, []alloc{{"old", 1}, {"new", 1}}}},
		},
		{
			name:        "在庫より多い材料は在庫の量までしか割り当てない",
			ingredients: []recipe.RecipeIngredient{ing("豚肉", 300, "g")},
			stocks:      []StockItem{stock("p", "豚肉", 100, "g", "", "")},
			want:        []lineView{{"豚肉", 300, []alloc{{"p", 100}}}},
		},
		{
			name:        "割り当てが 0 の在庫も Amount 0 で返る",
			ingredients: []recipe.RecipeIngredient{ing("玉ねぎ", 1, "個")},
			stocks: []StockItem{
				stock("a", "玉ねぎ", 2, "個", "2026-10-01", ""),
				stock("b", "玉ねぎ", 2, "個", "2026-10-03", ""),
			},
			want: []lineView{{"玉ねぎ", 1, []alloc{{"a", 1}, {"b", 0}}}},
		},
		{
			name:        "単位が違う在庫は候補にしない",
			ingredients: []recipe.RecipeIngredient{ing("豚肉", 200, "g")},
			stocks:      []StockItem{stock("p", "豚肉", 1, "パック", "", "")},
			want:        []lineView{},
			wantMissing: []string{"豚肉"},
		},
		{
			name:        "カタカナとひらがなの違う食材名は同じ食材として候補にする",
			ingredients: []recipe.RecipeIngredient{ing("たまねぎ", 1, "個")},
			stocks:      []StockItem{stock("t", "タマネギ", 3, "個", "", "")},
			want:        []lineView{{"たまねぎ", 1, []alloc{{"t", 1}}}},
		},
		{
			name:        "量 0 の材料と量る単位の材料は Lines にも Missing にも入らない",
			ingredients: []recipe.RecipeIngredient{ing("塩", 0, ""), ing("醤油", 2, "大さじ")},
			stocks:      []StockItem{stock("s", "醤油", 1, "本", "", "")},
			want:        []lineView{},
		},
		{
			name:        "常備品と同名の材料は Lines にも Missing にも入らない",
			ingredients: []recipe.RecipeIngredient{ing("牛乳", 200, "ml")},
			stocks:      []StockItem{stock("m", "牛乳", 1000, "ml", "", "")},
			staples:     []string{"ギュウニュウ", "牛乳"},
			want:        []lineView{},
		},
		{
			name:        "同じ食材名と単位の材料が 2 行あると量を合計して 1 行になる",
			ingredients: []recipe.RecipeIngredient{ing("玉ねぎ", 1, "個"), ing("玉ねぎ ", 1.5, "個")},
			stocks:      []StockItem{stock("o", "玉ねぎ", 2, "個", "", "")},
			want:        []lineView{{"玉ねぎ", 2.5, []alloc{{"o", 2}}}},
		},
		{
			name:        "在庫の無い材料は Missing に入る",
			ingredients: []recipe.RecipeIngredient{ing("玉ねぎ", 2, "個"), ing("じゃがいも", 3, "個")},
			stocks:      []StockItem{stock("o", "玉ねぎ", 3, "個", "", "")},
			want:        []lineView{{"玉ねぎ", 2, []alloc{{"o", 2}}}},
			wantMissing: []string{"じゃがいも"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PlanConsumption(tc.ingredients, tc.stocks, tc.staples, units)
			views := make([]lineView, 0)
			for _, l := range got.Lines {
				allocs := make([]alloc, 0)
				for _, c := range l.Candidates {
					allocs = append(allocs, alloc{c.StockID, c.Amount})
				}
				views = append(views, lineView{l.Name, l.RequiredAmount, allocs})
			}
			if !reflect.DeepEqual(views, tc.want) {
				t.Errorf("lines: got %#v\nwant %#v", views, tc.want)
			}
			wantMissing := tc.wantMissing
			if wantMissing == nil {
				wantMissing = []string{}
			}
			if !reflect.DeepEqual(got.Missing, wantMissing) {
				t.Errorf("missing: got %#v want %#v", got.Missing, wantMissing)
			}
		})
	}
}
