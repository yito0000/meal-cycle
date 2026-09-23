package shopping

import (
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func ing(name string, amount float64, unit string) recipe.RecipeIngredient {
	return recipe.RecipeIngredient{Name: name, Amount: amount, Unit: unit}
}

func batch(date, meal, recipeName string, ings ...recipe.RecipeIngredient) PlannedBatch {
	return PlannedBatch{
		FirstDate:        date,
		FirstMeal:        meal,
		RecipeID:         "id-" + recipeName,
		RecipeName:       recipeName,
		Dates:            []string{date},
		Meals:            []string{meal},
		BatchMemberCount: 1,
		Ingredients:      ings,
	}
}

func at(b PlannedBatch, position int) PlannedBatch {
	b.FirstPosition = position
	return b
}

func stock(name string, amount float64, unit string) StockAmount {
	return StockAmount{Name: name, Amount: amount, Unit: unit}
}

// view は比較用に Item から非公開フィールドを落とした形。
type view struct {
	Name    string
	Unit    string
	Amount  float64
	Status  StockStatus
	Recipes []string
}

func views(items []Item) []view {
	out := make([]view, 0, len(items))
	for _, it := range items {
		out = append(out, view{it.Name, it.Unit, it.Amount, it.StockStatus, it.Recipes})
	}
	return out
}

func TestAggregate(t *testing.T) {
	cases := []struct {
		name    string
		batches []PlannedBatch
		stocks  []StockAmount
		want    []view
	}{
		{
			name: "作り置き 0 件なら空",
			want: []view{},
		},
		{
			name:    "1 レシピ 1 材料で在庫なしなら none の 1 行",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "肉じゃが", ing("玉ねぎ", 2, "個"))},
			want:    []view{{"玉ねぎ", "個", 2, StockNone, []string{"肉じゃが"}}},
		},
		{
			name: "2 レシピの同名同単位は合算され、レシピ名は日付順",
			batches: []PlannedBatch{
				batch("2026-09-26", "dinner", "カレー", ing("玉ねぎ", 3, "個")),
				batch("2026-09-24", "dinner", "肉じゃが", ing("玉ねぎ", 2, "個")),
			},
			want: []view{{"玉ねぎ", "個", 5, StockNone, []string{"肉じゃが", "カレー"}}},
		},
		{
			name: "同じレシピが 2 回計上されると量は 2 回分でレシピ名は 1 回",
			batches: []PlannedBatch{
				batch("2026-09-24", "dinner", "肉じゃが", ing("玉ねぎ", 2, "個")),
				batch("2026-09-26", "dinner", "肉じゃが", ing("玉ねぎ", 2, "個")),
			},
			want: []view{{"玉ねぎ", "個", 4, StockNone, []string{"肉じゃが"}}},
		},
		{
			name: "同じ日の同じ時間帯の 2 品はセル内の順にレシピ名が入る",
			batches: []PlannedBatch{
				at(batch("2026-09-24", "dinner", "みそ汁", ing("玉ねぎ", 0.5, "個")), 1),
				at(batch("2026-09-24", "dinner", "肉じゃが", ing("玉ねぎ", 1, "個")), 0),
			},
			want: []view{{"玉ねぎ", "個", 1.5, StockNone, []string{"肉じゃが", "みそ汁"}}},
		},
		{
			name: "同名で単位が違う材料は 2 行",
			batches: []PlannedBatch{
				batch("2026-09-24", "dinner", "肉じゃが", ing("豚肉", 200, "g")),
				batch("2026-09-24", "dinner", "生姜焼き", ing("豚肉", 1, "パック")),
			},
			want: []view{
				{"豚肉", "g", 200, StockNone, []string{"肉じゃが"}},
				{"豚肉", "パック", 1, StockNone, []string{"生姜焼き"}},
			},
		},
		{
			name: "食材名と単位の前後の空白は無視する",
			batches: []PlannedBatch{
				batch("2026-09-24", "dinner", "肉じゃが", ing(" 玉ねぎ ", 1, " 個")),
				batch("2026-09-25", "dinner", "カレー", ing("玉ねぎ", 1, "個")),
			},
			stocks: []StockAmount{stock("玉ねぎ ", 2, "個 ")},
			want:   []view{{"玉ねぎ", "個", 2, StockCovered, []string{"肉じゃが", "カレー"}}},
		},
		{
			name:    "在庫が必要量より少なければ partial",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "肉じゃが", ing("玉ねぎ", 2, "個"))},
			stocks:  []StockAmount{stock("玉ねぎ", 1, "個")},
			want:    []view{{"玉ねぎ", "個", 2, StockPartial, []string{"肉じゃが"}}},
		},
		{
			name:    "在庫が必要量と等しければ covered",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "肉じゃが", ing("玉ねぎ", 2, "個"))},
			stocks:  []StockAmount{stock("玉ねぎ", 2, "個")},
			want:    []view{{"玉ねぎ", "個", 2, StockCovered, []string{"肉じゃが"}}},
		},
		{
			name:    "同名同単位の在庫 2 行は合計して判定する",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "肉じゃが", ing("玉ねぎ", 3, "個"))},
			stocks:  []StockAmount{stock("玉ねぎ", 1, "個"), stock("玉ねぎ", 2, "個")},
			want:    []view{{"玉ねぎ", "個", 3, StockCovered, []string{"肉じゃが"}}},
		},
		{
			name:    "同名で単位違いの在庫は判定に含めない",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "肉じゃが", ing("豚肉", 200, "g"))},
			stocks:  []StockAmount{stock("豚肉", 1, "パック")},
			want:    []view{{"豚肉", "g", 200, StockNone, []string{"肉じゃが"}}},
		},
		{
			name: "0.1 + 0.2 の必要量と 0.3 の在庫は covered",
			batches: []PlannedBatch{
				batch("2026-09-24", "dinner", "A", ing("酢", 0.1, "L")),
				batch("2026-09-25", "dinner", "B", ing("酢", 0.2, "L")),
			},
			stocks: []StockAmount{stock("酢", 0.3, "L")},
			want:   []view{{"酢", "L", 0.3, StockCovered, []string{"A", "B"}}},
		},
		{
			name:    "量 0 の材料は在庫が無ければ none",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "みそ汁", ing("塩", 0, ""))},
			want:    []view{{"塩", "", 0, StockNone, []string{"みそ汁"}}},
		},
		{
			name:    "量 0 の材料は同名同単位の在庫があれば covered",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "みそ汁", ing("塩", 0, ""))},
			stocks:  []StockAmount{stock("塩", 1, "")},
			want:    []view{{"塩", "", 0, StockCovered, []string{"みそ汁"}}},
		},
		{
			name: "covered は日付が早くても none / partial より後ろ",
			batches: []PlannedBatch{
				batch("2026-09-24", "dinner", "卵焼き", ing("卵", 2, "個")),
				batch("2026-09-26", "dinner", "カレー", ing("じゃがいも", 2, "個")),
			},
			stocks: []StockAmount{stock("卵", 5, "個")},
			want: []view{
				{"じゃがいも", "個", 2, StockNone, []string{"カレー"}},
				{"卵", "個", 2, StockCovered, []string{"卵焼き"}},
			},
		},
		{
			name: "同じまとまりの中は日付、食材名、単位の昇順",
			batches: []PlannedBatch{
				batch("2026-09-26", "dinner", "カレー", ing("じゃがいも", 2, "個")),
				batch("2026-09-24", "dinner", "生姜焼き", ing("豚ロース", 200, "g"), ing("生姜", 1, "かけ"), ing("生姜", 10, "g")),
			},
			want: []view{
				{"生姜", "g", 10, StockNone, []string{"生姜焼き"}},
				{"生姜", "かけ", 1, StockNone, []string{"生姜焼き"}},
				{"豚ロース", "g", 200, StockNone, []string{"生姜焼き"}},
				{"じゃがいも", "個", 2, StockNone, []string{"カレー"}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Aggregate(tc.batches, tc.stocks)
			if got == nil {
				t.Fatal("want non-nil slice")
			}
			if !reflect.DeepEqual(views(got), tc.want) {
				t.Errorf("got %#v\nwant %#v", views(got), tc.want)
			}
		})
	}
}

