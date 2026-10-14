import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  createRecipeWithIngredients,
  mealPlanCell,
  mealPlanItemRows,
  openItemMenu,
  openOverflowList,
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

// 4 週表示のセル内表示上限 (2 品) を 1 品超える 3 品を、来週の月曜・夕食に並べる。
async function openFourWeeksWithThreeDishes(page: Page) {
  for (const name of ["肉じゃが", "みそ汁", "ほうれん草の胡麻和え"]) {
    const id = await createRecipeWithIngredients(page, name, []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
  }
  await openNextWeek(page);
  await page.getByTestId("period-weeks-4").click();
  await expect(page.getByTestId("period-calendar")).toHaveAttribute("data-period-weeks", "4");
}

const moreOne = t.cell.more_items.replace("{{count}}", "1");

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

  test("「他 N 件」を押すとセルの全品の一覧が開く", async ({ page }) => {
    await openFourWeeksWithThreeDishes(page);
    const cell = mealPlanCell(page, nextWeekDay(0), "dinner");
    await expect(cell.getByRole("button", { name: moreOne })).toBeVisible();

    const list = await openOverflowList(page, nextWeekDay(0), "dinner");

    const rows = list.getByRole("listitem");
    await expect(rows).toHaveCount(3);
    await expect(rows.nth(0)).toContainText("肉じゃが");
    await expect(rows.nth(1)).toContainText("みそ汁");
    await expect(rows.nth(2)).toContainText("ほうれん草の胡麻和え");
  });

  test("畳まれた品に作った印を付けられる", async ({ page }) => {
    await openFourWeeksWithThreeDishes(page);
    const list = await openOverflowList(page, nextWeekDay(0), "dinner");

    const checkbox = list
      .getByRole("listitem")
      .filter({ hasText: "ほうれん草の胡麻和え" })
      .getByRole("checkbox", { name: t.cell.cooked });
    await checkbox.click();
    await expect(checkbox).toBeChecked();

    await page.keyboard.press("Escape");
    await page.getByTestId("period-weeks-1").click();
    await expect(page.getByTestId("period-calendar")).toHaveAttribute("data-period-weeks", "1");
    await expect(
      mealPlanItemRows(page, "ほうれん草の胡麻和え").getByRole("checkbox", { name: t.cell.cooked }),
    ).toBeChecked();
  });

  test("畳まれた品のメニューから削除できる", async ({ page }) => {
    await openFourWeeksWithThreeDishes(page);
    const list = await openOverflowList(page, nextWeekDay(0), "dinner");

    await list
      .getByRole("listitem")
      .filter({ hasText: "ほうれん草の胡麻和え" })
      .getByRole("button", { name: "ほうれん草の胡麻和え", exact: true })
      .click();
    await page.getByRole("menuitem", { name: t.item_menu.remove }).click();

    const cell = mealPlanCell(page, nextWeekDay(0), "dinner");
    const rows = cell.getByRole("listitem");
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toContainText("肉じゃが");
    await expect(rows.nth(1)).toContainText("みそ汁");
    await expect(cell.getByRole("button", { name: moreOne })).toHaveCount(0);
    await expect(page.getByTestId("cell-overflow-list")).toHaveCount(0);
  });

  test("一覧は Esc で閉じる", async ({ page }) => {
    await openFourWeeksWithThreeDishes(page);
    await openOverflowList(page, nextWeekDay(0), "dinner");

    await page.keyboard.press("Escape");

    await expect(page.getByTestId("cell-overflow-list")).toHaveCount(0);
    const cell = mealPlanCell(page, nextWeekDay(0), "dinner");
    await expect(cell.getByRole("listitem")).toHaveCount(2);
    await expect(cell.getByRole("button", { name: moreOne })).toBeVisible();
    const list = await openOverflowList(page, nextWeekDay(0), "dinner");
    await expect(list.getByRole("listitem")).toHaveCount(3);
  });
});
