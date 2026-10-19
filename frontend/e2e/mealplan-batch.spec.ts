import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  copyLeftover,
  createRecipeWithIngredients,
  dateAfter,
  deleteRecipe,
  mealPlanCell,
  mealPlanItemRows,
  openItemMenu,
  openSpreadDialog,
  setCooked,
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
const FRI = nextWeekDay(4);
const SAT = nextWeekDay(5);
const SUN = nextWeekDay(6);
const MON_AFTER = nextWeekDay(7);

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
  cooked: boolean;
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

// 品のメニューの「他の作り置きにまとめる」を開き、まとめ先を選ぶダイアログを返す。
async function openMergeDialog(page: Page, recipeName: string, nth = 0) {
  await openItemMenu(page, recipeName, nth);
  await page.getByRole("menuitem", { name: t.item_menu.merge_batch }).click();
  const dialog = page.getByRole("dialog", { name: t.batch.merge_dialog.title });
  await dialog.waitFor();
  return dialog;
}

// rows の中の date / meal の行の batchId。
function batchOf(rows: Row[], date: string, meal: string): string {
  const row = rowsIn(rows, date, meal)[0];
  if (!row) throw new Error(`no row at ${date} ${meal}`);
  return row.batchId;
}

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

  test("同一レシピの近い日付の作り置きにまとめる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await assignRecipe(page, TUE, "dinner", id);
    await openNextWeek(page);

    const dialog = await openMergeDialog(page, "肉じゃが", 1);
    await dialog.getByRole("button").filter({ hasText: MON }).click();

    await expect(dialog).toHaveCount(0);
    await expect
      .poll(async () => {
        const rows = await listNextWeek(page);
        return batchOf(rows, TUE, "dinner") === batchOf(rows, MON, "dinner");
      })
      .toBe(true);
  });

  test("レシピが違う作り置きにはまとめられない", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, MON, "dinner", miso);
    await spreadLeftover(page, [WED]);
    await assignRecipe(page, TUE, "dinner", nikujaga);
    await openNextWeek(page);

    const dialog = await openMergeDialog(page, "肉じゃが");

    await expect(dialog.getByRole("listitem")).toHaveCount(0);
    await expect(dialog).toContainText(t.batch.merge_dialog.empty);
  });

  test("日付レンジ外の作り置きにはまとめられない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    // 金曜の 6 日前と 5 日前 (前の週の土曜と日曜) の作り置き。
    const saturdayBefore = nextWeekDay(-2);
    await assignRecipe(page, saturdayBefore, "dinner", id);
    await spreadEntry(
      page,
      { date: saturdayBefore, meal: "dinner", position: 0 },
      [{ date: nextWeekDay(-1), meal: "dinner" }],
      true,
    );
    await assignRecipe(page, FRI, "dinner", id);
    await openNextWeek(page);

    const dialog = await openMergeDialog(page, "肉じゃが");

    await expect(dialog.getByRole("listitem")).toHaveCount(0);
    await expect(dialog).toContainText(t.batch.merge_dialog.empty);
  });

  test("1 日飛ばした同一レシピの行はまとめられる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await spreadLeftover(page, [TUE]);
    await assignRecipe(page, FRI, "dinner", id);
    await openNextWeek(page);

    const dialog = await openMergeDialog(page, "肉じゃが", 2);
    await dialog.getByRole("button").filter({ hasText: MON }).click();

    await expect(dialog).toHaveCount(0);
    await expect
      .poll(async () => new Set((await listNextWeek(page)).map((r) => r.batchId)).size)
      .toBe(1);
    expect(await listNextWeek(page)).toHaveLength(3);
  });

  test("別時間帯の同一レシピ作り置きにもまとめられる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await assignRecipe(page, TUE, "lunch", id);
    await openNextWeek(page);
    await page.getByRole("checkbox", { name: t.meal.lunch }).check();

    await mealPlanCell(page, TUE, "lunch")
      .getByRole("button", { name: "肉じゃが", exact: true })
      .click();
    await page.getByRole("menuitem", { name: t.item_menu.merge_batch }).click();
    const dialog = page.getByRole("dialog", { name: t.batch.merge_dialog.title });
    await dialog.getByRole("button").filter({ hasText: t.meal.dinner }).click();

    await expect(dialog).toHaveCount(0);
    await expect
      .poll(async () => {
        const rows = await listNextWeek(page);
        return batchOf(rows, TUE, "lunch") === batchOf(rows, MON, "dinner");
      })
      .toBe(true);
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

  test("起点より前の時間帯のセルは選べない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    await openSpreadDialog(page, "肉じゃが");

    for (const meal of ["breakfast", "lunch"]) {
      const earlier = spreadCell(page, MON, meal);
      await expect(earlier.getByRole("checkbox")).toHaveCount(0);
      await expect(
        earlier.getByRole("img", { name: t.spread.earlier_meal_cell_label }),
      ).toBeVisible();
    }
    await expect(spreadCell(page, TUE, "breakfast").getByRole("checkbox")).toBeEnabled();
  });

  test("表示期間の外のセルでも既存割当が示される", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, SAT, "dinner", nikujaga);
    await assignRecipe(page, MON_AFTER, "dinner", miso);
    await openNextWeek(page);
    await expect(mealPlanItemRows(page, "みそ汁")).toHaveCount(0);

    await openSpreadDialog(page, "肉じゃが");

    await expect(spreadCell(page, MON_AFTER, "dinner")).toHaveClass(/bg-amber-50/);
    await expect(spreadCell(page, SUN, "dinner")).not.toHaveClass(/bg-amber-50/);
  });

  test("起点の日付は格子の行見出しと同じ表記で示される", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    const dialog = await openSpreadDialog(page, "肉じゃが");

    const heading =
      (await dialog.locator("tbody tr").first().locator("td").first().textContent()) ?? "";
    expect(heading).not.toBe("");
    const hint = dialog.getByText(t.spread.origin_hint.split(" ")[0], { exact: false });
    await expect(hint).toContainText(heading);
    await expect(hint).not.toContainText(MON);
  });
});

