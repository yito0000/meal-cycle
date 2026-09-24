import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipe, openRecipeList } from "./helpers.js";

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

test.describe("S02 一覧からのレシピ編集導線 (暫定)", () => {
  test("行を押下すると当該レシピの編集画面に遷移する", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openRecipeList(page);

    await page.getByRole("cell", { name: "肉じゃが" }).click();
    await expect(page.locator("#recipe-name")).toHaveValue("肉じゃが");

    await page.locator("#recipe-name").fill("肉じゃが（改）");
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("row").locator("td:first-child")).toHaveText(["肉じゃが（改）"]);
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
});

test.describe("S03 から S04 編集画面への遷移", () => {
  test.skip("保存すると S02 に戻り行が強調される", () => {});
});
