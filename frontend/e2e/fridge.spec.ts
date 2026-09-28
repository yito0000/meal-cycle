import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { addStockItem, dateAfter, openFridge } from "./helpers.js";

const t = ja.pages.fridge;

function nameCells(page: Page) {
  return page.locator("tbody tr td:first-child");
}

function row(page: Page, name: string) {
  return page.locator("tbody tr").filter({ has: page.getByText(name, { exact: true }) });
}

function addRow(page: Page) {
  return page.getByRole("group", { name: t.add.label });
}

async function fillAddRow(
  page: Page,
  v: { name?: string; amount?: string; unit?: string; expiresAt?: string },
) {
  const g = addRow(page);
  if (v.name !== undefined) await g.getByRole("combobox", { name: t.field.name }).fill(v.name);
  if (v.amount !== undefined) await g.getByLabel(t.field.amount, { exact: true }).fill(v.amount);
  if (v.unit !== undefined) await g.getByLabel(t.field.unit, { exact: true }).fill(v.unit);
  if (v.expiresAt !== undefined) await g.getByLabel(t.field.expires_at).fill(v.expiresAt);
}

async function chooseSort(page: Page, label: string) {
  await page.getByRole("combobox", { name: t.sort.label }).click();
  await page.getByRole("option", { name: label }).click();
}

function dotOf(page: Page, name: string) {
  return row(page, name).locator("[data-expiry-status]");
}

test.describe("在庫一覧の表示", () => {
  test("量る単位の在庫は単位を量の前に表示する", async ({ page }) => {
    await addStockItem(page, { name: "味噌", amount: 3, unit: "大さじ" });
    await addStockItem(page, { name: "豚肉", amount: 300, unit: "g" });
    await openFridge(page);

    await expect(row(page, "味噌").locator("td").nth(1)).toHaveText("大さじ 3");
    await expect(row(page, "豚肉").locator("td").nth(1)).toHaveText("300 g");
  });
});

test.describe("賞味期限の状態表示", () => {
  test("当日期限は赤で表示される", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "ヨーグルト", amount: 1, unit: "個", expiresInDays: 0 });

    await openFridge(page);

    await expect(dotOf(page, "ヨーグルト")).toHaveAttribute("data-expiry-status", "urgent");
    await expect(row(page, "ヨーグルト")).toContainText(t.days_left.today);
    await expect(row(page, "ヨーグルト")).not.toHaveAttribute("data-expired");
  });

  test("期限切れは色ドットではなく期限切れバッジで表示される", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "豆腐", amount: 1, unit: "丁", expiresInDays: -2 });
    await addStockItem(page, { name: "牛乳", amount: 1, unit: "本", expiresInDays: 2 });

    await openFridge(page);

    const tofu = row(page, "豆腐");
    await expect(tofu.getByRole("img")).toHaveCount(0);
    await expect(tofu.getByText(t.expiry_status.expired, { exact: true })).toBeVisible();
    await expect(dotOf(page, "豆腐")).toHaveAttribute("data-expiry-status", "expired");
    await expect(tofu).toHaveAttribute("data-expired", "true");
    // 背景の具体色ではなく「期限内の行と区別できる」ことを検証する。
    const bg = (name: string) =>
      row(page, name).evaluate((el) => getComputedStyle(el).backgroundColor);
    expect(await bg("豆腐")).not.toBe(await bg("牛乳"));
    await expect(tofu).toContainText(t.days_left.overdue.replace("{{count}}", "2"));
    await expect(row(page, "牛乳")).not.toHaveAttribute("data-expired");
  });
});

