package main

import (
	"context"
	"testing"

	"github.com/yito0000/meal-cycle/internal/domain/mealplan"
)

func TestMealPlanService_ListWeek(t *testing.T) {
	ctx := context.Background()

	t.Run("should return entries when weekStart is a monday", func(t *testing.T) {
		svc := NewMealPlanService()
		got, err := svc.ListWeek(ctx, "2026-08-24")
		if err != nil {
			t.Fatalf("ListWeek returned error: %v", err)
		}
		if len(got) == 0 {
			t.Fatal("ListWeek returned no entries")
		}
	})

	t.Run("should keep every date inside monday..sunday of weekStart", func(t *testing.T) {
		svc := NewMealPlanService()
		got, _ := svc.ListWeek(ctx, "2026-08-24")

		valid := map[string]bool{
			"2026-08-24": true, "2026-08-25": true, "2026-08-26": true, "2026-08-27": true,
			"2026-08-28": true, "2026-08-29": true, "2026-08-30": true,
		}
		for _, e := range got {
			if !valid[e.Date] {
				t.Errorf("entry has date %q outside 2026-08-24..2026-08-30", e.Date)
			}
		}
	})

	t.Run("should include a slot with more than 2 items to exercise 他N件 collapse", func(t *testing.T) {
		svc := NewMealPlanService()
		got, _ := svc.ListWeek(ctx, "2026-08-24")

		type key struct {
			date string
			slot mealplan.Slot
		}
		counts := map[key]int{}
		for _, e := range got {
			counts[key{e.Date, e.Slot}]++
		}
		hasOverflow := false
		for _, n := range counts {
			if n > 2 {
				hasOverflow = true
				break
			}
		}
		if !hasOverflow {
			t.Error("fixture must include at least one (date, slot) with > 2 dishes for UI overflow test")
		}
	})

	t.Run("should return the same fixed set regardless of weekStart", func(t *testing.T) {
		svc := NewMealPlanService()
		a, _ := svc.ListWeek(ctx, "2026-08-24")
		b, _ := svc.ListWeek(ctx, "2027-01-04")
		if len(a) != len(b) {
			t.Fatalf("fixture size changed with weekStart: %d vs %d", len(a), len(b))
		}
		for i := range a {
			if a[i] != b[i] {
				t.Errorf("entry %d differs between weekStart calls: %+v vs %+v", i, a[i], b[i])
			}
		}
	})

	t.Run("should keep position unique within each (date, slot)", func(t *testing.T) {
		svc := NewMealPlanService()
		got, _ := svc.ListWeek(ctx, "2026-08-24")

		type key struct {
			date string
			slot mealplan.Slot
			pos  int
		}
		seen := map[key]bool{}
		for _, e := range got {
			k := key{e.Date, e.Slot, e.Position}
			if seen[k] {
				t.Errorf("duplicate (date=%s, slot=%s, position=%d)", e.Date, e.Slot, e.Position)
			}
			seen[k] = true
		}
	})
}
