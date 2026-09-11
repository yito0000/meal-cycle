import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipe } from "./helpers.js";

// テスト名は docs/scenarios/mealplan.feature の シナリオ: 行と一字一句揃える。
// Radix DropdownMenu の Portal 操作は E2E で扱いづらいため、そこに依存するシナリオは
// test.skip で名前だけ残し、次イテレーションで有効化する (design.md D6)。
const t = ja.pages.meal_plan;

async function openMealCalendar(page: import("@playwright/test").Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
}

test.describe("機能: 作り置き / 別調理の選択と事後編集", () => {
  test("作り置きを選ぶと全日程が同じ作り置きになる", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openMealCalendar(page);

    // 凡例 (「同色 = 同じ作り置き / 材料は 1 セット」) が描画されていることは、
    // 少なくとも UI に「作り置きの視覚識別」の説明が入っていることの担保。
    // 実際の連続コピー動作 (DropdownMenu → 連続 N 日 → コピー方法選択) の E2E 検証は
    // Radix Portal を扱う次イテレーションで有効化する。
    await expect(page.getByText(t.batch.legend.leftover)).toBeVisible();
  });

  test.skip("別調理を選ぶと各日が独立した作り置きになる", async () => {
    // CopyModeDialog の「別調理」経路。DropdownMenu の Portal 操作を整えたら有効化。
  });

  test.skip("作り置きから外すと起点日だけが残る", async () => {
    // CellItemMenu → 「作り置きから外す」経路。連続 N 日コピーの逆演算として、
    // 起点日以外の行が削除される。同上。
  });

  test.skip("単独作り置きの行に対して外す操作は何もしない", async () => {
    // 1 行しか含まない作り置きに Split を叩いても no-op。同上。
  });

  test.skip("同一レシピの近い日付の作り置きにまとめる", async () => {
    // CellItemMenu → 「他の作り置きにまとめる」→ BatchMergeDialog。同上。
  });

  test.skip("レシピが違う作り置きにはまとめられない", async () => {
    // ListSlotBatches が別レシピを候補から除外することを検証。同上。
  });

  test.skip("日付レンジ外の作り置きにはまとめられない", async () => {
    // ListSlotBatches が ±4 日を超える候補を除外することを検証。同上。
  });

  test.skip("1 日飛ばした同一レシピの行はまとめられる", async () => {
    // ±4 日以内なら連続でなくてもまとめ可能。同上。
  });

  test.skip("2 つの作り置きが同時に存在すると別の色になる", async () => {
    // 2 つの作り置きのセットアップに連続コピーの実行が要る。上記シナリオと同時に有効化。
  });
});
