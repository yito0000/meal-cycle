import { join } from "node:path";
import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipeDetailed, openRecipeDetail, openRecipeList } from "./helpers.js";

const t = ja.pages.recipes;
const im = t.import;
const d = t.detail;

const FIXTURES = join(import.meta.dirname, "fixtures", "recipe-import");

// OS のファイルダイアログは Playwright の filechooser で受ける。隠し input の click() が発火させる (add-recipe-bulk-import D2)。
async function chooseFrom(page: Page, menuItem: string, paths: string | string[]) {
  await openRecipeList(page);
  const chooser = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: im.button, exact: true }).click();
  await page.getByRole("menuitem", { name: menuItem }).click();
  await (await chooser).setFiles(paths);
}

function chooseFiles(page: Page, names: string[]) {
  return chooseFrom(
    page,
    im.choose_files,
    names.map((n) => join(FIXTURES, n)),
  );
}

function chooseFolder(page: Page, dir: string) {
  return chooseFrom(page, im.choose_folder, join(FIXTURES, dir));
}

function dialog(page: Page) {
  return page.getByRole("dialog");
}

function rows(page: Page) {
  return dialog(page).getByTestId("recipe-import-row");
}

function row(page: Page, fileName: string) {
  return rows(page).filter({ hasText: fileName });
}

function rowCheckbox(page: Page, fileName: string) {
  return dialog(page).getByRole("checkbox", {
    name: im.select_row.replace("{{name}}", fileName),
  });
}

function submitButton(page: Page, count?: number) {
  const name =
    count === undefined ? im.submit : im.submit_count.replace("{{count}}", String(count));
  return dialog(page).getByRole("button", { name, exact: true });
}

function issueText(code: keyof typeof im.issue, arg = "") {
  return im.issue[code].replace("{{arg}}", arg);
}

// S02 の一覧の名前列。ダイアログを開いている間は main が aria-hidden になるので、role ではなく CSS で引く。
function listNameCells(page: Page) {
  return page.locator("main table tbody tr td:first-child");
}

async function importSingle(page: Page, name: string) {
  await chooseFiles(page, [name]);
  await submitButton(page).click();
  await expect(dialog(page)).toHaveCount(0);
}

function ingredientItems(page: Page) {
  return page
    .getByRole("main")
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: d.section.ingredients, exact: true }) })
    .getByRole("listitem");
}

test.describe("S02 からのレシピのインポート", () => {
  test("選んだファイルがインポートダイアログに表示される", async ({ page }) => {
    await chooseFiles(page, ["nikujaga.md", "curry.md"]);

    await expect(rows(page)).toHaveCount(2);
    await expect(row(page, "nikujaga.md")).toBeVisible();
    await expect(row(page, "curry.md")).toBeVisible();
    await expect(page.locator("main").getByText(t.empty.no_recipes)).toBeVisible();
  });

  test("フォルダを選ぶと中の Markdown ファイルだけが読み込まれる", async ({ page }) => {
    await chooseFolder(page, "folder");

    await expect(rows(page)).toHaveCount(2);
    await expect(row(page, "a.md")).toBeVisible();
    await expect(row(page, "b.md")).toBeVisible();
    await expect(dialog(page)).not.toContainText("photo.png");
  });

  test("Markdown ファイルが 1 つも無いと知らせる", async ({ page }) => {
    await chooseFolder(page, "empty-folder");

    await expect(page.getByText(im.no_files)).toBeVisible();
    await expect(dialog(page)).toHaveCount(0);
  });
});

