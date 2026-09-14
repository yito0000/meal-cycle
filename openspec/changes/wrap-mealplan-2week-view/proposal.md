## Why

`add-mealplan-period-navigation` で入れた「2 週表示 = 14 列 1 行 + 横スクロール」レイアウトは、実装後の
目視確認で「4 週表示 (7 列 × 4 行折り返し) の方が全体像を把握しやすい」ことがわかった。2 週だけ横方向に
14 列並べる非対称は、Wails webview の既定幅ではセルが細長くなり視認性が落ちる。4 週の折り返し方針に
2 週も合わせる方が自然。

## What Changes

- S01 の 2 週表示のレイアウトを **「7 列 × 2 行折り返し」** に変更する
- 1 週表示 (7 列 × 1 行) と 4 週表示 (7 列 × 4 行折り返し) の中間として、行数だけが増える形になる
- 横スクロール (`overflow-x-auto`) は削除する。全期間で「1 週分 = 1 行」の一貫レイアウトになる
- **BREAKING**: `mealplan` spec の Scenario「2 週を選ぶと 14 日分が表示される」を
  「2 週を選ぶと 2 週分が行方向に折り返す」に置き換える (受け入れ基準の変更)
- ワイヤー / product.md の記述「2 週なら 14 列（横スクロール想定）」を「2 週は 7 列 × 2 行折り返し」に更新
- E2E テスト「2 週を選ぶと 14 日分が表示される」を新シナリオ名にリネーム、内容も新レイアウト検証に置換

## Non-goals

- **Go 側 (ListPeriod) の変更**: 取得データは変わらない (14 日分の Entry を返すまま)
- **1 週 / 4 週レイアウトの変更**: 現状維持
- **CellSizePreset の再調整**: 2 週 × 時間帯数の preset 値は現状のまま (`resize-mealplan-cells` の
  結果を継承)。もし折り返し後にセル密度が違和感を出すなら別 proposal で調整する
- **月間表示 (30 日カレンダー)**: 対象外 (`add-mealplan-period-navigation` の Non-goals から継続)
- **キーボードショートカット**: 対象外

## Capabilities

### Modified Capabilities

- `mealplan`:
  - **MODIFIED**: 「期間切替」Requirement の Scenario「2 週を選ぶと 14 日分が表示される」を
    「2 週を選ぶと 2 週分が行方向に折り返す」に差し替える (14 列 1 行 → 7 列 × 2 行折り返し)

## Impact

- **新設ファイル**: なし
- **更新ファイル**:
  - `frontend/src/components/mealplan/week-calendar.tsx` — `isWrapped` の分岐を `periodWeeks >= 2` に
    広げ、`blockCount` を `periodWeeks` そのものに変える (1 週 = 1 block、2 週 = 2 block、4 週 = 4 block)。
    横スクロール (`overflow-x-auto`) の class を削除する
  - `frontend/e2e/mealplan-assign.spec.ts` — 「2 週を選ぶと 14 日分が表示される」テストを
    「2 週を選ぶと 2 週分が行方向に折り返す」にリネーム + 検証内容を新レイアウトに合わせて更新
  - `docs/product.md` §S01 — 「1 週なら 7 列、2 週なら 14 列（横スクロール想定）、4 週は行方向に折り返す」
    の説明を「1 週 = 7 列 × 1 行、2 週 = 7 列 × 2 行折り返し、4 週 = 7 列 × 4 行折り返し」に更新
  - `docs/scenarios/mealplan.feature` — 「2 週を選ぶと 14 日分が表示される」シナリオを spec と揃えて
    「2 週を選ぶと 2 週分が行方向に折り返す」にリネーム / 内容更新
  - `openspec/specs/mealplan/spec.md` — spec 側 (archive 時に sync される delta 反映後) の対応シナリオも
    更新される。本 proposal の delta spec が source of truth
- **生成物 (再生成のみ)**: なし
- **migration**: なし

### 後続 proposal への引き継ぎ

- `resize-mealplan-cells` の 2 週プリセット (`{periodWeeks: 2, visibleMeals: N}`) は 7 列レイアウトで
  評価し直したくなる可能性がある。ただし本 proposal 内では触らず、必要ならフォロー proposal で扱う
- ワイヤー `wireframes/S01_meal_calendar.v2.drawio.svg` は本 proposal では更新しない (`.drawio.svg` の
  二重構造更新は独立作業として別途)。product.md の記述と実装は先に揃える
