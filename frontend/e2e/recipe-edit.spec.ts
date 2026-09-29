import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  createRecipe,
  createRecipeWithIngredients,
  dateAfter,
  detailBreadcrumb,
  openItemMenu,
  openRecipeDetail,
  openRecipeList,
} from "./helpers.js";

const t = ja.pages.recipes;
const e = t.edit;

// S02 から S04 を開く。
async function openNewRecipeForm(page: Page) {
  await openRecipeList(page);
  await page.getByRole("button", { name: t.new_button }).click();
  await expect(page.getByRole("heading", { name: e.title_new })).toBeVisible();
}

// 材料 1 行を足して埋める。名前欄はレシピ名と同じラベルなので行内に絞る。
async function fillFirstIngredient(page: Page, name: string, amount: string, unit: string) {
  await page.getByRole("button", { name: e.add_ingredient }).click();
  const row = page.getByRole("textbox", { name: e.label.name }).last();
  await row.fill(name);
  await page.getByRole("spinbutton", { name: e.label.amount }).fill(amount);
  await page.getByRole("textbox", { name: e.label.unit }).fill(unit);
}

// S03 詳細画面から「編集」で S04 に入る。
async function openEditFromDetail(page: Page, name: string) {
  await openRecipeDetail(page, name);
  await page.getByRole("main").getByRole("button", { name: t.detail.edit }).click();
  await expect(page.locator("#recipe-name")).toHaveValue(name);
}

const servingsText = (n: number) => t.detail.servings.replace("{{count}}", String(n));

// S03 詳細画面の見出し。S04 の見出し「レシピを編集: <名前>」と区別するため完全一致で引く。
function detailHeading(page: Page, name: string) {
  return page.getByRole("main").getByRole("heading", { name, exact: true });
}

test.describe("レシピ一覧の初期表示", () => {
  test("1 レシピに複数タグがある場合はまとめて表示される", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.locator("#recipe-name").fill("豚汁");
    await fillFirstIngredient(page, "大根", "100", "g");
    await page.getByRole("textbox", { name: e.label.tags }).fill("和食");
    await page.getByRole("button", { name: e.add_tag }).click();
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("row").locator("td:nth-child(2)")).toHaveText(["和食"]);
  });
});

test.describe("レシピの新規登録", () => {
  test("材料に自由テキストを入れてレシピを登録する", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.locator("#recipe-name").fill("肉じゃが");
    await page.locator("#recipe-servings").fill("2");
    await fillFirstIngredient(page, "じゃがいも", "2", "個");
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByRole("row").locator("td:first-child")).toHaveText(["肉じゃが"]);
  });

  test("名前が空のままでは保存できない", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByText(e.validation.name_required)).toBeVisible();
    await expect(page.getByRole("heading", { name: e.title_new })).toBeVisible();
  });
});

test.describe("未保存変更のキャンセル確認", () => {
  test("dirty 状態でキャンセルを押すと確認される", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.locator("#recipe-name").fill("肉じゃが");
    await page.getByRole("button", { name: e.cancel }).click();

    await expect(page.getByRole("dialog")).toContainText(e.unsaved_confirm.title);

    await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
  });
  test("編集モードで変更を破棄すると S03 に戻る", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await page.getByRole("button", { name: e.cancel }).click();
    await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();

    await expect(detailHeading(page, "肉じゃが")).toBeVisible();
    await expect(page.getByRole("main")).toContainText(servingsText(2));
  });
});

test.describe("S03 から S04 編集画面への遷移", () => {
  test("編集を押すと既存値がロードされた S04 に遷移する", async ({ page }) => {
    await createRecipe(page, "肉じゃが");

    await openEditFromDetail(page, "肉じゃが");

    await expect(
      page.getByRole("heading", { name: e.title_edit.replace("{{name}}", "肉じゃが") }),
    ).toBeVisible();
    await expect(detailHeading(page, "肉じゃが")).toHaveCount(0);
  });

  test("保存すると S02 に戻り行が強調される", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(detailHeading(page, "肉じゃが")).toHaveCount(0);
    await expect(page.getByRole("row").filter({ hasText: "肉じゃが" })).toHaveClass(/bg-accent/);
  });

  test("キャンセルすると S03 に戻る", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.getByRole("button", { name: e.cancel }).click();

    await expect(detailHeading(page, "肉じゃが")).toBeVisible();
    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.recipes,
      "肉じゃが",
    ]);
  });

  test("保存した内容が S03 に反映される", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await page.getByRole("button", { name: e.save }).click();
    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();

    await page.getByRole("cell", { name: "肉じゃが" }).click();
    await expect(detailHeading(page, "肉じゃが")).toBeVisible();
    await expect(page.getByRole("main")).toContainText(servingsText(3));
  });
});

test.describe("S04 のパンくず", () => {
  test("編集画面のパンくずに開いた元とレシピ名が表示される", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await page.goto("/");
    await openItemMenu(page, "肉じゃが");
    await page.getByRole("menuitem", { name: ja.pages.meal_plan.item_menu.view_detail }).click();
    await expect(detailHeading(page, "肉じゃが")).toBeVisible();

    await page.getByRole("main").getByRole("button", { name: t.detail.edit }).click();

    await expect(page.locator("#recipe-name")).toHaveValue("肉じゃが");
    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.meal_plan,
      "肉じゃが",
      e.breadcrumb_edit,
    ]);
  });

  test("新規登録画面のパンくず", async ({ page }) => {
    await openNewRecipeForm(page);

    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.recipes,
      e.title_new,
    ]);
  });

  test("パンくずで移るときも未保存の変更を確かめる", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.recipes }).click();

    await expect(page.getByRole("dialog")).toContainText(e.unsaved_confirm.title);
    await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();

    await expect(page.getByRole("heading", { name: t.title, exact: true })).toBeVisible();
    await page.getByRole("cell", { name: "肉じゃが" }).click();
    await expect(page.getByRole("main")).toContainText(servingsText(2));
  });
});
