import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipeDetailed, openRecipeDetail } from "./helpers.js";

const tr = ja.pages.recipes;

test.describe("レシピの所要時間", () => {
  async function editNikujaga(page: Page) {
    await createRecipeDetailed(page, { name: "肉じゃが" });
    await openRecipeDetail(page, "肉じゃが");
    await page.getByRole("main").getByRole("button", { name: tr.detail.edit }).click();
    await expect(page.locator("#recipe-name")).toHaveValue("肉じゃが");
  }

  test("所要時間を入れて保存すると詳細画面に表示される", async ({ page }) => {
    await editNikujaga(page);

    await page.getByLabel(tr.edit.label.cook_minutes).fill("30");
    await page.getByRole("button", { name: tr.edit.save }).click();
    await openRecipeDetail(page, "肉じゃが");

    await expect(page.getByRole("main")).toContainText(
      tr.detail.cook_minutes.replace("{{count}}", "30"),
    );
  });

  test("範囲外の所要時間では保存できない", async ({ page }) => {
    await editNikujaga(page);

    await page.getByLabel(tr.edit.label.cook_minutes).fill("1000");
    await page.getByRole("button", { name: tr.edit.save }).click();

    await expect(page.getByText(tr.edit.save_error.cook_minutes_out_of_range)).toBeVisible();
    await expect(page.locator("#recipe-name")).toHaveValue("肉じゃが");
  });
});
