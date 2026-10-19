import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  copyLeftover,
  createRecipe,
  createRecipeWithIngredients,
  dateAfter,
  deleteRecipe,
  detailBreadcrumb,
  mealPlanCell,
  mealPlanItemRows,
  openItemMenu,
  spreadEntry,
} from "./helpers.js";

const t = ja.pages.meal_plan;

async function openMealCalendar(page: import("@playwright/test").Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  // 初期ルートが献立カレンダーなので、遷移操作は不要。ヘッダで到達確認する。
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
}

// seedDinner は UI を経由せず、date の夕食に names の順で品を割り当てる。
async function seedDinner(page: import("@playwright/test").Page, date: string, names: string[]) {
  for (const name of names) {
    const id = await createRecipeWithIngredients(page, name, []);
    await assignRecipe(page, date, "dinner", id);
  }
}

// date の夕食の `⋯` から「時間帯全体を連続 N 日コピー」で days 日を選び、コピー方法に「作り置き」を選んで進める。
// コピー先に既存の割当があれば、続けて衝突のダイアログが開く。
async function copyWholeMeal(page: import("@playwright/test").Page, date: string, days: number) {
  await mealPlanCell(page, date, "dinner")
    .getByRole("button", { name: t.cell_actions.meal_menu })
    .click();
  await page
    .getByRole("menuitem", { name: t.meal_menu.copy_days.replace("{{days}}", "N") })
    .click();
  await page
    .getByRole("menuitem", { name: t.copy.days_option.replace("{{days}}", String(days)) })
    .click();
  const mode = page.getByRole("dialog", { name: t.copy.mode.title });
  await mode.getByText(t.copy.mode.leftover, { exact: true }).click();
  await mode.getByRole("button", { name: t.copy.mode.confirm }).click();
}

// 画面上の全品行を DOM 順 (= 日付順、セル内は position 順) で取る。
function allItemRows(page: import("@playwright/test").Page) {
  return page.locator('[role="listitem"]');
}

// shiftIsoDate は "YYYY-MM-DD" を days 日ぶんシフトする。UTC ではなくローカル日付として扱う。
function shiftIsoDate(iso: string, days: number): string {
  const [y, m, d] = iso.split("-").map((s) => Number.parseInt(s, 10));
  const dt = new Date(y, m - 1, d + days);
  const yy = dt.getFullYear();
  const mm = String(dt.getMonth() + 1).padStart(2, "0");
  const dd = String(dt.getDate()).padStart(2, "0");
  return `${yy}-${mm}-${dd}`;
}

test.describe("セル内の品表示", () => {
  test("品数超過は省略表示になる", async ({ page }) => {
    // 1 週 × 夕食のみの表示では、セル内表示上限は 4 件。
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁", "豚汁", "冷奴", "きんぴら"]);
    await openMealCalendar(page);

    const cell = mealPlanCell(page, dateAfter(0), "dinner");
    await expect(cell.getByRole("listitem")).toHaveText(["肉じゃが", "みそ汁", "豚汁", "冷奴"]);
    await expect(
      cell.getByRole("button", { name: t.cell.more_items.replace("{{count}}", "1") }),
    ).toBeVisible();
  });

  test("時間帯全体の `⋯` メニューには動かない項目を置かない", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁"]);
    await openMealCalendar(page);

    await page.getByRole("button", { name: t.cell_actions.meal_menu }).click();

    const items = page.getByRole("menuitem");
    await expect(items).toHaveText([
      t.meal_menu.clear,
      t.meal_menu.copy_days.replace("{{days}}", "N"),
    ]);
    await expect(page.getByRole("menuitem", { name: "並び替え" })).toHaveCount(0);
  });
});

