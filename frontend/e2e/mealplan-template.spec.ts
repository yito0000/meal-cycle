import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipeWithIngredients, deleteRecipe, mealPlanCell } from "./helpers.js";

const t = ja.pages.meal_plan;
const tt = t.template;

// 今週の月曜から offset 日後。S01 はサイドバーから開くと今週の 1 週表示になる。
function weekDay(offset: number): string {
  const now = new Date();
  const toMonday = (now.getDay() + 6) % 7;
  const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - toMonday + offset);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

async function api<T>(page: Page, method: string, ...args: unknown[]): Promise<T> {
  const res = await page.request.post(`/api/MealPlanService/${method}`, { data: args });
  if (!res.ok()) throw new Error(`${method} failed: ${res.status()} ${await res.text()}`);
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

// 今週の月〜水の夕食に肉じゃがの作り置き、木の夕食にカレーを置く。
async function planStewWeek(page: Page) {
  await page.goto("/");
  const nikujaga = await createRecipeWithIngredients(page, "肉じゃが", []);
  const curry = await createRecipeWithIngredients(page, "カレー", []);
  await api(page, "ApplyPlan", [
    { date: weekDay(0), meal: "dinner", recipeId: nikujaga, group: 0 },
    { date: weekDay(1), meal: "dinner", recipeId: nikujaga, group: 0 },
    { date: weekDay(2), meal: "dinner", recipeId: nikujaga, group: 0 },
    { date: weekDay(3), meal: "dinner", recipeId: curry, group: 1 },
  ]);
  return { nikujaga, curry };
}

async function saveStewTemplate(page: Page) {
  const ids = await planStewWeek(page);
  const tpl = await api<{ id: string }>(page, "SaveTemplate", weekDay(0), "煮物の週");
  return { ...ids, templateId: tpl.id };
}

async function openMealCalendar(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
}

async function openMenu(page: Page) {
  await page.getByTestId("template-menu").click();
}

async function openList(page: Page) {
  await openMenu(page);
  await page.getByRole("menuitem", { name: tt.open_list }).click();
  const dialog = page.getByRole("dialog", { name: tt.list_title });
  await dialog.waitFor();
  return dialog;
}

function templateCard(page: Page, name: string) {
  return page.getByRole("dialog").getByRole("region", { name });
}

test.describe("献立テンプレートの保存", () => {
  test("週の献立をテンプレートとして保存する", async ({ page }) => {
    await planStewWeek(page);
    await openMealCalendar(page);

    await openMenu(page);
    await page.getByRole("menuitem", { name: tt.save_week }).click();
    await page.getByLabel(tt.name_label).fill("煮物の週");
    await page.getByRole("button", { name: tt.save, exact: true }).click();

    await expect(page.getByText(tt.saved.replace("{{name}}", "煮物の週"))).toBeVisible();
    await openList(page);
    await expect(templateCard(page, "煮物の週")).toContainText(
      tt.item_count.replace("{{count}}", "4"),
    );
  });

  test("品の無い週は保存できない", async ({ page }) => {
    await openMealCalendar(page);

    await openMenu(page);

    await expect(page.getByRole("menuitem", { name: tt.save_week })).toHaveAttribute(
      "data-disabled",
      "",
    );
  });

  test("同じ名前のテンプレートは保存できない", async ({ page }) => {
    await saveStewTemplate(page);
    await openMealCalendar(page);

    await openMenu(page);
    await page.getByRole("menuitem", { name: tt.save_week }).click();
    await page.getByLabel(tt.name_label).fill("煮物の週");
    await page.getByRole("button", { name: tt.save, exact: true }).click();

    await expect(page.getByRole("alert")).toHaveText(t.errors.template_name_duplicate);
  });
});

test.describe("献立テンプレートの適用", () => {
  test("テンプレートを別の週に適用する", async ({ page }) => {
    await saveStewTemplate(page);
    await openMealCalendar(page);
    await page.getByTestId("period-next").click();

    await openList(page);
    await templateCard(page, "煮物の週").getByRole("button", { name: tt.apply }).click();

    await expect(page.getByRole("dialog", { name: tt.list_title })).toHaveCount(0);
    await expect(page.getByText(tt.applied.replace("{{count}}", "4"))).toBeVisible();
    for (const offset of [7, 8, 9]) {
      await expect(mealPlanCell(page, weekDay(offset), "dinner")).toContainText("肉じゃが");
    }
    await expect(mealPlanCell(page, weekDay(10), "dinner")).toContainText("カレー");
    const entries = await api<{ recipeName: string; batchId: string; batchMemberCount: number }[]>(
      page,
      "ListPeriod",
      weekDay(7),
      1,
    );
    const stew = entries.filter((e) => e.recipeName === "肉じゃが");
    expect(new Set(stew.map((e) => e.batchId)).size).toBe(1);
    expect(stew[0].batchMemberCount).toBe(3);
  });

  test("適用を元に戻すと割り当てた品が消える", async ({ page }) => {
    await saveStewTemplate(page);
    await openMealCalendar(page);
    await page.getByTestId("period-next").click();
    await openList(page);
    await templateCard(page, "煮物の週").getByRole("button", { name: tt.apply }).click();
    await expect(mealPlanCell(page, weekDay(10), "dinner")).toContainText("カレー");

    await page.getByRole("button", { name: t.toast.undo }).click();

    await expect(mealPlanCell(page, weekDay(10), "dinner")).not.toContainText("カレー");
    await expect(mealPlanCell(page, weekDay(7), "dinner")).not.toContainText("肉じゃが");
  });

  test("削除済みのレシピの品は除いて適用する", async ({ page }) => {
    const { curry } = await saveStewTemplate(page);
    await deleteRecipe(page, curry);
    await openMealCalendar(page);
    await page.getByTestId("period-next").click();

    await openList(page);
    await expect(templateCard(page, "煮物の週")).toContainText(
      tt.archived_item.replace("{{name}}", "カレー"),
    );
    await templateCard(page, "煮物の週").getByRole("button", { name: tt.apply }).click();

    await expect(
      page.getByText(tt.applied_skipped.replace("{{count}}", "3").replace("{{skipped}}", "1")),
    ).toBeVisible();
    await expect(mealPlanCell(page, weekDay(7), "dinner")).toContainText("肉じゃが");
    await expect(mealPlanCell(page, weekDay(10), "dinner")).not.toContainText("カレー");
  });
});

test.describe("献立テンプレートの削除", () => {
  test("テンプレートを削除する", async ({ page }) => {
    const { templateId } = await saveStewTemplate(page);
    await api(page, "ApplyTemplate", templateId, weekDay(7));
    await openMealCalendar(page);

    await openList(page);
    await templateCard(page, "煮物の週").getByRole("button", { name: tt.delete }).click();
    await page.getByRole("button", { name: tt.delete_confirm.confirm }).click();

    await expect(templateCard(page, "煮物の週")).toHaveCount(0);
    const entries = await api<unknown[]>(page, "ListPeriod", weekDay(7), 1);
    expect(entries).toHaveLength(4);
  });
});
