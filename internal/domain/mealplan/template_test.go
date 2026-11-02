package mealplan

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateTemplateName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{"前後の空白を除いた名前を返す", "  煮物の週  ", "煮物の週", nil},
		{"空白だけの名前は ErrTemplateNameEmpty", "   ", "", ErrTemplateNameEmpty},
		{"50 文字の名前は通る", strings.Repeat("あ", 50), strings.Repeat("あ", 50), nil},
		{"51 文字の名前は ErrTemplateNameTooLong", strings.Repeat("あ", 51), "", ErrTemplateNameTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateTemplateName(tc.in)
			if got != tc.want || err != tc.wantErr {
				t.Errorf("got (%q, %v), want (%q, %v)", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestTemplateItemsFromWeek(t *testing.T) {
	e := func(date string, meal Meal, pos int, recipe, batch string) Entry {
		return Entry{Date: date, Meal: meal, Position: pos, RecipeID: recipe, BatchID: batch}
	}
	type view struct {
		Day   int
		Meal  Meal
		Pos   int
		Rec   string
		Group int
	}
	views := func(items []TemplateItem) []view {
		out := make([]view, 0, len(items))
		for _, it := range items {
			out = append(out, view{it.DayOffset, it.Meal, it.Position, it.RecipeID, it.GroupNo})
		}
		return out
	}
	cases := []struct {
		name    string
		entries []Entry
		want    []view
	}{
		{
			name: "同じ batch_id の行に同じ group_no を振る",
			entries: []Entry{
				e("2026-11-02", MealDinner, 0, "nikujaga", "b1"),
				e("2026-11-03", MealDinner, 0, "nikujaga", "b1"),
				e("2026-11-05", MealDinner, 0, "curry", "b2"),
			},
			want: []view{{0, MealDinner, 0, "nikujaga", 0}, {1, MealDinner, 0, "nikujaga", 0}, {3, MealDinner, 0, "curry", 1}},
		},
		{
			name: "週の外に同じ batch_id の行があっても週の中の行だけを返す",
			entries: []Entry{
				e("2026-11-08", MealDinner, 0, "curry", "b1"),
				e("2026-11-09", MealDinner, 0, "curry", "b1"),
				e("2026-11-01", MealDinner, 0, "curry", "b0"),
			},
			want: []view{{6, MealDinner, 0, "curry", 0}},
		},
		{
			name: "day_offset は週の最初の日からの日数で、時間帯とセル内の順に並ぶ",
			entries: []Entry{
				e("2026-11-04", MealDinner, 1, "soup", "b3"),
				e("2026-11-04", MealDinner, 0, "main", "b2"),
				e("2026-11-04", MealBreakfast, 0, "toast", "b1"),
			},
			want: []view{{2, MealBreakfast, 0, "toast", 0}, {2, MealDinner, 0, "main", 1}, {2, MealDinner, 1, "soup", 2}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := views(TemplateItemsFromWeek("2026-11-02", tc.entries)); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v\nwant %#v", got, tc.want)
			}
		})
	}
}

func TestPlanItemsFromTemplate(t *testing.T) {
	items := []TemplateItem{
		{DayOffset: 3, Meal: MealDinner, Position: 0, RecipeID: "curry", GroupNo: 1},
		{DayOffset: 0, Meal: MealDinner, Position: 1, RecipeID: "soup", GroupNo: 2},
		{DayOffset: 0, Meal: MealDinner, Position: 0, RecipeID: "nikujaga", GroupNo: 0},
		{DayOffset: 0, Meal: MealLunch, Position: 0, RecipeID: "udon", GroupNo: 3},
	}
	cases := []struct {
		name        string
		archived    map[string]bool
		want        []PlanItem
		wantSkipped int
	}{
		{
			name: "PlanItem は (day_offset, meal, position) の順に並ぶ",
			want: []PlanItem{
				{Date: "2026-11-09", Meal: MealLunch, RecipeID: "udon", Group: 3},
				{Date: "2026-11-09", Meal: MealDinner, RecipeID: "nikujaga", Group: 0},
				{Date: "2026-11-09", Meal: MealDinner, RecipeID: "soup", Group: 2},
				{Date: "2026-11-12", Meal: MealDinner, RecipeID: "curry", Group: 1},
			},
		},
		{
			name:     "削除済みのレシピの行を除いて除いた数を返す",
			archived: map[string]bool{"curry": true, "udon": true},
			want: []PlanItem{
				{Date: "2026-11-09", Meal: MealDinner, RecipeID: "nikujaga", Group: 0},
				{Date: "2026-11-09", Meal: MealDinner, RecipeID: "soup", Group: 2},
			},
			wantSkipped: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, skipped := PlanItemsFromTemplate("2026-11-09", items, tc.archived)
			if !reflect.DeepEqual(got, tc.want) || skipped != tc.wantSkipped {
				t.Errorf("got (%#v, %d)\nwant (%#v, %d)", got, skipped, tc.want, tc.wantSkipped)
			}
		})
	}
}