test.describe("時間帯内の並び替え", () => {
  test("2 品目を上に動かすと 1 品目と入れ替わる", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁"]);
    await openMealCalendar(page);
    await expect(allItemRows(page)).toHaveText(["肉じゃが", "みそ汁"]);

    await openItemMenu(page, "みそ汁");
    await page.getByRole("menuitem", { name: t.item_menu.move_up }).click();

    await expect(allItemRows(page)).toHaveText(["みそ汁", "肉じゃが"]);
  });

  test("先頭の品は上へ、末尾の品は下へ移動できない", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁", "ごはん"]);
    await openMealCalendar(page);
    const moveUp = page.getByRole("menuitem", { name: t.item_menu.move_up });
    const moveDown = page.getByRole("menuitem", { name: t.item_menu.move_down });

    await openItemMenu(page, "肉じゃが");
    await expect(moveUp).toBeDisabled();
    await expect(moveDown).toBeEnabled();
    await page.keyboard.press("Escape");
    await expect(page.getByRole("menu")).toHaveCount(0);

    await openItemMenu(page, "ごはん");
    await expect(moveDown).toBeDisabled();
    await expect(moveUp).toBeEnabled();
  });

  test("1 品だけのセルには移動の項目を出さない", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが"]);
    await openMealCalendar(page);

    await openItemMenu(page, "肉じゃが");

    await expect(page.getByRole("menuitem", { name: t.item_menu.view_detail })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: t.item_menu.move_up })).toHaveCount(0);
    await expect(page.getByRole("menuitem", { name: t.item_menu.move_down })).toHaveCount(0);
  });

  test("作り置きに属する品も移動できる", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["みそ汁", "肉じゃが"]);
    await spreadEntry(
      page,
      { date: dateAfter(0), meal: "dinner", position: 1 },
      [
        { date: dateAfter(1), meal: "dinner" },
        { date: dateAfter(2), meal: "dinner" },
      ],
      true,
    );
    await openMealCalendar(page);
    // 今日が週末だと 2〜3 日目が翌週になるので、2 週表示で 3 日分を画面に入れる。
    await page.getByTestId("period-weeks-2").click();
    await expect(allItemRows(page)).toHaveText(["みそ汁", "肉じゃが", "肉じゃが", "肉じゃが"]);

    await openItemMenu(page, "肉じゃが", 0);
    await page.getByRole("menuitem", { name: t.item_menu.move_up }).click();

    await expect(allItemRows(page)).toHaveText(["肉じゃが", "みそ汁", "肉じゃが", "肉じゃが"]);
    const batchRows = mealPlanItemRows(page, "肉じゃが");
    const batchId = await batchRows.nth(0).getAttribute("data-batch-id");
    for (const i of [0, 1, 2]) {
      await expect(batchRows.nth(i)).toHaveAttribute("data-batch-member-count", "3");
      await expect(batchRows.nth(i)).toHaveAttribute("data-batch-id", batchId ?? "");
    }
  });
});

test.describe("並べ替えたあとの品の削除と移動", () => {
  test("並べ替えたセルで先頭の品を削除できる", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁", "ほうれん草の胡麻和え"]);
    await openMealCalendar(page);
    await expect(allItemRows(page)).toHaveText(["肉じゃが", "みそ汁", "ほうれん草の胡麻和え"]);

    for (const expected of [
      ["みそ汁", "肉じゃが", "ほうれん草の胡麻和え"],
      ["みそ汁", "ほうれん草の胡麻和え", "肉じゃが"],
    ]) {
      await openItemMenu(page, "肉じゃが");
      await page.getByRole("menuitem", { name: t.item_menu.move_down }).click();
      await expect(allItemRows(page)).toHaveText(expected);
    }

    await openItemMenu(page, "みそ汁");
    await page.getByRole("menuitem", { name: t.item_menu.remove }).click();

    await expect(allItemRows(page)).toHaveText(["ほうれん草の胡麻和え", "肉じゃが"]);
    for (const message of Object.values(t.errors)) {
      await expect(page.getByText(message, { exact: true })).toHaveCount(0);
    }
  });
});

