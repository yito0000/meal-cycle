import type { TFunction } from "i18next";
import { describe, expect, it } from "vitest";
import { StapleState } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/fridge/models";
import { StockStatus } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";
import { buildShoppingText } from "./shopping-text";

// キーと引数をそのまま文字列にし、組み立ての順と使ったキーだけを確かめる。
const t = ((key: string, opts?: Record<string, unknown>) =>
  opts && "name" in opts ? `${key}:${opts.name}` : key) as unknown as TFunction;

const period = { periodStart: "2026-10-05", periodDays: 7 };
const units = ["大さじ", "小さじ", "カップ"];
const onion = { name: "玉ねぎ", amount: 2, unit: "個", stockStatus: StockStatus.StockNone };
const soy = { name: "醤油", amount: 0, unit: "", stockStatus: StockStatus.StockNone };
const milk = { name: "牛乳", state: StapleState.StapleOut };
const bread = { name: "食パン", state: StapleState.StapleLow };

const H = "pages.shopping.copy.heading";
const ITEM = "pages.shopping.copy.item:玉ねぎ";
const SH = "pages.shopping.copy.staple_heading";
const PH = "pages.shopping.copy.pantry_heading";
const PANTRY = "pages.shopping.copy.pantry_item:醤油";
const MILK = "pages.shopping.copy.staple_item:牛乳";
const BREAD = "pages.shopping.copy.staple_item_low:食パン";

describe("buildShoppingText", () => {
  it.each([
    {
      name: "常備品が無いときは、これまでどおり材料と家にあるか確かめる材料だけを組み立てる",
      items: [onion, soy],
      staples: [],
      want: [H, ITEM, "", PH, PANTRY],
      count: 2,
    },
    {
      name: "常備品は材料のあとに空行と見出しを挟んで並ぶ",
      items: [onion],
      staples: [milk],
      want: [H, ITEM, "", SH, MILK],
      count: 2,
    },
    {
      name: "少ないの常備品には (少ない) が添えられ、ないの常備品には添えられない",
      items: [],
      staples: [milk, bread],
      want: [H, "", SH, MILK, BREAD],
      count: 2,
    },
    {
      name: "常備品は家にあるか確かめる材料より前に並ぶ",
      items: [onion, soy],
      staples: [milk, bread],
      want: [H, ITEM, "", SH, MILK, BREAD, "", PH, PANTRY],
      count: 4,
    },
    {
      name: "材料が無く常備品だけのときは、見出しのあとに空行と常備品の見出しが続く",
      items: [],
      staples: [milk],
      want: [H, "", SH, MILK],
      count: 1,
    },
    {
      name: "件数は材料、常備品、家にあるか確かめる材料の行の合計になる",
      items: [onion, soy],
      staples: [bread],
      want: [H, ITEM, "", SH, BREAD, "", PH, PANTRY],
      count: 3,
    },
  ])("$name", ({ items, staples, want, count }) => {
    const got = buildShoppingText(items, period, t, units, staples);
    expect(got.text.split("\n")).toEqual(want);
    expect(got.count).toBe(count);
  });
});
