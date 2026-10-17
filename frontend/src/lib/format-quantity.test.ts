import { describe, expect, it } from "vitest";
import { formatQuantity } from "./format-quantity";

// 量る単位は設定で変わるので、テストでは一覧を固定して渡す (add-settings-screen design.md D9)。
const measuringUnits = ["大さじ", "小さじ", "カップ"];

describe("formatQuantity", () => {
  it.each([
    { name: "量る単位は単位を前に置く", amount: 1, unit: "大さじ", want: "大さじ 1" },
    { name: "量る単位以外は量を前に置く", amount: 200, unit: "g", want: "200 g" },
    { name: "量が 0 なら単位だけを返す", amount: 0, unit: "少々", want: "少々" },
    { name: "単位が空文字なら量だけを返す", amount: 3, unit: "", want: "3" },
    {
      name: "単位の前後の空白を取り除いてから判定する",
      amount: 1,
      unit: " 大さじ ",
      want: "大さじ 1",
    },
    { name: "小数の量は小数第 2 位までに丸める", amount: 0.333, unit: "個", want: "0.33 個" },
    { name: "丸めた結果の末尾の 0 を出さない", amount: 1.5, unit: "個", want: "1.5 個" },
  ])("$name", ({ amount, unit, want }) => {
    expect(formatQuantity(amount, unit, measuringUnits)).toBe(want);
  });

  it("渡した量る単位の一覧に無い単位は量を前に置く", () => {
    expect(formatQuantity(1, "大さじ", [])).toBe("1 大さじ");
  });
});
