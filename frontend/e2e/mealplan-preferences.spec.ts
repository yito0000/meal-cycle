import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";

// テスト名は docs/scenarios/mealplan.feature の シナリオ: 行と一字一句揃える。
const t = ja.pages.meal_plan;

async function openMealCalendar(page: import("@playwright/test").Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
}

test.describe("機能: 献立カレンダーの表示", () => {
  test("時間帯切替が再起動後も保持される", async ({ page }) => {
    await openMealCalendar(page);

    // 既定は「夕食のみ」。朝食 / 昼食を on にして「3 食表示」にする。
    const breakfast = page.getByRole("checkbox", { name: t.meal.breakfast });
    const lunch = page.getByRole("checkbox", { name: t.meal.lunch });
    await breakfast.check();
    await lunch.check();
    await expect(breakfast).toBeChecked();
    await expect(lunch).toBeChecked();

    // reload で state 初期化を強制し、localStorage 経由で保持されているか確認する。
    await page.reload();
    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByRole("checkbox", { name: t.meal.breakfast })).toBeChecked();
    await expect(page.getByRole("checkbox", { name: t.meal.lunch })).toBeChecked();
    await expect(page.getByRole("checkbox", { name: t.meal.dinner })).toBeChecked();
  });

  test("期間長切替が再起動後も保持される", async ({ page }) => {
    await openMealCalendar(page);

    await page.getByTestId("period-weeks-2").click();
    await expect(page.getByTestId("period-weeks-2")).toHaveAttribute("aria-pressed", "true");

    await page.reload();
    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByTestId("period-weeks-2")).toHaveAttribute("aria-pressed", "true");
    await expect(page.getByTestId("period-calendar")).toHaveAttribute("data-period-weeks", "2");
  });

  test("期間の開始日は毎回「今週」にリセットされる", async ({ page }) => {
    await openMealCalendar(page);
    const todayBtn = page.getByTestId("period-today");
    const initial = await todayBtn.getAttribute("data-period-start");
    if (!initial) throw new Error("initial data-period-start missing");

    // ＞ を 2 回押して未来の週へ移動する。
    await page.getByTestId("period-next").click();
    await page.getByTestId("period-next").click();
    await expect(todayBtn).not.toHaveAttribute("data-period-start", initial);

    // reload 後、periodStart は「今日を含む週の月曜」に戻る (永続化対象外)。
    await page.reload();
    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByTestId("period-today")).toHaveAttribute("data-period-start", initial);
  });
});
