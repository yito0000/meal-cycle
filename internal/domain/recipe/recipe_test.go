package recipe

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRecipe_JSONTags(t *testing.T) {
	cases := []struct {
		name       string
		in         Recipe
		wantHas    []string
		wantNotHas []string
	}{
		{
			name:    "encodes name and servings with lowerCamelCase keys",
			in:      Recipe{Name: "肉じゃが", Servings: 2},
			wantHas: []string{`"name":"肉じゃが"`, `"servings":2`},
		},
		{
			name:    "encodes nil ingredients as null so frontend can guard with ?? []",
			in:      Recipe{Name: "肉じゃが"},
			wantHas: []string{`"ingredients":null`},
		},
		{
			name:    "encodes RecipeIngredient with lowerCamelCase keys and free-form name",
			in:      Recipe{Ingredients: []RecipeIngredient{{Name: "玉ねぎ", Amount: 1.5, Unit: "個"}}},
			wantHas: []string{`"name":"玉ねぎ"`, `"amount":1.5`, `"unit":"個"`},
		},
		{
			name:    "encodes Steps as empty string when unset (Markdown text, not array)",
			in:      Recipe{Name: "x"},
			wantHas: []string{`"steps":""`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			got := string(b)
			for _, want := range tc.wantHas {
				if !strings.Contains(got, want) {
					t.Errorf("expected %q in %s", want, got)
				}
			}
			for _, want := range tc.wantNotHas {
				if strings.Contains(got, want) {
					t.Errorf("did not expect %q in %s", want, got)
				}
			}
		})
	}
}
