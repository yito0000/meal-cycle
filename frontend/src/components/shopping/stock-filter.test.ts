import { describe, expect, it } from "vitest";
import { StockStatus } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { countByStockFilter, matchesStockFilter, type StockFilter } from "./stock-filter";

describe("matchesStockFilter", () => {
  it.each<{ filter: StockFilter; status: StockStatus; want: boolean }>([
    { filter: "all", status: StockStatus.StockNone, want: true },
    { filter: "all", status: StockStatus.StockPartial, want: true },
    { filter: "all", status: StockStatus.StockCovered, want: true },
    { filter: "needed", status: StockStatus.StockNone, want: true },
    { filter: "needed", status: StockStatus.StockPartial, want: true },
    { filter: "needed", status: StockStatus.StockCovered, want: false },
    { filter: "covered", status: StockStatus.StockNone, want: false },
    { filter: "covered", status: StockStatus.StockPartial, want: false },
    { filter: "covered", status: StockStatus.StockCovered, want: true },
  ])("$filter は $status に $want を返す", ({ filter, status, want }) => {
    expect(matchesStockFilter({ stockStatus: status }, filter)).toBe(want);
  });
});

describe("countByStockFilter", () => {
  it.each([
    { name: "空配列", items: [], want: { all: 0, needed: 0, covered: 0 } },
    {
      name: "none / partial / covered が 1 件ずつ",
      items: [
        { stockStatus: StockStatus.StockNone },
        { stockStatus: StockStatus.StockPartial },
        { stockStatus: StockStatus.StockCovered },
      ],
      want: { all: 3, needed: 2, covered: 1 },
    },
  ])("$name の件数を数える", ({ items, want }) => {
    expect(countByStockFilter(items)).toEqual(want);
  });
});
