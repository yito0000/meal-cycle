import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStockItem,
  assignRecipe,
  copyLeftover,
  createRecipeWithIngredients,
  dateAfter,
  openShoppingList,
  shoppingPantryItem,
} from "./helpers.js";

// 対象期間はサイドバーから開いたときの「今日から 7 日間」を基準にし、日付は今日からの相対で組み立てる。
const t = ja.pages.shopping;
const tm = ja.pages.meal_plan;

// 対象期間は yyyy/mm/dd で表示される (design.md D15)。引数は YYYY-MM-DD。
function periodText(start: string, end: string): string {
  const slash = (d: string) => d.replaceAll("-", "/");
  return t.period.replace("{{start}}", slash(start)).replace("{{end}}", slash(end));
}

// 開始日のカレンダーを開いて date (YYYY-MM-DD) を選ぶ。表示中の月に無ければ翌月へ送る。
async function pickStartDate(page: Page, date: string) {
  await page.getByLabel(t.picker.start).click();
  const cell = page.locator(`[data-day="${date}"]:not([data-outside])`);
  if ((await cell.count()) === 0) {
    // nav 内のボタンは [前月, 翌月] の順。
    await page.getByRole("dialog").locator("nav button").last().click();
  }
  await cell.getByRole("button").click();
}

// 今日を含む週の月曜 (S01 の既定の開始日) からの相対日付。
function mondayAfter(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - ((d.getDay() + 6) % 7) + days);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function monthDay(date: string): string {
  const [, m, d] = date.split("-").map(Number);
  return `${m}/${d}`;
}

function itemRow(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByRole("cell", { name, exact: true }) });
}

function dish(page: Page, recipeName: string) {
  return page.getByTestId("shopping-dish").filter({ hasText: recipeName });
}

async function ingredientNames(page: Page): Promise<string[]> {
  // 先頭はチェックの列なので、食材名は 2 列目。
  return page.locator("tbody tr td:nth-child(2)").allInnerTexts();
}

test.describe("買い物リストの対象期間", () => {
  test("S01 の表示期間を引き継ぐ", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: tm.title })).toBeVisible();
    await page.getByTestId("period-weeks-2").click();
    await page.getByRole("button", { name: tm.shopping_cta }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByTestId("shopping-period")).toHaveText(
      periodText(mondayAfter(0), mondayAfter(13)),
    );
    await expect(page.getByTestId("shopping-days-14")).toHaveAttribute("aria-pressed", "true");
  });

  test("S01 を先に開いていてもサイドバーからは既定期間になる", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("period-weeks-4").click();
    await openShoppingList(page);

    await expect(page.getByTestId("shopping-period")).toHaveText(
      periodText(dateAfter(0), dateAfter(6)),
    );
    await expect(page.getByTestId("shopping-days-7")).toHaveAttribute("aria-pressed", "true");
  });
});

test.describe("S07 上での期間の切替", () => {
  test("日数を切り替える", async ({ page }) => {
    await openShoppingList(page);

    await page.getByTestId("shopping-days-3").click();
    await expect(page.getByTestId("shopping-period")).toHaveText(
      periodText(dateAfter(0), dateAfter(2)),
    );
  });

  test("開始日を変える", async ({ page }) => {
    await openShoppingList(page);

    await pickStartDate(page, dateAfter(1));
    await expect(page.getByTestId("shopping-period")).toHaveText(
      periodText(dateAfter(1), dateAfter(7)),
    );
  });
});

test.describe("作り置き単位の計上", () => {
  test("作り置き 3 日分でも材料は 1 回分だけ計上される", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await copyLeftover(page, dateAfter(0), "dinner", 3, true);
    await openShoppingList(page);

    await expect(itemRow(page, "玉ねぎ")).toHaveCount(1);
    await expect(itemRow(page, "玉ねぎ")).toContainText("2 個");
  });
});

