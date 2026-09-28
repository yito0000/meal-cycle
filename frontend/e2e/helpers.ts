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

// copyLeftover は (date, meal) の 1 品目を days 日分に広げる。shareBatch=true で作り置き、false で別調理。
export async function copyLeftover(
  page: Page,
  date: string,
  meal: string,
  days: number,
  shareBatch: boolean,
) {
  const res = await page.request.post("/api/MealPlanService/CopyLeftover", {
    data: [date, meal, days, 0, "append", shareBatch],
  });
  if (!res.ok()) {
    throw new Error(`copy leftover ${date} failed: ${res.status()} ${await res.text()}`);
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

// S02 を開き、name の行を押して S03 ドロワーが見えるまで待つ。
export async function openRecipeDetail(page: Page, name: string) {
  await openRecipeList(page);
  await page.getByRole("cell", { name, exact: true }).click();
  await page.getByRole("dialog").waitFor();
}

// S01 の品行の `⋯` を押してメニューを開く。DropdownMenu の中身は body 直下の Portal に出るが、
// page 起点で getByRole("menuitem") を探せば DOM の親子関係に縛られずに届く (add-recipe-details D9)。
export async function openItemMenu(page: Page, recipeName: string) {
  const trigger = page.getByRole("button", { name: ja.pages.meal_plan.cell_actions.item_menu });
  const row = page
    .locator("div")
    .filter({ has: page.getByText(recipeName, { exact: true }) })
    .filter({ has: trigger })
    .last();
  await row.getByRole("button", { name: ja.pages.meal_plan.cell_actions.item_menu }).click();
  await page.getByRole("menu").waitFor();
}

// 開いている S03 ドロワーで「削除」を押し、確認ダイアログの「削除」を選んでドロワーが閉じるまで待つ。
// 確認ダイアログの「削除」ボタンはドロワーのものと同名なので、ダイアログのタイトルで絞る。
export async function deleteRecipeFromDetail(page: Page) {
  await page.getByRole("dialog").getByRole("button", { name: t.detail.delete }).click();
  const confirm = page.getByRole("dialog").filter({ hasText: t.delete_confirm.body }).last();
  await confirm.getByRole("button", { name: t.delete_confirm.confirm }).click();
  // 確認ダイアログとドロワーが順に閉じる間は dialog が 2 つあり、locator.waitFor だと strict 違反になる。
  await page.waitForFunction(() => document.querySelectorAll('[role="dialog"]').length === 0);
}

// deleteRecipe は UI を経由せずにレシピを削除する。S01 側のシナリオの前提づくりに使う。
export async function deleteRecipe(page: Page, id: string) {
  const res = await page.request.post("/api/RecipeService/Delete", { data: [id] });
  if (!res.ok()) {
    throw new Error(`delete ${id} failed: ${res.status()} ${await res.text()}`);
  }
}
