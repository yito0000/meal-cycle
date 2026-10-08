import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  copyLeftover,
  createRecipeWithIngredients,
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

// 夕食のみ表示ではセルが日付順に並ぶので、同名の品行も DOM 順 = 日付順になる。
function itemRows(page: Page, recipeName: string) {
  return page.getByRole("listitem").filter({ has: page.getByText(recipeName, { exact: true }) });
}

async function openRowMenu(page: Page, recipeName: string, nth: number) {
  await itemRows(page, recipeName)
    .nth(nth)
    .getByRole("button", { name: t.cell_actions.item_menu })
    .click();
  await page.getByRole("menu").waitFor();
}

async function expectCooked(page: Page, recipeName: string, flags: boolean[]) {
  const rows = itemRows(page, recipeName);
  await expect(rows).toHaveCount(flags.length);
  for (const [i, cooked] of flags.entries()) {
    await expect(rows.nth(i)).toHaveAttribute("data-cooked", String(cooked));
    await expect(rows.nth(i).getByRole("img", { name: t.cell.cooked })).toHaveCount(cooked ? 1 : 0);
  }
}

async function markCooked(page: Page, recipeName: string, nth = 0) {
  await openRowMenu(page, recipeName, nth);
  await page.getByRole("menuitem", { name: t.item_menu.mark_cooked, exact: true }).click();
}

test.describe("作った印", () => {
  test("作ったを選ぶと品に印が付く", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await openNextWeek(page);

    await markCooked(page, "肉じゃが");

    await expectCooked(page, "肉じゃが", [true]);
    await openItemMenu(page, "肉じゃが");
    await expect(page.getByRole("menuitem", { name: t.item_menu.unmark_cooked })).toBeVisible();
    await expect(
      page.getByRole("menuitem", { name: t.item_menu.mark_cooked, exact: true }),
    ).toHaveCount(0);
  });

  test("作り置きの 1 行に印を付けると同じ作り置きの全行に印が付く", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await copyLeftover(page, nextWeekDay(0), "dinner", 3, true);
    await openNextWeek(page);

    await markCooked(page, "カレー", 0);

    await expectCooked(page, "カレー", [true, true, true]);
  });

  test("別調理の行には印が広がらない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await assignRecipe(page, nextWeekDay(2), "dinner", id);
    await openNextWeek(page);

    await markCooked(page, "カレー", 0);

    await expectCooked(page, "カレー", [true, false]);
  });

  test("作った印を外す", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await copyLeftover(page, nextWeekDay(0), "dinner", 3, true);
    await openNextWeek(page);
    await markCooked(page, "カレー", 0);
    await expectCooked(page, "カレー", [true, true, true]);

    await openRowMenu(page, "カレー", 1);
    await page.getByRole("menuitem", { name: t.item_menu.unmark_cooked }).click();

    await expectCooked(page, "カレー", [false, false, false]);
  });

  test("作った印は再起動後も保持される", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await openNextWeek(page);
    await markCooked(page, "肉じゃが");
    await expectCooked(page, "肉じゃが", [true]);

    await page.reload();
    await openNextWeek(page);

    await expectCooked(page, "肉じゃが", [true]);
  });
});

test.describe("作り置きの編集と作った印", () => {
  test("作り置きとして広げた行は印を引き継ぐ", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", id);
    await openNextWeek(page);
    await markCooked(page, "カレー");

    await openRowMenu(page, "カレー", 0);
    await page.getByRole("menuitem", { name: t.item_menu.spread }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toContainText(t.spread.title);
    // 既定でチェックされる日をすべて外し、火曜・夕食だけを対象にする。
    const checked = dialog.getByRole("checkbox", { checked: true });
    while ((await checked.count()) > 0) {
      await checked.first().uncheck();
    }
    await dialog
      .locator(`td[data-date="${nextWeekDay(1)}"][data-meal="dinner"]`)
      .getByRole("checkbox")
      .check();
    await dialog.getByRole("radio", { name: t.spread.share_batch_label }).check();
    await dialog.getByRole("button", { name: t.spread.confirm }).click();
    await expect(dialog).toHaveCount(0);

    await expectCooked(page, "カレー", [true, true]);
  });

  test("入れ替えた品は印の無い状態になる", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, nextWeekDay(0), "dinner", nikujaga);
    await openNextWeek(page);
    await markCooked(page, "肉じゃが");

    await openItemMenu(page, "肉じゃが");
    await page.getByRole("menuitem", { name: t.item_menu.replace }).click();
    await page.getByRole("dialog").getByRole("button", { name: "カレー" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expectCooked(page, "カレー", [false]);

    await openItemMenu(page, "カレー");
    await page.getByRole("menuitem", { name: t.item_menu.replace }).click();
    await page.getByRole("dialog").getByRole("button", { name: "肉じゃが" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expectCooked(page, "肉じゃが", [false]);
  });
});
