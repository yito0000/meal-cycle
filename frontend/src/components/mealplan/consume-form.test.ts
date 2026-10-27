import { describe, expect, it } from "vitest";
import {
  type ConsumeRow,
  toConsumeRows,
  toConsumptions,
  validateConsumeRows,
} from "./consume-form";

const row = (amount: string, current = 3): ConsumeRow => ({
  stockId: "s1",
  expiresAt: "",
  current,
  amount,
});

describe("toConsumeRows", () => {
  it.each([
    { amount: 2, want: "2" },
    { amount: 0.5, want: "0.5" },
    { amount: 0, want: "0" },
  ])("初期値は候補の Amount $amount の表示 $want になる", ({ amount, want }) => {
    const rows = toConsumeRows({
      lines: [{ candidates: [{ stockId: "s1", expiresAt: "", currentAmount: 3, amount }] }],
    });
    expect(rows).toEqual([{ stockId: "s1", expiresAt: "", current: 3, amount: want }]);
  });
});

describe("validateConsumeRows", () => {
  it.each([
    { name: "今の量を超える値はエラーになる", amount: "5", error: true },
    { name: "負の値はエラーになる", amount: "-1", error: true },
    { name: "数値でない値はエラーになる", amount: "abc", error: true },
    { name: "空はエラーになる", amount: " ", error: true },
    { name: "今の量と等しい値は通る", amount: "3", error: false },
    { name: "0 は通る", amount: "0", error: false },
  ])("$name", ({ amount, error }) => {
    const got = validateConsumeRows([row(amount)]);
    expect(got.hasError).toBe(error);
    expect(got.errors[0] !== undefined).toBe(error);
  });
});

describe("toConsumptions", () => {
  it.each([
    {
      name: "0 の行は toConsumptions から除かれる",
      rows: [row("0"), { ...row("2"), stockId: "s2" }],
      want: [{ stockId: "s2", amount: 2 }],
    },
    { name: "すべて 0 なら空になる", rows: [row("0")], want: [] },
  ])("$name", ({ rows, want }) => {
    expect(toConsumptions(rows)).toEqual(want);
  });
});
