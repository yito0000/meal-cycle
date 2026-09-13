## 1. Go 側 config テーブル

- [x] 1.1 `internal/config/defaults.go` に `CellSizePreset` struct を追加する (fields:
      `PeriodWeeks int` / `VisibleMeals int` / `MinHeightRem int` / `FontSizeClass string` /
      `VisibleItems int`)。完了判定: `go build ./internal/...` が通る
- [x] 1.2 `Config` struct に `MealPlanCellSizePresets []CellSizePreset` フィールドを追加し、
      `Defaults()` で design.md D4 の表 (9 プリセット) を返す。完了判定: 下記 1.4 のテストが pass する
- [x] 1.3 既存 `MealPlanCellVisibleItems` は残し、コメントで「S09 実装時に `MealPlanCellSizePresets`
      に完全移行する。それまでの間は 1 週 × 1 時間帯行と同じ値を保つ」と deprecate 明記する。
      完了判定: 下記 1.4 のテストが pass する
- [x] 1.4 `internal/config/defaults_test.go` に以下のテストを追加する
    - `Defaults().MealPlanCellSizePresets` が 9 プリセットを返す (1/2/4 週 × 1/2/3 時間帯 の全組み合わせ)
    - 1 週 × 1 時間帯 のプリセット値 (MinHeightRem / FontSizeClass / VisibleItems) が design.md D4 の表と一致
    - 1 週 × 3 時間帯 のプリセット値が design.md D4 の表と一致
    - 4 週 × 3 時間帯 のプリセット値が design.md D4 の表と一致
    - `MealPlanCellVisibleItems` は 1 週 × 1 時間帯 プリセットの `VisibleItems` と等しい (deprecate 中の
      整合性チェック)
    - 完了判定: `go test ./internal/config/...` が pass する

## 2. フロント側 mirror

- [x] 2.1 `frontend/src/components/mealplan/mealplan-config.ts` に `CellSizePreset` 型と
      `CELL_SIZE_PRESETS` (Go 側と同じ 9 プリセットの配列) を追加する。完了判定: `npm run type-check`
      が通る
- [x] 2.2 `mealplan-config.ts` に `pickCellPreset(periodWeeks: number, visibleMeals: number):
      CellSizePreset` を追加する。未定義組み合わせは最も密度が高いプリセット (4 週 × 3 時間帯) に
      fallback する。完了判定: `npm run type-check` が通る
- [x] 2.3 `mealplan-config.ts` の冒頭に「Tailwind JIT が拾えるよう事前列挙する class 一覧」
      コメントを追加し、`min-h-16` / `min-h-20` / `min-h-24` / `min-h-32` / `min-h-40` / `min-h-48` /
      `text-xs` / `text-sm` / `text-base` の 9 個をリテラルで記載する。完了判定: `npm run build:dev`
      が通り、後続タスクで各 class が実際に描画に効く

## 3. `MealPlanCell` の props 化

- [x] 3.1 `frontend/src/components/mealplan/meal-plan-cell.tsx` の Props に `preset:
      CellSizePreset` を追加する。完了判定: `npm run type-check` が通る
- [x] 3.2 `min-h-16` ハードコードを `preset.MinHeightRem` 相当の Tailwind class に置き換える
      (D1 で採用した「事前列挙 + switch」パターン。完了判定: `npm run type-check` が通り、目視で
      1 週 × 夕食のみのセルが `min-h-48` 相当の高さで描画される)
- [x] 3.3 レシピ名の `text-sm` を `preset.FontSizeClass` に置き換える。完了判定: `npm run type-check`
      が通り、目視で 1 週 × 1 時間帯 のとき `text-base` で描画される
- [x] 3.4 `MEALPLAN_DEFAULTS.mealPlanCellVisibleItems` の参照を `preset.VisibleItems` に置き換える。
      完了判定: `npm run type-check` が通る
- [x] 3.5 セル padding を `p-2` → `p-3` に上げる (期間 4 週のときは `p-2` に落とす)。完了判定:
      目視で 1 週 × 1 時間帯 のセル余白が広くなり、4 週表示 (実装は後続 proposal だが preset で
      指定される値) では狭く保たれる
- [x] 3.6 空セル (`min-h-16 rounded-md border border-dashed`) 側も `preset.MinHeightRem` に合わせて
      高さを揃える。完了判定: 目視で空セルと割当済セルの高さが等しい
- [x] 3.7 空セル中央の `＋` ボタンの文字サイズを `text-sm` → `text-2xl` 相当に上げ、色を
      `text-neutral-400` → `text-neutral-500` に濃くする。完了判定: 目視で空セルの `＋` が
      視認しやすくなる
