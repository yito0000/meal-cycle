import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStaple,
  addStockItem,
  assignRecipe,
  createRecipeWithIngredients,
  dateAfter,
  openShoppingList,
} from "./helpers.js";

const tc = ja.pages.shopping.copy;

function copyButton(page: Page) {
  return page.getByTestId("shopping-copy-text");
}

// サイドバーから開いた S07 の対象期間 (今日から 7 日間) を添えた 1 行目。
function headingLine(): string {
  const slash = (d: string) => d.replaceAll("-", "/");
  return tc.heading
    .replace("{{start}}", slash(dateAfter(0)))
    .replace("{{end}}", slash(dateAfter(6)));
}

// 今日の夕食に材料つきのレシピを 1 件割り当てる。
async function planDinner(
  page: Page,
  recipe: string,
  ingredients: { name: string; amount: number; unit: string }[],
) {
  const id = await createRecipeWithIngredients(page, recipe, ingredients);
  await assignRecipe(page, dateAfter(0), "dinner", id);
}

// 「テキストでコピー」を押し、成功のトーストを待ってからクリップボードの中身を返す。
async function copyText(page: Page, count: number): Promise<string> {
  await copyButton(page).click();
  await expect(page.getByText(tc.copied.replace("{{count}}", String(count)))).toBeVisible();
  return page.evaluate(() => navigator.clipboard.readText());
}

test.describe("買い物リストのテキストコピー", () => {
  test.beforeEach(async ({ page }) => {
    await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  });

  test("在庫なしと一部在庫ありの材料がテキストでコピーされる", async ({ page }) => {
    await planDinner(page, "生姜焼き", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "豚ロース", amount: 200, unit: "g" },
      { name: "卵", amount: 2, unit: "個" },
    ]);
    await addStockItem(page, { name: "豚ロース", amount: 100, unit: "g" });
    await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
    await openShoppingList(page);

    const text = await copyText(page, 2);

    expect(text.split("\n")).toEqual([
      headingLine(),
      "- 玉ねぎ 2 個",
      "- 豚ロース 200 g (一部在庫あり)",
    ]);
    expect(text).not.toContain("卵");
  });

  test("家にあるか確かめる材料は見出しの下に続く", async ({ page }) => {
    await planDinner(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "醤油", amount: 2, unit: "大さじ" },
      { name: "塩", amount: 0, unit: "" },
    ]);
    await addStockItem(page, { name: "塩", amount: 1, unit: "" });
    await openShoppingList(page);

    const text = await copyText(page, 2);

    expect(text.split("\n")).toEqual([
      headingLine(),
      "- 玉ねぎ 2 個",
      "",
      tc.pantry_heading,
      "- 醤油",
    ]);
    expect(text).not.toContain("塩");
  });

  test("絞り込みにかかわらず同じ内容がコピーされる", async ({ page }) => {
    await planDinner(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "卵", amount: 2, unit: "個" },
    ]);
    await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
    await openShoppingList(page);

    await page.getByTestId("shopping-filter-covered").click();
    const text = await copyText(page, 1);

    expect(text).toContain("- 玉ねぎ 2 個");
    expect(text).not.toContain("卵");
  });

  test("買う必要のある材料が無いとコピーできない", async ({ page }) => {
    await planDinner(page, "卵焼き", [{ name: "卵", amount: 2, unit: "個" }]);
    await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
    await openShoppingList(page);

    await expect(copyButton(page)).toBeVisible();
    await expect(copyButton(page)).toBeDisabled();
  });

  test("献立が無い期間にはコピーのボタンを出さない", async ({ page }) => {
    await openShoppingList(page);

    await expect(page.getByText(ja.pages.shopping.empty.no_plans)).toBeVisible();
    await expect(copyButton(page)).toHaveCount(0);
  });
  test("常備品は材料のあとに見出しを付けて続く", async ({ page }) => {
    await planDinner(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "醤油", amount: 2, unit: "大さじ" },
    ]);
    await addStaple(page, "牛乳", "out");
    await addStaple(page, "食パン", "low");
    await openShoppingList(page);

    const text = await copyText(page, 4);

    expect(text.split("\n")).toEqual([
      headingLine(),
      "- 玉ねぎ 2 個",
      "",
      ja.pages.shopping.copy.staple_heading,
      "- 牛乳",
      "- 食パン (少ない)",
      "",
      tc.pantry_heading,
      "- 醤油",
    ]);
  });

  test("献立が無い期間でも常備品をコピーできる", async ({ page }) => {
    await addStaple(page, "牛乳", "out");
    await openShoppingList(page);

    const text = await copyText(page, 1);

    expect(text.split("\n")).toEqual([headingLine(), "", tc.staple_heading, "- 牛乳"]);
  });
});
