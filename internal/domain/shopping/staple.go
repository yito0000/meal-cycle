package shopping

import (
	"sort"

	"github.com/yito0000/meal-cycle/internal/domain/fridge"
)

// StapleItems は「少ない」「ない」の常備品を S07 の常備品の欄の項目にし、同名の材料を使うレシピ名を添える (add-fridge-staples design.md D6)。
func StapleItems(batches []PlannedBatch, staples []StapleRef) []StapleItem {
	out := make([]StapleItem, 0)
	// そろえた名前が同じ常備品が 2 件あっても、先の 1 件にだけレシピ名を寄せる。
	byName := map[string]int{}
	for _, st := range staples {
		if st.State == fridge.StapleEnough {
			continue
		}
		key := normalizeKey(st.Name)
		if _, ok := byName[key]; !ok {
			byName[key] = len(out)
		}
		out = append(out, StapleItem{ID: st.ID, Name: st.Name, State: st.State, Recipes: []string{}})
	}

	for _, b := range sortedBatches(batches) {
		if b.Cooked {
			continue
		}
		for _, ing := range b.Ingredients {
			i, ok := byName[normalizeKey(ing.Name)]
			if !ok {
				continue
			}
			if !containsString(out[i].Recipes, b.RecipeName) {
				out[i].Recipes = append(out[i].Recipes, b.RecipeName)
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.State == fridge.StapleOut) != (b.State == fridge.StapleOut) {
			return a.State == fridge.StapleOut
		}
		return a.Name < b.Name
	})
	return out
}
