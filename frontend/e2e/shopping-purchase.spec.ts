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

const t = ja.pages.shopping;
const tp = t.purchase;
const tf = ja.pages.fridge;

function itemRow(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByRole("cell", { name, exact: true }) });
}

function rowCheckbox(page: Page, name: string) {
  return page.getByLabel(tp.check_row.replace("{{name}}", name), { exact: true });
}

function registerButton(page: Page) {
  // 件数が埋め込まれるので、件数より前の部分で前方一致させる。
  const prefix = tp.register.split("{{count}}")[0];
  return page.getByRole("button").filter({ hasText: prefix });
}

function dialogRow(page: Page, index: number) {
  return page.getByRole("dialog").getByTestId("purchase-row").nth(index);
}

function stockRows(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByText(name, { exact: true }) });
}

// 今日の夕食に材料つきのレシピを 1 件割り当てる。
async function planDinner(
  page: Page,
  recipe: string,
  ingredients: { name: string; amount: number; unit: string }[],
  daysLater = 0,
) {
  const id = await createRecipeWithIngredients(page, recipe, ingredients);
  await assignRecipe(page, dateAfter(daysLater), "dinner", id);
}

test.describe("購入済チェック", () => {
  test("チェックを付けても在庫は増えない", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await checkShoppingItem(page, "玉ねぎ");
    await openFridge(page);

    await expect(page.getByText(tf.empty.no_items)).toBeVisible();
    await expect(stockRows(page, "玉ねぎ")).toHaveCount(0);
  });

  test("在庫ありの行にもチェックを付けられる", async ({ page }) => {
    await planDinner(page, "卵焼き", [{ name: "卵", amount: 2, unit: "個" }]);
    await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
    await openShoppingList(page);

    await expect(itemRow(page, "卵")).toHaveAttribute("data-stock-status", "covered");
    await checkShoppingItem(page, "卵");
    await expect(rowCheckbox(page, "卵")).toBeChecked();
  });

  test("見出しの操作で全行のチェックを付け外しする", async ({ page }) => {
    await planDinner(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "じゃがいも", amount: 3, unit: "個" },
    ]);
    await openShoppingList(page);

    const all = page.getByLabel(tp.check_all, { exact: true });
    await all.click();
    await expect(rowCheckbox(page, "玉ねぎ")).toBeChecked();
    await expect(rowCheckbox(page, "じゃがいも")).toBeChecked();

    await all.click();
    await expect(rowCheckbox(page, "玉ねぎ")).not.toBeChecked();
    await expect(rowCheckbox(page, "じゃがいも")).not.toBeChecked();
  });

  test("期間を切り替えるとチェックが外れる", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await checkShoppingItem(page, "玉ねぎ");
    await page.getByTestId("shopping-days-3").click();
    await page.getByTestId("shopping-days-7").click();

    await expect(rowCheckbox(page, "玉ねぎ")).not.toBeChecked();
  });
});