test.describe("インポートするファイルの形式", () => {
  test("frontmatter と本文の節がレシピの各欄になる", async ({ page }) => {
    await importSingle(page, "nikujaga.md");
    await openRecipeDetail(page, "肉じゃが");

    const main = page.getByRole("main");
    await expect(main).toContainText("和食");
    await expect(main).toContainText(d.servings.replace("{{count}}", "3"));
    await expect(ingredientItems(page).filter({ hasText: "じゃがいも" })).toContainText("3 個");
    await expect(main).toContainText("じゃがいもを切る");
    await expect(main).toContainText("https://example.com/recipes/nikujaga");
    await expect(main).not.toContainText("## 手順");
  });

  test("人数を省略すると 2 人分になる", async ({ page }) => {
    await importSingle(page, "no-servings.md");
    await openRecipeDetail(page, "豚汁");

    await expect(page.getByRole("main")).toContainText(d.servings.replace("{{count}}", "2"));
  });

  test("名前の無いファイルはエラーになる", async ({ page }) => {
    await chooseFiles(page, ["no-name.md"]);

    await expect(dialog(page)).toContainText(issueText("name_empty"));
  });

  test("frontmatter の無いファイルはエラーになる", async ({ page }) => {
    await chooseFiles(page, ["no-frontmatter.md"]);

    await expect(dialog(page)).toContainText(issueText("no_frontmatter"));
  });
});

test.describe("材料の量の表記ゆれの吸収", () => {
  test("分数と帯分数は小数になる", async ({ page }) => {
    await importSingle(page, "amounts.md");
    await openRecipeDetail(page, "表記ゆれ");

    await expect(ingredientItems(page).filter({ hasText: "玉ねぎ" })).toContainText("0.5 個");
    await expect(ingredientItems(page).filter({ hasText: "牛乳" })).toContainText("カップ 1.5");
  });

  test("単位が付いた量は量と単位に分かれる", async ({ page }) => {
    await importSingle(page, "amounts.md");
    await openRecipeDetail(page, "表記ゆれ");

    await expect(ingredientItems(page).filter({ hasText: "醤油" })).toContainText("大さじ 2");
    await expect(ingredientItems(page).filter({ hasText: "卵" })).toContainText("3 個");
  });

  test("量を持たない言葉は量 0 になる", async ({ page }) => {
    await importSingle(page, "amounts.md");
    await openRecipeDetail(page, "表記ゆれ");

    const salt = ingredientItems(page).filter({ hasText: "塩" });
    await expect(salt).toContainText("少々");
    await expect(salt).not.toContainText("0");
  });

  test("読み取れない量はエラーになる", async ({ page }) => {
    await chooseFiles(page, ["bad-amount.md"]);

    await expect(dialog(page)).toContainText(issueText("amount_invalid", "にんじん"));
  });
});

test.describe("1 件のインポートのプレビュー", () => {
  test("1 件ならプレビューを見てから取り込める", async ({ page }) => {
    await chooseFiles(page, ["nikujaga.md"]);

    await expect(dialog(page)).toContainText("じゃがいも");
    await expect(dialog(page)).toContainText("じゃがいもを切る");
    await expect(dialog(page)).toContainText("https://example.com/recipes/nikujaga");

    await submitButton(page).click();
    await expect(dialog(page)).toHaveCount(0);
    await expect(listNameCells(page)).toHaveText(["肉じゃが"]);
  });

  test("キャンセルすると何も登録されない", async ({ page }) => {
    await chooseFiles(page, ["nikujaga.md"]);
    await expect(dialog(page)).toContainText("じゃがいもを切る");

    await dialog(page).getByRole("button", { name: im.cancel, exact: true }).click();

    await expect(dialog(page)).toHaveCount(0);
    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
  });

  test("1 件がエラーなら取り込めない", async ({ page }) => {
    await chooseFiles(page, ["no-name.md"]);

    await expect(dialog(page)).toContainText(issueText("name_empty"));
    await expect(submitButton(page)).toBeDisabled();
  });
});

