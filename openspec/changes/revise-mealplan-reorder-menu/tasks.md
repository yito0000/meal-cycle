## 1. 時間帯全体のメニュー

- [x] 1.1 `frontend/src/components/mealplan/meal-menu.tsx` を更新する (design.md D1)
    - 「並び替え」の `DropdownMenuItem` と、その前の `DropdownMenuSeparator` を削除する
    - props から `onReorder` を削除する
    - 冒頭のコメントのメニュー項目の列挙から「並び替え」を外す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `frontend/src/components/mealplan/meal-plan-cell.tsx` を更新する
    - `MealAction` から `"reorder"` を外す
    - `MealMenu` に渡していた `onReorder` を削除する
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.3 `frontend/src/components/mealplan/week-calendar.tsx` の `handleMealAction` から `case "reorder"` の分岐を削除する。完了判定: `cd frontend && npm run type-check` が通る

## 2. 品のメニュー

- [x] 2.1 `frontend/src/components/mealplan/cell-item-menu.tsx` を更新する (design.md D2)
    - props に `position: number`、`cellItemCount: number`、`onMoveUp: () => void`、`onMoveDown: () => void` を足す
    - `cellItemCount >= 2` のときだけ、「この品のレシピ詳細を見る」の下に「上へ移動」「下へ移動」を出す
    - `position === 0` のとき「上へ移動」を `disabled` にする
    - `position === cellItemCount - 1` のとき「下へ移動」を `disabled` にする
    - 冒頭のコメントのメニュー項目の列挙に「上へ移動 / 下へ移動」を足す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/components/mealplan/meal-plan-cell.tsx` を更新する
    - `EntryAction` に `"move-up"` と `"move-down"` を足す
    - `CellItemMenu` に `position={entry.position}` と `cellItemCount={entries.length}` を渡す
    - `onMoveUp` / `onMoveDown` から `onEntryAction("move-up", entry)` / `onEntryAction("move-down", entry)` を呼ぶ
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.3 `frontend/src/components/mealplan/week-calendar.tsx` の `handleEntryAction` に分岐を足す (design.md D3)
    - `case "move-up"`：`reorderWithinCell(date, meal, entry.position, entry.position - 1)` を呼ぶ
    - `case "move-down"`：`reorderWithinCell(date, meal, entry.position + 1, entry.position)` を呼ぶ
    - `move-down` で 1 つ下の品を source にする理由 (`reorderWithinCell` は target の前に差し込む) を WHY のコメントで 1 行書く
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.4 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する
    - `pages.meal_plan.meal_menu.reorder` を削除する
    - `pages.meal_plan.reorder` (move_up / move_down / done) を削除する
    - `pages.meal_plan.item_menu.move_up`：「上へ移動」/ "Move up"
    - `pages.meal_plan.item_menu.move_down`：「下へ移動」/ "Move down"
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 3. E2E

- [x] 3.1 `frontend/e2e/mealplan-assign.spec.ts` に describe「時間帯内の並び替え」を追加する
    - 2 品目を上に動かすと 1 品目と入れ替わる
    - 先頭の品は上へ、末尾の品は下へ移動できない
    - 1 品だけのセルには移動の項目を出さない
    - 作り置きに属する品も移動できる (`copyLeftover` で月〜水曜の作り置きを作ってから、月曜に単独の品を先に置いた状態で確かめる)
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.2 `frontend/e2e/mealplan-assign.spec.ts` の describe「セル内の品表示」に Scenario を追加する
    - 時間帯全体の `⋯` メニューには動かない項目を置かない
    - 完了判定: `task test:e2e` で上記が pass する

## 4. ドキュメントと検証

- [x] 4.1 `docs/product.md` §4 S01 を更新する
    - 「セル内の操作」の品のメニューの項目列挙に「上へ移動 / 下へ移動」を足し、セルに 2 品以上あるときだけ出ることを書く
    - 「時間帯全体の操作」の項目列挙から「並び替え」を削除する
    - D&D の説明の「上下ボタン UI と同じ経路」を「品のメニューの上へ / 下へ移動と同じ経路」に改める
    - 完了判定: 差分を目視で確認する
- [x] 4.2 `docs/reviews/2026-09-25-design-ux-review.md` の R5 のチェックを付け、`revise-mealplan-reorder-menu` で対応した旨を追記する。完了判定: 差分を目視で確認する
- [x] 4.3 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 4.4 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 4.5 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 4.6 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 4.7 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 4.8 `openspec validate revise-mealplan-reorder-menu --strict` を実行する。完了判定: エラー 0 件
