import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  copyLeftover,
  createRecipeWithIngredients,
  dateAfter,
  mealPlanItemRows,
} from "./helpers.js";

// Radix DropdownMenu の Portal 操作は E2E で扱いづらいため、そこに依存するシナリオは
// test.skip で名前だけ残し、次イテレーションで有効化する (design.md D6)。

test.describe("事後編集による作り置きの外し / まとめ", () => {
  test.skip("作り置きから外すと起点日だけが残る", async () => {
    // CellItemMenu → 「作り置きから外す」経路。作り置きの逆演算として、
    // 起点日以外の行が削除される。DropdownMenu Portal 対応時に有効化。
  });

  test.skip("単独作り置きの行に対して外す操作は何もしない", async () => {
    // 1 行しか含まない作り置きに Split を叩いても no-op。同上。
  });

  test.skip("同一レシピの近い日付の作り置きにまとめる", async () => {
    // CellItemMenu → 「他の作り置きにまとめる」→ BatchMergeDialog。同上。
  });

  test.skip("レシピが違う作り置きにはまとめられない", async () => {
    // ListMealBatches が別レシピを候補から除外することを検証。同上。
  });

  test.skip("日付レンジ外の作り置きにはまとめられない", async () => {
    // ListMealBatches が ±4 日を超える候補を除外することを検証。同上。
  });

  test.skip("1 日飛ばした同一レシピの行はまとめられる", async () => {
    // ±4 日以内なら連続でなくてもまとめ可能。同上。
  });

  test.skip("別時間帯の同一レシピ作り置きにもまとめられる", async () => {
    // CellItemMenu → 「他の作り置きにまとめる」→ 別 meal 候補選択。DropdownMenu Portal 対応時に有効化。
  });
});

test.describe("作り置きを広げる (統合ダイアログ)", () => {
  test.skip("起点日 + 2 日ぶん同時間帯が既定でチェックされる", async () => {
    // CellItemMenu → 作り置きを広げる → 統合ダイアログ格子。DropdownMenu Portal 対応時に有効化。
  });

  test.skip("対象日 × 時間帯を自由に組み合わせて広げる", async () => {
    // 統合ダイアログで対象日 × 時間帯を任意にチェックして反映。DropdownMenu Portal 対応時に有効化。
  });

  test.skip("別調理を選ぶと各セルが独立した作り置きになる", async () => {
    // 統合ダイアログのモードセグメント「別調理」経路。同上。
  });

  test.skip("衝突セルは追加 / 上書きの指定に従って反映される", async () => {
    // 統合ダイアログの衝突挙動セグメント「追加」経路。同上。
  });

  test.skip("キャンセルすると割当は変わらない", async () => {
    // 統合ダイアログをキャンセルで閉じて no-op を検証。同上。
  });
});

test.describe("作り置きの枠色による識別", () => {
  test.skip("2 つの作り置きが同時に存在すると別の色になる", async () => {
    // 2 つの作り置きのセットアップに統合ダイアログの実行が要る。上記シナリオと同時に有効化。
  });

  test("作り置きの品行には 🔗 アイコンを表示しない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await copyLeftover(page, dateAfter(0), "dinner", 2, true);
    await page.goto("/");
    await expect(page.getByRole("heading", { name: ja.pages.meal_plan.title })).toBeVisible();
    // 今日が日曜だと 2 日目が翌週になるので、2 週表示で両方を画面に入れる。
    await page.getByTestId("period-weeks-2").click();

    const rows = mealPlanItemRows(page, "カレー");
    await expect(rows).toHaveCount(2);
    for (const row of await rows.all()) {
      await expect(row).toHaveAttribute("data-batch-member-count", "2");
      await expect(row).not.toHaveClass(/border-transparent/);
      await expect(row).not.toContainText("🔗");
    }
  });
});

test.describe("1 行 移動 (ドラッグ&ドロップ)", () => {
  test.skip("別セルにドロップすると 1 行だけ移動する", async () => {
    // Playwright の drag は onDragStart/onDragOver/onDrop を確実には triggering しないため、
    // 手動 (Windows ネイティブビルド) で担保する。proposal: add-mealplan-drag-move
  });

  test.skip("作り置みに属する行はドラッグできない", async () => {
    // batch member 行は draggable=false。個別移動は禁止 (design.md D1)。
    // 手動確認で担保。
  });

  test.skip("別時間帯にドロップしても batch_id は維持される", async () => {
    // 別 meal 間 (dinner → lunch 等) の 1 行移動。同上。
  });

  test.skip("同一セル内でドロップすると並び替えとして扱う", async () => {
    // 同一 (date, meal) で他の entry にドロップ → ReorderMeal 経由で position 差替え。同上。
  });

  test.skip("既存割当のあるセルにドロップすると末尾に追加される", async () => {
    // ドロップ先セルに既存行があるとき、末尾 (position = max+1) に挿入。同上。
  });

  test.skip("ドラッグした品行では品のメニューが開かない", async () => {
    // Playwright の D&D は発火が安定しないため、Windows 実機で確かめる (revise-mealplan-item-row design.md Risks)。
  });
});
