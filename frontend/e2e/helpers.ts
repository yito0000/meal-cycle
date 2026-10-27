import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };

const t = ja.pages.recipes;

// サイドバーから S02 に入る。ルータが memory history なので URL 直打ちでは
// 遷移しない (design.md D8)。
export async function openRecipeList(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: t.title }).click();
}

// createRecipe は UI を経由せず shim と同じ HTTP 経路で 1 件作る。
// 前提条件を整えるだけのステップを画面操作で書くと、検証したいシナリオが
// 別のシナリオの成否に引きずられるため。
export async function createRecipe(page: Page, name: string, tags: string[] = []) {
  const res = await page.request.post("/api/RecipeService/Create", {
    data: [
      {
        id: "",
        name,
        servings: 2,
        tags,
        ingredients: [{ name: "玉ねぎ", amount: 1, unit: "個" }],
        ingredientCount: 1,
        steps: "",
        notes: "",
        createdAt: "",
        updatedAt: "",
      },
    ],
  });
  if (!res.ok()) {
    throw new Error(`create ${name} failed: ${res.status()} ${await res.text()}`);
  }
}

const tf = ja.pages.fridge;

// サイドバーから S05 に入る。
export async function openFridge(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: ja.nav.fridge }).click();
  await page.getByRole("heading", { name: tf.title }).waitFor();
}

// dateAfter はローカル日付で n 日後の YYYY-MM-DD を返す。e2e-server も同じマシンの
// ローカル日付で残り日数を数えるので、ここで UTC を使うと深夜帯に 1 日ずれる。
export function dateAfter(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() + days);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// addStockItem は UI を経由せず 1 行作る (createRecipe と同じ理由)。
export async function addStockItem(
  page: Page,
  item: { name: string; amount: number; unit: string; expiresInDays?: number },
) {
  const res = await page.request.post("/api/FridgeService/Add", {
    data: [
      {
        id: "",
        name: item.name,
        amount: item.amount,
        unit: item.unit,
        expiresAt: item.expiresInDays === undefined ? "" : dateAfter(item.expiresInDays),
        addedAt: "",
        updatedAt: "",
        expiryStatus: "",
        daysLeft: null,
      },
    ],
  });
  if (!res.ok()) {
    throw new Error(`add stock ${item.name} failed: ${res.status()} ${await res.text()}`);
  }
}

const ts = ja.pages.shopping;

// サイドバーから S07 に入る。search 無しのリンクなので対象期間は既定 (今日から 7 日間) になる。
export async function openShoppingList(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: ja.nav.shopping }).click();
  await page.getByRole("heading", { name: ts.title }).waitFor();
}

// S07 の材料の行に購入済チェックを付ける。
export async function checkShoppingItem(page: Page, name: string) {
  await page.getByLabel(ts.purchase.check_row.replace("{{name}}", name), { exact: true }).check();
}

// S07 の量を持たない材料の欄から、食材名が name の項目を取る。
export function shoppingPantryItem(page: Page, name: string) {
  return page
    .getByTestId("shopping-pantry-item")
    .filter({ has: page.getByText(name, { exact: true }) });
}

// createRecipeWithIngredients は材料を指定してレシピを 1 件作り、ID を返す。
export async function createRecipeWithIngredients(
  page: Page,
  name: string,
  ingredients: { name: string; amount: number; unit: string }[],
): Promise<string> {
  const res = await page.request.post("/api/RecipeService/Create", {
    data: [
      {
        id: "",
        name,
        servings: 2,
        tags: [],
        ingredients,
        ingredientCount: ingredients.length,
        steps: "",
        notes: "",
        createdAt: "",
        updatedAt: "",
      },
    ],
  });
  if (!res.ok()) {
    throw new Error(`create ${name} failed: ${res.status()} ${await res.text()}`);
  }
  return (await res.json()).id as string;
}

export async function assignRecipe(page: Page, date: string, meal: string, recipeId: string) {
  const res = await page.request.post("/api/MealPlanService/AssignRecipe", {
    data: [date, meal, recipeId],
  });
  if (!res.ok()) {
    throw new Error(`assign ${date} ${meal} failed: ${res.status()} ${await res.text()}`);
  }
}