test.describe("品の指定は並び順に左右されない", () => {
  test("同じレシピの 2 品を区別して並べ替えられる", async ({ page }) => {
    const curry = await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, dateAfter(0), "dinner", curry);
    await copyLeftover(page, dateAfter(0), "dinner", 2, true);
    await assignRecipe(page, dateAfter(0), "dinner", curry);
    await openMealCalendar(page);
    const rows = mealPlanCell(page, dateAfter(0), "dinner").locator('[role="listitem"]');
    await expect(rows).toHaveText(["カレー", "カレー"]);
    await expect(rows.nth(0)).toHaveAttribute("data-batch-member-count", "2");
    await expect(rows.nth(1)).toHaveAttribute("data-batch-member-count", "1");

    await openItemMenu(page, "カレー", 1);
    await page.getByRole("menuitem", { name: t.item_menu.move_up }).click();

    await expect(rows.nth(0)).toHaveAttribute("data-batch-member-count", "1");
    await expect(rows.nth(0)).toHaveClass(/border-transparent/);
    await expect(rows.nth(1)).toHaveAttribute("data-batch-member-count", "2");
    await expect(rows.nth(1)).not.toHaveClass(/border-transparent/);
  });
});

// 「元に戻す」付きの通知。sonner の toast は main の外に描かれるので page 起点で探す。
function undoButton(page: import("@playwright/test").Page) {
  return page.getByRole("button", { name: t.toast.undo, exact: true });
}

test.describe("品の削除と時間帯クリアの取り消し", () => {
  test("削除した品を元に戻す", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁", "ほうれん草の胡麻和え"]);
    await openMealCalendar(page);
    await expect(allItemRows(page)).toHaveText(["肉じゃが", "みそ汁", "ほうれん草の胡麻和え"]);

    await openItemMenu(page, "みそ汁");
    await page.getByRole("menuitem", { name: t.item_menu.remove }).click();
    await expect(allItemRows(page)).toHaveText(["肉じゃが", "ほうれん草の胡麻和え"]);

    await undoButton(page).click();

    await expect(allItemRows(page)).toHaveText(["肉じゃが", "みそ汁", "ほうれん草の胡麻和え"]);
  });

  test("削除すると通知に元に戻すが表示される", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが"]);
    await openMealCalendar(page);

    await openItemMenu(page, "肉じゃが");
    await page.getByRole("menuitem", { name: t.item_menu.remove }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(allItemRows(page)).toHaveCount(0);
    await expect(page.getByText(t.toast.removed.replace("{{name}}", "肉じゃが"))).toBeVisible();
    await expect(undoButton(page)).toBeVisible();
  });

  test("クリアした時間帯を元に戻す", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁"]);
    await openMealCalendar(page);
    await expect(allItemRows(page)).toHaveText(["肉じゃが", "みそ汁"]);

    await page.getByRole("button", { name: t.cell_actions.meal_menu }).click();
    await page.getByRole("menuitem", { name: t.meal_menu.clear }).click();
    await expect(allItemRows(page)).toHaveCount(0);
    await expect(page.getByText(t.toast.cleared.replace("{{count}}", "2"))).toBeVisible();

    await undoButton(page).click();

    await expect(allItemRows(page)).toHaveText(["肉じゃが", "みそ汁"]);
  });

  test("作り置きの 1 行を削除して元に戻すと同じ作り置きに戻る", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["カレー"]);
    await copyLeftover(page, dateAfter(0), "dinner", 3, true);
    await openMealCalendar(page);
    // 今日が週末だと 2〜3 日目が翌週になるので、2 週表示で 3 日分を画面に入れる。
    await page.getByTestId("period-weeks-2").click();
    const rows = mealPlanItemRows(page, "カレー");
    await expect(rows).toHaveCount(3);
    const batchId = await rows.nth(0).getAttribute("data-batch-id");

    await openItemMenu(page, "カレー", 1);
    await page.getByRole("menuitem", { name: t.item_menu.remove }).click();
    await expect(rows).toHaveCount(2);

    await undoButton(page).click();

    await expect(rows).toHaveCount(3);
    for (const i of [0, 1, 2]) {
      await expect(rows.nth(i)).toHaveAttribute("data-batch-id", batchId ?? "");
      await expect(rows.nth(i)).toHaveAttribute("data-batch-member-count", "3");
    }
  });
});

