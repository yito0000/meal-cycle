import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipeDetailed, dateAfter, mealPlanCell } from "./helpers.js";

const t = ja.pages.meal_plan;
const ts = t.picker.suggestion;
const tset = ja.pages.settings;

async function openPickerToday(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
  await mealPlanCell(page, dateAfter(0), "dinner")
    .getByRole("button", { name: t.cell_actions.add_first })
    .click();
  const dialog = page.getByRole("dialog", { name: t.picker.title });
  await dialog.waitFor();
  return dialog;
}

function byTime(page: Page, minutes = 20) {
  return page
    .getByRole("dialog")
    .getByRole("region", { name: ts.by_time_title.replace("{{minutes}}", String(minutes)) });
}

function minutesText(n: number): string {
  return ts.by_time_reason.replace("{{minutes}}", String(n));
}

test.describe("時短の候補", () => {
  test("所要時間の短いレシピが候補に出る", async ({ page }) => {
    await createRecipeDetailed(page, { name: "豚キムチ", cookMinutes: 15 });
    await createRecipeDetailed(page, { name: "肉じゃが", cookMinutes: 40 });

    await openPickerToday(page);

    const item = byTime(page).getByTestId("suggestion-item").filter({ hasText: "豚キムチ" });
    await expect(item).toHaveCount(1);
    await expect(item).toContainText(minutesText(15));
    await expect(
      byTime(page).getByTestId("suggestion-item").filter({ hasText: "肉じゃが" }),
    ).toHaveCount(0);
  });

  test("所要時間を持たないレシピは候補に出ない", async ({ page }) => {
    await createRecipeDetailed(page, { name: "豚汁" });

    await openPickerToday(page);

    await expect(byTime(page).getByText(ts.none)).toBeVisible();
  });

  test("S09 の分数を変えると候補が変わる", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが", cookMinutes: 30 });
    await page.goto("/");
    await page.getByRole("link", { name: ja.nav.settings }).click();
    await page.getByLabel(tset.field.quick_cook_minutes, { exact: true }).fill("30");
    await page.getByRole("button", { name: tset.save, exact: true }).click();
    await expect(page.getByText(tset.toast.saved)).toBeVisible();

    await page.getByRole("link", { name: ja.nav.meal_plan }).click();
    await mealPlanCell(page, dateAfter(0), "dinner")
      .getByRole("button", { name: t.cell_actions.add_first })
      .click();

    await expect(
      byTime(page, 30).getByTestId("suggestion-item").filter({ hasText: "肉じゃが" }),
    ).toHaveCount(1);
  });
});
