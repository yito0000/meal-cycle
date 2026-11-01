package suggestion

import (
	"reflect"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/recipe"
)

func slots(n int) []WeekSlot {
	out := make([]WeekSlot, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, WeekSlot{Date: day(i), Meal: "dinner"})
	}
	return out
}

func TestSplitSlots(t *testing.T) {
	cases := []struct {
		name  string
		n     int
		max   int
		sizes []int
	}{
		{"5 コマ・上限 3 は 3, 2 になる", 5, 3, []int{3, 2}},
		{"4 コマ・上限 3 は 2, 2 になる", 4, 3, []int{2, 2}},
		{"7 コマ・上限 3 は 3, 2, 2 になる", 7, 3, []int{3, 2, 2}},
		{"上限 1 はコマの数だけの組になる", 4, 1, []int{1, 1, 1, 1}},
		{"0 コマは空になる", 0, 3, []int{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitSlots(slots(tc.n), tc.max)
			sizes := make([]int, 0, len(got))
			flat := make([]WeekSlot, 0)
			for _, g := range got {
				sizes = append(sizes, len(g))
				flat = append(flat, g...)
			}
			if !reflect.DeepEqual(sizes, tc.sizes) {
				t.Errorf("sizes: got %v, want %v", sizes, tc.sizes)
			}
			if !reflect.DeepEqual(flat, slots(tc.n)) {
				t.Errorf("slots must keep date order: %v", flat)
			}
		})
	}
}

func TestWeekPlans(t *testing.T) {
	cand := func(name string) Candidate { return Candidate{RecipeID: "id-" + name, RecipeName: name} }
	two := SplitSlots(slots(5), 3)
	three := SplitSlots(slots(3), 1)
	type planView struct {
		Kind    WeekPlanKind
		Recipes []string
	}
	view := func(plans []WeekPlan) []planView {
		out := make([]planView, 0, len(plans))
		for _, p := range plans {
			names := make([]string, 0, len(p.Items))
			for _, it := range p.Items {
				names = append(names, it.RecipeName)
			}
			out = append(out, planView{p.Kind, names})
		}
		return out
	}

	cases := []struct {
		name      string
		groups    [][]WeekSlot
		byStock   []Candidate
		byRecency []Candidate
		rest      []recipe.Recipe
		want      []planView
	}{
		{
			name:      "使い切り優先は byStock の順に当てる / 久しぶり優先は byRecency の順に当てる / バランスは交互に当てる",
			groups:    three,
			byStock:   []Candidate{cand("A"), cand("B")},
			byRecency: []Candidate{cand("X"), cand("Y")},
			want: []planView{
				{WeekPlanStock, []string{"A", "B", "X"}},
				{WeekPlanRecency, []string{"X", "Y", "A"}},
				{WeekPlanBalanced, []string{"A", "X", "B"}},
			},
		},
		{
			name:      "1 案の中で同じレシピを 2 回使わない",
			groups:    two,
			byStock:   []Candidate{cand("カレー")},
			byRecency: []Candidate{cand("カレー"), cand("肉じゃが")},
			want: []planView{
				{WeekPlanStock, []string{"カレー", "肉じゃが"}},
			},
		},
		{
			name:   "レシピが足りない組は案に含めない",
			groups: three,
			rest:   []recipe.Recipe{rec("A"), rec("B")},
			want:   []planView{{WeekPlanStock, []string{"A", "B"}}},
		},
		{
			name:   "前の案と同じ案は除かれる",
			groups: two,
			rest:   []recipe.Recipe{rec("A"), rec("B"), rec("C")},
			want:   []planView{{WeekPlanStock, []string{"A", "B"}}},
		},
		{
			name:   "コマが無いと案は空になる",
			groups: [][]WeekSlot{},
			rest:   []recipe.Recipe{rec("A")},
			want:   []planView{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WeekPlans(tc.groups, tc.byStock, tc.byRecency, tc.rest)
			if !reflect.DeepEqual(view(got), tc.want) {
				t.Errorf("got %#v\nwant %#v", view(got), tc.want)
			}
		})
	}

	t.Run("組の日付と根拠が入る", func(t *testing.T) {
		stock := Candidate{RecipeID: "id-肉じゃが", RecipeName: "肉じゃが", Stocks: []StockReason{{"玉ねぎ", 2}}}
		got := WeekPlans(two, []Candidate{stock}, nil, []recipe.Recipe{rec("カレー")})
		it := got[0].Items[0]
		if !reflect.DeepEqual(it.Dates, []string{day(0), day(1), day(2)}) || it.Meal != "dinner" || len(it.Stocks) != 1 {
			t.Errorf("item: %#v", it)
		}
		if got[0].Items[1].Stocks == nil {
			t.Error("stocks must be non-nil")
		}
	})
}
