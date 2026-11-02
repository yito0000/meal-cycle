import { describe, expect, it } from "vitest";
import type { Recipe } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/recipe/models";
import { collectTags, filterRecipes } from "./recipe-filter";

function recipe(name: string, tags: string[], archivedAt = ""): Recipe {
  return { name, tags, archivedAt } as Recipe;
}

describe("collectTags", () => {
  it.each<{ name: string; recipes: Recipe[]; want: string[] }>([
    {
      name: "重複を除いて名前の昇順に返す",
      recipes: [recipe("A", ["和食", "簡単"]), recipe("B", ["中華", "和食"])],
      want: ["中華", "和食", "簡単"].sort((a, b) => a.localeCompare(b, "ja")),
    },
    {
      name: "削除済みのレシピのタグを含めない",
      recipes: [recipe("A", ["和食"]), recipe("B", ["中華"], "2026-10-01T00:00:00Z")],
      want: ["和食"],
    },
  ])("$name", ({ recipes, want }) => {
    expect(collectTags(recipes)).toEqual(want);
  });
});

describe("filterRecipes", () => {
  const nikujaga = recipe("肉じゃが", ["和食"]);
  const hiyayakko = recipe("冷奴", ["和食", "簡単"]);
  const yasai = recipe("肉野菜炒め", ["中華"]);
  const mixed = recipe("フュージョン", ["和食・洋食"]);
  const all = [nikujaga, hiyayakko, yasai, mixed];

  it.each<{ name: string; query: string; tags: string[]; want: Recipe[] }>([
    {
      name: "タグが空なら名前の検索だけで絞る",
      query: "肉",
      tags: [],
      want: [nikujaga, yasai],
    },
    {
      name: "選んだタグをすべて持つレシピだけを返す",
      query: "",
      tags: ["和食", "簡単"],
      want: [hiyayakko],
    },
    {
      name: "タグを完全一致で比べる (「和食」は「和食・洋食」に一致しない)",
      query: "",
      tags: ["和食"],
      want: [nikujaga, hiyayakko],
    },
    {
      name: "名前の検索とタグの両方を満たすレシピだけを返す",
      query: "肉",
      tags: ["和食"],
      want: [nikujaga],
    },
  ])("$name", ({ query, tags, want }) => {
    expect(filterRecipes(all, { query, tags })).toEqual(want);
  });
});
