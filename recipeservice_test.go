package main

import (
	"context"
	"testing"
)

func TestRecipeService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should return all fixture recipes when List is called", func(t *testing.T) {
		svc := NewRecipeService()
		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if len(got) != len(fixtureRecipes) {
			t.Errorf("want %d recipes, got %d", len(fixtureRecipes), len(got))
		}
	})

	t.Run("should return recipes whose Name field is non-empty for every entry", func(t *testing.T) {
		svc := NewRecipeService()
		got, _ := svc.List(ctx)
		for i, r := range got {
			if r.Name == "" {
				t.Errorf("recipe %d has empty Name (id=%q)", i, r.ID)
			}
		}
	})

	t.Run("should include at least one recipe with two or more tags", func(t *testing.T) {
		svc := NewRecipeService()
		got, _ := svc.List(ctx)
		found := false
		for _, r := range got {
			if len(r.Tags) >= 2 {
				found = true
				break
			}
		}
		if !found {
			t.Error("fixture must include at least one recipe with 2+ tags for UI multi-tag rendering test")
		}
	})

	t.Run("should include at least one recipe with zero tags", func(t *testing.T) {
		svc := NewRecipeService()
		got, _ := svc.List(ctx)
		found := false
		for _, r := range got {
			if len(r.Tags) == 0 {
				found = true
				break
			}
		}
		if !found {
			t.Error("fixture must include at least one recipe with zero tags for empty-tag column test")
		}
	})

	t.Run("should return a fresh slice so mutating the result does not affect subsequent calls", func(t *testing.T) {
		svc := NewRecipeService()
		first, _ := svc.List(ctx)
		if len(first) == 0 {
			t.Fatal("List returned no recipes")
		}
		originalName := first[0].Name
		first[0].Name = "MUTATED"

		second, _ := svc.List(ctx)
		if second[0].Name != originalName {
			t.Errorf("mutation leaked into subsequent List call: want %q, got %q", originalName, second[0].Name)
		}
	})
}
