import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { addStockItem, openFridge } from "./helpers.js";

const t = ja.pages.fridge;
const tr = t.review;

function row(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByText(name, { exact: true }) });
}

function progress(done: number, total: number): string {
  return tr.progress.replace("{{done}}", String(done)).replace("{{total}}", String(total));
}

async function startReview(page: Page) {
  await page.getByRole("button", { name: tr.start, exact: true }).click();
  await expect(page.getByRole("button", { name: tr.finish })).toBeVisible();
}

async function seedOnionAndMilk(page: Page) {
  await page.goto("/");
  await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個", expiresInDays: 7 });
  await addStockItem(page, { name: "牛乳", amount: 1, unit: "本", expiresInDays: 3 });
  await openFridge(page);
}

test.describe("在庫の見直し", () => {
  test("見直すと各行に 3 つのボタンが出る", async ({ page }) => {
    await seedOnionAndMilk(page);

    await startReview(page);

    for (const name of ["玉ねぎ", "牛乳"]) {
      const r = row(page, name);
      await expect(r.getByRole("button", { name: tr.ok, exact: true })).toBeVisible();
      await expect(r.getByRole("button", { name: tr.reduced, exact: true })).toBeVisible();
      await expect(r.getByRole("button", { name: tr.gone, exact: true })).toBeVisible();
      await expect(r.getByRole("button", { name: t.row.edit })).toHaveCount(0);
      await expect(r.getByRole("button", { name: t.row.delete })).toHaveCount(0);
    }
    await expect(page.getByText(progress(0, 2))).toBeVisible();
  });

  test("ある を押すと在庫を変えずに見直し済みになる", async ({ page }) => {
    await seedOnionAndMilk(page);
    await startReview(page);

    await row(page, "玉ねぎ").getByRole("button", { name: tr.ok, exact: true }).click();

    await expect(row(page, "玉ねぎ")).toContainText("3 個");
    await expect(row(page, "玉ねぎ")).toHaveAttribute("data-reviewed", "true");
    await expect(page.getByText(progress(1, 2))).toBeVisible();
  });

  test("減った で量を打ち直す", async ({ page }) => {
    await seedOnionAndMilk(page);
    await startReview(page);

    await row(page, "玉ねぎ").getByRole("button", { name: tr.reduced, exact: true }).click();
    const amount = row(page, "玉ねぎ").getByLabel(t.field.amount, { exact: true });
    await expect(amount).toBeFocused();
    await page.keyboard.type("1");
    await page.keyboard.press("Enter");

    await expect(row(page, "玉ねぎ")).toContainText("1 個");
    await expect(row(page, "玉ねぎ")).toHaveAttribute("data-reviewed", "true");

    await page.getByRole("link", { name: ja.nav.recipes }).click();
    await openFridge(page);
    await expect(row(page, "玉ねぎ")).toContainText("1 個");
  });

  test("減った で量を 0 にすると保存されない", async ({ page }) => {
    await seedOnionAndMilk(page);
    await startReview(page);

    await row(page, "玉ねぎ").getByRole("button", { name: tr.reduced, exact: true }).click();
    const amount = row(page, "玉ねぎ").getByLabel(t.field.amount, { exact: true });
    await amount.fill("0");
    await amount.press("Enter");

    await expect(row(page, "玉ねぎ").getByRole("alert")).toHaveText(tr.amount_zero);
    await page.keyboard.press("Escape");
    await expect(row(page, "玉ねぎ")).toContainText("3 個");
    await expect(row(page, "玉ねぎ")).not.toHaveAttribute("data-reviewed", "true");
  });

  test("ない を押すと行が消え、元に戻せる", async ({ page }) => {
    await seedOnionAndMilk(page);
    await startReview(page);

    await row(page, "牛乳").getByRole("button", { name: tr.gone, exact: true }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(row(page, "牛乳")).toHaveCount(0);
    await expect(page.getByText(t.toast.deleted.replace("{{name}}", "牛乳"))).toBeVisible();

    await page.getByRole("button", { name: t.toast.undo }).click();

    await expect(row(page, "牛乳")).toContainText("1 本");
    await expect(row(page, "牛乳")).not.toHaveAttribute("data-reviewed", "true");
    await expect(page.getByText(progress(0, 2))).toBeVisible();
  });

  test("操作のあとは次の行にフォーカスが移る", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "牛乳", amount: 1, unit: "本", expiresInDays: 1 });
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個", expiresInDays: 2 });
    await addStockItem(page, { name: "にんじん", amount: 2, unit: "本", expiresInDays: 3 });
    await openFridge(page);
    await startReview(page);

    await row(page, "牛乳").getByRole("button", { name: tr.ok, exact: true }).click();

    await expect(
      row(page, "玉ねぎ").getByRole("button", { name: tr.ok, exact: true }),
    ).toBeFocused();
  });

  test("見直しを終えると通常の表示に戻る", async ({ page }) => {
    await seedOnionAndMilk(page);
    await startReview(page);
    await row(page, "玉ねぎ").getByRole("button", { name: tr.ok, exact: true }).click();

    await page.getByRole("button", { name: tr.finish }).click();

    for (const name of ["玉ねぎ", "牛乳"]) {
      await expect(row(page, name).getByRole("button", { name: t.row.edit })).toBeVisible();
      await expect(row(page, name).getByRole("button", { name: t.row.delete })).toBeVisible();
    }
    await expect(page.locator("tbody tr[data-reviewed]")).toHaveCount(0);

    await startReview(page);
    await expect(page.getByText(progress(0, 2))).toBeVisible();
  });
});
