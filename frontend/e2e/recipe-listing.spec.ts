import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipe, openRecipeList } from "./helpers.js";

// テスト名は docs/scenarios/recipe.feature の シナリオ: 行と一字一句揃える。
// 文言は ja.json から引き、日本語をテストにハードコードしない (design.md D9)。
const t = ja.pages.recipes;

// 一覧の名前列。ヘッダ行には td が無いので本文行だけが残る。
function nameCells(page: Page) {
  return page.getByRole("row").locator("td:first-child");
}

test.describe("機能: レシピの一覧と詳細閲覧", () => {
  test.describe("レシピ未登録", () => {
    test("登録済みレシピが 1 件もない状態で一覧を開く", async ({ page }) => {
      await openRecipeList(page);

      await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
      await expect(page.getByRole("button", { name: t.new_button })).toBeVisible();
      await expect(page.getByRole("table")).toHaveCount(0);
    });

    test("登録済みレシピを更新日 desc（デフォルト）で一覧表示する", async ({ page }) => {
      // updated_at は RFC3339 の秒精度なので、間隔を空けないと 3 件が同着になる。
      await createRecipe(page, "肉じゃが");
      await page.waitForTimeout(1100);
      await createRecipe(page, "豚汁");
      await page.waitForTimeout(1100);
      await createRecipe(page, "鮭のムニエル");

      await openRecipeList(page);

      await expect(nameCells(page)).toHaveText(["鮭のムニエル", "豚汁", "肉じゃが"]);
      for (const column of [t.column.name, t.column.tags, t.column.updated_at]) {
        await expect(page.getByRole("columnheader", { name: column })).toBeVisible();
      }
    });
  });

  test.describe("サンプルレシピ登録済み", () => {
    test.use({ seed: true });

    test("名前で部分一致検索する", async ({ page }) => {
      await openRecipeList(page);

      await page.getByRole("searchbox", { name: t.search.placeholder }).fill("鮭");

      await expect(nameCells(page)).toHaveText(["鮭のムニエル"]);
    });

    test("名前 昇順に並べ替える", async ({ page }) => {
      await openRecipeList(page);
      const before = await nameCells(page).allInnerTexts();

      await page.getByRole("combobox", { name: t.sort.label }).click();
      await page.getByRole("option", { name: t.sort.name_asc }).click();

      // 期待順はブラウザの localeCompare で作る。アプリ側 (compareRecipes) と
      // 同じ照合順を使い、テストで照合規則を書き直さないため。
      const expected = await page.evaluate(
        (names) => [...names].sort((a, b) => a.localeCompare(b)),
        before,
      );
      await expect(nameCells(page)).toHaveText(expected);
    });

    test("検索に一致するレシピが無い場合は空状態が表示される", async ({ page }) => {
      await openRecipeList(page);

      await page.getByRole("searchbox", { name: t.search.placeholder }).fill("該当なし");

      await expect(page.getByText(t.empty.no_match)).toBeVisible();
      await expect(page.getByRole("table")).toHaveCount(0);
    });
  });

  // S03 詳細ドロワーは add-recipe-detail-drawer で実装する。
  test.skip("レシピ一覧から詳細ドロワーを開く", () => {});
  test.skip("詳細ドロワーを背景クリックで閉じる", () => {});
});