test.describe("同一食材の合算", () => {
  test("複数レシピで必要な同一食材は 1 行に合算される", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
    ]);
    const curry = await createRecipeWithIngredients(page, "カレー", [
      { name: "玉ねぎ", amount: 3, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", nikujaga);
    await assignRecipe(page, dateAfter(2), "dinner", curry);
    await openShoppingList(page);

    await expect(itemRow(page, "玉ねぎ")).toHaveCount(1);
    await expect(itemRow(page, "玉ねぎ")).toContainText("5 個");
    await expect(itemRow(page, "玉ねぎ")).toContainText("肉じゃが, カレー");
  });

  test("同一時間帯内の複数品も合算対象になる", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 1, unit: "個" },
    ]);
    const misoshiru = await createRecipeWithIngredients(page, "みそ汁", [
      { name: "玉ねぎ", amount: 0.5, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(1), "dinner", nikujaga);
    await assignRecipe(page, dateAfter(1), "dinner", misoshiru);
    await openShoppingList(page);

    await expect(itemRow(page, "玉ねぎ")).toHaveCount(1);
    await expect(itemRow(page, "玉ねぎ")).toContainText("1.5 個");
    await expect(itemRow(page, "玉ねぎ")).toContainText("肉じゃが, みそ汁");
  });

  test("ひらがなとカタカナの違う食材名は 1 行に合算される", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "たまねぎ", amount: 2, unit: "個" },
    ]);
    const curry = await createRecipeWithIngredients(page, "カレー", [
      { name: "タマネギ", amount: 1, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", nikujaga);
    await assignRecipe(page, dateAfter(2), "dinner", curry);
    await openShoppingList(page);

    expect(await ingredientNames(page)).toEqual(["たまねぎ"]);
    await expect(itemRow(page, "たまねぎ")).toContainText("3 個");
    await expect(itemRow(page, "たまねぎ")).toContainText("肉じゃが, カレー");
  });

  test("表記の違う在庫も同じ食材として数える", async ({ page }) => {
    const curry = await createRecipeWithIngredients(page, "カレー", [
      { name: "タマネギ", amount: 2, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", curry);
    await addStockItem(page, { name: "たまねぎ", amount: 3, unit: "個" });
    await openShoppingList(page);

    await expect(itemRow(page, "タマネギ")).toHaveAttribute("data-stock-status", "covered");
    await expect(itemRow(page, "タマネギ")).toContainText(t.stock_status.covered);
  });

  test("漢字とかなの違う食材名は別の行になる", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
    ]);
    const curry = await createRecipeWithIngredients(page, "カレー", [
      { name: "たまねぎ", amount: 1, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", nikujaga);
    await assignRecipe(page, dateAfter(2), "dinner", curry);
    await openShoppingList(page);

    await expect(itemRow(page, "玉ねぎ")).toContainText("2 個");
    await expect(itemRow(page, "たまねぎ")).toContainText("1 個");
  });
});

test.describe("在庫の状態表示", () => {
  test("在庫で一部を賄える食材は一部在庫ありになる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(1), "dinner", id);
    await addStockItem(page, { name: "玉ねぎ", amount: 1, unit: "個" });
    await openShoppingList(page);

    const row = itemRow(page, "玉ねぎ");
    await expect(row).toContainText("2 個");
    await expect(row).toHaveAttribute("data-stock-status", "partial");
    await expect(row).toContainText(t.stock_status.partial);
  });

  test("在庫で足りる食材は在庫ありになる", async ({ page }) => {
    const tamagoyaki = await createRecipeWithIngredients(page, "卵焼き", [
      { name: "卵", amount: 2, unit: "個" },
    ]);
    const curry = await createRecipeWithIngredients(page, "カレー", [
      { name: "じゃがいも", amount: 2, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", tamagoyaki);
    await assignRecipe(page, dateAfter(2), "dinner", curry);
    await addStockItem(page, { name: "卵", amount: 5, unit: "個" });
    await openShoppingList(page);

    await expect(itemRow(page, "卵")).toHaveAttribute("data-stock-status", "covered");
    await expect(itemRow(page, "卵")).toContainText(t.stock_status.covered);
    expect(await ingredientNames(page)).toEqual(["じゃがいも", "卵"]);
  });

  test("期限切れの在庫は在庫として扱わない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "ホットミルク", [
      { name: "牛乳", amount: 1, unit: "本" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await addStockItem(page, { name: "牛乳", amount: 1, unit: "本", expiresInDays: -2 });
    await openShoppingList(page);

    await expect(itemRow(page, "牛乳")).toHaveAttribute("data-stock-status", "none");
    await expect(itemRow(page, "牛乳")).toContainText(t.stock_status.none);
  });
});

test.describe("期間の献立一覧", () => {
  test("作り置きは献立の一覧で 1 行にまとまる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await copyLeftover(page, dateAfter(0), "dinner", 3, true);
    await openShoppingList(page);

    const row = dish(page, "肉じゃが");
    await expect(row).toHaveCount(1);
    const [startMonth] = dateAfter(0).split("-").slice(1).map(Number);
    const [endMonth, endDay] = dateAfter(2).split("-").slice(1).map(Number);
    const end = endMonth === startMonth ? String(endDay) : monthDay(dateAfter(2));
    await expect(row.getByTestId("shopping-dish-dates")).toHaveText(
      t.dishes.date_range.replace("{{start}}", monthDay(dateAfter(0))).replace("{{end}}", end),
    );
    await expect(row.getByTestId("shopping-dish-meals")).toHaveText(t.dishes.meal.dinner);
    await expect(row.getByRole("img", { name: t.dishes.batch })).toBeVisible();
  });

  test("期間より前から続く作り置きには印が付く", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "カレー", [
      { name: "玉ねぎ", amount: 3, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(-1), "dinner", id);
    await copyLeftover(page, dateAfter(-1), "dinner", 2, true);
    await openShoppingList(page);

    const row = dish(page, "カレー");
    await expect(row.getByTestId("shopping-dish-started-before")).toHaveText(
      t.dishes.started_before,
    );
    await expect(row.getByTestId("shopping-dish-dates")).toHaveText(monthDay(dateAfter(0)));
  });
});

test.describe("量を持たない材料の欄", () => {
  test("量を持たない材料は材料テーブルとは別の欄に並ぶ", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "塩", amount: 0, unit: "" },
      { name: "醤油", amount: 2, unit: "大さじ" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openShoppingList(page);

    expect(await ingredientNames(page)).toEqual(["玉ねぎ"]);
    await expect(page.getByRole("heading", { name: t.pantry.title })).toBeVisible();
    for (const name of ["塩", "醤油"]) {
      const item = shoppingPantryItem(page, name);
      await expect(item).toHaveCount(1);
      await expect(item).toHaveAttribute("data-stock-status", "none");
      await expect(item).not.toContainText(t.stock_status.none);
      await expect(item).not.toContainText(t.stock_status.covered);
    }
  });

  test("量を持たない材料は単位が違う在庫でも在庫ありになる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "醤油", amount: 2, unit: "大さじ" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await addStockItem(page, { name: "醤油", amount: 1, unit: "本" });
    await openShoppingList(page);

    const item = shoppingPantryItem(page, "醤油");
    await expect(item).toHaveAttribute("data-stock-status", "covered");
    await expect(item).toContainText(t.stock_status.covered);
  });

  test("量を持たない材料にはチェックを付けられない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
      { name: "醤油", amount: 2, unit: "大さじ" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openShoppingList(page);

    await expect(shoppingPantryItem(page, "醤油")).toHaveCount(1);
    await expect(shoppingPantryItem(page, "醤油").getByRole("checkbox")).toHaveCount(0);
    await expect(
      page.getByLabel(t.purchase.check_row.replace("{{name}}", "醤油"), { exact: true }),
    ).toHaveCount(0);
  });

  test("表記の違う量なし材料は 1 項目にまとまる", async ({ page }) => {
    const itamemono = await createRecipeWithIngredients(page, "炒め物", [
      { name: "ごま油", amount: 1, unit: "小さじ" },
    ]);
    const namul = await createRecipeWithIngredients(page, "ナムル", [
      { name: "ゴマ油", amount: 1, unit: "大さじ" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", itamemono);
    await assignRecipe(page, dateAfter(2), "dinner", namul);
    await openShoppingList(page);

    await expect(page.getByTestId("shopping-pantry-item")).toHaveCount(1);
    await expect(shoppingPantryItem(page, "ごま油")).toContainText("炒め物");
    await expect(shoppingPantryItem(page, "ごま油")).toContainText("ナムル");
  });
});

test.describe("買い物リストの空状態", () => {
  test("期間に献立が無い", async ({ page }) => {
    await openShoppingList(page);

    await expect(page.getByText(t.empty.no_plans)).toBeVisible();
    await expect(page.getByRole("heading", { name: t.dishes.title })).toHaveCount(0);
  });

  test("献立のレシピに材料が無い", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "ご飯", []);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openShoppingList(page);

    await expect(page.getByText(t.empty.no_ingredients)).toBeVisible();
    await expect(page.getByRole("heading", { name: t.dishes.title })).toBeVisible();
    await expect(dish(page, "ご飯")).toHaveCount(1);
  });

  test("量を持たない材料しか無い", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "醤油", amount: 2, unit: "大さじ" },
      { name: "塩", amount: 0, unit: "" },
    ]);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openShoppingList(page);

    await expect(page.getByTestId("shopping-no-quantified")).toHaveText(t.empty.no_quantified);
    await expect(page.getByRole("table")).toHaveCount(0);
    await expect(
      page.getByRole("button").filter({ hasText: t.purchase.register.split("{{count}}")[0] }),
    ).toHaveCount(0);
    await expect(shoppingPantryItem(page, "醤油")).toHaveCount(1);
    await expect(shoppingPantryItem(page, "塩")).toHaveCount(1);
  });
});
