import i18next from "i18next";
import { describe, expect, it } from "vitest";
import type { Entry } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/mealplan/models";
import ja from "../../locales/ja.json";
import { buildMealPlanText } from "./mealplan-text";

const i18n = i18next.createInstance();
i18n.init({
  lng: "ja",
  resources: { ja: { translation: ja } },
  initAsync: false,
  interpolation: { escapeValue: false },
});
const t = i18n.t.bind(i18n);

function entry(date: string, meal: string, position: number, recipeName: string): Entry {
  return { date, meal, position, recipeName } as Entry;
}

const period = { start: "2026-11-02", end: "2026-11-08" };
const HEADING = "献立 11/2(月)〜11/8(日)";

describe("buildMealPlanText", () => {
  it.each<{ name: string; entries: Entry[]; lines: string[]; dayCount: number }>([
    {
      name: "1 行目に期間の見出しを置く",
      entries: [entry("2026-11-02", "dinner", 0, "肉じゃが")],
      lines: [HEADING, "11/2(月) 夕: 肉じゃが"],
      dayCount: 1,
    },
    {
      name: "同じ時間帯の品をセル内の順に「、」で区切る",
      entries: [
        entry("2026-11-02", "dinner", 1, "味噌汁"),
        entry("2026-11-02", "dinner", 0, "肉じゃが"),
      ],
      lines: [HEADING, "11/2(月) 夕: 肉じゃが、味噌汁"],
      dayCount: 1,
    },
    {
      name: "同じ日の行を朝、昼、夕の順に並べる",
      entries: [
        entry("2026-11-03", "dinner", 0, "カレー"),
        entry("2026-11-03", "breakfast", 0, "トースト"),
        entry("2026-11-03", "lunch", 0, "サンドイッチ"),
      ],
      lines: [HEADING, "11/3(火) 朝: トースト", "11/3(火) 昼: サンドイッチ", "11/3(火) 夕: カレー"],
      dayCount: 1,
    },
    {
      name: "献立の無い日付と時間帯の行を出さない",
      entries: [
        entry("2026-11-04", "dinner", 0, "カレー"),
        entry("2026-11-02", "dinner", 0, "肉じゃが"),
      ],
      lines: [HEADING, "11/2(月) 夕: 肉じゃが", "11/4(水) 夕: カレー"],
      dayCount: 2,
    },
    {
      name: "dayCount は行のある日付の数を返す",
      entries: [
        entry("2026-11-02", "lunch", 0, "うどん"),
        entry("2026-11-02", "dinner", 0, "肉じゃが"),
        entry("2026-11-05", "dinner", 0, "鍋"),
      ],
      lines: [HEADING, "11/2(月) 昼: うどん", "11/2(月) 夕: 肉じゃが", "11/5(木) 夕: 鍋"],
      dayCount: 2,
    },
    {
      name: "献立が 0 件なら dayCount は 0",
      entries: [],
      lines: [HEADING],
      dayCount: 0,
    },
  ])("$name", ({ entries, lines, dayCount }) => {
    const got = buildMealPlanText(entries, period, t, "ja");
    expect(got.text).toBe(lines.join("\n"));
    expect(got.dayCount).toBe(dayCount);
  });
});
