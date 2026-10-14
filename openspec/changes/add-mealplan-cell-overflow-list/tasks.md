## 1. 品行の切り出し

- [ ] 1.1 `frontend/src/components/mealplan/meal-item-row.tsx` (新設) に `MealItemRow` を書く (design.md D3)
    - `meal-plan-cell.tsx` の品行の JSX (チェックボックス、レシピ名のボタン、`CellItemMenu`、ドラッグの開始と終了、ghost の色付け) をそのまま移す
    - 品のメニューの開閉 (`open` の state と、閉じたときにレシピ名のボタンへフォーカスを戻す処理) を部品の中に持つ
    - `draggable` が false のときは、`onDragStart` / `onDragEnd` / `onDrop` を付けない
    - 完了判定: `cd frontend && npm run type-check` が通る
- [ ] 1.2 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) の品行の描画を `MealItemRow` に置き換え、`menuPosition`、`menuButtonRef`、`pressedCheckboxRef` を消す。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 1.3 `task test:e2e` を実行する。完了判定: describe「セル内の品表示」「時間帯内の並び替え」「作った印」の既存テストが pass する (切り出しで挙動が変わっていない)

## 2. 一覧

- [ ] 2.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` (更新) に `pages.meal_plan.cell.overflow_list_label`
  (「{{date}} {{meal}} の品」/ "Dishes for {{date}} {{meal}}") を足す。完了判定: 両言語で同じキーがそろっている
- [ ] 2.2 `frontend/src/components/mealplan/cell-overflow-list.tsx` (新設) に `CellOverflowList` を書く (design.md D1、D2)
    - `PopoverTrigger` は「他 N 件」の `<button>` にする (文言は既存の `pages.meal_plan.cell.more_items`)
    - `PopoverContent` に、セルの全品を position 順に `MealItemRow` (`draggable={false}`、`fontClass="text-sm"`) で並べる
    - `PopoverContent` に `aria-label` として `overflow_list_label` を付ける
    - 完了判定: `cd frontend && npm run type-check` が通る
- [ ] 2.3 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) の「他 N 件」の `<div>` を `CellOverflowList` に置き換える。
  `overflow > 0` のときだけ描く (design.md D5)。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 2.4 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) の冒頭のコメントの「品数超過は『他 N 件』で折り畳む」に、押すと全品の一覧が開くことを 1 行で足す。
  完了判定: 差分を目視で確認する

## 3. E2E

- [ ] 3.1 `frontend/e2e/helpers.ts` (更新) に `openOverflowList(page, date, meal)` (セルの「他 N 件」を押し、一覧の locator を返す) を足す。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 3.2 `frontend/e2e/mealplan-item-row.spec.ts` (更新) の describe「セル内の品表示」にシナリオを足す (4 週表示に切り替え、API で 3 品を割り当てる)
    - 「他 N 件」を押すとセルの全品の一覧が開く
    - 畳まれた品に作った印を付けられる
    - 畳まれた品のメニューから削除できる
    - 一覧は Esc で閉じる
    - 完了判定: `task test:e2e` で上記が pass する

## 4. ドキュメントと検証

- [ ] 4.1 `docs/product.md` (更新) §4 S01 の「超過は『他 N 件』で省略」に、押すとそのセルの全品の一覧が開き、作った印と品のメニューを使えることを足す。
  完了判定: 差分を目視で確認する
- [ ] 4.2 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R29 にチェックを入れ、`add-mealplan-cell-overflow-list` で対応した旨を追記する。完了判定: 差分を目視で確認する
- [ ] 4.3 `task dev` で起動し、1 週表示で単独の品を別のセルへドラッグする。完了判定: ghost が薄い緑、元の位置が灰色で表示され、品が移動することを目視で確認する
- [ ] 4.4 `task dev` で 4 週表示にし、一覧の品のメニューから「入れ替え」を選ぶ。完了判定: レシピ選択ダイアログが開き、一覧が閉じることを目視で確認する (design.md D4)
- [ ] 4.5 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [ ] 4.6 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [ ] 4.7 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [ ] 4.8 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [ ] 4.9 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [ ] 4.10 `openspec validate add-mealplan-cell-overflow-list --strict` を実行する。完了判定: エラー 0 件
