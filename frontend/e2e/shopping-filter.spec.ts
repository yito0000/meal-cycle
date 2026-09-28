import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStockItem,
  assignRecipe,
  checkShoppingItem,
  createRecipeWithIngredients,
  dateAfter,
  openFridge,
  openShoppingList,
} from "./helpers.js";

const tp = ja.pages.shopping.purchase;
const tfl = ja.pages.shopping.filter;

function itemRow(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByRole("cell", { name, exact: true }) });
}

function rowCheckbox(page: Page, name: string) {
  return page.getByLabel(tp.check_row.replace("{{name}}", name), { exact: true });
}

function registerButton(page: Page) {
  const prefix = tp.register.split("{{count}}")[0];
  return page.getByRole("button").filter({ hasText: prefix });
}

function filterButton(page: Page, filter: "all" | "needed" | "covered") {
  return page.getByTestId(`shopping-filter-${filter}`);
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

// じゃがいも (在庫なし) と 卵 (在庫あり) の 2 行を用意する。
async function planPotatoAndEgg(page: Page) {
  await planDinner(page, "肉じゃが", [
    { name: "じゃがいも", amount: 2, unit: "個" },
    { name: "卵", amount: 2, unit: "個" },
  ]);
  await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
}

test.describe("在庫の状態での絞り込み", () => {
  test("在庫なし・一部で絞り込む", async ({ page }) => {
    await planDinner(page, "肉じゃが", [
      { name: "じゃがいも", amount: 2, unit: "個" },
      { name: "玉ねぎ", amount: 3, unit: "個" },
      { name: "卵", amount: 2, unit: "個" },
    ]);
    await addStockItem(page, { name: "玉ねぎ", amount: 1, unit: "個" });
    await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
    await openShoppingList(page);
    await expect(itemRow(page, "玉ねぎ")).toHaveAttribute("data-stock-status", "partial");

    await filterButton(page, "needed").click();

    await expect(itemRow(page, "じゃがいも")).toBeVisible();
    await expect(itemRow(page, "玉ねぎ")).toBeVisible();
    await expect(itemRow(page, "卵")).toHaveCount(0);
  });

  test("在庫ありで絞り込む", async ({ page }) => {
    await planPotatoAndEgg(page);
    await openShoppingList(page);

    await filterButton(page, "covered").click();

    await expect(page.locator("tbody tr")).toHaveCount(1);
    await expect(itemRow(page, "卵")).toBeVisible();
  });

  test("選択肢に件数が表示される", async ({ page }) => {
    await planDinner(page, "肉じゃが", [
      { name: "じゃがいも", amount: 2, unit: "個" },
      { name: "にんじん", amount: 1, unit: "本" },
      { name: "玉ねぎ", amount: 3, unit: "個" },
      { name: "卵", amount: 2, unit: "個" },
    ]);
    await addStockItem(page, { name: "玉ねぎ", amount: 1, unit: "個" });
    await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
    await openShoppingList(page);

    await expect(filterButton(page, "all")).toHaveText(tfl.all.replace("{{count}}", "4"));
    await expect(filterButton(page, "needed")).toHaveText(tfl.needed.replace("{{count}}", "3"));
    await expect(filterButton(page, "covered")).toHaveText(tfl.covered.replace("{{count}}", "1"));
  });

  test("開き直すと絞り込みはすべてに戻る", async ({ page }) => {
    await planPotatoAndEgg(page);
    await openShoppingList(page);
    await filterButton(page, "needed").click();
    await expect(itemRow(page, "卵")).toHaveCount(0);

    await openFridge(page);
    await openShoppingList(page);

    await expect(filterButton(page, "all")).toHaveAttribute("aria-pressed", "true");
    await expect(itemRow(page, "卵")).toBeVisible();
  });

  test("該当する行が無いとその旨を表示する", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "じゃがいも", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await filterButton(page, "covered").click();

    await expect(page.getByText(tfl.no_match)).toBeVisible();
    await expect(page.locator("tbody tr")).toHaveCount(0);
  });

  test("絞り込みを切り替えてもチェックは残る", async ({ page }) => {
    await planPotatoAndEgg(page);
    await openShoppingList(page);

    await checkShoppingItem(page, "卵");
    await filterButton(page, "needed").click();
    await filterButton(page, "all").click();

    await expect(rowCheckbox(page, "卵")).toBeChecked();
  });

  test("全選択は表示中の行だけに効く", async ({ page }) => {
    await planPotatoAndEgg(page);
    await openShoppingList(page);

    await filterButton(page, "needed").click();
    await page.getByLabel(tp.check_all, { exact: true }).click();
    await filterButton(page, "all").click();

    await expect(rowCheckbox(page, "じゃがいも")).toBeChecked();
    await expect(rowCheckbox(page, "卵")).not.toBeChecked();
  });

  test("隠れた行のチェックも登録の対象になる", async ({ page }) => {
    await planPotatoAndEgg(page);
    await openShoppingList(page);

    await checkShoppingItem(page, "じゃがいも");
    await checkShoppingItem(page, "卵");
    await filterButton(page, "needed").click();

    await expect(registerButton(page)).toHaveText(tp.register.replace("{{count}}", "2"));
    await registerButton(page).click();

    const rows = page.getByRole("dialog").getByTestId("purchase-row");
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0).getByLabel(ja.pages.fridge.field.name)).toHaveValue("じゃがいも");
    await expect(rows.nth(1).getByLabel(ja.pages.fridge.field.name)).toHaveValue("卵");
  });

  test("登録して在庫が足りた行は在庫なし・一部の表示から消える", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "じゃがいも", amount: 2, unit: "個" }]);
    await openShoppingList(page);
    await expect(itemRow(page, "じゃがいも")).toHaveAttribute("data-stock-status", "none");

    await filterButton(page, "needed").click();
    await checkShoppingItem(page, "じゃがいも");
    await registerButton(page).click();
    await page.getByRole("dialog").getByRole("button", { name: tp.submit }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    await expect(itemRow(page, "じゃがいも")).toHaveCount(0);
    await filterButton(page, "covered").click();
    await expect(itemRow(page, "じゃがいも")).toBeVisible();
  });
});