test.describe("複数件のインポートの選択", () => {
  test("エラーの無いファイルは既定で選ばれている", async ({ page }) => {
    await chooseFiles(page, ["nikujaga.md", "curry.md", "no-name.md"]);

    await expect(rows(page)).toHaveCount(3);
    await expect(rowCheckbox(page, "nikujaga.md")).toBeChecked();
    await expect(rowCheckbox(page, "curry.md")).toBeChecked();
    await expect(row(page, "no-name.md")).toContainText(issueText("name_empty"));
    await expect(rowCheckbox(page, "no-name.md")).toBeDisabled();
    await expect(submitButton(page, 2)).toBeEnabled();
  });

  test("チェックを外したファイルは取り込まれない", async ({ page }) => {
    await chooseFiles(page, ["nikujaga.md", "curry.md"]);
    await expect(rowCheckbox(page, "nikujaga.md")).toBeChecked();
    await expect(rowCheckbox(page, "curry.md")).toBeChecked();

    await rowCheckbox(page, "curry.md").uncheck();
    await submitButton(page, 1).click();

    await expect(dialog(page)).toHaveCount(0);
    await expect(listNameCells(page)).toHaveText(["肉じゃが"]);
  });

  test("すべて解除すると取り込めない", async ({ page }) => {
    await chooseFiles(page, ["nikujaga.md", "curry.md"]);
    await expect(rows(page)).toHaveCount(2);

    await dialog(page).getByRole("button", { name: im.deselect_all }).click();

    await expect(dialog(page).getByRole("checkbox", { checked: true })).toHaveCount(0);
    await expect(submitButton(page, 0)).toBeDisabled();
  });
});

test.describe("同名のレシピの警告", () => {
  test("登録済みのレシピと同じ名前なら警告が出る", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが" });
    await chooseFiles(page, ["nikujaga.md", "curry.md"]);

    await expect(row(page, "nikujaga.md")).toContainText(im.status.duplicate);
    await expect(rowCheckbox(page, "nikujaga.md")).not.toBeChecked();
    await expect(rowCheckbox(page, "curry.md")).toBeChecked();
  });

  test("同時に読み込んだファイルどうしで名前が同じなら警告が出る", async ({ page }) => {
    await chooseFiles(page, ["curry.md", "curry-2.md"]);

    await expect(row(page, "curry.md")).toContainText(im.status.duplicate);
    await expect(row(page, "curry-2.md")).toContainText(im.status.duplicate);
  });

  test("1 件でも同名なら警告を添えてプレビューする", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが" });
    await chooseFiles(page, ["nikujaga.md"]);

    await expect(dialog(page)).toContainText("じゃがいもを切る");
    await expect(dialog(page)).toContainText(im.duplicate_warning);
    await expect(submitButton(page)).toBeEnabled();
  });

  test("警告を承知で取り込むと別のレシピとして追加される", async ({ page }) => {
    const existingId = await createRecipeDetailed(page, { name: "肉じゃが", notes: "元のメモ" });
    await chooseFiles(page, ["nikujaga.md", "curry.md"]);
    await expect(rowCheckbox(page, "nikujaga.md")).not.toBeChecked();

    await rowCheckbox(page, "nikujaga.md").check();
    await submitButton(page, 2).click();

    await expect(dialog(page)).toHaveCount(0);
    await expect(listNameCells(page).filter({ hasText: "肉じゃが" })).toHaveCount(2);
    const res = await page.request.post("/api/RecipeService/GetByID", { data: [existingId] });
    const existing = await res.json();
    expect(existing.notes).toBe("元のメモ");
    expect(existing.ingredients.map((i: { name: string }) => i.name)).toEqual(["玉ねぎ"]);
  });
});

test.describe("インポートの確定", () => {
  test("取り込んだ件数が通知される", async ({ page }) => {
    await chooseFiles(page, ["nikujaga.md", "curry.md", "no-servings.md"]);
    await expect(rows(page)).toHaveCount(3);

    await submitButton(page, 3).click();

    await expect(dialog(page)).toHaveCount(0);
    await expect(page.getByText(im.done.replace("{{count}}", "3"))).toBeVisible();
    await expect(listNameCells(page)).toHaveCount(3);
  });
});