// 広げたあとの通知は sonner の toast で main の外に描かれるので、page 起点で探す。
function spreadUndoButton(page: Page) {
  return page.getByRole("button", { name: t.toast.undo, exact: true });
}

// 月曜・夕食の肉じゃがの統合ダイアログで、既定の火曜・夕食に加えて extraDinners の夕食をチェックして反映する。
async function spreadFromMonday(page: Page, extraDinners: string[], overwrite: boolean) {
  const dialog = await openSpreadDialog(page, "肉じゃが");
  await expect(spreadCell(page, TUE, "dinner").getByRole("checkbox")).toBeChecked();
  for (const date of extraDinners) {
    await setCheck(page, date, "dinner", true);
  }
  if (overwrite) {
    await dialog.getByText(t.spread.conflict_overwrite_label, { exact: true }).click();
  }
  await dialog.getByRole("button", { name: t.spread.confirm }).click();
  await expect(dialog).toHaveCount(0);
}

test.describe("作り置きを広げる操作の取り消し", () => {
  test("広げた品を元に戻す", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);
    await spreadFromMonday(page, [WED], false);
    await waitForRowCount(page, 3);

    await spreadUndoButton(page).click();

    const rows = await waitForRowCount(page, 1);
    expect(rows[0].date).toBe(MON);
    expect(rows[0].batchMemberCount).toBe(1);
    const shown = mealPlanItemRows(page, "肉じゃが");
    await expect(shown).toHaveCount(1);
    await expect(shown).toHaveClass(/border-transparent/);
  });

  test("上書きで消えた品を元に戻す", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    const salad = await createRecipeWithIngredients(page, "サラダ", []);
    await assignRecipe(page, MON, "dinner", nikujaga);
    await assignRecipe(page, TUE, "dinner", miso);
    await assignRecipe(page, TUE, "dinner", salad);
    await openNextWeek(page);
    await spreadFromMonday(page, [], true);
    await expect(mealPlanItemRows(page, "みそ汁")).toHaveCount(0);

    await spreadUndoButton(page).click();

    await expect(mealPlanItemRows(page, "みそ汁")).toHaveCount(1);
    const rows = await waitForRowCount(page, 3);
    expect(rowsIn(rows, TUE, "dinner").map((r) => r.recipeName)).toEqual(["みそ汁", "サラダ"]);
    await expect(mealPlanCell(page, TUE, "dinner").getByRole("listitem")).toHaveText([
      "みそ汁",
      "サラダ",
    ]);
  });

  test("上書きで消えた品の作った印が戻る", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, MON, "dinner", nikujaga);
    await assignRecipe(page, TUE, "dinner", miso);
    await setCooked(page, TUE, "dinner", 0, true);
    await openNextWeek(page);
    await spreadFromMonday(page, [], true);
    await expect(mealPlanItemRows(page, "みそ汁")).toHaveCount(0);

    await spreadUndoButton(page).click();

    await expect(mealPlanItemRows(page, "みそ汁")).toHaveCount(1);
    const rows = await waitForRowCount(page, 2);
    expect(rowsIn(rows, TUE, "dinner")[0].cooked).toBe(true);
    await expect(
      mealPlanItemRows(page, "みそ汁").getByRole("checkbox", {
        name: t.cell.cooked.replace("{{name}}", "みそ汁"),
      }),
    ).toBeChecked();
  });

  test("広げると通知にセルの数と置き換えた品の数が示される", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, MON, "dinner", nikujaga);
    await assignRecipe(page, TUE, "dinner", miso);
    await openNextWeek(page);

    await spreadFromMonday(page, [WED], true);

    const message = t.toast.spread_overwritten
      .replace("{{cells}}", "2")
      .replace("{{replaced}}", "1");
    await expect(page.locator("[data-sonner-toast]").getByText(message)).toBeVisible();
    await expect(spreadUndoButton(page)).toBeVisible();
  });
});

