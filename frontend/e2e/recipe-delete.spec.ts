import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  createRecipeDetailed,
  dateAfter,
  deleteRecipe,
  deleteRecipeFromDetail,
  detailBreadcrumb,
  openItemMenu,
  openRecipeDetail,
} from "./helpers.js";

const t = ja.pages.recipes;
const mp = ja.pages.meal_plan;

async function openMealCalendar(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: ja.nav.meal_plan }).click();
  await expect(page.getByRole("heading", { name: mp.title })).toBeVisible();
}

async function openDetailFromMealCalendar(page: Page, recipeName: string) {
  await openItemMenu(page, recipeName);
  await page.getByRole("menuitem", { name: mp.item_menu.view_detail }).click();
  await detailBreadcrumb(page).getByText(recipeName, { exact: true }).waitFor();
}

test.describe("S03 詳細画面からのレシピの削除", () => {
  test("確認してレシピを削除すると一覧から消える", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが" });
    await createRecipeDetailed(page, { name: "豚汁" });
    await openRecipeDetail(page, "肉じゃが");

    await page.getByRole("main").getByRole("button", { name: t.detail.delete }).click();
    const confirm = page.getByRole("dialog").filter({ hasText: t.delete_confirm.body });
    await expect(confirm).toBeVisible();
    await expect(confirm).toContainText(t.delete_confirm.title.replace("{{name}}", "肉じゃが"));

    await confirm.getByRole("button", { name: t.delete_confirm.confirm }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByRole("heading", { name: t.title, exact: true })).toBeVisible();
    await expect(
      page.getByText(t.delete_confirm.done.replace("{{name}}", "肉じゃが")),
    ).toBeVisible();
    await expect(page.getByRole("cell", { name: "肉じゃが", exact: true })).toHaveCount(0);
    await expect(page.getByRole("cell", { name: "豚汁", exact: true })).toBeVisible();
  });

  test("確認ダイアログでキャンセルすると削除しない", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが" });
    await openRecipeDetail(page, "肉じゃが");

    await page.getByRole("main").getByRole("button", { name: t.detail.delete }).click();
    const confirm = page.getByRole("dialog").filter({ hasText: t.delete_confirm.body });
    await confirm.getByRole("button", { name: t.delete_confirm.cancel }).click();

    await expect(confirm).toHaveCount(0);
    await expect(
      page.getByRole("main").getByRole("heading", { name: "肉じゃが", exact: true }),
    ).toBeVisible();

    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.recipes }).click();
    await expect(page.getByRole("cell", { name: "肉じゃが", exact: true })).toBeVisible();
  });

  test("S01 から開いた詳細画面で削除すると S01 に戻る", async ({ page }) => {
    const id = await createRecipeDetailed(page, { name: "肉じゃが" });
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openMealCalendar(page);
    const periodStart = await page.getByTestId("period-today").getAttribute("data-period-start");

    await openDetailFromMealCalendar(page, "肉じゃが");
    await deleteRecipeFromDetail(page);

    await expect(page.getByRole("heading", { name: mp.title })).toBeVisible();
    await expect(page.getByTestId("period-today")).toHaveAttribute(
      "data-period-start",
      periodStart ?? "",
    );
  });
});

test.describe("削除済みレシピの除外範囲", () => {
  test("削除済みレシピはレシピ選択ダイアログに出ない", async ({ page }) => {
    const id = await createRecipeDetailed(page, { name: "肉じゃが" });
    await createRecipeDetailed(page, { name: "豚汁" });
    await deleteRecipe(page, id);
    await openMealCalendar(page);

    await page.getByRole("button", { name: mp.cell_actions.add_first }).first().click();
    const picker = page.getByRole("dialog");
    await expect(picker).toContainText(mp.picker.title);

    await expect(picker.getByRole("button", { name: "豚汁" })).toBeVisible();
    await expect(picker.getByRole("button", { name: "肉じゃが" })).toHaveCount(0);
  });

  test("全レシピを削除すると空状態になる", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが" });
    await openRecipeDetail(page, "肉じゃが");

    await deleteRecipeFromDetail(page);

    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
  });
});

test.describe("削除済みレシピを使った献立の保持", () => {
  test("削除したレシピの品が S01 に残る", async ({ page }) => {
    const id = await createRecipeDetailed(page, { name: "肉じゃが" });
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openRecipeDetail(page, "肉じゃが");
    await deleteRecipeFromDetail(page);

    await openMealCalendar(page);

    await expect(page.getByText("肉じゃが", { exact: true })).toBeVisible();
  });
});

test.describe("S03 レシピ詳細画面の表示", () => {
  test("削除済みレシピは内容と削除済みの旨を表示し、操作ボタンを出さない", async ({ page }) => {
    const id = await createRecipeDetailed(page, {
      name: "肉じゃが",
      ingredients: [{ name: "玉ねぎ", amount: 1, unit: "個" }],
      steps: "玉ねぎを切る",
      notes: "肉は 30 分漬ける",
    });
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await deleteRecipe(page, id);
    await openMealCalendar(page);

    await openDetailFromMealCalendar(page, "肉じゃが");

    const detail = page.getByRole("main");
    await expect(detail.getByRole("heading", { name: "肉じゃが" })).toBeVisible();
    await expect(detail).toContainText("玉ねぎ");
    await expect(detail).toContainText("玉ねぎを切る");
    await expect(detail).toContainText("肉は 30 分漬ける");
    await expect(detail).toContainText(t.detail.archived);
    await expect(detail.getByRole("button", { name: t.detail.edit })).toHaveCount(0);
    await expect(detail.getByRole("button", { name: t.detail.delete })).toHaveCount(0);
  });
});
