import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipe, detailBreadcrumb, openRecipeList } from "./helpers.js";

const t = ja.pages.recipes;

function nameCells(page: Page) {
  return page.getByRole("row").locator("td:first-child");
}

function chip(page: Page, tag: string) {
  return page
    .getByRole("group", { name: t.tag_filter.aria_label })
    .getByRole("button", { name: tag, exact: true });
}

test.describe("タグでの絞り込み", () => {
  test("タグを選ぶとそのタグを持つレシピだけが残る", async ({ page }) => {
    await createRecipe(page, "肉じゃが", ["和食"]);
    await createRecipe(page, "麻婆豆腐", ["中華"]);
    await createRecipe(page, "冷奴", ["和食", "簡単"]);
    await openRecipeList(page);

    await chip(page, "和食").click();

    await expect(chip(page, "和食")).toHaveAttribute("aria-pressed", "true");
    await expect(nameCells(page)).toHaveCount(2);
    await expect(nameCells(page).filter({ hasText: "肉じゃが" })).toHaveCount(1);
    await expect(nameCells(page).filter({ hasText: "冷奴" })).toHaveCount(1);
  });

  test("複数のタグを選ぶとすべてを持つレシピだけが残る", async ({ page }) => {
    await createRecipe(page, "肉じゃが", ["和食"]);
    await createRecipe(page, "冷奴", ["和食", "簡単"]);
    await openRecipeList(page);

    await chip(page, "和食").click();
    await chip(page, "簡単").click();

    await expect(nameCells(page)).toHaveText(["冷奴"]);
  });

  test("選んだタグをもう一度押すと解除される", async ({ page }) => {
    await createRecipe(page, "肉じゃが", ["和食"]);
    await createRecipe(page, "麻婆豆腐", ["中華"]);
    await openRecipeList(page);
    await chip(page, "和食").click();
    await expect(nameCells(page)).toHaveCount(1);

    await chip(page, "和食").click();

    await expect(nameCells(page)).toHaveCount(2);
    await expect(page.getByRole("button", { name: t.tag_filter.clear })).toHaveCount(0);
  });

  test("タグと名前の検索を同時に使う", async ({ page }) => {
    await createRecipe(page, "肉じゃが", ["和食"]);
    await createRecipe(page, "冷奴", ["和食"]);
    await createRecipe(page, "肉野菜炒め", ["中華"]);
    await openRecipeList(page);

    await chip(page, "和食").click();
    await page.getByRole("searchbox", { name: t.search.placeholder }).fill("肉");

    await expect(nameCells(page)).toHaveText(["肉じゃが"]);
  });

  test("該当するレシピが無いと一致しない旨が出る", async ({ page }) => {
    await createRecipe(page, "冷奴", ["和食", "簡単"]);
    await createRecipe(page, "麻婆豆腐", ["中華"]);
    await openRecipeList(page);

    await chip(page, "簡単").click();
    await chip(page, "中華").click();

    await expect(page.getByText(t.empty.no_match)).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
  });
});

test.describe("S02 一覧から S03 詳細画面へ移る", () => {
  test("S02 に戻ると選んでいたタグが保たれている", async ({ page }) => {
    await createRecipe(page, "肉じゃが", ["和食"]);
    await createRecipe(page, "麻婆豆腐", ["中華"]);
    await openRecipeList(page);
    await chip(page, "和食").click();

    await page.getByRole("cell", { name: "肉じゃが" }).click();
    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.recipes }).click();

    await expect(chip(page, "和食")).toHaveAttribute("aria-pressed", "true");
    await expect(nameCells(page)).toHaveText(["肉じゃが"]);
  });

  test("サイドバーから開き直すとタグの選択は外れる", async ({ page }) => {
    await createRecipe(page, "肉じゃが", ["和食"]);
    await createRecipe(page, "麻婆豆腐", ["中華"]);
    await openRecipeList(page);
    await chip(page, "和食").click();

    await page.getByRole("link", { name: ja.nav.fridge }).click();
    await page.getByRole("heading", { name: ja.pages.fridge.title }).waitFor();
    await page.getByRole("link", { name: ja.nav.recipes }).click();

    await expect(chip(page, "和食")).toHaveAttribute("aria-pressed", "false");
    await expect(chip(page, "中華")).toHaveAttribute("aria-pressed", "false");
    await expect(nameCells(page)).toHaveCount(2);
  });
});
