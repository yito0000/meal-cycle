import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStockItem,
  assignRecipe,
  createRecipeWithIngredients,
  dateAfter,
  mealPlanItemRows,
  openFridge,
} from "./helpers.js";

const t = ja.pages.meal_plan;
const tc = t.consume;

type Ingredient = { name: string; amount: number; unit: string };

// 今日の夕食に 1 品置き、S01 (今日を含む週) を開く。
async function planToday(page: Page, recipe: string, ingredients: Ingredient[]) {
  const id = await createRecipeWithIngredients(page, recipe, ingredients);
  await assignRecipe(page, dateAfter(0), "dinner", id);
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
}

function cookedCheckbox(page: Page, recipe: string) {
  return mealPlanItemRows(page, recipe)
    .first()
    .getByRole("checkbox", { name: t.cell.cooked.replace("{{name}}", recipe) });
}

function dialog(page: Page) {
  return page.getByRole("dialog", { name: tc.title });
}

function consumeRows(page: Page, ingredient: string) {
  return dialog(page).getByRole("region", { name: ingredient }).getByTestId("consume-row");
}

function amountInput(page: Page, ingredient: string, nth = 0) {
  return consumeRows(page, ingredient).nth(nth).getByRole("textbox", { name: tc.column.amount });
}

function stockRow(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByText(name, { exact: true }) });
}

async function markCooked(page: Page, recipe: string) {
  await cookedCheckbox(page, recipe).click();
  await expect(mealPlanItemRows(page, recipe).first()).toHaveAttribute("data-cooked", "true");
}

test.describe("作った印を付けたときの在庫の消費", () => {
  test("印を付けると減らせる在庫がダイアログに並ぶ", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await planToday(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);

    await markCooked(page, "肉じゃが");

    await expect(dialog(page)).toBeVisible();
    await expect(consumeRows(page, "玉ねぎ")).toHaveCount(1);
    await expect(amountInput(page, "玉ねぎ")).toHaveValue("2");
  });

  test("減らせる在庫が無いとダイアログを開かない", async ({ page }) => {
    await planToday(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);

    await markCooked(page, "肉じゃが");

    // 候補の取得は印の保存のあとに走るので、取得が終わる程度に待ってから開いていないことを見る。
    await page.waitForLoadState("networkidle");
    await expect(dialog(page)).toHaveCount(0);
  });

  test("減らさないを選んでも印は残る", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await planToday(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await markCooked(page, "肉じゃが");

    await dialog(page).getByRole("button", { name: tc.skip }).click();

    await expect(dialog(page)).toHaveCount(0);
    await expect(mealPlanItemRows(page, "肉じゃが").first()).toHaveAttribute("data-cooked", "true");
    await openFridge(page);
    await expect(stockRow(page, "玉ねぎ")).toContainText("3 個");
  });

  test("印を外しても在庫は変わらない", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await planToday(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await markCooked(page, "肉じゃが");
    await dialog(page).getByRole("button", { name: tc.submit }).click();
    await expect(dialog(page)).toHaveCount(0);

    await cookedCheckbox(page, "肉じゃが").click();
    await expect(mealPlanItemRows(page, "肉じゃが").first()).toHaveAttribute(
      "data-cooked",
      "false",
    );

    await page.waitForLoadState("networkidle");
    await expect(dialog(page)).toHaveCount(0);
    await openFridge(page);
    await expect(stockRow(page, "玉ねぎ")).toContainText("1 個");
  });
});
