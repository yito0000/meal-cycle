package shopping

import "github.com/yito0000/meal-cycle/internal/domain/ingredient"

func normalizeKey(s string) string {
	return ingredient.NormalizeName(s)
}
