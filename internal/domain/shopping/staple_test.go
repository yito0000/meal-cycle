package shopping

import (
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
)

func TestStapleItems(t *testing.T) {
	type itemView struct {
		Name    string
		State   fridge.StapleState
		Recipes []string
	}
	cases := []struct {
		name    string
		batches []PlannedBatch
		staples []StapleRef
		want    []itemView
	}{
		{
			name:    "enough の常備品は返らない",
			staples: []StapleRef{{ID: "1", Name: "卵", State: fridge.StapleEnough}},
			want:    []itemView{},
		},
		{
			name: "out が low より先に並び、それぞれ食材名の昇順になる",
			staples: []StapleRef{
				{ID: "1", Name: "牛乳", State: fridge.StapleLow},
				{ID: "2", Name: "食パン", State: fridge.StapleOut},
				{ID: "3", Name: "バター", State: fridge.StapleLow},
				{ID: "4", Name: "チーズ", State: fridge.StapleOut},
			},
			want: []itemView{
				{"チーズ", fridge.StapleOut, []string{}},
				{"食パン", fridge.StapleOut, []string{}},
				{"バター", fridge.StapleLow, []string{}},
				{"牛乳", fridge.StapleLow, []string{}},
			},
		},
		{
			name:    "献立に無い常備品は Recipes が空で返る",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "炒め物", ing("玉ねぎ", 1, "個"))},
			staples: []StapleRef{{ID: "1", Name: "牛乳", State: fridge.StapleOut}},
			want:    []itemView{{"牛乳", fridge.StapleOut, []string{}}},
		},
		{
			name: "Recipes は献立の順で重複なく並ぶ",
			batches: []PlannedBatch{
				batch("2026-09-26", "dinner", "シチュー", ing("牛乳", 200, "ml")),
				at(batch("2026-09-24", "dinner", "スープ", ing("牛乳", 100, "ml")), 1),
				at(batch("2026-09-24", "dinner", "グラタン", ing("牛乳", 300, "ml"), ing("牛乳", 50, "ml")), 0),
				batch("2026-09-24", "breakfast", "ミルクティー", ing("ギュウニュウ", 100, "ml")),
			},
			staples: []StapleRef{{ID: "1", Name: "牛乳", State: fridge.StapleLow}},
			want:    []itemView{{"牛乳", fridge.StapleLow, []string{"グラタン", "スープ", "シチュー"}}},
		},
		{
			name: "作った印の付いた組のレシピは Recipes に入らない",
			batches: []PlannedBatch{
				cooked(batch("2026-09-24", "dinner", "グラタン", ing("牛乳", 300, "ml"))),
				batch("2026-09-25", "dinner", "シチュー", ing("牛乳", 200, "ml")),
			},
			staples: []StapleRef{{ID: "1", Name: "牛乳", State: fridge.StapleOut}},
			want:    []itemView{{"牛乳", fridge.StapleOut, []string{"シチュー"}}},
		},
		{
			name:    "表記をそろえた名前が同じ常備品が 2 件あると、先の 1 件にだけ Recipes が入る",
			batches: []PlannedBatch{batch("2026-09-24", "dinner", "ミルクスープ", ing("みるく", 200, "ml"))},
			staples: []StapleRef{
				{ID: "1", Name: "ミルク", State: fridge.StapleOut},
				{ID: "2", Name: "みるく", State: fridge.StapleOut},
			},
			want: []itemView{
				{"みるく", fridge.StapleOut, []string{}},
				{"ミルク", fridge.StapleOut, []string{"ミルクスープ"}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StapleItems(tc.batches, tc.staples)
			views := make([]itemView, 0, len(got))
			for _, g := range got {
				views = append(views, itemView{g.Name, g.State, g.Recipes})
			}
			if !reflect.DeepEqual(views, tc.want) {
				t.Errorf("got %#v\nwant %#v", views, tc.want)
			}
		})
	}
}
