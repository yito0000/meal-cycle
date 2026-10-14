import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  copyLeftover,
  createRecipeWithIngredients,
  dateAfter,
  mealPlanItemRows,
  openItemMenu,
  openSpreadDialog,
  spreadCell,
  spreadEntry,
} from "./helpers.js";

const t = ja.pages.meal_plan;

// 曜日を固定するため、来週の月曜を起点にする。統合ダイアログの 4 行 (月〜木) も来週に収まる。
function nextWeekDay(offsetFromMonday: number): string {
  const now = new Date();
  const toMonday = (now.getDay() + 6) % 7;
  const d = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate() - toMonday + 7 + offsetFromMonday,
  );
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

const MON = nextWeekDay(0);
const TUE = nextWeekDay(1);
const WED = nextWeekDay(2);
const THU = nextWeekDay(3);

async function openNextWeek(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
  await page.getByTestId("period-next").click();
}

type Row = {
  date: string;
  meal: string;
  recipeName: string;
  batchId: string;
  batchMemberCount: number;
};

async function listNextWeek(page: Page): Promise<Row[]> {
  const res = await page.request.post("/api/MealPlanService/ListPeriod", { data: [MON, 1] });
  if (!res.ok()) throw new Error(`list period failed: ${res.status()} ${await res.text()}`);
  return (await res.json()) as Row[];
}

function rowsIn(rows: Row[], date: string, meal: string): Row[] {
  return rows.filter((r) => r.date === date && r.meal === meal);
}

// 統合ダイアログは閉じたあとに非同期で反映するので、行数が期待どおりになるまで待ってから中身を見る。
async function waitForRowCount(page: Page, count: number): Promise<Row[]> {
  await expect.poll(async () => (await listNextWeek(page)).length).toBe(count);
  return listNextWeek(page);
}

async function setCheck(page: Page, date: string, meal: string, checked: boolean) {
  await spreadCell(page, date, meal).getByRole("checkbox").setChecked(checked);
}

// spreadLeftover は MON の夕食の 1 品目を、days に挙げた日の夕食へ作り置きとして広げる。
async function spreadLeftover(page: Page, days: string[]) {
  await spreadEntry(
    page,
    { date: MON, meal: "dinner", position: 0 },
    days.map((date) => ({ date, meal: "dinner" })),
    true,
  );
}

function dissolveDialog(page: Page) {
  return page.getByRole("dialog", { name: t.dissolve.title });
}

// 月・火・水曜・夕に同じ作り置きの「肉じゃが」を並べ、火曜の行で「作り置きをやめる」を選ぶ。
async function chooseDissolveOnTuesday(page: Page) {
  const id = await createRecipeWithIngredients(page, "肉じゃが", []);
  await assignRecipe(page, MON, "dinner", id);
  await spreadLeftover(page, [TUE, WED]);
  await openNextWeek(page);
  await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(3);

  await openItemMenu(page, "肉じゃが", 1);
  await page.getByRole("menuitem", { name: t.item_menu.split_batch }).click();
  await expect(dissolveDialog(page)).toBeVisible();
}

// Radix DropdownMenu の Portal 操作に依存するまとめるシナリオは、test.skip で名前だけ残している (design.md D6)。
test.describe("作り置きをやめる操作とまとめる操作", () => {
  test("作り置きをやめると最初の日だけが残る", async ({ page }) => {
    await chooseDissolveOnTuesday(page);
    const before = rowsIn(await listNextWeek(page), MON, "dinner")[0];

    await dissolveDialog(page).getByRole("button", { name: t.dissolve.confirm }).click();

    await expect(dissolveDialog(page)).toHaveCount(0);
    const rows = await waitForRowCount(page, 1);
    expect(rows[0].date).toBe(MON);
    expect(rows[0].batchMemberCount).toBe(1);
    expect(rows[0].batchId).not.toBe(before.batchId);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(1);
  });

  test("作り置きをやめる前に削除される行の数を確認する", async ({ page }) => {
    await chooseDissolveOnTuesday(page);

    await expect(dissolveDialog(page)).toContainText(
      t.dissolve.description.replace("{{count}}", "2"),
    );
    expect(await listNextWeek(page)).toHaveLength(3);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(3);
  });

  test("確認でキャンセルすると作り置きは変わらない", async ({ page }) => {
    await chooseDissolveOnTuesday(page);
    const before = await listNextWeek(page);

    await dissolveDialog(page).getByRole("button", { name: t.dissolve.cancel }).click();

    await expect(dissolveDialog(page)).toHaveCount(0);
    const after = await listNextWeek(page);
    expect(after).toHaveLength(3);
    for (const r of after) {
      expect(r.batchId).toBe(before[0].batchId);
      expect(r.batchMemberCount).toBe(3);
    }
  });

  test("単独の行には「作り置きをやめる」項目が出ない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    await openItemMenu(page, "肉じゃが");

    await expect(page.getByRole("menuitem", { name: t.item_menu.spread })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: t.item_menu.split_batch })).toHaveCount(0);
  });

  test.skip("同一レシピの近い日付の作り置きにまとめる", async () => {
    // CellItemMenu → 「他の作り置きにまとめる」→ BatchMergeDialog。DropdownMenu Portal 対応時に有効化。
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
    // CellItemMenu → 「他の作り置きにまとめる」→ 別 meal 候補選択。同上。
  });
});