test.describe("S01 の操作結果の通知", () => {
  test("失敗した操作は通知で知らされる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await openMealCalendar(page);

    await page.getByRole("button", { name: t.cell_actions.add_first }).first().click();
    const dialog = page.getByRole("dialog");
    await expect(dialog.getByRole("button", { name: "肉じゃが" })).toBeVisible();
    await deleteRecipe(page, id);
    await dialog.getByRole("button", { name: "肉じゃが" }).click();

    const message = t.errors.unknown_recipe;
    await expect(page.locator("[data-sonner-toast]").getByText(message)).toBeVisible();
    await expect(page.getByRole("main").getByText(message)).toHaveCount(0);
    await expect(allItemRows(page)).toHaveCount(0);
  });
});

test.describe("期間切替", () => {
  test("期間切替の既定は 1 週", async ({ page }) => {
    await openMealCalendar(page);
    const oneWeek = page.getByTestId("period-weeks-1");
    await expect(oneWeek).toHaveAttribute("aria-pressed", "true");
    const calendar = page.getByTestId("period-calendar");
    await expect(calendar).toHaveAttribute("data-period-weeks", "1");
  });

  test("2 週を選ぶと 14 日分が表示される", async ({ page }) => {
    await openMealCalendar(page);
    await page.getByTestId("period-weeks-2").click();
    const calendar = page.getByTestId("period-calendar");
    await expect(calendar).toHaveAttribute("data-period-weeks", "2");
    // 折り返しレイアウトなので横スクロールは発生しない。
    await expect(calendar).toHaveCSS("overflow-x", "visible");
  });

  test("4 週を選ぶと 4 週分が行方向に折り返す", async ({ page }) => {
    await openMealCalendar(page);
    await page.getByTestId("period-weeks-4").click();
    const calendar = page.getByTestId("period-calendar");
    await expect(calendar).toHaveAttribute("data-period-weeks", "4");
  });
});

test.describe("前後ナビゲーション", () => {
  test("`＞` を押すと表示中の期間長ぶん未来にシフトする", async ({ page }) => {
    await openMealCalendar(page);
    const todayBtn = page.getByTestId("period-today");
    const before = await todayBtn.getAttribute("data-period-start");
    if (!before) throw new Error("initial data-period-start missing");

    await page.getByTestId("period-next").click();

    // periodStart が 7 日進んでいることを確認する。
    const expected = shiftIsoDate(before, 7);
    await expect(todayBtn).toHaveAttribute("data-period-start", expected);
  });

  test("「今週」ボタンで現在の週に戻る", async ({ page }) => {
    await openMealCalendar(page);
    const todayBtn = page.getByTestId("period-today");
    const initial = await todayBtn.getAttribute("data-period-start");
    if (!initial) throw new Error("initial data-period-start missing");

    await page.getByTestId("period-next").click();
    await page.getByTestId("period-next").click();
    await expect(todayBtn).not.toHaveAttribute("data-period-start", initial);

    await todayBtn.click();
    await expect(todayBtn).toHaveAttribute("data-period-start", initial);
  });
});

