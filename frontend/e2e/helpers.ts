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

const tf = ja.pages.fridge;

// サイドバーから S05 に入る。
export async function openFridge(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: ja.nav.fridge }).click();
  await page.getByRole("heading", { name: tf.title }).waitFor();
}

// dateAfter はローカル日付で n 日後の YYYY-MM-DD を返す。e2e-server も同じマシンの
// ローカル日付で残り日数を数えるので、ここで UTC を使うと深夜帯に 1 日ずれる。
export function dateAfter(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() + days);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// addStockItem は UI を経由せず 1 行作る (createRecipe と同じ理由)。
export async function addStockItem(
  page: Page,
  item: { name: string; amount: number; unit: string; expiresInDays?: number },
) {
  const res = await page.request.post("/api/FridgeService/Add", {
    data: [
      {
        id: "",
        name: item.name,
        amount: item.amount,
        unit: item.unit,
        expiresAt: item.expiresInDays === undefined ? "" : dateAfter(item.expiresInDays),
        addedAt: "",
        updatedAt: "",
        expiryStatus: "",
        daysLeft: null,
      },
    ],
  });
  if (!res.ok()) {
    throw new Error(`add stock ${item.name} failed: ${res.status()} ${await res.text()}`);
  }
}

const ts = ja.pages.shopping;

// サイドバーから S07 に入る。search 無しのリンクなので対象期間は既定 (今日から 7 日間) になる。
export async function openShoppingList(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: ja.nav.shopping }).click();
  await page.getByRole("heading", { name: ts.title }).waitFor();
}

// createRecipeWithIngredients は材料を指定してレシピを 1 件作り、ID を返す。
export async function createRecipeWithIngredients(
  page: Page,
  name: string,
  ingredients: { name: string; amount: number; unit: string }[],
): Promise<string> {
  const res = await page.request.post("/api/RecipeService/Create", {
    data: [
      {
        id: "",
        name,
        servings: 2,
        tags: [],
        ingredients,
        ingredientCount: ingredients.length,
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
  return (await res.json()).id as string;
}

export async function assignRecipe(page: Page, date: string, meal: string, recipeId: string) {
  const res = await page.request.post("/api/MealPlanService/AssignRecipe", {
    data: [date, meal, recipeId],
  });
  if (!res.ok()) {
    throw new Error(`assign ${date} ${meal} failed: ${res.status()} ${await res.text()}`);
  }
}

// copyLeftover は (date, meal) の 1 品目を days 日分に広げる。shareBatch=true で作り置き、false で別調理。
export async function copyLeftover(
  page: Page,
  date: string,
  meal: string,
  days: number,
  shareBatch: boolean,
) {
  const res = await page.request.post("/api/MealPlanService/CopyLeftover", {
    data: [date, meal, days, 0, "append", shareBatch],
  });
  if (!res.ok()) {
    throw new Error(`copy leftover ${date} failed: ${res.status()} ${await res.text()}`);
  }
}
