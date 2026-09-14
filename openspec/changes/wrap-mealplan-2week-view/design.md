## Context

- 動機は [proposal.md](proposal.md) 参照。`add-mealplan-period-navigation` で導入した 2 週レイアウトが
  「14 列 1 行 + 横スクロール」だが、目視確認で「4 週の折り返しに揃えた方が全体像が掴みやすい」判断に
  なった
- 実装は `frontend/src/components/mealplan/week-calendar.tsx` の `isWrapped`, `columnsPerBlock`,
  `blockCount`, `dateBlocks` 分岐で 1 か所にまとまっている
- 依存関係: `add-mealplan-period-navigation` が **先に archive されている前提**。本 proposal の delta は
  archive 済み `openspec/specs/mealplan/spec.md` の「期間切替」Requirement を上書きする

## Goals / Non-Goals

**Goals:**

- 2 週表示を「7 列 × 2 行折り返し」に統一する。1 週 (1 行) / 4 週 (4 行) との一貫性を取る
- 横スクロールを排除し、期間長に関係なく全域が縦だけで俯瞰できるようにする
- 実装 diff を最小化する (`isWrapped` の閾値と `blockCount` の式を変えるだけで届く)

**Non-Goals:**

- CellSizePreset の再チューニング (`resize-mealplan-cells` で決めた値を継続)
- 4 週表示のレイアウト変更 (継続で 7 列 × 4 行)
- Go / SQL / bindings への影響 (ListPeriod は 14 日分の Entry を返すまま。取得と描画は独立)

## Decisions

### D1. 折り返し閾値をどう表現するか

**選択肢:**

- (a) `isWrapped = periodWeeks >= 2` に閾値を 1 段下げ、`blockCount = isWrapped ? periodWeeks : 1`
  で 1 週 = 1 ブロック / 2 週 = 2 ブロック / 4 週 = 4 ブロックとする。実装 diff が最小 ← 採用
- (b) `isWrapped` を廃止し、常に `blockCount = periodWeeks`、`columnsPerBlock = 7` に統一する。
  1 週表示も「1 ブロック × 7 列」の形になり、より均質。 Cons: 1 週表示のときの `minWidth` 計算などの
  周辺コードが余計な条件を含むことになる (差分が広がる)
- (c) `periodWeeks` から「行数」「列数」を返す純関数 (`layoutFor(periodWeeks)`) に抽出する。
  Pros: テスタブル / 拡張しやすい。 Cons: 現状用途で 3 値だけなのでオーバーエンジニアリング

**採用: (a)**。理由: 現状の分岐は既に `isWrapped` を持ち、閾値を `=== 4` から `>= 2` にするだけで
2 週も折り返し対象になる。`blockCount` の式は `isWrapped ? periodWeeks : 1` になり、4 週の挙動は
変わらない。差分は数行で済む。将来「8 週」等が増えるなら (c) に寄せる。

### D2. 横スクロール属性の扱い

**選択肢:**

- (a) `overflow-x-auto` を完全に外す ← 採用
- (b) 残しておく (どのケースでも折り返しになるので実際には発火しないが、保険)

**採用: (a)**。理由: 「発火しない属性」を残すのは意図の曖昧化。折り返しに統一するなら削除して
CSS のセマンティクスを明示的にする。仮に将来 3 週 (21 日) や 5 週表示 (35 日) を足すなら、その
proposal でレイアウトを再検討する。

### D3. Scenario 名の差し替え方針

**選択肢:**

- (a) 既存 Scenario「2 週を選ぶと 14 日分が表示される」の名前 + 内容を **RENAMED + MODIFIED** で
  分けて記述する
- (b) 「期間切替」Requirement 全体を MODIFIED し、Scenario ブロックをまるごと置き換える ← 採用

**採用: (b)**。理由: 本 proposal は「2 週の Scenario 名」だけでなく「Requirement 本文の説明
(取得起点日は変わらず、表示される日数だけが 7 / 14 / 28 に広がる)」にも「レイアウトは全期間で
折り返す」旨を追記したい。RENAMED は Scenario 単体には使えるが Requirement 本文の説明と両立させると
delta が読みにくい。全体 MODIFIED の方が spec の archive 後の姿と 1:1 で対応し、レビューコストが低い。

## Risks / Trade-offs

- [`add-mealplan-period-navigation` の archive 順序] → 本 proposal を先に archive すると
  「期間切替」Requirement が main spec に無い状態で MODIFIED が走り validate 失敗する。
  archive 順序は `add-mealplan-period-navigation` → `wrap-mealplan-2week-view` を厳守する
- [2 週表示のセルの縦幅] → 折り返しにすると、`resize-mealplan-cells` の `{periodWeeks: 2}` preset
  (現状 `min-h-24` 前後) が「1 行だった前提」で決められていた場合に、縦積みで縦長になる可能性がある。
  ただし現状 preset の `minHeightRem` は 24〜32 でセル数自体は 14 のまま (折り返し前と同じ) なので、
  縦幅そのものはそれほど変わらない。実装後の目視で違和感が出たら別 proposal で調整する
- [ワイヤーとの一時的乖離] → `wireframes/S01_meal_calendar.v2.drawio.svg` は 2 週想定を含んでいない
  (現状は 1 週 × 3 食のみ描画) ので実害は無い。product.md §S01 のテキストだけ先に揃える

## Migration Plan

- フロントのみの変更。DB / API 変更なし
- 実装後の見え方: 2 週表示を選ぶと 14 セルが 7 列 × 2 行に折り返される。横スクロール bar が消える
- ロールバック: `week-calendar.tsx` の `isWrapped = periodWeeks === 4` に戻し、`overflow-x-auto` の
  class を復活させる。spec 側は archive 済みの main spec を手動で戻す (通常 revert)
