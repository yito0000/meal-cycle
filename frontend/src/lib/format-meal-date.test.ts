import { describe, expect, it } from "vitest";
import { formatMealDate } from "./format-meal-date";

describe("formatMealDate", () => {
  it.each([
    { name: "ja は月/日(曜日) になる", iso: "2026-10-20", language: "ja", want: "10/20(火)" },
    { name: "en は 曜日, 月/日 になる", iso: "2026-10-20", language: "en", want: "Tue, 10/20" },
    { name: "月末はローカル日付のまま示す", iso: "2026-10-31", language: "ja", want: "10/31(土)" },
    { name: "月初はローカル日付のまま示す", iso: "2026-11-01", language: "ja", want: "11/1(日)" },
  ])("$name", ({ iso, language, want }) => {
    expect(formatMealDate(iso, language)).toBe(want);
  });

  it("同じ言語で 2 回呼んでも同じ文字列を返す", () => {
    expect(formatMealDate("2026-10-20", "ja")).toBe(formatMealDate("2026-10-20", "ja"));
  });
});
