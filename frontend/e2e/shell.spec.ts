import type { Locator } from "@playwright/test";
import ja from "../src/locales/ja.json" with { type: "json" };
import { expect, test } from "./fixtures.js";
import { openFridge } from "./helpers.js";

// Tailwind v4 のパレットは oklch で返るので、その場合は明度 L をそのまま使う。
function toLuminance(color: string) {
  const nums = (color.match(/\d+(\.\d+)?/g) ?? []).map(Number);
  if (color.startsWith("oklch")) return nums[0];
  const [r, g, b] = nums;
  return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
}

async function textLuminance(locator: Locator) {
  return toLuminance(await locator.evaluate((el) => getComputedStyle(el).color));
}

// 背景が透明な要素は親の背景が見えるので、色の付いた祖先まで遡る。
async function backgroundLuminance(locator: Locator) {
  const bg = await locator.evaluate((el) => {
    for (let e: Element | null = el; e; e = e.parentElement) {
      const c = getComputedStyle(e).backgroundColor;
      if (c !== "transparent" && !/rgba\(.*,\s*0\)$/.test(c)) return c;
    }
    return "rgb(255, 255, 255)";
  });
  return toLuminance(bg);
}

test.describe("配色の固定", () => {
  test.use({ colorScheme: "dark" });

  test("OS がダークモードでも見出しは暗い文字で表示される", async ({ page }) => {
    await page.goto("/");

    const heading = page.getByRole("heading", { level: 1, name: ja.pages.meal_plan.title });
    await expect(heading).toBeVisible();
    expect(await textLuminance(heading)).toBeLessThan(0.3);
    expect(await backgroundLuminance(heading)).toBeGreaterThan(0.9);
  });

  test("OS がダークモードでも入力欄の文字は暗い色で表示される", async ({ page }) => {
    await page.goto("/");
    await openFridge(page);

    const input = page
      .getByRole("group", { name: ja.pages.fridge.add.label })
      .getByRole("combobox", { name: ja.pages.fridge.field.name });
    await input.fill("玉ねぎ");

    await expect(input).toHaveValue("玉ねぎ");
    expect(await textLuminance(input)).toBeLessThan(0.3);
    expect(await backgroundLuminance(input)).toBeGreaterThan(0.9);
  });
});