// copyLeftover は (date, meal) の全品を days 日分に広げる。shareBatch=true で作り置き、false で別調理。
// 2 品以上あるセルで 1 品だけを広げたいときは spreadEntry を使う。
export async function copyLeftover(
  page: Page,
  date: string,
  meal: string,
  days: number,
  shareBatch: boolean,
) {
  const res = await page.request.post("/api/MealPlanService/CopyLeftover", {
    data: [date, meal, days, "append", shareBatch],
  });
  if (!res.ok()) {
    throw new Error(`copy leftover ${date} failed: ${res.status()} ${await res.text()}`);
  }
}

// entryId は (date, meal, position) にいまある品の ID を返す。サービスは品を ID で指すが、
// テストの前提は並び順で書いたほうが読みやすいため、ここで引き直す (add-mealplan-entry-id design.md D7)。
export async function entryId(
  page: Page,
  date: string,
  meal: string,
  position: number,
): Promise<string> {
  const res = await page.request.post("/api/MealPlanService/ListPeriod", { data: [date, 1] });
  if (!res.ok()) {
    throw new Error(`list ${date} failed: ${res.status()} ${await res.text()}`);
  }
  const entries = (await res.json()) as {
    id: string;
    date: string;
    meal: string;
    position: number;
  }[];
  const found = entries.find((e) => e.date === date && e.meal === meal && e.position === position);
  if (!found) {
    throw new Error(`no entry at ${date} ${meal} ${position}`);
  }
  return found.id;
}

// spreadEntry は (date, meal, position) の 1 品を targets のセルへ広げる。
export async function spreadEntry(
  page: Page,
  source: { date: string; meal: string; position: number },
  targets: { date: string; meal: string }[],
  shareBatch: boolean,
) {
  const id = await entryId(page, source.date, source.meal, source.position);
  const res = await page.request.post("/api/MealPlanService/SpreadEntry", {
    data: [id, targets, "append", shareBatch],
  });
  if (!res.ok()) {
    throw new Error(`spread ${source.date} failed: ${res.status()} ${await res.text()}`);
  }
}

// setCooked は UI を経由せず (date, meal, position) の品に作った印を付け外しする。S07 側のシナリオの前提づくりに使う。
export async function setCooked(
  page: Page,
  date: string,
  meal: string,
  position: number,
  cooked: boolean,
) {
  const id = await entryId(page, date, meal, position);
  const res = await page.request.post("/api/MealPlanService/SetCooked", {
    data: [id, cooked],
  });
  if (!res.ok()) {
    throw new Error(`set cooked ${date} ${meal} failed: ${res.status()} ${await res.text()}`);
  }
}

// createRecipeDetailed は S03 の表示を確かめるために、各項目を指定してレシピを 1 件作り ID を返す。
// 省略した項目は createRecipe と同じ既定値にする。材料 0 件は ingredients: [] を明示する。
export async function createRecipeDetailed(
  page: Page,
  input: {
    name: string;
    tags?: string[];
    servings?: number;
    ingredients?: { name: string; amount: number; unit: string }[];
    steps?: string;
    notes?: string;
  },
): Promise<string> {
  const ingredients = input.ingredients ?? [{ name: "玉ねぎ", amount: 1, unit: "個" }];
  const res = await page.request.post("/api/RecipeService/Create", {
    data: [
      {
        id: "",
        name: input.name,
        servings: input.servings ?? 2,
        tags: input.tags ?? [],
        ingredients,
        ingredientCount: ingredients.length,
        steps: input.steps ?? "",
        notes: input.notes ?? "",
        createdAt: "",
        updatedAt: "",
      },
    ],
  });
  if (!res.ok()) {
    throw new Error(`create ${input.name} failed: ${res.status()} ${await res.text()}`);
  }
  return (await res.json()).id as string;
}

// S03 と S04 の上部にあるパンくず。サイドバーも navigation なので aria-label で絞る。
export function detailBreadcrumb(page: Page) {
  return page.getByRole("navigation", { name: ja.breadcrumb.aria_label });
}

// S02 を開き、name の行を押して S03 詳細画面のパンくずが見えるまで待つ。
export async function openRecipeDetail(page: Page, name: string) {
  await openRecipeList(page);
  await page.getByRole("cell", { name, exact: true }).click();
  await detailBreadcrumb(page).getByText(name, { exact: true }).waitFor();
}

// S01 の品行。夕食のみ表示ではセルが日付順に並ぶので、同名の品行も DOM 順 = 日付順になる。
// 品のメニューが開いている間は Radix が品行に aria-hidden を付けるので、role ではなく属性で探す。
export function mealPlanItemRows(page: Page, recipeName: string) {
  return page
    .locator('[role="listitem"]')
    .filter({ has: page.getByText(recipeName, { exact: true }) });
}

