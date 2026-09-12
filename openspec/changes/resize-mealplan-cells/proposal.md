## Why

S01 献立カレンダーの現行セル (`min-h-16` = 64px、`text-sm`) は、既定表示 (1 週間 × 夕食のみ = 7 セル) に対して
小さすぎる。7 セルしか描画されないのに各セルは画面の 1/7 幅 × 64px 高しか使わず、画面の縦半分以上が余白のまま
残る。1 か月表示なら適正だが 1 週間なら情報密度を上げるべきで、レシピ名も現状の `text-sm` は視認性が低い。
「作り置き 3 日ぶんを 1 バッチで見せる」設計 (`add-mealplan-batch-linking` で導入した色帯 + 🔗) の効きも、
セルが小さいと見にくい。

## What Changes

- CSS Grid の `grid-auto-rows` (現状 `min-h-16` 固定) を、**期間 × 表示時間帯数** の 2 次元テーブルから引く
  可変値に変更する
- 目安 (実際の値は design.md で決定): 1 週 × 1 時間帯 は `min-h-48` (192px)、1 週 × 3 時間帯 は `min-h-32`、
  2 週 × 3 時間帯 は `min-h-24`、4 週は現状の `min-h-16`
- あわせて、広いセル高を活かすためのセル内デザインを調整する:
  - レシピ名の typography を段階的に上げる (期間が広いとき `text-sm` → `text-base`)
  - セル内で折り畳まずに見せる品数上限 (`MealPlanCellVisibleItems`、現状固定 2) を「セル高さから逆算」で
    3〜4 に伸ばす
  - padding を `p-2` → `p-3` に厚く、`＋` と `⋯` のタップ / クリックのヒットボックスを広げる
- **`＋` ボタンの視認性 / 押しやすさを改善する**:
  - 空セル `＋` は既にセル全体クリック可能だが、視覚的にもっと目立たせる
    (文字サイズを `text-sm` → `text-2xl` 相当に、色を `text-neutral-400` → `text-neutral-500` に濃く)
  - 割当済セル右上 `＋` は現状 `text-xs` (12px 相当) と小さすぎるので `text-base` + `p-1.5` に拡大、
    hover ハイライト背景を追加してヒットボックスを明確化
- 「期間 × 時間帯数 → セル高さ / typography / visible items」のテーブルは `internal/config/defaults.go`
  に集約 (call site にリテラル閾値を書かないプロジェクト規約に合わせる)
- ワイヤーフレーム [S01_meal_calendar.drawio.svg](../../../docs/wireframes/S01_meal_calendar.drawio.svg) を
  新レイアウト (1 週 × 夕食のみ / 1 週 × 3 時間帯) で更新する
- [docs/product.md](../../../docs/product.md) §S01 の描画説明と §1 可変値表を追記する

## Non-goals

- **縦横をひっくり返す (day-as-row) レイアウト**: 「作り置きの横方向色帯」「4 週折り返しモデル」を壊すため
  今回は採らない
- **カレンダー / アジェンダの切替ビュー**: 機能過多、MVP 対象外
- **ドラッグ&ドロップでのセル間並び替え**: 別 proposal (`add-mealplan-dnd` 相当) で扱う
- **期間切替 UI (1 週 / 2 週 / 4 週) の実装**: 現在は 1 週固定。本 proposal ではテーブルに 2 週 / 4 週の
  行を用意するに留め、UI 切替ボタン自体の実装は後続 proposal (`add-mealplan-period-switch` 相当)
- **設定画面 S09 からのセル高さ変更 UI**: `internal/config/defaults.go` に置くことで S09 実装時に自然に
  つながる素地は作るが、UI 側の設定操作は S09 の proposal で扱う

## Capabilities

### New Capabilities

なし。

### Modified Capabilities

- `mealplan`:
  - **MODIFIED**: 「セル内の品表示」Requirement — 「1..N 品はレシピ名を 1 行 1 品で描画する」を、
    **期間 × 表示時間帯数** に応じてセル高さと表示件数上限が可変になる旨で拡張。「品数超過は先頭 2 品を表示し
    3 行目に "他 N 件"」を「先頭 N 品 (`config.Defaults()` から期間 × 時間帯数 で決定) を表示、超過は "他 M 件"」
    に一般化

## Impact

- **新設ファイル**:
  - なし (既存ファイルの拡張のみ)
- **更新ファイル**:
  - [internal/config/defaults.go](../../../internal/config/defaults.go) — `MealPlanCellSizePresets`
    (期間 × 時間帯数 → セル高 / typography / visible items) を追加。既存の `MealPlanCellVisibleItems`
    (int 固定) は「表の 1 セル」に置き換わるので削除または後方互換のため deprecate 明記
  - [internal/config/defaults_test.go](../../../internal/config/defaults_test.go) — 新テーブルの
    デフォルト値を検証するテストを追加 (「デフォルト値変更時はこのテストと product.md 両方を更新」の
    プロジェクト規約に合わせる)
  - [frontend/src/components/mealplan/mealplan-config.ts](../../../frontend/src/components/mealplan/mealplan-config.ts)
    — Go 側 `MealPlanCellSizePresets` の mirror を追加 (S09 で ConfigService 経由で読めるようになるまでは
    手動同期する既存パターンに従う)
  - [frontend/src/components/mealplan/week-calendar.tsx](../../../frontend/src/components/mealplan/week-calendar.tsx)
    — Grid の `gridAutoRows` を「期間 × 表示時間帯数」から算出。MealPlanCell に typography /
    visibleItems props を渡す
  - [frontend/src/components/mealplan/meal-plan-cell.tsx](../../../frontend/src/components/mealplan/meal-plan-cell.tsx)
    — typography / padding / visibleItems を props で受ける。現状ハードコードの `text-sm` / `p-2` / `min-h-16` /
    定数 `MEALPLAN_DEFAULTS.mealPlanCellVisibleItems` を置き換える。あわせて `＋` ボタン (空セル中央 /
    右上両方) を拡大して押しやすくする
  - [docs/wireframes/S01_meal_calendar.drawio.svg](../../../docs/wireframes/S01_meal_calendar.drawio.svg) —
    新レイアウトを反映 (可視 SVG + drawio 用 XML の二重構造)。1 週 × 夕食のみ (推奨サイズ) と
    1 週 × 3 時間帯 (縦がやや詰まる版) の 2 状態を描く
  - [docs/product.md](../../../docs/product.md) §S01 描画説明 / §1 可変値表 (`MealPlanCellSizePresets`
    の追加)
- **生成物 (再生成のみ)**: なし (config の struct 追加は Wails サービスの公開シグネチャに影響しないため、
  `wails3 generate bindings` の再生成は不要)
- **migration**: なし (DB スキーマ変更なし)

### 後続 proposal への引き継ぎ

- **期間切替 UI (`add-mealplan-period-switch` 相当)**: 期間切替 (1 週 / 2 週 / 4 週) の実装は別 proposal。
  本 proposal で `MealPlanCellSizePresets` に 2 週 / 4 週の行を用意しておくので、切替 UI 実装時に
  参照するだけで済む
- **設定画面 S09 (`add-settings-screen` 相当)**: セル高さプリセットの UI からの変更、および
  「1 週 × 夕食のみ = 特大 / 中 / 標準」の 3 段階からの選択を提供。本 proposal は Go 側 config に載せる
  ことで S09 実装時に mirror が自然につながる
- **`add-mealplan-multi-week`**: 現行の週データ取得 API (`ListWeek`) は 7 日固定。2 週 / 4 週表示のときは
  複数週の取得が必要になる。本 proposal はセルサイズだけを扱うのでこれは対象外
