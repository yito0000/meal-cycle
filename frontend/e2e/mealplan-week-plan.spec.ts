import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipeWithIngredients, mealPlanCell, mealPlanItemRows } from "./helpers.js";

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

test.describe("週の候補の呼び出し", () => {
  test("週の候補ダイアログを開く", async ({ page }) => {
    await recipes(page, ["カレー", "肉じゃが"]);
    await openNextWeek(page);
    await page.getByTestId("period-weeks-2").click();
    await openWeekPlan(page);

    expect(await itemDates(page, "stock")).toEqual([range(0, 2), range(3, 4)]);
  });
});

test.describe("週の候補の適用", () => {
  // 案は 1 つだけになり、月〜水に先頭のレシピ、木〜金に次のレシピが当たる (一度も作っていないレシピは名前の順)。
  async function applyCurryAndNikujaga(page: Page) {
    await recipes(page, ["肉じゃが", "カレー"]);
    await openNextWeek(page);
    await openWeekPlan(page);
    await expect(planOf(page, "stock").getByTestId("week-plan-item").first()).toContainText(
      "カレー",
    );
    await planOf(page, "stock").getByRole("button", { name: tw.apply }).click();
  }

  test("案を適用すると作り置きとして割り当たる", async ({ page }) => {
    await applyCurryAndNikujaga(page);

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByText(tw.applied.replace("{{count}}", "5"))).toBeVisible();
    for (let i = 0; i < 3; i++) {
      await expect(
        mealPlanCell(page, nextWeekDay(i), "dinner").getByText("カレー", { exact: true }),
      ).toBeVisible();
    }
    for (let i = 3; i < 5; i++) {
      await expect(
        mealPlanCell(page, nextWeekDay(i), "dinner").getByText("肉じゃが", { exact: true }),
      ).toBeVisible();
    }
    const curry = mealPlanItemRows(page, "カレー");
    await expect(curry).toHaveCount(3);
    // 同じ作り置きの行は同じ batch_id を持ち、S01 で同じ枠色になる。
    const batches = await curry.evaluateAll((rows) =>
      rows.map((r) => r.getAttribute("data-batch-id") ?? ""),
    );
    expect(new Set(batches).size).toBe(1);
    expect(batches[0]).not.toBe("");
  });

  test("適用を元に戻すと割り当てた行が消える", async ({ page }) => {
    await applyCurryAndNikujaga(page);
    await expect(mealPlanItemRows(page, "カレー")).toHaveCount(3);

    await page.getByRole("button", { name: t.toast.undo }).click();

    await expect(mealPlanItemRows(page, "カレー")).toHaveCount(0);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(0);
  });
});
