import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipe, detailBreadcrumb, openRecipeList } from "./helpers.js";

// 文言は ja.json から引き、日本語をテストにハードコードしない (design.md D9)。
const t = ja.pages.recipes;

// 一覧の名前列。ヘッダ行には td が無いので本文行だけが残る。
function nameCells(page: Page) {
  return page.getByRole("row").locator("td:first-child");
}

test.describe("レシピ一覧の初期表示", () => {
  test("レシピが 0 件の場合は空状態を示す", async ({ page }) => {
    await openRecipeList(page);

    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
    await expect(page.getByRole("button", { name: t.new_button })).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
  });
});

test.describe("レシピ名検索", () => {
  test.use({ seed: true });

  test("検索語がレシピ名に部分一致する", async ({ page }) => {
    await openRecipeList(page);

    await page.getByRole("searchbox", { name: t.search.placeholder }).fill("鮭");

    await expect(nameCells(page)).toHaveText(["鮭のムニエル"]);
  });

  test("検索語がどのレシピにも一致しない", async ({ page }) => {
    await openRecipeList(page);

    await page.getByRole("searchbox", { name: t.search.placeholder }).fill("該当なし");

    await expect(page.getByText(t.empty.no_match)).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
  });
});

test.describe("ソート切替", () => {
  test("初期状態は更新日 降順", async ({ page }) => {
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

  test.describe("サンプルレシピ登録済み", () => {
    test.use({ seed: true });

    test("名前 昇順に切り替える", async ({ page }) => {
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
  });
});

test.describe("S02 一覧から S03 詳細画面へ移る", () => {
  test("行を押すと S03 詳細画面に移る", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openRecipeList(page);

    await page.getByRole("cell", { name: "肉じゃが" }).click();

    await expect(page.getByRole("main").getByRole("heading", { name: "肉じゃが" })).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
    await expect(page.locator("#recipe-name")).toHaveCount(0);
  });

  test("S02 に戻ると検索語と並び順が保たれている", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await createRecipe(page, "じゃがバター");
    await createRecipe(page, "豚汁");
    await openRecipeList(page);

    const search = page.getByRole("searchbox", { name: t.search.placeholder });
    const sort = page.getByRole("combobox", { name: t.sort.label });
    await search.fill("じゃが");
    await sort.click();
    await page.getByRole("option", { name: t.sort.name_asc }).click();

    await page.getByRole("cell", { name: "肉じゃが" }).click();
    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.recipes }).click();

    await expect(search).toHaveValue("じゃが");
    await expect(sort).toHaveText(t.sort.name_asc);
    await expect(nameCells(page)).toHaveCount(2);
  });

  test("サイドバーから開き直すと検索語と並び順は既定に戻る", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openRecipeList(page);

    await page.getByRole("searchbox", { name: t.search.placeholder }).fill("じゃが");
    await page.getByRole("combobox", { name: t.sort.label }).click();
    await page.getByRole("option", { name: t.sort.name_asc }).click();

    await page.getByRole("link", { name: ja.nav.fridge }).click();
    await page.getByRole("heading", { name: ja.pages.fridge.title }).waitFor();
    await page.getByRole("link", { name: ja.nav.recipes }).click();

    await expect(page.getByRole("searchbox", { name: t.search.placeholder })).toHaveValue("");
    await expect(page.getByRole("combobox", { name: t.sort.label })).toHaveText(
      t.sort.updated_desc,
    );
  });
});
