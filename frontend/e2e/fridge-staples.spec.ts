import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { addStaple, addStockItem, openFridge, stapleRow } from "./helpers.js";

const t = ja.pages.fridge;
const ts = t.staples;

function panel(page: Page) {
  return page.getByRole("region", { name: ts.title });
}

function stapleInput(page: Page) {
  return panel(page).getByRole("combobox", { name: ts.name_placeholder });
}

async function stapleOrder(page: Page): Promise<string[]> {
  return page
    .getByTestId("staple-row")
    .evaluateAll((rows) => rows.map((r) => r.querySelector("span")?.textContent ?? ""));
}

function stateButton(page: Page, name: string, state: "enough" | "low" | "out") {
  return stapleRow(page, name).getByRole("button", { name: ts.state[state], exact: true });
}

async function addFromPanel(page: Page, name: string) {
  await stapleInput(page).fill(name);
  await panel(page).getByRole("button", { name: ts.add, exact: true }).click();
}

test.describe("常備品の永続化", () => {
  test("常備品は在庫の一覧に出ない", async ({ page }) => {
    await openFridge(page);
    await addFromPanel(page, "食パン");

    await expect(stapleRow(page, "食パン")).toBeVisible();
    await expect(page.getByText(t.empty.no_items)).toBeVisible();
    await expect(page.locator("tbody tr")).toHaveCount(0);
  });
});

test.describe("常備品の欄の表示", () => {
  test("常備品が食材名の順に並ぶ", async ({ page }) => {
    await addStaple(page, "ヨーグルト");
    await addStaple(page, "チーズ");
    await addStaple(page, "バター");
    await openFridge(page);

    await expect(page.getByTestId("staple-row")).toHaveCount(3);
    expect(await stapleOrder(page)).toEqual(["チーズ", "バター", "ヨーグルト"]);
    for (const name of ["チーズ", "バター", "ヨーグルト"]) {
      await expect(
        stapleRow(page, name).getByRole("group", {
          name: ts.state_group.replace("{{name}}", name),
        }),
      ).toBeVisible();
      await expect(stapleRow(page, name).getByRole("button", { name: ts.delete })).toBeVisible();
    }
  });

  test("常備品が無い場合はその旨を示す", async ({ page }) => {
    await openFridge(page);

    await expect(panel(page).getByText(ts.empty)).toBeVisible();
    await expect(stapleInput(page)).toBeVisible();
  });

  test("在庫の検索は常備品の一覧を絞り込まない", async ({ page }) => {
    await addStockItem(page, { name: "豚ひき肉", amount: 200, unit: "g" });
    await addStaple(page, "牛乳");
    await openFridge(page);

    await page.getByPlaceholder(t.search.placeholder).fill("豚");

    await expect(page.locator("tbody tr")).toHaveCount(1);
    await expect(page.locator("tbody tr").first()).toContainText("豚ひき肉");
    await expect(stapleRow(page, "牛乳")).toBeVisible();
  });
});

test.describe("常備品の追加", () => {
  test("常備品を追加すると状態はあるになる", async ({ page }) => {
    await openFridge(page);
    await addFromPanel(page, "牛乳");

    await expect(stapleRow(page, "牛乳")).toHaveAttribute("data-state", "enough");
    await expect(stateButton(page, "牛乳", "enough")).toHaveAttribute("aria-pressed", "true");
    await expect(stapleInput(page)).toHaveValue("");
    await expect(stapleInput(page)).toBeFocused();
  });

  test("Enter で続けて追加できる", async ({ page }) => {
    await openFridge(page);
    await stapleInput(page).fill("牛乳");
    await stapleInput(page).press("Enter");
    await expect(stapleRow(page, "牛乳")).toBeVisible();
    await stapleInput(page).fill("食パン");
    await stapleInput(page).press("Enter");

    await expect(stapleRow(page, "牛乳")).toBeVisible();
    await expect(stapleRow(page, "食パン")).toBeVisible();
  });
});

test.describe("常備品の入力検証", () => {
  test("食材名が空だと常備品を追加できない", async ({ page }) => {
    await openFridge(page);
    await addFromPanel(page, "   ");

    await expect(panel(page).getByRole("alert")).toHaveText(ts.errors.name_empty);
    await expect(page.getByTestId("staple-row")).toHaveCount(0);
  });

  test("同じ常備品は 2 つ登録できない", async ({ page }) => {
    await addStaple(page, "ミルク");
    await openFridge(page);
    await addFromPanel(page, "みるく");

    await expect(panel(page).getByRole("alert")).toHaveText(ts.errors.duplicate);
    await expect(page.getByTestId("staple-row")).toHaveCount(1);
  });

  test("同名の在庫があっても常備品を追加できる", async ({ page }) => {
    await addStockItem(page, { name: "牛乳", amount: 1, unit: "本" });
    await openFridge(page);
    await addFromPanel(page, "牛乳");

    await expect(stapleRow(page, "牛乳")).toBeVisible();
    await expect(page.locator("tbody tr").filter({ hasText: "牛乳" })).toHaveCount(1);
  });
});

test.describe("常備品の状態の切替", () => {
  test("状態を押すだけで切り替わる", async ({ page }) => {
    await addStaple(page, "牛乳");
    await openFridge(page);

    await stateButton(page, "牛乳", "out").click();
    await expect(stateButton(page, "牛乳", "out")).toHaveAttribute("aria-pressed", "true");
    await expect(stapleRow(page, "牛乳")).toHaveAttribute("data-state", "out");

    await page.getByRole("link", { name: ja.nav.shopping }).click();
    await openFridge(page);
    await expect(stapleRow(page, "牛乳")).toHaveAttribute("data-state", "out");
  });

  test("状態を切り替えても並び順は変わらない", async ({ page }) => {
    await addStaple(page, "チーズ");
    await addStaple(page, "バター");
    await addStaple(page, "ヨーグルト");
    await openFridge(page);

    await stateButton(page, "バター", "low").click();
    await expect(stapleRow(page, "バター")).toHaveAttribute("data-state", "low");
    expect(await stapleOrder(page)).toEqual(["チーズ", "バター", "ヨーグルト"]);
  });
});

test.describe("常備品の削除と元に戻す", () => {
  test("常備品を削除してトーストの元に戻すで復元する", async ({ page }) => {
    await addStaple(page, "牛乳", "low");
    await openFridge(page);

    await stapleRow(page, "牛乳").getByRole("button", { name: ts.delete }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(stapleRow(page, "牛乳")).toHaveCount(0);
    await expect(page.getByText(ts.toast.deleted.replace("{{name}}", "牛乳"))).toBeVisible();

    await page.getByRole("button", { name: t.toast.undo }).click();
    await expect(stapleRow(page, "牛乳")).toHaveAttribute("data-state", "low");
  });
});
