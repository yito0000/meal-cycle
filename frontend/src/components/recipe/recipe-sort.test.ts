import { describe, expect, it } from "vitest";
import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { compareRecipes, type RecipeSortKey } from "./recipe-sort";

// .ts 生成と .js 生成で型の付き方が違うため、比較に使うフィールドだけを持たせる (design.md Risks)。
function recipe(name: string, updatedAt: string): Recipe {
  return { name, updatedAt } as Recipe;
}

describe("compareRecipes", () => {
  it.each<{ name: string; key: RecipeSortKey; a: Recipe; b: Recipe; want: number }>([
    {
      name: "updated-desc で updatedAt が新しいほうが先に並ぶ",
      key: "updated-desc",
      a: recipe("A", "2026-10-15T10:00:00Z"),
      b: recipe("B", "2026-10-14T10:00:00Z"),
      want: -1,
    },
    {
      name: "updated-desc で updatedAt が古いほうが後に並ぶ",
      key: "updated-desc",
      a: recipe("A", "2026-10-14T10:00:00Z"),
      b: recipe("B", "2026-10-15T10:00:00Z"),
      want: 1,
    },
    {
      name: "updated-desc で updatedAt が同じなら 0 を返す",
      key: "updated-desc",
      a: recipe("A", "2026-10-15T10:00:00Z"),
      b: recipe("B", "2026-10-15T10:00:00Z"),
      want: 0,
    },
  ])("$name", ({ key, a, b, want }) => {
    expect(Math.sign(compareRecipes(a, b, key))).toBe(want);
  });

  it("name-asc で名前の昇順に並ぶ", () => {
    const recipes = [
      recipe("Curry", "2026-10-15T10:00:00Z"),
      recipe("Apple pie", "2026-10-13T10:00:00Z"),
      recipe("Beef stew", "2026-10-14T10:00:00Z"),
    ];
    const got = recipes.sort((a, b) => compareRecipes(a, b, "name-asc")).map((r) => r.name);
    expect(got).toEqual(["Apple pie", "Beef stew", "Curry"]);
  });
});
