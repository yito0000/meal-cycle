## Context

- 動機は [proposal.md](proposal.md) 参照。前提として `add-mealplan-batch-linking` が実装済みで、
  `MealPlanCell` は `text-sm` / `p-2` / `min-h-16` / `MEALPLAN_DEFAULTS.mealPlanCellVisibleItems` を
  ハードコードしている状態
- 現状のレイアウト定義:
  - [week-calendar.tsx](../../../frontend/src/components/mealplan/week-calendar.tsx) の
    `gridTemplateColumns: "6rem repeat(7, minmax(0, 1fr))"` (Grid 直書き)
  - [meal-plan-cell.tsx](../../../frontend/src/components/mealplan/meal-plan-cell.tsx) の
    `min-h-16 rounded-md border border-neutral-200 bg-white p-2` (Tailwind class 直書き)
  - [mealplan-config.ts](../../../frontend/src/components/mealplan/mealplan-config.ts) の
    `MEALPLAN_DEFAULTS.mealPlanCellVisibleItems = 2` (フロント側単発定数)
  - [defaults.go](../../../internal/config/defaults.go) の `MealPlanCellVisibleItems int` (int 単一値)
- 期間切替 (1 週 / 2 週 / 4 週) は product.md では想定済みだが実装上は 1 週固定。本 proposal では
  「テーブルを持っておく」に留め、切替 UI 実装は後続 proposal に譲る
- Tailwind v4 の CSS-first `@theme` (`src/index.css`) を使っており、`tailwind.config.js` は無い。
  動的な class 文字列 (`text-${size}`) は JIT の抽出対象から外れやすいので、class は事前列挙形が安全

## Goals / Non-Goals

**Goals:**

- 1 週 × 夕食のみ (S01 の既定表示) で「セルがスカスカ」問題を解消する。1 週間の 7 セルが縦幅の
  半分以上を使うようにする
- 「期間 × 表示時間帯数」の全 9 組み合わせ (1/2/4 週 × 1/2/3 時間帯) について、セル高さ / typography /
  visible items が破綻しない
- セルサイズと typography を可変にするための **1 つの源** (`config.Defaults()`) を用意する
- Wails bindings 経由でフロントに自動配信されるよう Go 側 struct を追加する (S09 実装時に UI から
  変更できる素地)

**Non-Goals:**

- 期間切替 UI (1 週 / 2 週 / 4 週) 自体の実装 (別 proposal)
- CSS カスタムプロパティを介した動的テーマ (Tailwind class 事前列挙で足りる)
- ユーザーごとの永続的なセルサイズ設定 (S09 で扱う)
- ワイヤーフレームの完全リビルド (更新は 1 週 × 1 時間帯 と 1 週 × 3 時間帯 の 2 状態のみ)

## Decisions

### D1. セル高さの決定モデル

**選択肢:**

- (a) CSS `minmax()` を Grid の `gridAutoRows` に埋め込む — Pros: CSS のみで完結。 Cons: 値を動的に
  差し替えるとき React の `style` 経由で `${cellMinHeight}px` を渡す必要があり、Tailwind class ではなく
  raw CSS になる
- (b) React 側で計算した Tailwind class を `MealPlanCell` に渡す ← 採用 — Pros: JIT が事前抽出できる、
  デザインシステムに載る、`hover:` 等のバリアント対応も自然。 Cons: 選択肢を事前列挙する必要がある
  (`min-h-16` / `min-h-24` / `min-h-32` / `min-h-48` の 4 段階など)
- (c) `--cell-min-h` などの CSS custom property + `min-h-[var(--cell-min-h)]` — Pros: (a) の変種で
  動的にできる。 Cons: Tailwind v4 の arbitrary value 記法は使えるが、選択肢がむしろ増えてルール化が
  難しい

**採用: (b)**。理由: プロジェクトが shadcn/ui + Tailwind クラスを直書きする方針で統一されており、
生の `style` 属性への逃げは既存コードにほぼ無い。事前列挙する 4 段階 (`min-h-16` / `min-h-24` /
`min-h-32` / `min-h-48`) は本 proposal で十分カバーできる。