test.describe("空セルへの 1 品目割当", () => {
  test("空セルの `＋` からレシピを選んで割当する", async ({ page }) => {
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
});

test.describe("割当済セルへの品追加", () => {
  test("1 品ある時間帯にもう 1 品追加すると 2 品目が下に並ぶ", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが"]);
    await createRecipe(page, "みそ汁");
    await openMealCalendar(page);

    const cell = mealPlanCell(page, dateAfter(0), "dinner");
    await cell.getByRole("button", { name: t.cell_actions.add_more }).click();
    await page.getByRole("dialog").getByRole("button", { name: "みそ汁" }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(cell.getByRole("listitem")).toHaveText(["肉じゃが", "みそ汁"]);
  });
});

test.describe("1 品の入替 / 削除 / 詳細表示", () => {
  test("1 品だけ入れ替える", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁"]);
    await createRecipe(page, "豚汁");
    await openMealCalendar(page);

    await openItemMenu(page, "みそ汁");
    await page.getByRole("menuitem", { name: t.item_menu.replace }).click();
    await page.getByRole("dialog").getByRole("button", { name: "豚汁" }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(mealPlanCell(page, dateAfter(0), "dinner").getByRole("listitem")).toHaveText([
      "肉じゃが",
      "豚汁",
    ]);
  });

  test("1 品だけ削除する", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁"]);
    await openMealCalendar(page);

    await openItemMenu(page, "みそ汁");
    await page.getByRole("menuitem", { name: t.item_menu.remove }).click();

    await expect(mealPlanCell(page, dateAfter(0), "dinner").getByRole("listitem")).toHaveText([
      "肉じゃが",
    ]);
  });

  test("詳細を開くと S02 を経由せずに S01 に戻れる", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openMealCalendar(page);

    await openItemMenu(page, "肉じゃが");
    await page.getByRole("menuitem", { name: t.item_menu.view_detail }).click();

    await expect(page.getByRole("main").getByRole("heading", { name: "肉じゃが" })).toBeVisible();

    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.meal_plan }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByText("肉じゃが", { exact: true })).toBeVisible();
  });
});

test.describe("時間帯全体のクリア", () => {
  test("時間帯右下 `⋯ → 時間帯全体をクリア` で全削除", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁"]);
    await openMealCalendar(page);

    const cell = mealPlanCell(page, dateAfter(0), "dinner");
    await cell.getByRole("button", { name: t.cell_actions.meal_menu }).click();
    await page.getByRole("menuitem", { name: t.meal_menu.clear }).click();

    await expect(cell.getByRole("listitem")).toHaveCount(0);
    await expect(cell.getByRole("button", { name: t.cell_actions.add_first })).toBeVisible();
  });
});

test.describe("作り置きコピー (時間帯全体)", () => {
  test("2 品構成を 3 日分コピーする", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが", "みそ汁"]);
    await openMealCalendar(page);
    // 今日が週末だと 2〜3 日目が翌週になるので、2 週表示で 3 日分を画面に入れる。
    await page.getByTestId("period-weeks-2").click();

    await copyWholeMeal(page, dateAfter(0), 3);

    for (const days of [0, 1, 2]) {
      await expect(mealPlanCell(page, dateAfter(days), "dinner").getByRole("listitem")).toHaveText([
        "肉じゃが",
        "みそ汁",
      ]);
    }
    await expect(mealPlanCell(page, dateAfter(3), "dinner").getByRole("listitem")).toHaveCount(0);
  });
});

test.describe("作り置きコピー先の衝突解決", () => {
  test("追加を選ぶと元の品と併存する", async ({ page }) => {
    await seedDinner(page, dateAfter(0), ["肉じゃが"]);
    await seedDinner(page, dateAfter(1), ["豚汁"]);
    await openMealCalendar(page);
    await page.getByTestId("period-weeks-2").click();

    await copyWholeMeal(page, dateAfter(0), 3);
    const conflict = page.getByRole("dialog", { name: t.copy.conflict.title });
    await conflict.getByRole("button", { name: t.copy.conflict.append }).click();

    await expect(conflict).toHaveCount(0);
    await expect(mealPlanCell(page, dateAfter(1), "dinner").getByRole("listitem")).toHaveText([
      "豚汁",
      "肉じゃが",
    ]);
  });
});
