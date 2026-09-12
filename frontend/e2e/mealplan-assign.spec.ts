import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipe } from "./helpers.js";

// テスト名は docs/scenarios/mealplan.feature の シナリオ: 行と一字一句揃える。
// 現状は「空セルに 1 品目を割り当てる」のみ実装済み。残りは実装追随後に skip を外す。
const t = ja.pages.meal_plan;

async function openMealCalendar(page: import("@playwright/test").Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  // 初期ルートが献立カレンダーなので、遷移操作は不要。ヘッダで到達確認する。
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
}

test.describe("機能: 献立へのレシピ割当（1 時間帯に 0..N 品）", () => {
  test("空セルに 1 品目を割り当てる", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openMealCalendar(page);

    // 初期は夕食のみ表示 = 7 セルすべて空セル。最初の `＋` を押す。
    const addFirstButtons = page.getByRole("button", { name: t.cell_actions.add_first });
    await expect(addFirstButtons.first()).toBeVisible();
    await addFirstButtons.first().click();

    // レシピ選択ダイアログが開き、S02 本画面には遷移しない。
    await expect(page.getByRole("dialog")).toContainText(t.picker.title);

    // 検索で絞り込み、選択する。
    await page.getByPlaceholder(t.picker.search_placeholder).fill("肉");
    await page.getByRole("button", { name: "肉じゃが" }).click();

    // dialog が閉じ、セルに 1 品目が現れる。
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByText("肉じゃが").first()).toBeVisible();
  });

  test.skip("既に 1 品ある時間帯にもう 1 品追加する", async () => {
    // 実装は Group 7 で通した右上 `＋` 経路。E2E は次イテレーションで有効化する。
  });

  test.skip("品数が 3 を超えたら「他 N 件」で省略される", async () => {
    // MealPlanCell は overflow 表示済み。E2E は Group 10 の次イテレーションで有効化する。
  });

  test.skip("1 品だけ入れ替える", async () => {
    // CellItemMenu → replace は実装済み。Radix DropdownMenu の Portal を選ぶ手順を整えたら有効化する。
  });

  test.skip("1 品だけ削除する", async () => {
    // 同上。
  });

  test.skip("時間帯全体をクリアする", async () => {
    // MealMenu → clear は実装済み。
  });

  test.skip("割当済みの 1 品から S03 詳細を開き、S01 に戻る", async () => {
    // S03 レシピ詳細ドロワー自体が別 proposal (add-recipe-detail-drawer) で未実装。
  });
});

test.describe("機能: 作り置きコピー（同一レシピの連続割当）", () => {
  test.skip("1 品を連続 3 日に作り置きコピー", async () => {
    // CellItemMenu の「連続 N 日」→ CopyLeftover。実装済みだが E2E は次イテレーションで。
  });

  test.skip("時間帯全体（複数品）を連続 N 日コピー", async () => {
    // MealMenu の「時間帯全体を連続 N 日」→ CopyLeftover。
  });

  test.skip("コピー先に既存の割当があると追加 or 上書きを確認する", async () => {
    // CopyConflictDialog は実装済み。
  });
});
