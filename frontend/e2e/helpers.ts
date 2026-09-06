import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };

const t = ja.pages.recipes;

// サイドバーから S02 に入る。ルータが memory history なので URL 直打ちでは
// 遷移しない (design.md D8)。
export async function openRecipeList(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: t.title }).click();
}

// createRecipe は UI を経由せず shim と同じ HTTP 経路で 1 件作る。
// 前提条件を整えるだけのステップを画面操作で書くと、検証したいシナリオが
// 別のシナリオの成否に引きずられるため。
export async function createRecipe(page: Page, name: string, tags: string[] = []) {
  const res = await page.request.post("/api/RecipeService/Create", {
    data: [
      {
        id: "",
        name,
        servings: 2,
        tags,
        ingredients: [{ name: "玉ねぎ", amount: 1, unit: "個" }],
        ingredientCount: 1,
        steps: "",
        notes: "",
        createdAt: "",
        updatedAt: "",
      },
    ],
  });
  if (!res.ok()) {
    throw new Error(`create ${name} failed: ${res.status()} ${await res.text()}`);
  }
}
