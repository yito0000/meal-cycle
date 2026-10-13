import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import {
  addStockItem,
  assignRecipe,
  createRecipe,
  createRecipeWithIngredients,
  dateAfter,
  detailBreadcrumb,
  openItemMenu,
  openRecipeDetail,
  openRecipeList,
} from "./helpers.js";

const t = ja.pages.recipes;
const e = t.edit;

// S02 から S04 を開く。
async function openNewRecipeForm(page: Page) {
  await openRecipeList(page);
  await page.getByRole("button", { name: t.new_button }).click();
  await expect(page.getByRole("heading", { name: e.title_new })).toBeVisible();
}

// 材料 1 行を足して埋める。名前欄はレシピ名と同じラベルなので行内に絞る。
async function fillFirstIngredient(page: Page, name: string, amount: string, unit: string) {
  await page.getByRole("button", { name: e.add_ingredient }).click();
  await ingredientNameInput(page).fill(name);
  await page.getByRole("spinbutton", { name: e.label.amount }).fill(amount);
  await ingredientUnitInput(page).fill(unit);
}

// 材料行の名前欄は候補補完つきの combobox、単位欄は datalist つきで combobox になる。
function ingredientNameInput(page: Page) {
  return page.getByRole("combobox", { name: e.label.name, exact: true }).last();
}

function ingredientUnitInput(page: Page) {
  return page.getByRole("combobox", { name: e.label.unit, exact: true }).last();
}

// S03 詳細画面から「編集」で S04 に入る。
async function openEditFromDetail(page: Page, name: string) {
  await openRecipeDetail(page, name);
  await page.getByRole("main").getByRole("button", { name: t.detail.edit }).click();
  await expect(page.locator("#recipe-name")).toHaveValue(name);
}

// dirty な S04 からサイドバーで S02 に戻る。未保存の確認は破棄で抜ける。
async function openRecipeListDiscarding(page: Page) {
  await page.getByRole("link", { name: t.title }).click();
  await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();
  await expect(page.getByRole("heading", { name: t.title, exact: true })).toBeVisible();
}

const servingsText = (n: number) => t.detail.servings.replace("{{count}}", String(n));

// S03 詳細画面の見出し。S04 の見出し「レシピを編集: <名前>」と区別するため完全一致で引く。
function detailHeading(page: Page, name: string) {
  return page.getByRole("main").getByRole("heading", { name, exact: true });
}

test.describe("レシピ一覧の初期表示", () => {
  test("1 レシピに複数タグがある場合はまとめて表示される", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.locator("#recipe-name").fill("豚汁");
    await fillFirstIngredient(page, "大根", "100", "g");
    await page.getByRole("textbox", { name: e.label.tags }).fill("和食");
    await page.getByRole("button", { name: e.add_tag }).click();
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("row").locator("td:nth-child(2)")).toHaveText(["和食"]);
  });
});

test.describe("レシピの新規登録", () => {
  test("材料に自由テキストを入れてレシピを登録する", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.locator("#recipe-name").fill("肉じゃが");
    await page.locator("#recipe-servings").fill("2");
    await fillFirstIngredient(page, "じゃがいも", "2", "個");
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByRole("row").locator("td:first-child")).toHaveText(["肉じゃが"]);
  });

  test("名前が空のままでは保存できない", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByText(e.validation.name_required)).toBeVisible();
    await expect(page.getByRole("heading", { name: e.title_new })).toBeVisible();
  });
});

test.describe("未保存変更のキャンセル確認", () => {
  test("dirty 状態でキャンセルを押すと確認される", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.locator("#recipe-name").fill("肉じゃが");
    await page.getByRole("button", { name: e.cancel }).click();

    await expect(page.getByRole("dialog")).toContainText(e.unsaved_confirm.title);

    await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
  });
  test("編集モードで変更を破棄すると S03 に戻る", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await page.getByRole("button", { name: e.cancel }).click();
    await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();

    await expect(detailHeading(page, "肉じゃが")).toBeVisible();
    await expect(page.getByRole("main")).toContainText(servingsText(2));
  });

  test("dirty 状態でサイドバーを押すと確認される", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.locator("#recipe-name").fill("肉じゃが");
    await page.getByRole("link", { name: ja.nav.meal_plan }).click();

    await expect(page.getByRole("dialog")).toContainText(e.unsaved_confirm.title);
    // ダイアログが開いている間は背面が aria-hidden になるので、role ではなく id で S04 を確かめる。
    await expect(page.locator("#recipe-name")).toHaveValue("肉じゃが");
  });

  test("サイドバーからの確認で破棄を選ぶと押した画面へ移る", async ({ page }) => {
    await openNewRecipeForm(page);
    await page.locator("#recipe-name").fill("肉じゃが");
    await page.getByRole("link", { name: ja.nav.meal_plan }).click();

    await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();

    await expect(page.getByRole("heading", { name: ja.pages.meal_plan.title })).toBeVisible();
    await openRecipeList(page);
    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
  });

  test("サイドバーからの確認で編集を続けるを選ぶと入力が残る", async ({ page }) => {
    await openNewRecipeForm(page);
    await page.locator("#recipe-name").fill("肉じゃが");
    await page.getByRole("link", { name: ja.nav.meal_plan }).click();

    await page.getByRole("button", { name: e.unsaved_confirm.keep_editing }).click();

    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.locator("#recipe-name")).toHaveValue("肉じゃが");
  });

  test("dirty でない状態のサイドバーは確認をスキップする", async ({ page }) => {
    await openNewRecipeForm(page);

    await page.getByRole("link", { name: ja.nav.fridge }).click();

    await expect(page.getByRole("heading", { name: ja.pages.fridge.title })).toBeVisible();
    await expect(page.getByRole("dialog")).toHaveCount(0);
  });

  test("保存に成功したあとの移動では確認されない", async ({ page }) => {
    await openNewRecipeForm(page);
    await page.locator("#recipe-name").fill("肉じゃが");

    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(page.getByRole("row").locator("td:first-child")).toHaveText(["肉じゃが"]);
  });
});