- [x] 3.8 割当済セル右上の `＋` ボタンを `text-xs` から `text-base` に拡大、`p-1.5` の padding と
      `hover:bg-neutral-100 rounded` の hover ハイライトを付ける。完了判定: 目視で右上 `＋` が
      押しやすくなる (ヒットボックスがマウス移動で視覚的にわかる)

## 4. `WeekCalendar` の配線

- [x] 4.1 `frontend/src/components/mealplan/week-calendar.tsx` に、現在の期間 (実装上は 1 週固定)
      と `visibleMeals.size` から `pickCellPreset` を呼び出し、`preset` を各 `MealPlanCell` に渡す
      経路を追加する。完了判定: `npm run type-check` が通る
- [x] 4.2 Grid の `gridAutoRows` を preset 由来にする (現状は暗黙のため、`className` に `min-h-*` を
      通す実装なら `MealPlanCell` 側で完結。追加設定は不要ならこの task は削除できるので、実装時に
      判断)。完了判定: 目視で 1 週 × 1 時間帯 のときセルが均一に高くなる
- [x] 4.3 期間切替 UI は本 proposal のスコープ外だが、preset を「1 週固定」で決め打ちで渡している
      箇所にコメントで「後続 `add-mealplan-period-switch` で state から取る」と明記する。完了判定:
      grep で該当コメントが 1 箇所に存在する

## 5. ワイヤーフレーム更新

`docs/wireframes/README.md` の規約「構造変更は `Sxx_<name>.v2.drawio.svg` に切り出す (旧ワイヤーは残す)」
に従い、原本 `S01_meal_calendar.drawio.svg` は触らず、新規レイアウトは
`docs/wireframes/S01_meal_calendar.v2.drawio.svg` に描画する。product.md からの参照は原本のままにする
(v2 は本 proposal 用の追加ビジョン記録)。

- [x] 5.1 `docs/wireframes/S01_meal_calendar.v2.drawio.svg` に、1 週 × 夕食のみの
      レイアウト (推奨サイズ min-h-48 / text-base / 最大 4 品) を描画する。完了判定: SVG を開くと
      「① 既定：1 週 × 夕食のみ」ラベル付きの 1 週 × 1 時間帯 版が見える
- [x] 5.2 同 v2 ワイヤーに 1 週 × 3 時間帯 のレイアウト (min-h-32 / text-base / 最大 3 品) も並べて
      描画する。完了判定: SVG に「② 3 食表示：1 週 × 3 時間帯」ラベル付きの 2 状態目が並列表示される
- [x] 5.3 ワイヤー内テキスト (レシピ名、`＋`、`⋯` メニュー) を実際の UI と揃える (「肉じゃが」
      「みそ汁」など)。完了判定: 目視で spec の Scenario 例と符合する

## 6. ドキュメント更新

- [x] 6.1 `docs/product.md` §S01 のセル描画説明に「セル高さ / typography / 表示件数は
      期間 × 時間帯数で可変」の 1 行を追加する。完了判定: 目視で該当段落に追記されている
- [x] 6.2 `docs/product.md` §1 の可変値表に `MealPlanCellSizePresets` を 1 行追加する
      (source は `internal/config/defaults.go`、既定は 9 プリセット、UI 変更は S09 で対応)。完了判定:
      目視で表に行が追加されている

## 7. lint / format / テスト最終ゲート

- [x] 7.1 `task lint` を実行し、Go / Frontend の両方でエラーがないことを確認する。完了判定:
      exit code 0
- [x] 7.2 `task format` を実行し、`git diff` の差分が整形のみであることを確認する。完了判定:
      diff が整形差分に限られる
- [x] 7.3 `go test ./internal/... ./` を実行し、config を含む全パッケージのテストが pass することを
      確認する。完了判定: exit code 0
- [x] 7.4 `npm --prefix frontend run type-check && npm --prefix frontend run build:dev` を実行し、
      フロントの型チェックとビルドが通ることを確認する。完了判定: exit code 0
- [x] 7.5 `task test:e2e` を実行し、既存 E2E シナリオが引き続き pass することを確認する。
      完了判定: exit code 0
- [x] 7.6 `task dev` で起動し、S01 (1 週 × 夕食のみ) を開いて「セルが縦に大きく、レシピ名が
      `text-base` で描画される」ことを目視で確認する。完了判定: 起動画面のスクリーンショットで
      1 セルが画面の縦 1/4 以上を占める
      (注: CLAUDE.md の「動作確認の 2 階建て構成」に沿って、ユーザーが目視確認して良好と判断)