test.describe("作り置きを広げる操作", () => {
  test("起点日の翌日の同じ時間帯が既定でチェックされる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");

    await expect(spreadCell(page, TUE, "dinner").getByRole("checkbox")).toBeChecked();
    await expect(dialog.getByRole("checkbox", { checked: true })).toHaveCount(1);
    const source = spreadCell(page, MON, "dinner");
    await expect(source.getByRole("checkbox")).toHaveCount(0);
    await expect(source.getByRole("img", { name: t.spread.source_cell_label })).toBeVisible();
  });

  test("対象日 × 時間帯を自由に組み合わせて広げる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");
    await setCheck(page, WED, "lunch", true);
    await setCheck(page, THU, "dinner", true);
    await dialog.getByRole("button", { name: t.spread.confirm }).click();

    const rows = await waitForRowCount(page, 4);
    const sourceBatch = rowsIn(rows, MON, "dinner")[0].batchId;
    for (const [date, meal] of [
      [TUE, "dinner"],
      [WED, "lunch"],
      [THU, "dinner"],
    ]) {
      const cell = rowsIn(rows, date, meal);
      expect(cell.map((r) => r.recipeName)).toEqual(["肉じゃが"]);
      expect(cell[0].batchId).toBe(sourceBatch);
    }
    const shown = mealPlanItemRows(page, "肉じゃが");
    await expect(shown).toHaveCount(3);
    for (const row of await shown.all()) {
      await expect(row).toHaveAttribute("data-batch-member-count", "4");
      await expect(row).not.toHaveClass(/border-transparent/);
    }
  });

  test("チェックしていない日には追加されない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");
    await setCheck(page, TUE, "dinner", false);
    await setCheck(page, THU, "dinner", true);
    await dialog.getByRole("button", { name: t.spread.confirm }).click();

    const rows = await waitForRowCount(page, 2);
    expect(rowsIn(rows, THU, "dinner").map((r) => r.recipeName)).toEqual(["肉じゃが"]);
    expect(rowsIn(rows, TUE, "dinner")).toHaveLength(0);
    expect(rowsIn(rows, WED, "dinner")).toHaveLength(0);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(2);
  });

  test("別調理を選ぶと各セルが独立した作り置きになる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");
    await setCheck(page, WED, "dinner", true);
    await dialog.getByText(t.spread.distinct_label, { exact: true }).click();
    await dialog.getByRole("button", { name: t.spread.confirm }).click();

    const rows = await waitForRowCount(page, 3);
    expect(new Set(rows.map((r) => r.batchId)).size).toBe(3);
    const shown = mealPlanItemRows(page, "肉じゃが");
    await expect(shown).toHaveCount(3);
    for (const row of await shown.all()) {
      await expect(row).toHaveAttribute("data-batch-member-count", "1");
      await expect(row).toHaveClass(/border-transparent/);
    }
  });

  test("別調理は別の時間帯のセルにも効く", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");
    await setCheck(page, WED, "lunch", true);
    await dialog.getByText(t.spread.distinct_label, { exact: true }).click();
    await dialog.getByRole("button", { name: t.spread.confirm }).click();

    const rows = await waitForRowCount(page, 3);
    expect(rowsIn(rows, TUE, "dinner").map((r) => r.recipeName)).toEqual(["肉じゃが"]);
    expect(rowsIn(rows, WED, "lunch").map((r) => r.recipeName)).toEqual(["肉じゃが"]);
    for (const r of rows) {
      expect(r.batchMemberCount).toBe(1);
    }
    for (const row of await mealPlanItemRows(page, "肉じゃが").all()) {
      await expect(row).toHaveClass(/border-transparent/);
    }
  });

  test("衝突セルは追加 / 上書きの指定に従って反映される", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, MON, "dinner", nikujaga);
    await assignRecipe(page, TUE, "lunch", miso);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");
    await setCheck(page, TUE, "dinner", false);
    await setCheck(page, TUE, "lunch", true);
    await dialog.getByText(t.spread.conflict_append_label, { exact: true }).click();
    await dialog.getByRole("button", { name: t.spread.confirm }).click();

    const rows = await waitForRowCount(page, 3);
    const lunch = rowsIn(rows, TUE, "lunch");
    expect(lunch.map((r) => r.recipeName)).toEqual(["みそ汁", "肉じゃが"]);
    expect(lunch[1].batchId).toBe(rowsIn(rows, MON, "dinner")[0].batchId);
  });

  test("上書きを選ぶと対象セルの既存の品が置き換わる", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, MON, "dinner", nikujaga);
    await assignRecipe(page, TUE, "dinner", miso);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");
    await expect(spreadCell(page, TUE, "dinner").getByRole("checkbox")).toBeChecked();
    await dialog.getByText(t.spread.conflict_overwrite_label, { exact: true }).click();
    await dialog.getByRole("button", { name: t.spread.confirm }).click();

    await expect(mealPlanItemRows(page, "みそ汁")).toHaveCount(0);
    const rows = await waitForRowCount(page, 2);
    expect(rowsIn(rows, TUE, "dinner").map((r) => r.recipeName)).toEqual(["肉じゃが"]);
  });

  test("同じ作り置きがすでに入っているセルは選べない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await copyLeftover(page, MON, "dinner", 2, true);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが", 0);

    const tue = spreadCell(page, TUE, "dinner");
    await expect(tue.getByRole("checkbox")).toHaveCount(0);
    await expect(tue.getByRole("img", { name: t.spread.same_batch_cell_label })).toBeVisible();
    await expect(dialog.getByRole("checkbox", { checked: true })).toHaveCount(0);
  });

  test("起点日 + 未来 3 日ぶんの 4 行だけが表示される", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");

    await expect(dialog.locator("tbody tr")).toHaveCount(4);
    for (const date of [MON, TUE, WED, THU]) {
      await expect(spreadCell(page, date, "dinner")).toHaveCount(1);
    }
    await expect(dialog.locator(`td[data-date="${nextWeekDay(-1)}"]`)).toHaveCount(0);
    await expect(dialog.locator(`td[data-date="${nextWeekDay(4)}"]`)).toHaveCount(0);
    await expect(dialog.getByRole("button", { name: /さらに/ })).toHaveCount(0);
  });

  test("キャンセルすると割当は変わらない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");
    await setCheck(page, WED, "lunch", true);
    await setCheck(page, THU, "dinner", true);
    await dialog.getByRole("button", { name: t.spread.cancel }).click();

    await expect(dialog).toHaveCount(0);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(1);
    expect(await listNextWeek(page)).toHaveLength(1);
  });
});

test.describe("まとめ候補の一意性", () => {
  test("同じ日の昼食と夕食にまたがる作り置きは候補に 1 行だけ出る", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "lunch", id);
    await spreadEntry(
      page,
      { date: MON, meal: "lunch", position: 0 },
      [{ date: MON, meal: "dinner" }],
      true,
    );
    await assignRecipe(page, TUE, "dinner", id);
    await openNextWeek(page);

    // 夕食だけの表示なので、品行は月曜・夕食 (0 番目) と火曜・夕食 (1 番目) になる。
    await openItemMenu(page, "肉じゃが", 1);
    await page.getByRole("menuitem", { name: t.item_menu.merge_batch }).click();

    const dialog = page.getByRole("dialog", { name: t.batch.merge_dialog.title });
    const candidates = dialog.getByRole("listitem");
    await expect(candidates).toHaveCount(1);
    await expect(candidates).toContainText(MON);
    await expect(candidates).toContainText(t.meal.lunch);
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