test.describe("数値欄の入力途中の保持", () => {
  // fill は欄を丸ごと置き換えるので、消した直後に別の値が入る不具合を見逃す。1 文字ずつ操作する。
  test("人数を消して打ち直せる", async ({ page }) => {
    await openNewRecipeForm(page);
    const servings = page.locator("#recipe-servings");
    await expect(servings).toHaveValue("1");

    await servings.focus();
    await servings.press("End");
    await servings.press("Backspace");
    await expect(servings).toHaveValue("");
    await servings.pressSequentially("4");

    await expect(servings).toHaveValue("4");
  });

  test("人数が空のままでは保存できない", async ({ page }) => {
    await openNewRecipeForm(page);
    await page.locator("#recipe-name").fill("x");

    await page.locator("#recipe-servings").fill("");
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByText(e.save_error.servings_out_of_range)).toBeVisible();
    await expect(page.getByRole("heading", { name: e.title_new })).toBeVisible();
    await openRecipeListDiscarding(page);
    await expect(page.getByText(t.empty.no_recipes)).toBeVisible();
  });

  test("量を消して打ち直せる", async ({ page }) => {
    await openNewRecipeForm(page);
    await page.getByRole("button", { name: e.add_ingredient }).click();
    const amount = page.getByRole("spinbutton", { name: e.label.amount });
    await amount.pressSequentially("200");
    await expect(amount).toHaveValue("200");

    for (let i = 0; i < 3; i++) await amount.press("Backspace");
    await expect(amount).toHaveValue("");
    await amount.pressSequentially("50");

    await expect(amount).toHaveValue("50");
  });

  test("量が空の材料は量 0 として保存される", async ({ page }) => {
    await openNewRecipeForm(page);
    await page.locator("#recipe-name").fill("野菜炒め");
    await page.getByRole("button", { name: e.add_ingredient }).click();
    await ingredientNameInput(page).fill("塩");
    await ingredientUnitInput(page).fill("少々");
    await expect(page.getByRole("spinbutton", { name: e.label.amount })).toHaveValue("");

    await page.getByRole("button", { name: e.save }).click();
    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await openRecipeDetail(page, "野菜炒め");

    const ingredients = page
      .getByRole("main")
      .locator("section")
      .filter({
        has: page.getByRole("heading", { name: t.detail.section.ingredients, exact: true }),
      });
    await expect(ingredients).toContainText("塩");
    await expect(ingredients).toContainText("少々");
    await expect(ingredients).not.toContainText("0");
  });
});

test.describe("S03 から S04 編集画面への遷移", () => {
  test("編集を押すと既存値がロードされた S04 に遷移する", async ({ page }) => {
    await createRecipe(page, "肉じゃが");

    await openEditFromDetail(page, "肉じゃが");

    await expect(
      page.getByRole("heading", { name: e.title_edit.replace("{{name}}", "肉じゃが") }),
    ).toBeVisible();
    await expect(detailHeading(page, "肉じゃが")).toHaveCount(0);
  });

  test("保存すると S02 に戻り行が強調される", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await page.getByRole("button", { name: e.save }).click();

    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();
    await expect(detailHeading(page, "肉じゃが")).toHaveCount(0);
    await expect(page.getByRole("row").filter({ hasText: "肉じゃが" })).toHaveClass(/bg-accent/);
  });

  test("キャンセルすると S03 に戻る", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.getByRole("button", { name: e.cancel }).click();

    await expect(detailHeading(page, "肉じゃが")).toBeVisible();
    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.recipes,
      "肉じゃが",
    ]);
  });

  test("保存した内容が S03 に反映される", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await page.getByRole("button", { name: e.save }).click();
    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();

    await page.getByRole("cell", { name: "肉じゃが" }).click();
    await expect(detailHeading(page, "肉じゃが")).toBeVisible();
    await expect(page.getByRole("main")).toContainText(servingsText(3));
  });
});

