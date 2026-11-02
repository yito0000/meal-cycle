import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { assignRecipe, createRecipeWithIngredients } from "./helpers.js";

const t = ja.pages.meal_plan;
const tc = t.text_copy;

// 今週の月曜から offset 日後。S01 はサイドバーから開くと今週の 1 週表示になる。
function thisWeekDay(offset: number): string {
  const now = new Date();
  const toMonday = (now.getDay() + 6) % 7;
  const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - toMonday + offset);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// アプリと同じく Intl に任せた「11/2(月)」の表記。
function mealDate(date: string): string {
  const [y, m, d] = date.split("-").map(Number);
  return new Intl.DateTimeFormat("ja", {
    month: "numeric",
    day: "numeric",
    weekday: "short",
  }).format(new Date(y, m - 1, d));
}

function line(date: string, meal: string, items: string[]): string {
  return tc.line
    .replace("{{date}}", mealDate(date))
    .replace("{{meal}}", meal)
    .replace("{{items}}", items.join(tc.separator));
}

async function openMealCalendar(page: Page) {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
}

async function plan(page: Page, date: string, meal: string, name: string) {
  const id = await createRecipeWithIngredients(page, name, []);
  await assignRecipe(page, date, meal, id);
}

async function copyText(page: Page, days: number): Promise<string> {
  await page.getByTestId("mealplan-copy-text").click();
  await expect(page.getByText(tc.copied.replace("{{count}}", String(days)))).toBeVisible();
  // Windows のクリップボードは読み出し時に改行を CRLF に変えるので、比較の前に LF にそろえる。
  const text = await page.evaluate(() => navigator.clipboard.readText());
  return text.replaceAll("\r\n", "\n");
}

test.describe("献立のテキストコピー", () => {
  test.beforeEach(async ({ page }) => {
    await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  });

  test("表示中の期間の献立をコピーする", async ({ page }) => {
    const mon = thisWeekDay(0);
    const wed = thisWeekDay(2);
    await plan(page, mon, "dinner", "肉じゃが");
    await plan(page, mon, "dinner", "味噌汁");
    await plan(page, wed, "dinner", "カレー");
    await openMealCalendar(page);

    const text = await copyText(page, 2);

    const heading = tc.heading
      .replace("{{start}}", mealDate(mon))
      .replace("{{end}}", mealDate(thisWeekDay(6)));
    expect(text).toBe(
      [
        heading,
        line(mon, tc.meal_dinner, ["肉じゃが", "味噌汁"]),
        line(wed, tc.meal_dinner, ["カレー"]),
      ].join("\n"),
    );
  });

  test("夕食のみの表示でも昼食の献立を含める", async ({ page }) => {
    const tue = thisWeekDay(1);
    await plan(page, tue, "lunch", "サンドイッチ");
    await plan(page, tue, "dinner", "カレー");
    await openMealCalendar(page);
    await expect(page.getByRole("checkbox", { name: t.meal.lunch })).not.toBeChecked();

    const text = await copyText(page, 1);

    const lunch = line(tue, tc.meal_lunch, ["サンドイッチ"]);
    const dinner = line(tue, tc.meal_dinner, ["カレー"]);
    expect(text).toContain(`${lunch}\n${dinner}`);
  });

  test("献立の無い期間ではコピーできない", async ({ page }) => {
    await openMealCalendar(page);

    await expect(page.getByTestId("mealplan-copy-text")).toBeDisabled();
  });
});
