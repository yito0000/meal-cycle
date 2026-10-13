import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  createRecipeWithIngredients,
  mealPlanItemRows,
  openItemMenu,
} from "./helpers.js";

const t = ja.pages.meal_plan;

// 今週の曜日だと実行日によって過去の日付になるため、来週の月曜を起点にする。
function nextWeekDay(offsetFromMonday: number): string {
  const now = new Date();
  const toMonday = (now.getDay() + 6) % 7;
  const d = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate() - toMonday + 7 + offsetFromMonday,
  );
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

async function openNextWeek(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
  await page.getByTestId("period-next").click();
}

function nameButton(page: Page, recipeName: string) {
  return mealPlanItemRows(page, recipeName).getByRole("button", { name: recipeName, exact: true });
}

test.describe("セル内の品表示", () => {
  test("複数品は position 順に縦積みされる", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", nikujaga);
    await assignRecipe(page, nextWeekDay(0), "dinner", miso);
    await openNextWeek(page);

    const rows = page.getByRole("listitem");
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toContainText("肉じゃが");
    await expect(rows.nth(1)).toContainText("みそ汁");
    for (const row of await rows.all()) {
      await expect(row.getByRole("checkbox", { name: t.cell.cooked })).toBeVisible();
      await expect(row).not.toContainText("⋯");
      await expect(row.getByRole("button")).toHaveCount(1);
    }
  });

  test("品行をクリックすると品のメニューが開く", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await openNextWeek(page);

    await openItemMenu(page, "肉じゃが");

    const menu = page.getByRole("menu");
    await expect(menu.getByRole("menuitem", { name: t.item_menu.replace })).toBeVisible();
    await expect(menu.getByRole("menuitem", { name: t.item_menu.remove })).toBeVisible();
    const rowBox = await mealPlanItemRows(page, "肉じゃが").boundingBox();
    const menuBox = await menu.boundingBox();
    if (!rowBox || !menuBox) throw new Error("bounding box missing");
    expect(menuBox.x).toBeGreaterThanOrEqual(rowBox.x + rowBox.width);
  });

  test("キーボードで品のメニューを開く", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await openNextWeek(page);

    await nameButton(page, "肉じゃが").focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("menu")).toBeVisible();

    await page.keyboard.press("Escape");
    await expect(page.getByRole("menu")).toHaveCount(0);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(1);
    await expect(nameButton(page, "肉じゃが")).toBeFocused();
  });

  test("1 行に収まらないレシピ名は 2 行まで折り返される", async ({ page }) => {
    // 1920×1080 (拡大率 100%) の画面での CSS 幅 (proposal.md Why)。
    await page.setViewportSize({ width: 1088, height: 612 });
    const longName = "ほうれん草とベーコンの胡麻和えサラダ";
    const two = await createRecipeWithIngredients(page, "ほうれん草の胡麻和え", []);
    const clamped = await createRecipeWithIngredients(page, longName, []);
    await assignRecipe(page, nextWeekDay(0), "dinner", two);
    await assignRecipe(page, nextWeekDay(1), "dinner", clamped);
    await openNextWeek(page);

    const measure = (name: string) =>
      nameButton(page, name)
        .locator("span")
        .evaluate((el) => ({
          lines: Math.round(el.clientHeight / Number.parseFloat(getComputedStyle(el).lineHeight)),
          overflows: el.scrollHeight > el.clientHeight,
        }));

    expect((await measure("ほうれん草の胡麻和え")).lines).toBe(2);
    expect(await measure(longName)).toEqual({ lines: 2, overflows: true });
  });
});
