import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { assignRecipe, createRecipeWithIngredients, dateAfter, mealPlanCell } from "./helpers.js";

test.describe("GUI へのエージェントの書き込みの反映", () => {
  test("ウィンドウに戻ると割り当てた献立が表示される", async ({ page }) => {
    const today = dateAfter(0);
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);

    await page.goto("/");
    await expect(page.getByRole("heading", { name: ja.pages.meal_plan.title })).toBeVisible();
    const cell = mealPlanCell(page, today, "dinner");
    await expect(cell).toBeVisible();
    await expect(cell.getByText("肉じゃが", { exact: true })).toHaveCount(0);

    // MCP サーバーは別プロセスから同じ DB に書くので、GUI のキャッシュを通らない E2E サーバーの API で代わりに書く。
    await assignRecipe(page, today, "dinner", id);
    await page.evaluate(() => window.dispatchEvent(new Event("focus")));

    await expect(cell.getByText("肉じゃが", { exact: true })).toBeVisible();
  });
});
