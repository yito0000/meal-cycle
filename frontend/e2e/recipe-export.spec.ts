import { readdirSync } from "node:fs";
import { join } from "node:path";
import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipeDetailed, deleteRecipe, openRecipeDetail, openRecipeList } from "./helpers.js";

const t = ja.pages.recipes;
const ex = t.export;
const d = t.detail;

function exportButton(page: Page) {
  return page.getByRole("button", { name: ex.button, exact: true });
}

// e2e-server は -export-dir をフォルダ選択ダイアログの戻り値として返す (add-recipe-export D1)。
async function exportRecipes(page: Page, exportDir: string, count: number): Promise<string> {
  await openRecipeList(page);
  await exportButton(page).click();
  await expect(page.getByText(ex.done.replace("{{count}}", String(count)))).toBeVisible();
  const created = readdirSync(exportDir);
  expect(created).toHaveLength(1);
  return join(exportDir, created[0]);
}

test.describe("S02 からのレシピのエクスポート", () => {
  test("全レシピがフォルダに書き出される", async ({ page, exportDir }) => {
    await page.goto("/");
    await createRecipeDetailed(page, { name: "肉じゃが" });
    await createRecipeDetailed(page, { name: "味噌汁" });

    const dir = await exportRecipes(page, exportDir, 2);

    expect(readdirSync(dir).sort()).toEqual(["味噌汁.md", "肉じゃが.md"].sort());
    await expect(page.getByText(dir, { exact: true })).toBeVisible();
  });

  test("レシピが 0 件ならエクスポートできない", async ({ page }) => {
    await openRecipeList(page);

    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
    await expect(exportButton(page)).toBeDisabled();
  });
});

test.describe("エクスポートするファイルの形式", () => {
  test("書き出したファイルはインポートで元と同じレシピになる", async ({ page, exportDir }) => {
    await page.goto("/");
    const id = await createRecipeDetailed(page, {
      name: "肉じゃが",
      servings: 3,
      tags: ["和食"],
      ingredients: [
        { name: "じゃがいも", amount: 3, unit: "個" },
        { name: "醤油", amount: 2, unit: "大さじ" },
        { name: "塩", amount: 0, unit: "少々" },
      ],
      steps: "じゃがいもを切る\n\n弱火で煮る",
      notes: "前日に作ると味が染みる",
    });

    const dir = await exportRecipes(page, exportDir, 1);
    await deleteRecipe(page, id);

    const chooser = page.waitForEvent("filechooser");
    await page.getByRole("button", { name: t.import.button, exact: true }).click();
    await page.getByRole("menuitem", { name: t.import.choose_files }).click();
    await (await chooser).setFiles(join(dir, "肉じゃが.md"));
    const dialog = page.getByRole("dialog");
    await dialog.getByRole("button", { name: t.import.submit, exact: true }).click();
    await expect(dialog).toHaveCount(0);

    await openRecipeDetail(page, "肉じゃが");
    const main = page.getByRole("main");
    await expect(main).toContainText(d.servings.replace("{{count}}", "3"));
    await expect(main).toContainText("和食");
    const ingredients = main
      .locator("section")
      .filter({ has: page.getByRole("heading", { name: d.section.ingredients, exact: true }) })
      .getByRole("listitem");
    await expect(ingredients).toHaveCount(3);
    await expect(ingredients.filter({ hasText: "じゃがいも" })).toContainText("3 個");
    await expect(ingredients.filter({ hasText: "醤油" })).toContainText("大さじ 2");
    await expect(ingredients.filter({ hasText: "塩" })).toContainText("少々");
    await expect(main).toContainText("じゃがいもを切る");
    await expect(main).toContainText("弱火で煮る");
    await expect(main).toContainText("前日に作ると味が染みる");
  });
});
