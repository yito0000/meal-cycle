import type { Page } from "@playwright/test";
import en from "../src/locales/en.json" with { type: "json" };
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStockItem,
  assignRecipe,
  createRecipeWithIngredients,
  dateAfter,
  openFridge,
  openShoppingList,
} from "./helpers.js";

const t = ja.pages.settings;
const tm = ja.pages.meal_plan;
const tsh = ja.pages.shopping;

// サイドバーから S09 に入る。ルータが memory history なので URL 直打ちでは遷移しない。
async function openSettings(page: Page) {
  if (!page.url().startsWith("http")) {
    await page.goto("/");
  }
  await page.getByRole("link", { name: ja.nav.settings }).click();
  await page.getByRole("heading", { name: t.title }).waitFor();
}

function redDays(page: Page) {
  return page.getByLabel(t.field.expiry_red, { exact: true });
}

function yellowDays(page: Page) {
  return page.getByLabel(t.field.expiry_yellow, { exact: true });
}

function saveButton(page: Page) {
  return page.getByRole("button", { name: t.save, exact: true });
}

async function save(page: Page) {
  await saveButton(page).click();
  await expect(page.getByText(t.toast.saved)).toBeVisible();
}

function listField(page: Page, label: string) {
  return page.getByRole("group", { name: label, exact: true });
}

function chips(page: Page, label: string) {
  return listField(page, label).locator("li[data-chip]");
}

async function addToList(page: Page, label: string, value: string) {
  const field = listField(page, label);
  await field.getByRole("textbox").fill(value);
  await field.getByRole("button", { name: t.add, exact: true }).click();
}

function dayChip(days: number): string {
  return t.day_chip.replace("{{count}}", String(days));
}

function removeButton(page: Page, label: string, value: string) {
  return listField(page, label).getByRole("button", {
    name: t.remove.replace("{{value}}", value),
    exact: true,
  });
}

async function chooseOption(page: Page, label: string, option: string) {
  await page.getByRole("combobox", { name: label }).click();
  await page.getByRole("option", { name: option, exact: true }).click();
}

function stockDot(page: Page, name: string) {
  return page
    .locator("tbody tr")
    .filter({ has: page.getByText(name, { exact: true }) })
    .locator("[data-expiry-status]");
}

test.describe("S09 設定画面の表示", () => {
  test("サイドバーから設定画面を開く", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: tm.title })).toBeVisible();

    await openSettings(page);

    const link = page.getByRole("link", { name: ja.nav.settings });
    await expect(link).toHaveClass(/bg-blue-100/);
    await expect(page.getByRole("heading", { name: t.section.fridge })).toBeVisible();
    await expect(redDays(page)).toHaveValue("3");
    await expect(yellowDays(page)).toHaveValue("7");
  });
});

test.describe("設定の保存", () => {
  test("保存した値は開き直しても残る", async ({ page }) => {
    await openSettings(page);
    await redDays(page).fill("2");
    await save(page);

    await page.reload();
    await openSettings(page);

    await expect(redDays(page)).toHaveValue("2");
  });

  test("保存せずに離れると変更は捨てられる", async ({ page }) => {
    await openSettings(page);
    await redDays(page).fill("2");

    await openFridge(page);
    await openSettings(page);

    await expect(redDays(page)).toHaveValue("3");
  });

  test("値を変えるまで保存は押せない", async ({ page }) => {
    await openSettings(page);

    await expect(saveButton(page)).toBeDisabled();
  });
});

test.describe("設定値の検証", () => {
  test("🔴 の日数が 🟡 の日数以上だと保存されない", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "A", amount: 1, unit: "個", expiresInDays: 5 });
    await openSettings(page);

    await redDays(page).fill("7");
    await yellowDays(page).fill("7");
    await saveButton(page).click();

    await expect(page.getByText(t.errors.red_not_below_yellow)).toBeVisible();
    await openFridge(page);
    await expect(stockDot(page, "A")).toHaveAttribute("data-expiry-status", "soon");
  });

  test("同じ量る単位は追加できない", async ({ page }) => {
    await openSettings(page);

    await addToList(page, t.field.measuring_units, "大さじ");

    await expect(listField(page, t.field.measuring_units)).toContainText(t.errors.unit_duplicate);
    await expect(chips(page, t.field.measuring_units)).toHaveText(["大さじ", "小さじ", "カップ"]);
  });

  test("買い物リストの 7 / 14 / 28 日は消せない", async ({ page }) => {
    await openSettings(page);

    const label = t.field.shopping_presets;
    await expect(removeButton(page, label, dayChip(3))).toBeVisible();
    await expect(removeButton(page, label, dayChip(4))).toBeVisible();
    for (const days of [7, 14, 28]) {
      await expect(
        listField(page, label).locator(`li[data-chip="${dayChip(days)}"]`),
      ).toBeVisible();
      await expect(removeButton(page, label, dayChip(days))).toHaveCount(0);
    }
  });
});

