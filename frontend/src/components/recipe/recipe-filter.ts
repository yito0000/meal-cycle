import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";

type FilterableRecipe = Pick<Recipe, "name" | "tags" | "archivedAt">;

// タグは保存された文字列のまま完全一致で扱い、表記の揺れはそろえない (add-recipe-tag-filter design.md D3)。
export function collectTags(recipes: readonly FilterableRecipe[]): string[] {
  const set = new Set<string>();
  for (const r of recipes) {
    if (r.archivedAt) continue;
    for (const tag of r.tags ?? []) set.add(tag);
  }
  return [...set].sort((a, b) => a.localeCompare(b, "ja"));
}

export function filterRecipes<T extends FilterableRecipe>(
  recipes: readonly T[],
  { query, tags }: { query: string; tags: readonly string[] },
): T[] {
  const q = query.trim().toLowerCase();
  return recipes.filter((r) => {
    if (q && !r.name.toLowerCase().includes(q)) return false;
    const own = r.tags ?? [];
    return tags.every((tag) => own.includes(tag));
  });
}