test.describe("削除済みレシピの品は広げられない", () => {
  test("削除済みレシピの品を広げようとすると通知で知らされる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await deleteRecipe(page, id);
    await openNextWeek(page);

    await spreadFromMonday(page, [], false);

    await expect(
      page.locator("[data-sonner-toast]").getByText(t.errors.archived_recipe),
    ).toBeVisible();
    expect(await listNextWeek(page)).toHaveLength(1);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(1);
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

// days に来週の外の日を含めると、来週の 1 週表示には作り置きの端の行だけが出る。
async function openBatchLeavingNextWeek(page: Page, from: string, days: string[]) {
  const id = await createRecipeWithIngredients(page, "肉じゃが", []);
  await assignRecipe(page, from, "dinner", id);
  await spreadEntry(
    page,
    { date: from, meal: "dinner", position: 0 },
    days.map((date) => ({ date, meal: "dinner" })),
    true,
  );
  await openNextWeek(page);
}

test.describe("表示期間をまたぐ作り置き", () => {
  test("期間の外に続く作り置きの行にも枠色が付く", async ({ page }) => {
    await openBatchLeavingNextWeek(page, SUN, [MON_AFTER]);

    const row = mealPlanItemRows(page, "肉じゃが");
    await expect(row).toHaveCount(1);
    await expect(row).toHaveAttribute("data-batch-member-count", "2");
    await expect(row).not.toHaveClass(/border-transparent/);
  });

  test("期間の外に続く作り置きの行はドラッグできない", async ({ page }) => {
    await openBatchLeavingNextWeek(page, SUN, [MON_AFTER]);

    const row = mealPlanItemRows(page, "肉じゃが");
    await expect(row).toHaveCount(1);
    await expect(row).toHaveAttribute("draggable", "false");
  });

  test("期間の外に続く作り置きの行にも作り置きをやめる項目が出る", async ({ page }) => {
    await openBatchLeavingNextWeek(page, SUN, [MON_AFTER]);

    await openItemMenu(page, "肉じゃが");

    await expect(page.getByRole("menuitem", { name: t.item_menu.split_batch })).toBeVisible();
  });

  test("作り置きをやめる確認には期間の外で削除される行も数える", async ({ page }) => {
    await openBatchLeavingNextWeek(page, SAT, [SUN, MON_AFTER]);
    await expect(mealPlanItemRows(page, "肉じゃが")).toHaveCount(2);

    await openItemMenu(page, "肉じゃが", 1);
    await page.getByRole("menuitem", { name: t.item_menu.split_batch }).click();

    await expect(dissolveDialog(page)).toContainText(
      t.dissolve.description.replace("{{count}}", "2"),
    );
  });
});

test.describe("作り置きをやめたときに残る行", () => {
  test("同じ日の昼食と夕食にまたがる作り置きをやめると昼食の行が残る", async ({ page }) => {
    const tonjiru = await createRecipeWithIngredients(page, "豚汁", []);
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "lunch", tonjiru);
    await assignRecipe(page, MON, "lunch", nikujaga);
    await spreadEntry(
      page,
      { date: MON, meal: "lunch", position: 1 },
      [{ date: MON, meal: "dinner" }],
      true,
    );
    await openNextWeek(page);
    await page.getByRole("checkbox", { name: t.meal.lunch }).check();
    const lunch = mealPlanCell(page, MON, "lunch");
    await expect(lunch.getByRole("listitem")).toHaveCount(2);

    await lunch
      .getByRole("listitem")
      .filter({ has: page.getByText("肉じゃが", { exact: true }) })
      .getByRole("button", { name: "肉じゃが", exact: true })
      .click();
    await page.getByRole("menuitem", { name: t.item_menu.split_batch }).click();
    await dissolveDialog(page).getByRole("button", { name: t.dissolve.confirm }).click();

    await expect(dissolveDialog(page)).toHaveCount(0);
    const rows = await waitForRowCount(page, 2);
    expect(rowsIn(rows, MON, "lunch").map((r) => r.recipeName)).toEqual(["豚汁", "肉じゃが"]);
    expect(rowsIn(rows, MON, "dinner")).toHaveLength(0);
    await expect(mealPlanCell(page, MON, "dinner").getByRole("listitem")).toHaveCount(0);
  });
});

test.describe("作り置きの枠色による識別", () => {
  test.skip("2 つの作り置きが同時に存在すると別の色になる", async () => {
    // 枠色は作り置き ID のハッシュで 3 色から選ぶので、2 つの作り置きが同じ色になることがある。
    // spec と実装が食い違っており (docs/reviews R8)、直すまでは有効にしても通らない回がある。
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

// 品行の HTML5 D&D は、Playwright の dragTo (Chromium のネイティブのドラッグ) で発火する。
test.describe("1 行 移動 (ドラッグ&ドロップ)", () => {
  test("別セルにドロップすると 1 行だけ移動する", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    await mealPlanItemRows(page, "肉じゃが").dragTo(mealPlanCell(page, TUE, "dinner"));

    await expect(mealPlanCell(page, MON, "dinner").getByRole("listitem")).toHaveCount(0);
    await expect(mealPlanCell(page, TUE, "dinner").getByRole("listitem")).toHaveText(["肉じゃが"]);
    const rows = await waitForRowCount(page, 1);
    expect(rowsIn(rows, TUE, "dinner")).toHaveLength(1);
  });

  test("作り置きに属する行はドラッグできない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await spreadLeftover(page, [TUE, WED]);
    await openNextWeek(page);
    const before = await listNextWeek(page);
    const wed = mealPlanCell(page, WED, "dinner").getByRole("listitem");
    await expect(wed).toHaveAttribute("draggable", "false");

    const from = await wed.boundingBox();
    const to = await mealPlanCell(page, THU, "dinner").boundingBox();
    if (!from || !to) throw new Error("no bounding box");
    await page.mouse.move(from.x + 4, from.y + from.height / 2);
    await page.mouse.down();
    await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 5 });
    await expect(page.getByRole("group", { name: t.dnd.cell_drop_label })).toHaveCount(0);
    await page.mouse.up();

    expect(await listNextWeek(page)).toEqual(before);
    await expect(mealPlanCell(page, THU, "dinner").getByRole("listitem")).toHaveCount(0);
  });

  test("別時間帯にドロップしても batch_id は維持される", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);
    await page.getByRole("checkbox", { name: t.meal.lunch }).check();
    const before = rowsIn(await listNextWeek(page), MON, "dinner")[0];

    await mealPlanItemRows(page, "肉じゃが").dragTo(mealPlanCell(page, MON, "lunch"));

    await expect(mealPlanCell(page, MON, "dinner").getByRole("listitem")).toHaveCount(0);
    await expect(mealPlanCell(page, MON, "lunch").getByRole("listitem")).toHaveText(["肉じゃが"]);
    const lunch = rowsIn(await listNextWeek(page), MON, "lunch");
    expect(lunch).toHaveLength(1);
    expect(lunch[0].batchId).toBe(before.batchId);
  });

  test("同一セル内でドロップすると並び替えとして扱う", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, MON, "dinner", nikujaga);
    await assignRecipe(page, MON, "dinner", miso);
    await openNextWeek(page);
    const items = mealPlanCell(page, MON, "dinner").getByRole("listitem");
    await expect(items).toHaveText(["肉じゃが", "みそ汁"]);

    await mealPlanItemRows(page, "みそ汁").dragTo(mealPlanItemRows(page, "肉じゃが"));

    await expect(items).toHaveText(["みそ汁", "肉じゃが"]);
  });

  test("既存割当のあるセルにドロップすると末尾に追加される", async ({ page }) => {
    const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
    const miso = await createRecipeWithIngredients(page, "みそ汁", []);
    await assignRecipe(page, MON, "dinner", nikujaga);
    await assignRecipe(page, TUE, "dinner", miso);
    await openNextWeek(page);

    // 品行ではなくセルの余白に落とす。品行の上に落とすと、その位置への差し込みとして扱われる。
    const target = mealPlanCell(page, TUE, "dinner");
    const box = await target.boundingBox();
    if (!box) throw new Error("no bounding box");
    await mealPlanItemRows(page, "肉じゃが").dragTo(target, {
      targetPosition: { x: box.width / 2, y: box.height - 4 },
    });

    await expect(mealPlanCell(page, MON, "dinner").getByRole("listitem")).toHaveCount(0);
    await expect(target.getByRole("listitem")).toHaveText(["みそ汁", "肉じゃが"]);
  });

  test("ドラッグした品行では品のメニューが開かない", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, MON, "dinner", id);
    await openNextWeek(page);

    // レシピ名のボタンを掴む。ボタンの上で押してもクリックではなくドラッグとして扱われることを確かめる。
    await mealPlanItemRows(page, "肉じゃが")
      .getByRole("button", { name: "肉じゃが", exact: true })
      .dragTo(mealPlanCell(page, TUE, "dinner"));

    await expect(mealPlanCell(page, TUE, "dinner").getByRole("listitem")).toHaveText(["肉じゃが"]);
    await expect(page.getByRole("menu")).toHaveCount(0);
  });
});