func TestDishes(t *testing.T) {
	cases := []struct {
		name    string
		batches []PlannedBatch
		want    []string
	}{
		{name: "作り置き 0 件なら空", want: []string{}},
		{
			name: "FirstDate の昇順",
			batches: []PlannedBatch{
				batch("2026-09-26", "dinner", "カレー"),
				batch("2026-09-24", "dinner", "肉じゃが"),
			},
			want: []string{"肉じゃが", "カレー"},
		},
		{
			name: "同じ日なら朝 / 昼 / 夕、同じ時間帯ならセル内の順",
			batches: []PlannedBatch{
				at(batch("2026-09-24", "dinner", "A"), 1),
				batch("2026-09-24", "breakfast", "Z"),
				at(batch("2026-09-24", "dinner", "B"), 0),
				batch("2026-09-24", "lunch", "Y"),
			},
			want: []string{"Z", "Y", "B", "A"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Dishes(tc.batches)
			names := make([]string, 0, len(got))
			for _, d := range got {
				names = append(names, d.RecipeName)
			}
			if !reflect.DeepEqual(names, tc.want) {
				t.Errorf("got %v, want %v", names, tc.want)
			}
		})
	}

	t.Run("識別子 / 日付 / 時間帯 / 作り置き行数 / 期間前フラグがそのまま移る", func(t *testing.T) {
		b := PlannedBatch{
			FirstDate:           "2026-09-24",
			BatchID:             "b1",
			RecipeID:            "r1",
			RecipeName:          "カレー",
			Dates:               []string{"2026-09-24", "2026-09-25"},
			Meals:               []string{"breakfast", "dinner"},
			BatchMemberCount:    3,
			StartedBeforePeriod: true,
		}
		got := Dishes([]PlannedBatch{b})
		want := []Dish{{
			BatchID:             "b1",
			RecipeID:            "r1",
			RecipeName:          "カレー",
			Dates:               []string{"2026-09-24", "2026-09-25"},
			Meals:               []string{"breakfast", "dinner"},
			BatchMemberCount:    3,
			StartedBeforePeriod: true,
		}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})

	t.Run("日付や時間帯が nil でも空配列で返す", func(t *testing.T) {
		got := Dishes([]PlannedBatch{{FirstDate: "2026-09-24", RecipeName: "A"}})
		if got[0].Dates == nil || got[0].Meals == nil {
			t.Errorf("want non-nil slices, got %#v", got[0])
		}
	})
}
