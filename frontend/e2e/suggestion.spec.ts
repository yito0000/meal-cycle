import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStockItem,
  assignRecipe,
  createRecipeWithIngredients,
  dateAfter,
  deleteRecipe,
  mealPlanCell,
  mealPlanItemRows,
  setCooked,
} from "./helpers.js";

const t = ja.pages.meal_plan;
const ts = t.picker.suggestion;

function onion() {
  return [{ name: "玉ねぎ", amount: 1, unit: "個" }];
}

// 今日の夕食の空セルの「＋」からレシピ選択ダイアログを開く。
async function openPickerToday(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
  await mealPlanCell(page, dateAfter(0), "dinner")
    .getByRole("button", { name: t.cell_actions.add_first })
    .click();
  const dialog = page.getByRole("dialog", { name: t.picker.title });
  await dialog.waitFor();
  return dialog;
}

function section(page: Page, title: string) {
  return page.getByRole("dialog").getByRole("region", { name: title });
}

function byStock(page: Page) {
  return section(page, ts.by_stock_title);
}

async function itemNames(page: Page, title: string): Promise<string[]> {
  return section(page, title)
    .getByTestId("suggestion-item")
    .evaluateAll((items) => items.map((i) => i.querySelector("span")?.textContent ?? ""));
}

test.describe("使い切りの候補", () => {
  test("期限の近い在庫を使うレシピが候補に出る", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 2, unit: "個", expiresInDays: 2 });
    await createRecipeWithIngredients(page, "肉じゃが", onion());

    await openPickerToday(page);

    const item = byStock(page).getByTestId("suggestion-item").filter({ hasText: "肉じゃが" });
    await expect(item).toHaveCount(1);
    await expect(item).toContainText(
      ts.by_stock_reason.replace("{{names}}", "玉ねぎ").replace("{{days}}", "2"),
    );
  });

  test("使える在庫の多いレシピが先に並ぶ", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 2, unit: "個", expiresInDays: 2 });
    await addStockItem(page, { name: "にんじん", amount: 2, unit: "本", expiresInDays: 5 });
    await createRecipeWithIngredients(page, "オニオンスープ", onion());
    await createRecipeWithIngredients(page, "ポトフ", [
      { name: "玉ねぎ", amount: 1, unit: "個" },
      { name: "にんじん", amount: 1, unit: "本" },
    ]);

    await openPickerToday(page);

    await expect(byStock(page).getByTestId("suggestion-item")).toHaveCount(2);
    expect(await itemNames(page, ts.by_stock_title)).toEqual(["ポトフ", "オニオンスープ"]);
  });

  test("期限に余裕のある在庫と期限切れの在庫は候補の根拠にならない", async ({ page }) => {
    await addStockItem(page, { name: "卵", amount: 6, unit: "個", expiresInDays: 10 });
    await addStockItem(page, { name: "豆腐", amount: 1, unit: "丁", expiresInDays: -2 });
    await createRecipeWithIngredients(page, "豆腐の卵とじ", [
      { name: "卵", amount: 2, unit: "個" },
      { name: "豆腐", amount: 1, unit: "丁" },
    ]);

    await openPickerToday(page);

    await expect(byStock(page).getByText(ts.none)).toBeVisible();
    await expect(byStock(page).getByTestId("suggestion-item")).toHaveCount(0);
  });

  test("候補を押すとそのレシピが割り当たる", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 2, unit: "個", expiresInDays: 2 });
    await createRecipeWithIngredients(page, "肉じゃが", onion());

    await openPickerToday(page);
    await byStock(page).getByTestId("suggestion-item").filter({ hasText: "肉じゃが" }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(
      mealPlanCell(page, dateAfter(0), "dinner").getByText("肉じゃが", { exact: true }),
    ).toBeVisible();
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(1);
  });

  test("検索中は候補の欄を隠す", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 2, unit: "個", expiresInDays: 2 });
    await createRecipeWithIngredients(page, "肉じゃが", onion());
    await createRecipeWithIngredients(page, "カレー", []);

    const dialog = await openPickerToday(page);
    await expect(byStock(page)).toBeVisible();
    await dialog.getByPlaceholder(t.picker.search_placeholder).fill("カレー");

    await expect(byStock(page)).toHaveCount(0);
    await expect(dialog.getByRole("button", { name: "カレー", exact: true })).toBeVisible();
    await expect(dialog.getByRole("button", { name: "肉じゃが", exact: true })).toHaveCount(0);
  });

  test("削除済みのレシピは候補に出ない", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 2, unit: "個", expiresInDays: 2 });
    const id = await createRecipeWithIngredients(page, "肉じゃが", onion());
    await deleteRecipe(page, id);

    await openPickerToday(page);

    await expect(byStock(page).getByText(ts.none)).toBeVisible();
  });
});

// daysAgo 日前の夕食にレシピを置き、作った印を付ける。
async function cookedAgo(page: Page, recipeId: string, daysAgo: number) {
  await assignRecipe(page, dateAfter(-daysAgo), "dinner", recipeId);
  await setCooked(page, dateAfter(-daysAgo), "dinner", 0, true);
}

function byRecency(page: Page) {
  return section(page, ts.by_recency_title.replace("{{days}}", "10"));
}

test.describe("久しぶりの候補", () => {
  test("しばらく作っていないレシピが候補に出る", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "麻婆豆腐", []);
    await cookedAgo(page, id, 14);

    await openPickerToday(page);

    const item = byRecency(page).getByTestId("suggestion-item").filter({ hasText: "麻婆豆腐" });
    await expect(item).toHaveCount(1);
    await expect(item).toContainText(ts.by_recency_reason.replace("{{days}}", "14"));
  });

  test("最近作ったレシピは候補に出ない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "麻婆豆腐", []);
    await cookedAgo(page, id, 3);

    await openPickerToday(page);

    await expect(byRecency(page).getByText(ts.none)).toBeVisible();
  });

  test("予定のあるレシピは候補に出ない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "麻婆豆腐", []);
    await cookedAgo(page, id, 14);
    await assignRecipe(page, dateAfter(1), "dinner", id);

    await openPickerToday(page);

    await expect(byRecency(page).getByTestId("suggestion-item")).toHaveCount(0);
  });

  test("一度も作っていないレシピは候補に出ない", async ({ page }) => {
    await createRecipeWithIngredients(page, "鮭のホイル焼き", []);

    await openPickerToday(page);

    await expect(byRecency(page).getByText(ts.none)).toBeVisible();
  });
});