test.describe("S04 のパンくず", () => {
  test("編集画面のパンくずに開いた元とレシピ名が表示される", async ({ page }) => {
    const id = await createRecipeWithIngredients(page, "肉じゃが", []);
    await assignRecipe(page, dateAfter(0), "dinner", id);
    await page.goto("/");
    await openItemMenu(page, "肉じゃが");
    await page.getByRole("menuitem", { name: ja.pages.meal_plan.item_menu.view_detail }).click();
    await expect(detailHeading(page, "肉じゃが")).toBeVisible();

    await page.getByRole("main").getByRole("button", { name: t.detail.edit }).click();

    await expect(page.locator("#recipe-name")).toHaveValue("肉じゃが");
    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.meal_plan,
      "肉じゃが",
      e.breadcrumb_edit,
    ]);
  });

  test("新規登録画面のパンくず", async ({ page }) => {
    await openNewRecipeForm(page);

    await expect(detailBreadcrumb(page).getByRole("listitem")).toHaveText([
      ja.nav.recipes,
      e.title_new,
    ]);
  });

  test("パンくずで移るときも未保存の変更を確かめる", async ({ page }) => {
    await createRecipe(page, "肉じゃが");
    await openEditFromDetail(page, "肉じゃが");

    await page.locator("#recipe-servings").fill("3");
    await detailBreadcrumb(page).getByRole("button", { name: ja.nav.recipes }).click();

    await expect(page.getByRole("dialog")).toContainText(e.unsaved_confirm.title);
    await page.getByRole("button", { name: e.unsaved_confirm.discard }).click();

    await expect(page.getByRole("heading", { name: t.title, exact: true })).toBeVisible();
    await page.getByRole("cell", { name: "肉じゃが" }).click();
    await expect(page.getByRole("main")).toContainText(servingsText(2));
  });
});

test.describe("材料名と単位の候補補完", () => {
  test("在庫の食材名が材料名の候補に出る", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "しめじ", amount: 1, unit: "パック" });
    await openNewRecipeForm(page);

    await page.getByRole("button", { name: e.add_ingredient }).click();
    await ingredientNameInput(page).fill("しめ");

    await expect(page.getByRole("option", { name: /しめじ/ })).toBeVisible();
  });

  test("他のレシピの材料名が材料名の候補に出る", async ({ page }) => {
    await page.goto("/");
    await createRecipeWithIngredients(page, "肉じゃが", [
      { name: "玉ねぎ", amount: 2, unit: "個" },
    ]);
    await openNewRecipeForm(page);

    await page.getByRole("button", { name: e.add_ingredient }).click();
    await ingredientNameInput(page).fill("玉ね");

    await expect(page.getByRole("option", { name: /玉ねぎ/ })).toBeVisible();
  });

  test("材料名の候補を選ぶと単位が埋まる", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await openNewRecipeForm(page);

    await page.getByRole("button", { name: e.add_ingredient }).click();
    const nameInput = ingredientNameInput(page);
    await nameInput.fill("玉ね");
    await nameInput.press("ArrowDown");
    await nameInput.press("Enter");

    await expect(nameInput).toHaveValue("玉ねぎ");
    await expect(ingredientUnitInput(page)).toHaveValue("個");
  });

  test("材料行の単位を先に入れていれば上書きしない", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "玉ねぎ", amount: 3, unit: "個" });
    await openNewRecipeForm(page);

    await page.getByRole("button", { name: e.add_ingredient }).click();
    await ingredientUnitInput(page).fill("g");
    const nameInput = ingredientNameInput(page);
    await nameInput.fill("玉ね");
    await page.getByRole("option", { name: /玉ねぎ/ }).click();

    await expect(nameInput).toHaveValue("玉ねぎ");
    await expect(ingredientUnitInput(page)).toHaveValue("g");
  });

  test("保存したレシピの材料名が次から候補に出る", async ({ page }) => {
    await openNewRecipeForm(page);
    await page.locator("#recipe-name").fill("ズッキーニのソテー");
    await fillFirstIngredient(page, "ズッキーニ", "1", "本");
    await page.getByRole("button", { name: e.save }).click();
    await expect(page.getByRole("heading", { name: t.title })).toBeVisible();

    await page.getByRole("button", { name: t.new_button }).click();
    await page.getByRole("button", { name: e.add_ingredient }).click();
    await ingredientNameInput(page).fill("ズッキ");

    await expect(page.getByRole("option", { name: /ズッキーニ/ })).toBeVisible();
  });

  test("単位欄に既知の単位が候補として示される", async ({ page }) => {
    await page.goto("/");
    await addStockItem(page, { name: "豚肉", amount: 1, unit: "パック" });
    await openNewRecipeForm(page);

    await page.getByRole("button", { name: e.add_ingredient }).click();
    const listId = await ingredientUnitInput(page).getAttribute("list");

    await expect(page.locator(`datalist[id="${listId}"] option[value="パック"]`)).toHaveCount(1);
  });
});
