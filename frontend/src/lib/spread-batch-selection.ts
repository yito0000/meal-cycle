// 統合ダイアログ「作り置きを広げる」の選択状態を、既存 API 呼び出し
// (CopyLeftover / CopyToOtherMeal) 用に分解するヘルパ。
//
// UI 側で 1 品を任意の (対象日, 対象時間帯) に広げる操作を、
// - 同 meal ぶんは CopyLeftover (連続 N 日相当)
// - 別 meal ぶんは CopyToOtherMeal (1 セルずつ)
// に振り分ける。サーバ API 統合は Non-goal。

export type SpreadTarget = { date: string; meal: string };

export type SpreadConflictMode = "append" | "overwrite";

export type SpreadSelection = {
  targets: SpreadTarget[];
  shareBatch: boolean;
  conflictMode: SpreadConflictMode;
};

export type SpreadSource = { date: string; meal: string; position: number };

export type SpreadPlan = {
  sameMealDates: string[];
  crossMeal: SpreadTarget[];
};

// 選択セルを「同 meal 対象日群」と「別 meal 対象セル群」に分ける。
// 起点セル自身は除外する (統合ダイアログの格子でも選択不可)。
export function planSpread(source: SpreadSource, selection: SpreadSelection): SpreadPlan {
  const sameMealDates: string[] = [];
  const crossMeal: SpreadTarget[] = [];
  for (const t of selection.targets) {
    if (t.date === source.date && t.meal === source.meal) continue;
    if (t.meal === source.meal) sameMealDates.push(t.date);
    else crossMeal.push(t);
  }
  return { sameMealDates, crossMeal };
}

// ISO 日付 (YYYY-MM-DD) の差分 (b - a) を日数で返す。ローカルタイム基準
// (ダイアログの対象日レンジ計算と揃える)。
export function dateGap(a: string, b: string): number {
  const [ay, am, ad] = a.split("-").map(Number);
  const [by, bm, bd] = b.split("-").map(Number);
  const da = new Date(ay, am - 1, ad).getTime();
  const db = new Date(by, bm - 1, bd).getTime();
  return Math.round((db - da) / 86_400_000);
}