test.describe("冷蔵庫への一括登録", () => {
  test("チェックが無いと登録ボタンを押せない", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await expect(registerButton(page)).toBeDisabled();
  });

  test("ダイアログにチェックした材料が初期値つきで並ぶ", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 5, unit: "個" }]);
    await planDinner(page, "生姜焼き", [{ name: "豚ロース", amount: 200, unit: "g" }], 1);
    await openShoppingList(page);

    await checkShoppingItem(page, "玉ねぎ");
    await checkShoppingItem(page, "豚ロース");
    await registerButton(page).click();

    await expect(page.getByRole("dialog").getByTestId("purchase-row")).toHaveCount(2);
    const expected = [
      ["玉ねぎ", "5", "個"],
      ["豚ロース", "200", "g"],
    ];
    for (const [i, [name, amount, unit]] of expected.entries()) {
      const row = dialogRow(page, i);
      await expect(row.getByLabel(tf.field.name)).toHaveValue(name);
      await expect(row.getByLabel(tf.field.amount)).toHaveValue(amount);
      await expect(row.getByLabel(tf.field.unit)).toHaveValue(unit);
      await expect(row.getByLabel(tf.field.expires_at)).toHaveValue("");
    }
  });

  test("量の指定が無い材料は量が空になる", async ({ page }) => {
    await planDinner(page, "おにぎり", [{ name: "塩", amount: 0, unit: "" }]);
    await openShoppingList(page);

    await expect(itemRow(page, "塩")).toContainText(t.amount_unspecified);
    await checkShoppingItem(page, "塩");
    await registerButton(page).click();

    await expect(dialogRow(page, 0).getByLabel(tf.field.amount)).toHaveValue("");
  });

  test("量と期限を直して登録すると在庫に加わる", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 5, unit: "個" }]);
    await openShoppingList(page);

    await checkShoppingItem(page, "玉ねぎ");
    await registerButton(page).click();
    const row = dialogRow(page, 0);
    await row.getByLabel(tf.field.amount).fill("3");
    await row.getByLabel(tf.field.expires_at).fill(dateAfter(9));
    await page.getByRole("dialog").getByRole("button", { name: tp.submit }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);

    await openFridge(page);
    const stock = stockRows(page, "玉ねぎ");
    await expect(stock).toHaveCount(1);
    await expect(stock).toContainText("3 個");
    await expect(stock).toContainText(dateAfter(9));
  });

  test("登録すると在庫の状態が変わりチェックが外れる", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);
    await expect(itemRow(page, "玉ねぎ")).toHaveAttribute("data-stock-status", "none");

    await checkShoppingItem(page, "玉ねぎ");
    await registerButton(page).click();
    await page.getByRole("dialog").getByRole("button", { name: tp.submit }).click();

    await expect(page.getByText(tp.registered.replace("{{count}}", "1"))).toBeVisible();
    await expect(itemRow(page, "玉ねぎ")).toHaveAttribute("data-stock-status", "covered");
    await expect(rowCheckbox(page, "玉ねぎ")).not.toBeChecked();
  });

  test("キャンセルすると在庫もチェックも変わらない", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await checkShoppingItem(page, "玉ねぎ");
    await registerButton(page).click();
    await page.getByRole("dialog").getByRole("button", { name: tp.cancel }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(rowCheckbox(page, "玉ねぎ")).toBeChecked();
    await expect(itemRow(page, "玉ねぎ")).toHaveAttribute("data-stock-status", "none");
  });
});

test.describe("一括登録の入力検証", () => {
  test("量が空の行があると登録されない", async ({ page }) => {
    await planDinner(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 5, unit: "個" },
      { name: "塩", amount: 0, unit: "" },
    ]);
    await openShoppingList(page);

    await page.getByLabel(tp.check_all, { exact: true }).click();
    await registerButton(page).click();
    await page.getByRole("dialog").getByRole("button", { name: tp.submit }).click();

    const salt = page
      .getByRole("dialog")
      .getByTestId("purchase-row")
      .filter({
        has: page.getByText(tf.errors.amount_out_of_range),
      });
    await expect(salt).toHaveCount(1);
    await expect(salt.getByLabel(tf.field.name)).toHaveValue("塩");
    await expect(page.getByRole("dialog")).toBeVisible();

    await page.getByRole("dialog").getByRole("button", { name: tp.cancel }).click();
    await openFridge(page);
    await expect(page.getByText(tf.empty.no_items)).toBeVisible();
  });

  test("食材名を空にした行があると登録されない", async ({ page }) => {
    await planDinner(page, "肉じゃが", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await checkShoppingItem(page, "玉ねぎ");
    await registerButton(page).click();
    await dialogRow(page, 0).getByLabel(tf.field.name).fill("");
    await page.getByRole("dialog").getByRole("button", { name: tp.submit }).click();

    await expect(dialogRow(page, 0)).toContainText(tf.errors.name_empty);
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: tp.cancel }).click();
    await expect(itemRow(page, "玉ねぎ")).toHaveAttribute("data-stock-status", "none");
  });
});
