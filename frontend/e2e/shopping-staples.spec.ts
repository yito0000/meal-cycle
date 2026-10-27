import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStaple,
  assignRecipe,
  checkShoppingItem,
  createRecipeWithIngredients,
  dateAfter,
  openFridge,
  openShoppingList,
  shoppingPantryItem,
  shoppingStapleItem,
  stapleRow,
} from "./helpers.js";

const t = ja.pages.shopping;
const ts = t.staples;

function itemRow(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByRole("cell", { name, exact: true }) });
}

function boughtButton(page: Page, name: string) {
  return shoppingStapleItem(page, name).getByRole("button", { name: ts.bought_button });
}

async function planDinner(
  page: Page,
  recipe: string,
  ingredients: { name: string; amount: number; unit: string }[],
) {
  const id = await createRecipeWithIngredients(page, recipe, ingredients);
  await assignRecipe(page, dateAfter(0), "dinner", id);
}

async function stapleItemNames(page: Page): Promise<string[]> {
  return page
    .getByTestId("shopping-staple-item")
    .evaluateAll((items) => items.map((i) => i.querySelector("span")?.textContent ?? ""));
}

test.describe("常備品の欄", () => {
  test("少ないとないの常備品が献立に関係なく並ぶ", async ({ page }) => {
    await addStaple(page, "牛乳", "low");
    await addStaple(page, "食パン", "out");
    await addStaple(page, "卵", "enough");
    await planDinner(page, "炒め物", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await expect(page.getByTestId("shopping-staple-item")).toHaveCount(2);
    expect(await stapleItemNames(page)).toEqual(["食パン", "牛乳"]);
    await expect(shoppingStapleItem(page, "食パン")).toContainText(
      ja.pages.fridge.staples.state.out,
    );
    await expect(shoppingStapleItem(page, "牛乳")).toContainText(ja.pages.fridge.staples.state.low);
    await expect(shoppingStapleItem(page, "卵")).toHaveCount(0);
  });

  test("常備品を使うレシピが用途として添えられる", async ({ page }) => {
    await addStaple(page, "牛乳", "out");
    await planDinner(page, "グラタン", [{ name: "牛乳", amount: 300, unit: "ml" }]);
    await openShoppingList(page);

    await expect(shoppingStapleItem(page, "牛乳")).toContainText(
      ts.recipes.replace("{{recipes}}", "グラタン"),
    );
  });

  test("常備品はチェックと絞り込みの件数の対象にならない", async ({ page }) => {
    await addStaple(page, "牛乳", "out");
    await planDinner(page, "肉じゃが", [{ name: "じゃがいも", amount: 2, unit: "個" }]);
    await openShoppingList(page);

    await expect(shoppingStapleItem(page, "牛乳").getByRole("checkbox")).toHaveCount(0);
    await expect(page.getByTestId("shopping-filter-all")).toHaveText(
      t.filter.all.replace("{{count}}", "1"),
    );
    await expect(page.getByTestId("shopping-filter-needed")).toHaveText(
      t.filter.needed.replace("{{count}}", "1"),
    );
  });
});

test.describe("常備品と同名の材料", () => {
  test("常備品と同名の材料は材料テーブルに出ない", async ({ page }) => {
    await addStaple(page, "牛乳", "enough");
    await planDinner(page, "グラタン", [
      { name: "牛乳", amount: 300, unit: "ml" },
      { name: "玉ねぎ", amount: 1, unit: "個" },
    ]);
    await openShoppingList(page);

    await expect(page.locator("tbody tr")).toHaveCount(1);
    await expect(itemRow(page, "玉ねぎ")).toHaveCount(1);
    await expect(shoppingStapleItem(page, "牛乳")).toHaveCount(0);
    await expect(shoppingPantryItem(page, "牛乳")).toHaveCount(0);
  });

  test("量を持たない材料も常備品に寄せられる", async ({ page }) => {
    await addStaple(page, "砂糖", "out");
    await planDinner(page, "卵焼き", [
      { name: "砂糖", amount: 1, unit: "大さじ" },
      { name: "卵", amount: 2, unit: "個" },
    ]);
    await openShoppingList(page);

    await expect(shoppingPantryItem(page, "砂糖")).toHaveCount(0);
    await expect(shoppingStapleItem(page, "砂糖")).toContainText(
      ts.recipes.replace("{{recipes}}", "卵焼き"),
    );
  });
});

test.describe("S07 の常備品の「買った」", () => {
  test("買ったを押すと常備品の欄から消える", async ({ page }) => {
    await addStaple(page, "牛乳", "out");
    await openShoppingList(page);

    await boughtButton(page, "牛乳").click();
    await expect(shoppingStapleItem(page, "牛乳")).toHaveCount(0);
    await expect(page.getByRole("button", { name: ts.undo })).toBeVisible();

    await openFridge(page);
    await expect(stapleRow(page, "牛乳")).toHaveAttribute("data-state", "enough");
  });

  test("買ったを元に戻すと元の状態で欄に戻る", async ({ page }) => {
    await addStaple(page, "食パン", "low");
    await openShoppingList(page);

    await boughtButton(page, "食パン").click();
    await expect(shoppingStapleItem(page, "食パン")).toHaveCount(0);
    await page.getByRole("button", { name: ts.undo }).click();

    await expect(shoppingStapleItem(page, "食パン")).toHaveAttribute("data-state", "low");
  });

  test("買ったを押しても購入済チェックは外れない", async ({ page }) => {
    await addStaple(page, "牛乳", "out");
    await planDinner(page, "炒め物", [{ name: "玉ねぎ", amount: 2, unit: "個" }]);
    await openShoppingList(page);
    await checkShoppingItem(page, "玉ねぎ");

    await boughtButton(page, "牛乳").click();
    await expect(shoppingStapleItem(page, "牛乳")).toHaveCount(0);

    await expect(
      page.getByLabel(t.purchase.check_row.replace("{{name}}", "玉ねぎ"), { exact: true }),
    ).toBeChecked();
  });
});

test.describe("買い物リストの空状態", () => {
  test("献立の材料がすべて常備品と同名", async ({ page }) => {
    await addStaple(page, "牛乳", "enough");
    await planDinner(page, "ホットミルク", [{ name: "牛乳", amount: 200, unit: "ml" }]);
    await openShoppingList(page);

    await expect(page.getByTestId("shopping-only-staples")).toHaveText(t.empty.only_staples);
    await expect(page.getByRole("table")).toHaveCount(0);
    await expect(
      page.getByRole("button").filter({ hasText: t.purchase.register.split("{{count}}")[0] }),
    ).toHaveCount(0);
    await expect(page.getByText(t.empty.no_ingredients)).toHaveCount(0);
  });

  test("献立が無い期間でも買う常備品は表示される", async ({ page }) => {
    await addStaple(page, "食パン", "out");
    await openShoppingList(page);

    await expect(page.getByText(t.empty.no_plans)).toBeVisible();
    await expect(shoppingStapleItem(page, "食パン")).toHaveCount(1);
    await expect(page.getByRole("heading", { name: t.dishes.title })).toHaveCount(0);
  });
});
