import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { addStockItem, assignRecipe, createRecipeWithIngredients } from "./helpers.js";

const t = ja.pages.meal_plan;
const tw = t.week_plan;

// 来週の月曜から offset 日後。今週だと実行日によって過去の日付が混ざるので、来週を対象にする。
function nextWeekDay(offset: number): string {
  const now = new Date();
  const toMonday = (now.getDay() + 6) % 7;
  const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - toMonday + 7 + offset);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function monthDay(date: string): string {
  const [, m, d] = date.split("-").map(Number);
  return `${m}/${d}`;
}

// 連続する日付の範囲の表示 (S07 の献立の欄と同じ「9/24〜26」)。月をまたぐときは終端にも月を付ける。
function range(from: number, to: number): string {
  const a = nextWeekDay(from);
  const b = nextWeekDay(to);
  const [, am] = a.split("-").map(Number);
  const [, bm, bd] = b.split("-").map(Number);
  return am === bm ? `${monthDay(a)}〜${bd}` : `${monthDay(a)}〜${monthDay(b)}`;
}

async function openNextWeek(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
  await page.getByTestId("period-next").click();
}

async function openWeekPlan(page: Page) {
  await page.getByTestId("week-plan-cta").click();
  const dialog = page.getByRole("dialog", { name: tw.title });
  await dialog.waitFor();
  return dialog;
}

function plans(page: Page) {
  return page.getByRole("dialog", { name: tw.title }).getByTestId("week-plan");
}

function planOf(page: Page, kind: "stock" | "recency" | "balanced") {
  return page.getByRole("dialog", { name: tw.title }).locator(`[data-kind="${kind}"]`);
}

async function itemDates(page: Page, kind: "stock" | "recency" | "balanced"): Promise<string[]> {
  return planOf(page, kind)
    .getByTestId("week-plan-item")
    .evaluateAll((items) => items.map((i) => i.querySelector("span")?.textContent ?? ""));
}

async function recipes(page: Page, names: string[]) {
  for (const n of names) {
    await createRecipeWithIngredients(page, n, []);
  }
}

test.describe("週の候補の対象", () => {
  test("空いた 5 コマは 3 日と 2 日の 2 組になる", async ({ page }) => {
    await recipes(page, ["カレー", "肉じゃが"]);
    await openNextWeek(page);
    await openWeekPlan(page);

    await expect(plans(page)).not.toHaveCount(0);
    expect(await itemDates(page, "stock")).toEqual([range(0, 2), range(3, 4)]);
  });

  test("品のあるセルは対象にしない", async ({ page }) => {
    await recipes(page, ["カレー", "肉じゃが"]);
    const id = await createRecipeWithIngredients(page, "味噌汁", []);
    await assignRecipe(page, nextWeekDay(2), "dinner", id);
    await openNextWeek(page);
    await openWeekPlan(page);

    expect(await itemDates(page, "stock")).toEqual([range(0, 1), range(3, 4)]);
  });

  test("空いたコマが無いとその旨を示す", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "カレー", []);
    for (let i = 0; i < 5; i++) {
      await assignRecipe(page, nextWeekDay(i), "dinner", id);
    }
    await openNextWeek(page);
    await openWeekPlan(page);

    await expect(page.getByTestId("week-plan-no-slots")).toHaveText(tw.no_slots);
    await expect(plans(page)).toHaveCount(0);
  });
});

test.describe("週の候補の案", () => {
  test("使い切り優先の案は期限の近い在庫を使うレシピから当てる", async ({ page }) => {
    await addStockItem(page, { name: "玉ねぎ", amount: 2, unit: "個", expiresInDays: 2 });
    await createRecipeWithIngredients(page, "カレー", []);
    await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 1, unit: "個" },
    ]);
    await openNextWeek(page);
    await openWeekPlan(page);

    const first = planOf(page, "stock").getByTestId("week-plan-item").first();
    await expect(first).toContainText("肉じゃが");
    await expect(first).toContainText(range(0, 2));
    await expect(first).toContainText("玉ねぎ");
  });

  test("同じ内容の案はまとめる", async ({ page }) => {
    await recipes(page, ["カレー", "肉じゃが"]);
    await openNextWeek(page);
    await openWeekPlan(page);

    await expect(plans(page)).toHaveCount(1);
  });
});
