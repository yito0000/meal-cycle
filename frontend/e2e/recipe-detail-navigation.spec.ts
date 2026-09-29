import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  createRecipe,
  createRecipeWithIngredients,
  detailBreadcrumb,
  openItemMenu,
  openRecipeDetail,
} from "./helpers.js";

const t = ja.pages.recipes;
const mp = ja.pages.meal_plan;

// 来週の金曜をローカル日付の YYYY-MM-DD で返す。S01 の週は月曜始まり。
function nextWeekFriday(): string {
  const now = new Date();
  const offset = (now.getDay() + 6) % 7;
  const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() - offset + 7 + 4);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

test.describe("S03 から開いた元の画面へ戻る操作", () => {
  test.beforeEach(async ({ page }) => {
    await createRecipe(page, "肉じゃが");
  });

  test("パンくずに開いた元とレシピ名が表示される", async ({ page }) => {
    await openRecipeDetail(page, "肉じゃが");

    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.recipes,
      "肉じゃが",
    ]);
  });

  test("パンくずの先頭を押すと開いた元の画面に戻る", async ({ page }) => {
    await openRecipeDetail(page, "肉じゃが");

    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.recipes }).click();

    await expect(page.getByRole("heading", { name: t.title, exact: true })).toBeVisible();
    await expect(detailBreadcrumb(page)).toHaveCount(0);
  });

  test("Esc キーで開いた元の画面に戻る", async ({ page }) => {
    await openRecipeDetail(page, "肉じゃが");

    await page.keyboard.press("Escape");

    await expect(page.getByRole("heading", { name: t.title, exact: true })).toBeVisible();
    await expect(detailBreadcrumb(page)).toHaveCount(0);
  });

  test("確認ダイアログを開いているときの Esc キーはダイアログだけを閉じる", async ({ page }) => {
    await openRecipeDetail(page, "肉じゃが");
    await page.getByRole("main").getByRole("button", { name: t.detail.delete }).click();
    await expect(page.getByRole("dialog")).toContainText(t.delete_confirm.body);

    await page.keyboard.press("Escape");

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(
      page.getByRole("main").getByRole("heading", { name: "肉じゃが", exact: true }),
    ).toBeVisible();
  });
});

test.describe("S01 献立カレンダーから S03 詳細画面へ移る", () => {
  test("S01 から開いてパンくずで戻ると表示期間が保たれている", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, nextWeekFriday(), "dinner", id);
    await page.goto("/");
    await expect(page.getByRole("heading", { name: mp.title })).toBeVisible();
    await page.getByTestId("period-next").click();
    const nextWeekStart = await page.getByTestId("period-today").getAttribute("data-period-start");

    await openItemMenu(page, "肉じゃが");
    await page.getByRole("menuitem", { name: mp.item_menu.view_detail }).click();

    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.meal_plan,
      "肉じゃが",
    ]);

    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.meal_plan }).click();

    await expect(page.getByRole("heading", { name: mp.title })).toBeVisible();
    await expect(page.getByTestId("period-today")).toHaveAttribute(
      "data-period-start",
      nextWeekStart ?? "",
    );
    await expect(page.getByText("肉じゃが", { exact: true })).toBeVisible();
  });
});
