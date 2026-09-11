## Context

- 動機は [proposal.md](proposal.md) 参照。現行は `MealPlanService.ListWeek(weekStart)` が 7 日固定で
  取得。フロントは `useQuery(["mealplan", "week", weekStart], () => ListWeek(...))` で 1 週ぶんだけ
  取得している
- 期間切替 UI は product.md §S01 で計画されていたが未実装
- 期間切替は `resize-mealplan-cells` の `CellSizePreset` (期間 × slot 数) を受け取る前提で設計されている
  (この proposal で状態を渡す起点ができる)

## Goals / Non-Goals

**Goals:**

- S01 で過去 / 未来の週に移動できる (最低限「1 週ずつ」)
- 2 週 / 4 週表示への切替 (product.md §S01 の計画を実現)
- サーバ API を「1 週固定」から「任意の連続期間」に拡張

**Non-Goals:**

- キーボードショートカット (`←` / `→`) 対応
- URL クエリ / 起動時状態の永続化 (S09 で扱う)
- 月間表示 (30 日カレンダー) — 4 週で十分

## Decisions

### D1. サービス API の切り替え方式

**選択肢:**

- (a) `ListWeek(weekStart)` を `ListPeriod(periodStart, periodWeeks)` に **置き換え** — Pros: シンプルで
  1 メソッド。 Cons: bindings のシグネチャが変わり、フロント全経路の呼び出しを一括更新する必要がある
- (b) `ListPeriod` を新規追加し、`ListWeek` は **`ListPeriod(weekStart, 1)` を呼ぶ互換 wrapper** として
  残す — Pros: 段階的移行、既存 E2E テストが影響を受けにくい。 Cons: メソッドが 1 つ増える
- (c) `ListWeek(weekStart, weeksCount)` の第 2 引数を追加。既定値は 1 (`ListWeek(weekStart)` は
  引数なしで動く形) — Cons: Go は default 引数が無いため、Wails bindings は必ず引数を要求する形になり
  「既定値の恩恵」が得られない
- (d) `ListPeriod(periodStart, periodWeeks)` に置き換え + フロントを一括更新 ← 採用

**採用: (d)**。理由: 現状 `ListWeek` の呼び出し箇所はフロントの route コンポーネントの 1 箇所と E2E shim
1 箇所だけで、`wails3 generate bindings` の再生成 + 型エラーで漏れなく更新できる。互換 wrapper (b) は
「後で消す」の TODO が残るコストの方が大きい。

### D2. `periodStart` の週境界

**選択肢:**

- (a) `periodStart` は必ず **月曜日** に揃える。ユーザーが「水曜日」を渡しても月曜日にスナップする
- (b) `periodStart` は任意の日を許容する
- (c) 月曜スナップは呼び出し側 (フロント) の責任、サービスは受け取った日付をそのまま扱う ← 採用

**採用: (c)**。理由: 「今日を含む週の月曜」への正規化はフロントで `date-fns` 等で行うのが自然。サービス
API は「任意の起点 + 日数」で汎用的にする方がテストしやすく、S09 で「日曜始まり」に切り替えたい要望が
出たときにサーバを触らなくて済む。バリデーションは既存 `ValidateDate` (YYYY-MM-DD 形式) のみ。

### D3. 前後ナビの単位

**選択肢:**

- (a) `＜` / `＞` は現在の **期間長** ぶんシフト (2 週表示なら 14 日戻る / 進む) ← 採用
- (b) `＜` / `＞` は常に **1 週 (7 日) ぶん** シフト (期間長に関わらず)

**採用: (a)**。理由: 2 週表示中に `＞` を押して次の 2 週間に飛ぶ挙動が直感的。7 日ずつシフトすると、
2 週表示中に押すたびに「1 週目が消えて 3 週目が現れる」の見え方になり違和感。product.md §S01 の
シナリオ (前後ナビゲーション) にも合致。

### D4. TanStack Query の cache key

**選択肢:**

- (a) `["mealplan", "period", periodStart, periodWeeks]` ← 採用
- (b) `["mealplan", "week", weekStart]` を残しつつ内部で `periodWeeks` を state に持つ — Cons:
  cache が期間ごとに分離されない

**採用: (a)**。理由: 期間切替のたびに再取得すべきなので、cache key に含めるのが正しい。既存の
`invalidateQueries({ queryKey: ["mealplan", "week"] })` は `["mealplan"]` に一段短くしてまとめて
invalidate する形に変える。

## Risks / Trade-offs

- [`wails3 generate bindings` の再生成で bindings shim の型が壊れる] → 既に何度か経験しており、パターン
  として `frontend/e2e/bindings-shim/mealplanservice.ts` の該当メソッドを合わせて更新するだけ
- [4 週表示は 28 日 × N slot でセル数が増えるので、`ListWeek` の SQL が重くなる可能性] → 28 日でも数百行
  規模。SQLite で問題になるスケールではない。indexed date range スキャンで十分
- [`resize-mealplan-cells` proposal と実装順序の依存関係] → 本 proposal は `periodWeeks` を state で持つ
  だけ。CellSizePreset の適用 (2 週 → セル狭く、4 週 → セル最小) は resize-mealplan-cells 側の
  実装に委ねる。両 proposal は独立に apply できるが、両方 apply して初めて視覚的完成度になる

## Migration Plan

- サービス API 変更のみ (DB スキーマ変更なし)
- 実装後の見え方: S01 右上に「1 週 / 2 週 / 4 週」トグル + `＜` `＞` `今週` ボタンが追加
- ロールバック: サービスは `ListPeriod` を消して `ListWeek` に戻す、フロントは PeriodNav コンポーネントを
  非表示にする。DB は無変更なので影響なし
