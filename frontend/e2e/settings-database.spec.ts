import { readdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { addStockItem, assignRecipe, createRecipeWithIngredients, dateAfter } from "./helpers.js";

const t = ja.pages.settings;
const td = t.database;

async function openSettings(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: ja.nav.settings }).click();
  await page.getByRole("heading", { name: t.title }).waitFor();
}

async function exportData(page: Page) {
  await page.getByRole("button", { name: td.export }).click();
  await expect(page.getByText(td.exported)).toBeVisible();
}

// 「復元する」→「ファイルを選ぶ」→ 確認。E2E の picker は書き出し先のフォルダの最も新しい .db か、open-target のパスを選ぶ。
async function reserveFromFile(page: Page) {
  await page.getByRole("button", { name: td.restore, exact: true }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("button", { name: td.pick_file }).click();
  await dialog.getByRole("button", { name: td.confirm_pick_file }).click();
}

function pendingText(name: string): string {
  return td.pending.replace("{{name}}", name);
}

test.describe("データの書き出し", () => {
  test("データを 1 つのファイルに書き出す", async ({ page, exportDir }) => {
    await page.goto("/");
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await openSettings(page);

    await exportData(page);

    const files = readdirSync(exportDir).filter((f) => /^meal-cycle-\d{8}-\d{6}\.db$/.test(f));
    expect(files).toHaveLength(1);
    await expect(page.getByText(join(exportDir, files[0]))).toBeVisible();
  });
});

test.describe("復元の予約", () => {
  test("書き出したファイルを選んで復元を予約する", async ({ page, exportDir }) => {
    await openSettings(page);
    await exportData(page);
    const exported = readdirSync(exportDir).find((f) => f.endsWith(".db")) ?? "";

    await reserveFromFile(page);

    await expect(page.getByTestId("restore-pending")).toContainText(pendingText(exported));
  });

  test("データのファイルでないものは予約できない", async ({ page, exportDir }) => {
    const note = join(exportDir, "note.db");
    writeFileSync(note, "this is not a database file");
    writeFileSync(join(exportDir, "open-target"), note);
    await openSettings(page);

    await reserveFromFile(page);

    await expect(page.getByRole("dialog").getByRole("alert")).toHaveText(td.errors.not_database);
    await page.getByRole("dialog").getByRole("button", { name: td.back }).click();
    await page.keyboard.press("Escape");
    await expect(page.getByTestId("restore-pending")).toHaveCount(0);
  });
});

test.describe("復元の予約の取り消しと終了", () => {
  test("予約を取り消す", async ({ page }) => {
    await openSettings(page);
    await exportData(page);
    await reserveFromFile(page);
    await expect(page.getByTestId("restore-pending")).toBeVisible();

    await page.getByRole("button", { name: td.cancel_pending }).click();

    await expect(page.getByTestId("restore-pending")).toHaveCount(0);
  });

  test("予約したあとに S09 を開き直しても予約中と表示される", async ({ page, exportDir }) => {
    await openSettings(page);
    await exportData(page);
    const exported = readdirSync(exportDir).find((f) => f.endsWith(".db")) ?? "";
    await reserveFromFile(page);
    await expect(page.getByTestId("restore-pending")).toBeVisible();

    await page.getByRole("link", { name: ja.nav.meal_plan }).click();
    await page.getByRole("heading", { name: ja.pages.meal_plan.title }).waitFor();
    await openSettings(page);

    await expect(page.getByTestId("restore-pending")).toContainText(pendingText(exported));
  });
});