test.describe("在庫の並び替え", () => {
  test("デフォルトで賞味期限の近い順に並ぶ", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "牛乳", amount: 1, unit: "本", expiresInDays: 2 });
    await addStockItem(page, { name: "卵", amount: 6, unit: "個", expiresInDays: 10 });
    await addStockItem(page, { name: "豚肉", amount: 300, unit: "g", expiresInDays: 5 });
    await addStockItem(page, { name: "米", amount: 5, unit: "kg" });

    await openFridge(page);

    await expect(nameCells(page)).toHaveText(["牛乳", "豚肉", "卵", "米"]);
    await expect(dotOf(page, "牛乳")).toHaveAttribute("data-expiry-status", "urgent");
    await expect(dotOf(page, "豚肉")).toHaveAttribute("data-expiry-status", "soon");
    await expect(dotOf(page, "卵")).toHaveAttribute("data-expiry-status", "fresh");
    await expect(dotOf(page, "米")).toHaveCount(0);
    for (const column of Object.values(t.column)) {
      await expect(page.getByRole("columnheader", { name: column })).toBeVisible();
    }
  });

  test("食材名でソートする", async ({ page }) => {
    await page.goto("/");
    // 照合順が環境で揺れないよう、読みの順がそのまま並ぶかな表記で揃える。
    await addStockItem(page, { name: "にんじん", amount: 1, unit: "本", expiresInDays: 1 });
    await addStockItem(page, { name: "きゃべつ", amount: 1, unit: "玉", expiresInDays: 9 });
    await addStockItem(page, { name: "たまご", amount: 6, unit: "個", expiresInDays: 5 });

    await openFridge(page);
    await chooseSort(page, t.sort.name_asc);

    await expect(nameCells(page)).toHaveText(["きゃべつ", "たまご", "にんじん"]);
  });

  test("追加日新しい順でソートする", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "一番目", amount: 1, unit: "", expiresInDays: 1 });
    await addStockItem(page, { name: "二番目", amount: 1, unit: "", expiresInDays: 3 });
    await addStockItem(page, { name: "三番目", amount: 1, unit: "", expiresInDays: 2 });

    await openFridge(page);
    await chooseSort(page, t.sort.added_desc);

    await expect(nameCells(page)).toHaveText(["三番目", "二番目", "一番目"]);
  });
});

test.describe("食材名検索", () => {
  test("検索語が食材名に部分一致する", async ({ page }) => {
    await page.goto("/");
    for (const name of ["豚ひき肉", "豚バラ肉", "玉ねぎ"]) {
      await addStockItem(page, { name, amount: 1, unit: "" });
    }

    await openFridge(page);
    await page.getByRole("searchbox", { name: t.search.placeholder }).fill("豚");

    // 並び順は照合順に依存するので、表示される集合だけを比べる。
    await expect(nameCells(page)).toHaveCount(2);
    expect((await nameCells(page).allTextContents()).sort()).toEqual(
      ["豚ひき肉", "豚バラ肉"].sort(),
    );
  });
});

test.describe("インライン追加行", () => {
  test("賞味期限つきで在庫を追加する", async ({ page }) => {
    await openFridge(page);

    await fillAddRow(page, { name: "玉ねぎ", amount: "3", unit: "個", expiresAt: dateAfter(7) });
    await addRow(page).getByRole("button", { name: t.add.submit }).click();

    await expect(row(page, "玉ねぎ")).toContainText("3 個");
    await expect(dotOf(page, "玉ねぎ")).toHaveAttribute("data-expiry-status", "soon");
    const nameInput = addRow(page).getByRole("combobox", { name: t.field.name });
    await expect(nameInput).toHaveValue("");
    await expect(addRow(page).getByLabel(t.field.amount, { exact: true })).toHaveValue("");
    await expect(addRow(page).getByLabel(t.field.unit, { exact: true })).toHaveValue("");
    await expect(nameInput).toBeFocused();
  });

  test("賞味期限を省略して追加する", async ({ page }) => {
    await openFridge(page);

    await fillAddRow(page, { name: "玉ねぎ", amount: "1", unit: "個" });
    // 量や単位の欄での Enter でも追加できる。
    await addRow(page).getByLabel(t.field.unit, { exact: true }).press("Enter");

    await expect(row(page, "玉ねぎ")).toContainText("1 個");
    await expect(dotOf(page, "玉ねぎ")).toHaveCount(0);
  });
});

