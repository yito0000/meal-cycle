import { describe, expect, it } from "vitest";
import {
  formatDate,
  isSameLocalDate,
  parseDate,
  periodDates,
  weekDates,
  weekStartOf,
} from "./week-utils";

describe("formatDate", () => {
  it.each([
    { name: "1 桁の月と日をゼロ埋めする", input: new Date(2026, 0, 5), want: "2026-01-05" },
    { name: "2 桁の月と日はそのまま", input: new Date(2026, 9, 15), want: "2026-10-15" },
  ])("$name", ({ input, want }) => {
    expect(formatDate(input)).toBe(want);
  });
});

describe("parseDate", () => {
  it.each([
    { name: "undefined", input: undefined },
    { name: "形式の違う文字列 (スラッシュ区切り)", input: "2026/10/15" },
    // 繰り上がりで 3/2 にならないこと
    { name: "存在しない日付", input: "2026-02-30" },
  ])("$name に null を返す", ({ input }) => {
    expect(parseDate(input)).toBeNull();
  });

  it("YYYY-MM-DD をローカル日付として返す", () => {
    const got = parseDate("2026-10-15");
    expect(got).not.toBeNull();
    expect(got?.getFullYear()).toBe(2026);
    expect(got?.getMonth()).toBe(9);
    expect(got?.getDate()).toBe(15);
    expect(got?.getHours()).toBe(0);
  });
});

describe("weekStartOf", () => {
  it.each([
    { name: "月曜はそのまま返す", input: "2026-10-12", want: "2026-10-12" },
    { name: "水曜は同じ週の月曜を返す", input: "2026-10-14", want: "2026-10-12" },
    { name: "日曜は前の月曜を返す", input: "2026-10-18", want: "2026-10-12" },
    { name: "月をまたぐ日曜は前月の月曜を返す", input: "2026-11-01", want: "2026-10-26" },
  ])("$name ($input → $want)", ({ input, want }) => {
    const d = parseDate(input);
    if (d === null) throw new Error(`invalid fixture: ${input}`);
    expect(formatDate(weekStartOf(d))).toBe(want);
  });
});

describe("periodDates", () => {
  it("開始日から 14 日分を月末をまたいでも日を飛ばさずに返す", () => {
    const got = periodDates(new Date(2026, 9, 26), 14).map(formatDate);
    expect(got).toEqual([
      "2026-10-26",
      "2026-10-27",
      "2026-10-28",
      "2026-10-29",
      "2026-10-30",
      "2026-10-31",
      "2026-11-01",
      "2026-11-02",
      "2026-11-03",
      "2026-11-04",
      "2026-11-05",
      "2026-11-06",
      "2026-11-07",
      "2026-11-08",
    ]);
  });
});

describe("weekDates", () => {
  it("7 要素を返す", () => {
    expect(weekDates(new Date(2026, 9, 12))).toHaveLength(7);
  });
});

describe("isSameLocalDate", () => {
  it.each([
    {
      name: "同じ日の異なる時刻",
      a: new Date(2026, 9, 15, 0, 0),
      b: new Date(2026, 9, 15, 23, 59),
      want: true,
    },
    {
      name: "異なる日",
      a: new Date(2026, 9, 15, 23, 59),
      b: new Date(2026, 9, 16, 0, 0),
      want: false,
    },
  ])("$name に $want を返す", ({ a, b, want }) => {
    expect(isSameLocalDate(a, b)).toBe(want);
  });
});
