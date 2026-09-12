package mealplan

import (
	"errors"
	"reflect"
	"testing"
)

func TestCopyDates(t *testing.T) {
	cases := []struct {
		name    string
		start   string
		days    int
		want    []string
		wantErr error
	}{
		{"single day", "2026-08-24", 1, []string{"2026-08-24"}, nil},
		{"three days", "2026-08-24", 3, []string{"2026-08-24", "2026-08-25", "2026-08-26"}, nil},
		{"cross month end", "2026-01-30", 3, []string{"2026-01-30", "2026-01-31", "2026-02-01"}, nil},
		{"zero days", "2026-08-24", 0, []string{}, nil},
		{"negative days", "2026-08-24", -1, nil, ErrInvalidCopyDays},
		{"malformed date", "2026/08/24", 3, nil, ErrInvalidDate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CopyDates(tc.start, tc.days)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPlanCopy(t *testing.T) {
	source := []Entry{
		{Date: "2026-08-24", Meal: MealDinner, Position: 0, RecipeID: "r1", RecipeName: "肉じゃが"},
		{Date: "2026-08-24", Meal: MealDinner, Position: 1, RecipeID: "r2", RecipeName: "みそ汁"},
	}

	t.Run("existing empty, append starts at 0", func(t *testing.T) {
		got, err := PlanCopy("2026-08-25", source, nil, CopyModeAppend, ShareBatchDistinct)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].Position != 0 || got[1].Position != 1 {
			t.Errorf("positions = %d, %d; want 0, 1", got[0].Position, got[1].Position)
		}
		if got[0].Date != "2026-08-25" {
			t.Errorf("date = %q, want 2026-08-25", got[0].Date)
		}
	})

	t.Run("existing has 1 item, append starts at 1", func(t *testing.T) {
		existing := []Entry{{Date: "2026-08-25", Meal: MealDinner, Position: 0, RecipeID: "r9"}}
		got, err := PlanCopy("2026-08-25", source, existing, CopyModeAppend, ShareBatchDistinct)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].Position != 1 || got[1].Position != 2 {
			t.Errorf("positions = %d, %d; want 1, 2", got[0].Position, got[1].Position)
		}
	})

	t.Run("existing has 2 items, overwrite starts at 0 and ignores existing", func(t *testing.T) {
		existing := []Entry{
			{Date: "2026-08-25", Meal: MealDinner, Position: 0, RecipeID: "r9"},
			{Date: "2026-08-25", Meal: MealDinner, Position: 1, RecipeID: "r8"},
		}
		got, err := PlanCopy("2026-08-25", source, existing, CopyModeOverwrite, ShareBatchDistinct)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].Position != 0 || got[1].Position != 1 {
			t.Errorf("positions = %d, %d; want 0, 1", got[0].Position, got[1].Position)
		}
		for _, e := range got {
			if e.RecipeID == "r9" || e.RecipeID == "r8" {
				t.Errorf("overwrite must not include existing recipeIDs; got %s", e.RecipeID)
			}
		}
	})

	t.Run("append skips duplicate recipeID when target is source date", func(t *testing.T) {
		existing := []Entry{
			{Date: "2026-08-24", Meal: MealDinner, Position: 0, RecipeID: "r1"},
			{Date: "2026-08-24", Meal: MealDinner, Position: 1, RecipeID: "r2"},
		}
		got, err := PlanCopy("2026-08-24", source, existing, CopyModeAppend, ShareBatchDistinct)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want 0 entries (both source dishes already present), got %d", len(got))
		}
	})

	t.Run("invalid mode returns ErrInvalidCopyMode", func(t *testing.T) {
		_, err := PlanCopy("2026-08-25", source, nil, CopyMode("replace"), ShareBatchDistinct)
		if !errors.Is(err, ErrInvalidCopyMode) {
			t.Errorf("err = %v, want ErrInvalidCopyMode", err)
		}
	})
}

func TestPlanCopy_ShareBatch(t *testing.T) {
	sourceShared := []Entry{
		{Date: "2026-08-24", Meal: MealDinner, Position: 0, RecipeID: "r1", BatchID: "b-source"},
	}

	t.Run("shareBatch=true keeps source BatchID on every returned Entry", func(t *testing.T) {
		got, err := PlanCopy("2026-08-25", sourceShared, nil, CopyModeAppend, ShareBatchLeftover)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].BatchID != "b-source" {
			t.Errorf("BatchID = %q, want b-source", got[0].BatchID)
		}
	})

	t.Run("shareBatch=false assigns a fresh BatchID per returned Entry", func(t *testing.T) {
		multi := []Entry{
			{Date: "2026-08-24", Meal: MealDinner, Position: 0, RecipeID: "r1", BatchID: "b-source"},
			{Date: "2026-08-24", Meal: MealDinner, Position: 1, RecipeID: "r2", BatchID: "b-source"},
		}
		got, err := PlanCopy("2026-08-25", multi, nil, CopyModeAppend, ShareBatchDistinct)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].BatchID == "" || got[1].BatchID == "" {
			t.Errorf("BatchIDs must be non-empty, got %q / %q", got[0].BatchID, got[1].BatchID)
		}
		if got[0].BatchID == got[1].BatchID {
			t.Errorf("BatchIDs must differ per row, got both = %q", got[0].BatchID)
		}
		if got[0].BatchID == "b-source" || got[1].BatchID == "b-source" {
			t.Errorf("BatchIDs must not equal source b-source, got %q / %q", got[0].BatchID, got[1].BatchID)
		}
	})

	t.Run("shareBatch=true / shareBatch=false produce disjoint BatchID sets", func(t *testing.T) {
		shared, err := PlanCopy("2026-08-25", sourceShared, nil, CopyModeAppend, ShareBatchLeftover)
		if err != nil {
			t.Fatalf("shared err: %v", err)
		}
		distinct, err := PlanCopy("2026-08-25", sourceShared, nil, CopyModeAppend, ShareBatchDistinct)
		if err != nil {
			t.Fatalf("distinct err: %v", err)
		}
		sharedSet := map[string]bool{}
		for _, e := range shared {
			sharedSet[e.BatchID] = true
		}
		for _, e := range distinct {
			if sharedSet[e.BatchID] {
				t.Errorf("distinct BatchID %q also appears in shared set", e.BatchID)
			}
		}
	})
}
