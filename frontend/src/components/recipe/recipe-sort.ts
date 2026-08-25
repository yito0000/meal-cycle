import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

export type RecipeSortKey = "updated-desc" | "name-asc";

export const RECIPE_SORT_KEYS: RecipeSortKey[] = ["updated-desc", "name-asc"];

export const DEFAULT_RECIPE_SORT: RecipeSortKey = "updated-desc";

export function compareRecipes(a: Recipe, b: Recipe, key: RecipeSortKey): number {
  switch (key) {
    case "name-asc":
      return a.name.localeCompare(b.name);
    case "updated-desc":
      if (a.updatedAt === b.updatedAt) return 0;
      return a.updatedAt < b.updatedAt ? 1 : -1;
  }
}