### D2. Config の source of truth

**選択肢:**

- (a) フロント定数のみ (`mealplan-config.ts` の `CELL_SIZE_TABLE`) — Pros: Go 側に追加不要、
  変更コストが低い。 Cons: S09 設定画面から変更できない、call site にリテラルを書かない規約から
  外れる (現行 `MealPlanCellVisibleItems` は Go 側にある)
- (b) Go 側 `config.Defaults()` に追加 + フロント側 mirror ← 採用 — Pros: 既存の
  `MealPlanCellVisibleItems` と揃う、S09 実装時に ConfigService 経由で自然に受け渡せる、
  `defaults_test.go` で数値の source of truth を担保できる。 Cons: 手動 mirror が必要
- (c) Go 側のみ、`ConfigService.GetDefaults()` をフロントから読む — Pros: mirror 不要。 Cons:
  `ConfigService` 自体が未実装 (S09 proposal で入る)、本 proposal で導入すると先取りしすぎ

**採用: (b)**。理由: プロジェクト規約 (可変値は `defaults.go`) と一致し、S09 で ConfigService が
入ったときに mirror を外すだけで済む。既存 `MealPlanCellVisibleItems` が同じ道を通っている前例あり。

### D3. 「期間 × 時間帯数 → セル高さ / typography / visible items」テーブル構造

**選択肢:**

- (a) 3×3 の 2 次元配列 (`[3][3]CellSizePreset`) — Pros: コンパクト。 Cons: 添字 (0=1 週 / 1=2 週 /
  2=4 週) の意味が call site で不明瞭
- (b) `map[string]CellSizePreset`、キーは `"1w-1s"` のような文字列 — Pros: キーが読める。 Cons:
  タイポで silent 失敗、JSON serialize がやや不自然
- (c) `[]CellSizePreset` の平坦配列 (各要素が `Period` / `Meals` を持つ) ← 採用 — Pros: enum 的な
  組み合わせが構造体に載る、フロントで `find` するだけ、JSON serialize が自然、未定義組み合わせは
  fallback ロジック 1 箇所で扱える。 Cons: 検索が O(N) だが N=9 で問題ない

**採用: (c)**。理由: JSON serialize (`json.Marshal(config.Defaults())`) 経由で S09 の設定 UI に
そのまま渡せる形が保てる。未定義組み合わせは「一番近い上位密度」に fallback するロジックを 1 箇所に
集約できる。

```go
type CellSizePreset struct {
    PeriodWeeks   int    `json:"periodWeeks"`   // 1 / 2 / 4
    VisibleMeals  int    `json:"visibleMeals"`  // 1 / 2 / 3
    MinHeightRem  int    `json:"minHeightRem"`  // Tailwind 単位 (4 / 6 / 8 / 12 → min-h-16 / -24 / -32 / -48)
    FontSizeClass string `json:"fontSizeClass"` // "text-base" | "text-sm" | "text-xs"
    VisibleItems  int    `json:"visibleItems"`  // セル内で折り畳まずに見せる品数
}
```

### D4. Typography のブレークポイント

**選択肢:**

- (a) セル高さから逆算 (`minHeight >= 32 → text-base`) — Pros: ロジックが 1 本。 Cons: 判定が
  暗黙的、閾値をどこに置くかで挙動が変わる
- (b) `CellSizePreset` に `fontSizeClass` を明示的に持つ ← 採用 — Pros: 各組み合わせに対して意図が
  読める、Tailwind class 名を事前列挙できる。 Cons: 冗長

**採用: (b)**。9 個の組み合わせに対してどの typography を割り当てるかは UX 判断なので、
コードで暗黙的に決めるより表で明示する方が変更しやすい。

割り当て案 (design 内での仮値、実装時に微調整):