test.describe("既定に戻す", () => {
  test("既定に戻すとすべての値が既定値になる", async ({ page }) => {
    await openSettings(page);
    await redDays(page).fill("2");
    await addToList(page, t.field.measuring_units, "合");
    await save(page);

    await page.getByRole("button", { name: t.reset, exact: true }).click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: t.reset_confirm.confirm, exact: true })
      .click();

    await expect(page.getByText(t.toast.reset)).toBeVisible();
    await expect(redDays(page)).toHaveValue("3");
    await expect(chips(page, t.field.measuring_units)).toHaveText(["大さじ", "小さじ", "カップ"]);
  });

  test("確認で取り消すと値は変わらない", async ({ page }) => {
    await openSettings(page);
    await redDays(page).fill("2");
    await save(page);

    await page.getByRole("button", { name: t.reset, exact: true }).click();
    await page
      .getByRole("dialog")
      .getByRole("button", { name: t.reset_confirm.cancel, exact: true })
      .click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(redDays(page)).toHaveValue("2");
  });
});

test.describe("保存した値の各画面への反映", () => {
  test("期限アラートの日数を変えると S05 の色が変わる", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "A", amount: 1, unit: "個", expiresInDays: 5 });
    await openSettings(page);

    await redDays(page).fill("5");
    await save(page);
    await openFridge(page);

    await expect(stockDot(page, "A")).toHaveAttribute("data-expiry-status", "urgent");
  });

  test("買い物リストの日数の候補を足すと S07 に出る", async ({ page }) => {
    await openSettings(page);

    await addToList(page, t.field.shopping_presets, "10");
    await save(page);
    await openShoppingList(page);

    const label = (days: number) => tsh.picker.day_option.replace("{{count}}", String(days));
    await expect(page.locator('[data-testid^="shopping-days-"]')).toHaveText(
      [3, 4, 7, 10, 14, 28].map(label),
    );
  });

  test("サイドバーから開いたときの日数を変えると S07 の対象期間が変わる", async ({ page }) => {
    await openSettings(page);

    await chooseOption(page, t.field.shopping_default, dayChip(14));
    await save(page);
    await openShoppingList(page);

    const slash = (d: string) => d.replaceAll("-", "/");
    await expect(
      page.getByText(
        tsh.period
          .replace("{{start}}", slash(dateAfter(0)))
          .replace("{{end}}", slash(dateAfter(13))),
      ),
    ).toBeVisible();
    await expect(page.getByTestId("shopping-days-14")).toHaveAttribute("aria-pressed", "true");
  });

  test("作り置きの日数の候補を変えるとコピーの選択肢が変わる", async ({ page }) => {
    await page.goto("/");
    const id = await createRecipeWithIngredients(page, "カレー", []);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await openSettings(page);

    const label = t.field.leftover_presets;
    await removeButton(page, label, dayChip(3)).click();
    await removeButton(page, label, dayChip(4)).click();
    await addToList(page, label, "5");
    await save(page);

    await page.getByRole("link", { name: ja.nav.meal_plan }).click();
    await page.getByRole("button", { name: tm.cell_actions.meal_menu }).first().click();
    await page
      .getByRole("menuitem", { name: tm.meal_menu.copy_days.replace("{{days}}", "N") })
      .click();
    const days = (n: number) => tm.copy.days_option.replace("{{days}}", String(n));
    await expect(page.getByRole("menu").last().getByRole("menuitem")).toHaveText([
      days(2),
      days(5),
    ]);
  });

  test("量る単位に足した単位は量の前に表示される", async ({ page }) => {
    await openSettings(page);
    await addToList(page, t.field.measuring_units, "合");
    await save(page);

    await addStockItem(page, { name: "米", amount: 2, unit: "合" });
    await openFridge(page);

    const row = page.locator("tbody tr").filter({ has: page.getByText("米", { exact: true }) });
    await expect(row.locator("td").nth(1)).toHaveText("合 2");
  });
});

test.describe("表示言語の切替", () => {
  test("English を保存すると画面が英語になる", async ({ page }) => {
    await openSettings(page);

    await chooseOption(page, t.field.language, "English");
    await saveButton(page).click();

    await expect(page.getByRole("link", { name: en.nav.meal_plan })).toBeVisible();
  });

  test("保存した表示言語で起動する", async ({ page }) => {
    await openSettings(page);
    await chooseOption(page, t.field.language, "English");
    await saveButton(page).click();
    await expect(page.getByRole("link", { name: en.nav.meal_plan })).toBeVisible();

    await page.reload();

    await expect(page.getByRole("heading", { name: en.pages.meal_plan.title })).toBeVisible();
    await expect(page.getByRole("heading", { name: tm.title })).toHaveCount(0);
  });
});
