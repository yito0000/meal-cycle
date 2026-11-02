import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  assignRecipe,
  copyLeftover,
  createRecipeWithIngredients,
  dateAfter,
  openShoppingList,
} from "./helpers.js";

const t = ja.pages.shopping;

function itemRow(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByRole("cell", { name, exact: true }) });
}

// 画面を通さずに世帯人数だけを変えて保存する。ほかの値は今の値をそのまま送る。
async function saveHousehold(page: Page, n: number) {
  const current = await (await page.request.post("/api/SettingsService/Get", { data: [] })).json();
  const res = await page.request.post("/api/SettingsService/Save", {
    data: [{ ...current, householdServings: n }],
  });
  if (!res.ok()) throw new Error(`save settings failed: ${res.status()} ${await res.text()}`);
}

function scaledNote(n: number): string {
  return t.scaled_note.replace("{{count}}", String(n));
}

test.describe("世帯人数による必要量の換算", () => {
  test("1 回きりの献立を世帯人数で換算する", async ({ page }) => {
    await page.goto("/");
    await saveHousehold(page, 4);
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "じゃがいも", amount: 3, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(1), "dinner", id);

    await openShoppingList(page);

    await expect(itemRow(page, "じゃがいも")).toContainText("6 個");
    await expect(page.getByTestId("shopping-scaled-note")).toHaveText(scaledNote(4));
  });

  test("作り置きは換算しない", async ({ page }) => {
    await page.goto("/");
    await saveHousehold(page, 4);
    const id = await createRecipeWithIngredients(page, "カレー", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(1), "dinner", id);
    await copyLeftover(page, dateAfter(1), "dinner", 3, true);

    await openShoppingList(page);

    await expect(itemRow(page, "玉ねぎ")).toContainText("2 個");
    await expect(page.getByTestId("shopping-scaled-note")).toHaveCount(0);
  });

  test("換算しないときはレシピの量のまま", async ({ page }) => {
    await page.goto("/");
    const id = await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "じゃがいも", amount: 3, unit: "個" },
    ]);
    await assignRecipe(page, dateAfter(1), "dinner", id);

    await openShoppingList(page);

    await expect(itemRow(page, "じゃがいも")).toContainText("3 個");
    await expect(page.getByTestId("shopping-scaled-note")).toHaveCount(0);
  });
});