| 期間 \ 時間帯数 | 1 時間帯 | 2 時間帯 | 3 時間帯 |
|---|---|---|---|
| 1 週 | `min-h-48` / `text-base` / 4 品 | `min-h-40` / `text-base` / 4 品 | `min-h-32` / `text-base` / 3 品 |
| 2 週 | `min-h-32` / `text-base` / 3 品 | `min-h-24` / `text-sm` / 3 品 | `min-h-24` / `text-sm` / 2 品 |
| 4 週 | `min-h-20` / `text-sm` / 2 品 | `min-h-16` / `text-sm` / 2 品 | `min-h-16` / `text-xs` / 2 品 |

### D5. ワイヤーフレーム更新の範囲

**選択肢:**

- (a) 全ワイヤー (1 週 × 3 時間帯 / 2 週 / 4 週) を新規で書き直す — Pros: 全状態が絵で見える。
  Cons: drawio XML 二重構造 (SVG + `content=` 属性の XML) の編集コストが大きく、実装完了までの
  スループットが下がる
- (b) 1 週 × 1 時間帯 (推奨サイズ) と 1 週 × 3 時間帯 (現行に近い密度) の 2 状態のみ更新 ← 採用 —
  Pros: ペルソナ既定と最頻使用状態を押さえる。 Cons: 2 週 / 4 週の見た目は絵で確認できない
  (テーブルの数値からの想像に依存)
- (c) ワイヤー更新は本 proposal では扱わず、後続の proposal に押し出す — Pros: proposal スコープが
  純粋にコード。 Cons: 「ワイヤーは視覚レイアウトの正」の規約 (docs/wireframes/README.md) と衝突する

**採用: (b)**。1 週 × 夕食のみは既定なので必ず絵で示す。3 時間帯 は「1 週の中で最も密度が高い」上限
なので、この 2 状態が押さえられれば読者は間の状態を想像できる。

### D6. `MealPlanCellVisibleItems` (既存 int) の扱い

**選択肢:**

- (a) 削除する — Pros: 単一の source of truth になる。 Cons: 過去の Wails bindings が持つフィールド
  が消えるので、フロント側 mirror にも影響
- (b) 残して deprecate、`CellSizePresets` の 1 週 × 1 時間帯行と同じ値を返す ← 採用 — Pros:
  bindings の diff が最小、S09 実装時に外せる。 Cons: 一時的に「同じ意味を 2 か所」で持つ

**採用: (b)**。本 proposal は「セル表示のリサイズ」だけを扱うので、既存フィールドの削除は
別の commit / proposal に切り出したほうがレビュー粒度がそろう。deprecate コメントで意図を明示する。

## Risks / Trade-offs

- [Tailwind の `min-h-*` / `text-*` class を事前列挙する必要がある] → CSS-first `@theme` (v4) のため、
  実際に使う class 文字列を静的に書いておかないと JIT が拾えない。`CellSizePreset` の値は 4 種類の
  `MinHeightRem` × 3 種類の `FontSizeClass` に絞られるので、`meal-plan-cell.tsx` 冒頭に「使う class の
  一覧」をコメントで残す
- [Go 側 struct を追加するので `wails3 generate bindings` の再生成が必要] → 本 proposal では config
  struct を Wails サービスから返すことはしないため、実は再生成不要。フロントは mirror で読む
- [未定義組み合わせ (仕様外の期間 / 時間帯数) に到達したときの fallback] → 「最も密度が高い」プリセット
  (4 週 × 3 時間帯) に落とす。log なし
- [S09 実装時に mirror を消す必要がある] → 本 proposal の deprecate コメントと defaults_test.go に
  「S09 実装時に mirror 経路を切り替える」と TODO を残す

## Migration Plan

- DB スキーマ変更なし、bindings 再生成なし、既存動作の破壊なし
- 実装後の見え方の変化: 1 週 × 夕食のみで S01 を開くとセルが約 3 倍高くなり、レシピ名が `text-base`
  で描画される (現状 `text-sm`)。既存の作り置き色帯 + 🔗 の視認性も同時に上がる
- ロールバック: `MealPlanCellSizePresets` の返り値を 1 種類 (`min-h-16` / `text-sm` / 2 品) に統一
  すれば現状に戻せる