test.describe("食材名の候補補完", () => {
  test("候補を選ぶと単位が埋まる", async ({ page }) => {
    await page.goto("/");
    const res = await page.request.post("/api/RecipeService/Create", {
      data: [
        {
          id: "",
          name: "きのこパスタ",
          servings: 1,
          tags: [],
          ingredients: [{ name: "しめじ", amount: 1, unit: "パック" }],
          ingredientCount: 1,
          steps: "",
          notes: "",
          createdAt: "",
          updatedAt: "",
        },
      ],
    });
    expect(res.ok()).toBe(true);

    await openFridge(page);
    const nameInput = addRow(page).getByRole("combobox", { name: t.field.name });
    await nameInput.fill("しめ");
    await expect(page.getByRole("option", { name: /しめじ/ })).toBeVisible();

    await nameInput.press("ArrowDown");
    await nameInput.press("Enter");

    await expect(nameInput).toHaveValue("しめじ");
    await expect(addRow(page).getByLabel(t.field.unit, { exact: true })).toHaveValue("パック");
    // 候補確定の Enter では追加されない。
    await expect(page.locator("tbody tr")).toHaveCount(0);
  });

  test("候補にない名前で追加する", async ({ page }) => {
    await openFridge(page);

    await fillAddRow(page, { name: "ズッキーニ", amount: "1", unit: "本" });
    await addRow(page).getByRole("button", { name: t.add.submit }).click();
    await expect(row(page, "ズッキーニ")).toContainText("1 本");

    await addRow(page).getByRole("combobox", { name: t.field.name }).fill("ズッキ");
    await expect(page.getByRole("option", { name: /ズッキーニ/ })).toBeVisible();
  });
});

test.describe("在庫の入力検証", () => {
  test("食材名が空だと追加できない", async ({ page }) => {
    await openFridge(page);

    await fillAddRow(page, { amount: "1" });
    await addRow(page).getByRole("button", { name: t.add.submit }).click();

    await expect(addRow(page).getByText(t.errors.name_empty)).toBeVisible();
    await expect(page.getByText(t.empty.no_items)).toBeVisible();
  });
});

test.describe("行内編集", () => {
  test("行内編集で在庫量を修正する", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await openFridge(page);

    await row(page, "玉ねぎ").getByRole("button", { name: t.row.edit }).click();
    const editing = page.locator("tbody tr[data-editing]");
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await editing.getByLabel(t.field.amount, { exact: true }).fill("2");
    await editing.getByRole("button", { name: t.row.save }).click();

    await expect(editing).toHaveCount(0);
    await expect(row(page, "玉ねぎ")).toContainText("2 個");
  });

  test("行内編集をキャンセルする", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await openFridge(page);

    await row(page, "玉ねぎ").getByRole("button", { name: t.row.edit }).click();
    const editing = page.locator("tbody tr[data-editing]");
    await editing.getByLabel(t.field.amount, { exact: true }).fill("2");
    await editing.getByRole("button", { name: t.row.cancel }).click();

    await expect(editing).toHaveCount(0);
    await expect(row(page, "玉ねぎ")).toContainText("3 個");
  });

  test("編集中は他の行を操作できない", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await addStockItem(page, { name: "牛乳", amount: 1, unit: "本" });
    await openFridge(page);

    await row(page, "玉ねぎ").getByRole("button", { name: t.row.edit }).click();

    await expect(row(page, "牛乳").getByRole("button", { name: t.row.edit })).toBeDisabled();
    await expect(row(page, "牛乳").getByRole("button", { name: t.row.delete })).toBeDisabled();
  });
});

test.describe("在庫の削除と元に戻す", () => {
  test("削除は即実行され、トーストの「元に戻す」で復元できる", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個", expiresInDays: 7 });
    await openFridge(page);

    await row(page, "玉ねぎ").getByRole("button", { name: t.row.delete }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(row(page, "玉ねぎ")).toHaveCount(0);
    await expect(page.getByText(t.toast.deleted.replace("{{name}}", "玉ねぎ"))).toBeVisible();

    await page.getByRole("button", { name: t.toast.undo }).click();

    await expect(row(page, "玉ねぎ")).toContainText("3 個");
    await expect(dotOf(page, "玉ねぎ")).toHaveAttribute("data-expiry-status", "soon");
  });
});
