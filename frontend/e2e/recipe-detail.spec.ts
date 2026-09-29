import type { Page } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { createRecipeDetailed, openRecipeDetail } from "./helpers.js";

const t = ja.pages.recipes;
const d = t.detail;

// S03 詳細画面の 1 欄 (材料 / 手順 / メモ)。欄見出しで絞る。
function section(page: Page, title: string) {
  return page
    .getByRole("main")
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: title, exact: true }) });
}

test.describe("S03 レシピ詳細画面の表示", () => {
  test("詳細画面に基本情報と材料と手順とメモが表示される", async ({ page }) => {
    await createRecipeDetailed(page, {
      name: "肉じゃが",
      tags: ["和食"],
      servings: 2,
      ingredients: [{ name: "玉ねぎ", amount: 1, unit: "個" }],
      steps: "玉ねぎを切る",
      notes: "肉は 30 分漬ける",
    });

    await openRecipeDetail(page, "肉じゃが");

    const main = page.getByRole("main");
    await expect(main.getByRole("heading", { name: "肉じゃが" })).toBeVisible();
    await expect(main).toContainText("和食");
    await expect(main).toContainText(d.servings.replace("{{count}}", "2"));
    await expect(section(page, d.section.ingredients)).toContainText("玉ねぎ");
    await expect(section(page, d.section.ingredients)).toContainText("1 個");
    await expect(section(page, d.section.steps)).toContainText("玉ねぎを切る");
    await expect(section(page, d.section.notes)).toContainText("肉は 30 分漬ける");
    await expect(main.getByRole("button", { name: d.edit })).toBeVisible();
    await expect(main.getByRole("button", { name: d.delete })).toBeVisible();
    await expect(page.getByRole("heading", { name: t.title, exact: true })).toHaveCount(0);
  });

  test("空の欄には未登録である旨が表示される", async ({ page }) => {
    await createRecipeDetailed(page, { name: "豚汁", ingredients: [] });

    await openRecipeDetail(page, "豚汁");

    await expect(page.getByRole("main").getByText(d.empty, { exact: true })).toHaveCount(3);
  });

  test("量が 0 の材料は量を出さずに表示される", async ({ page }) => {
    await createRecipeDetailed(page, {
      name: "焼き魚",
      ingredients: [{ name: "塩", amount: 0, unit: "少々" }],
    });

    await openRecipeDetail(page, "焼き魚");

    const ingredients = section(page, d.section.ingredients);
    await expect(ingredients).toContainText("塩");
    await expect(ingredients).toContainText("少々");
    await expect(ingredients).not.toContainText("0");
  });

  test("量る単位は量の前に表示される", async ({ page }) => {
    await createRecipeDetailed(page, {
      name: "照り焼き",
      ingredients: [
        { name: "醤油", amount: 2, unit: "大さじ" },
        { name: "玉ねぎ", amount: 1, unit: "個" },
      ],
    });

    await openRecipeDetail(page, "照り焼き");

    const items = section(page, d.section.ingredients).getByRole("listitem");
    await expect(items.filter({ hasText: "醤油" })).toContainText("大さじ 2");
    await expect(items.filter({ hasText: "玉ねぎ" })).toContainText("1 個");
  });
});

test.describe("手順とメモの Markdown 描画", () => {
  test("見出しと箇条書きが描画される", async ({ page }) => {
    await createRecipeDetailed(page, {
      name: "豚の生姜焼き",
      steps: "## 下ごしらえ\n- 玉ねぎを切る\n- 豚肉を漬ける",
    });

    await openRecipeDetail(page, "豚の生姜焼き");

    const steps = section(page, d.section.steps);
    await expect(steps.getByRole("heading", { name: "下ごしらえ" })).toBeVisible();
    await expect(steps.getByRole("listitem")).toHaveText(["玉ねぎを切る", "豚肉を漬ける"]);
    await expect(steps).not.toContainText("##");
  });

  test("画像記法は代替テキストだけが表示される", async ({ page }) => {
    await createRecipeDetailed(page, {
      name: "肉じゃが",
      notes: "![完成写真](https://example.com/a.jpg)",
    });

    await openRecipeDetail(page, "肉じゃが");

    const notes = section(page, d.section.notes);
    await expect(notes).toContainText("完成写真");
    await expect(page.getByRole("main").locator("img")).toHaveCount(0);
  });

  test("リンク記法は遷移しない文字列として表示される", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが", notes: "[参考](https://example.com)" });

    await openRecipeDetail(page, "肉じゃが");

    const main = page.getByRole("main");
    await expect(main.locator("a")).toHaveCount(0);
    await section(page, d.section.notes).getByText("参考").click();
    await expect(main.getByRole("heading", { name: "肉じゃが" })).toBeVisible();
  });

  test("HTML タグは文字列として表示される", async ({ page }) => {
    await createRecipeDetailed(page, { name: "肉じゃが", notes: "<b>強火</b>" });

    await openRecipeDetail(page, "肉じゃが");

    await expect(page.getByRole("main").locator("b")).toHaveCount(0);
    await expect(section(page, d.section.notes)).toContainText("<b>強火</b>");
  });
});