// S01 の品行のレシピ名を押して品のメニューを開く。同名の品が複数あるときは nth で何番目の行かを選ぶ。
// DropdownMenu の中身は body 直下の Portal に出るが、page 起点で getByRole("menuitem") を探せば
// DOM の親子関係に縛られずに届く (add-recipe-details D9)。
export async function openItemMenu(page: Page, recipeName: string, nth = 0) {
  await mealPlanItemRows(page, recipeName)
    .nth(nth)
    .getByRole("button", { name: recipeName, exact: true })
    .click();
  await page.getByRole("menu").waitFor();
}

// 開いている S03 詳細画面で「削除」を押し、確認ダイアログの「削除」を選んでダイアログが閉じるまで待つ。
// 確認ダイアログの「削除」ボタンは S03 のものと同名なので、S03 側は main で、確認側はダイアログで絞る。
export async function deleteRecipeFromDetail(page: Page) {
  await page.getByRole("main").getByRole("button", { name: t.detail.delete }).click();
  const confirm = page.getByRole("dialog").filter({ hasText: t.delete_confirm.body });
  await confirm.getByRole("button", { name: t.delete_confirm.confirm }).click();
  await page.waitForFunction(() => document.querySelectorAll('[role="dialog"]').length === 0);
}

// deleteRecipe は UI を経由せずにレシピを削除する。S01 側のシナリオの前提づくりに使う。
export async function deleteRecipe(page: Page, id: string) {
  const res = await page.request.post("/api/RecipeService/Delete", { data: [id] });
  if (!res.ok()) {
    throw new Error(`delete ${id} failed: ${res.status()} ${await res.text()}`);
  }
}

// 品のメニューから「作り置きを広げる」の統合ダイアログを開き、ダイアログの locator を返す。
export async function openSpreadDialog(page: Page, recipeName: string, nth = 0) {
  await openItemMenu(page, recipeName, nth);
  await page.getByRole("menuitem", { name: ja.pages.meal_plan.item_menu.spread }).click();
  const dialog = page.getByRole("dialog", { name: ja.pages.meal_plan.spread.title });
  await dialog.waitFor();
  return dialog;
}

// 統合ダイアログの格子の (date, meal) セル。選べるセルにはチェックボックスが、選べないセルには印が入る。
export function spreadCell(page: Page, date: string, meal: string) {
  return page
    .getByRole("dialog", { name: ja.pages.meal_plan.spread.title })
    .locator(`td[data-date="${date}"][data-meal="${meal}"]`);
}

// S01 の (date, meal) セル。
export function mealPlanCell(page: Page, date: string, meal: string) {
  return page.locator(`div[role="group"][data-date="${date}"][data-meal="${meal}"]`);
}

// S01 のセルの「他 N 件」を押し、開いた全品の一覧の locator を返す。
// 一覧の中で品のメニューを開くと Radix が一覧に aria-hidden を付けるので、role ではなく test id で探す。
export async function openOverflowList(page: Page, date: string, meal: string) {
  await mealPlanCell(page, date, meal)
    .getByRole("button", {
      name: new RegExp(`^${ja.pages.meal_plan.cell.more_items.replace("{{count}}", "\\d+")}$`),
    })
    .click();
  const list = page.getByTestId("cell-overflow-list");
  await list.waitFor();
  return list;
}

// addStaple は UI を経由せず常備品を 1 件作り、state が渡されたらその状態にする。
export async function addStaple(page: Page, name: string, state?: "enough" | "low" | "out") {
  const res = await page.request.post("/api/FridgeService/AddStaple", { data: [name] });
  if (!res.ok()) {
    throw new Error(`add staple ${name} failed: ${res.status()} ${await res.text()}`);
  }
  if (state === undefined || state === "enough") return;
  const id = (await res.json()).id as string;
  const set = await page.request.post("/api/FridgeService/SetStapleState", { data: [id, state] });
  if (!set.ok()) {
    throw new Error(`set staple ${name} failed: ${set.status()} ${await set.text()}`);
  }
}

// S05 の常備品の欄から、食材名が name の行を取る。
export function stapleRow(page: Page, name: string) {
  return page.getByTestId("staple-row").filter({ has: page.getByText(name, { exact: true }) });
}

// S07 の常備品の欄から、食材名が name の項目を取る。
export function shoppingStapleItem(page: Page, name: string) {
  return page
    .getByTestId("shopping-staple-item")
    .filter({ has: page.getByText(name, { exact: true }) });
}
