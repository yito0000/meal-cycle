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

// mealplan-cook-consume.spec.ts と同じ前提づくり。今日の夕食に 1 品置き、S01 (今日を含む週) を開く。
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

test.describe("料理で使った在庫の候補", () => {
  test("賞味期限の近い在庫から割り当てる", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個", expiresInDays: 10 });
    await addStockItem(page, { name: "玉ねぎ", amount: 1, unit: "個", expiresInDays: 2 });
    await planToday(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);

    await markCooked(page, "肉じゃが");

    const rows = consumeRows(page, "玉ねぎ");
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toHaveAttribute("data-expires-at", dateAfter(2));
    await expect(amountInput(page, "玉ねぎ", 0)).toHaveValue("1");
    await expect(rows.nth(1)).toHaveAttribute("data-expires-at", dateAfter(10));
    await expect(amountInput(page, "玉ねぎ", 1)).toHaveValue("1");
  });

  test("在庫の無い材料は名前だけ示す", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await planToday(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "じゃがいも", amount: 3, unit: "個" },
    ]);

    await markCooked(page, "肉じゃが");

    await expect(consumeRows(page, "玉ねぎ")).toHaveCount(1);
    const missing = dialog(page).getByTestId("consume-missing");
    await expect(missing).toContainText(tc.missing_title);
    await expect(missing).toContainText("じゃがいも");
  });
});

test.describe("料理で使った在庫の反映", () => {
  test("減らすと在庫の量が減る", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await planToday(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await markCooked(page, "肉じゃが");

    await dialog(page).getByRole("button", { name: tc.submit }).click();

    await expect(page.getByText(tc.consumed.replace("{{count}}", "1"))).toBeVisible();
    await openFridge(page);
    await expect(stockRow(page, "玉ねぎ")).toContainText("1 個");
  });

  test("使い切った在庫は消える", async ({ page }) => {
    await addStockItem(page, { name: "豚肉", amount: 200, unit: "g" });
    await planToday(page, "生姜焼き", [{ name: "豚肉", amount: 150, unit: "g" }]);
    await markCooked(page, "生姜焼き");

    await amountInput(page, "豚肉").fill("200");
    await dialog(page).getByRole("button", { name: tc.submit }).click();

    await expect(dialog(page)).toHaveCount(0);
    await openFridge(page);
    await expect(stockRow(page, "豚肉")).toHaveCount(0);
  });

  test("今の量より多く減らそうとすると何も変わらない", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await addStockItem(page, { name: "豚肉", amount: 200, unit: "g" });
    await planToday(page, "生姜焼き", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "豚肉", amount: 200, unit: "g" },
    ]);
    await markCooked(page, "生姜焼き");

    await amountInput(page, "玉ねぎ").fill("5");
    await dialog(page).getByRole("button", { name: tc.submit }).click();

    await expect(consumeRows(page, "玉ねぎ").getByRole("alert")).toHaveText(
      tc.errors.amount_out_of_range,
    );
    await expect(dialog(page)).toBeVisible();
    await dialog(page).getByRole("button", { name: tc.skip }).click();
    await openFridge(page);
    await expect(stockRow(page, "玉ねぎ")).toContainText("3 個");
    await expect(stockRow(page, "豚肉")).toContainText("200 g");
  });
});
