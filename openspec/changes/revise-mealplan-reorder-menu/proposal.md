## Why

S01 の時間帯右下の `⋯` メニューには「並び替え」がある。
押すと並び替えの UI は開かず、汎用のエラー (「処理に失敗しました」) が出る。
利用者から見ると、機能が未実装なのではなく、アプリが壊れているように見える
(docs/reviews/2026-09-25-design-ux-review.md R5)。

並び替えそのものは、同じセルの中で品をドラッグ&ドロップすればできる。
ただし、作り置きに属する品はドラッグできないので、作り置きの品をセルの中で上下に動かす手段は無い。
「並び替え」の項目を外すだけでは、この品の並びを変えられないまま残る。

## What Changes

- S01 の時間帯右下の `⋯` メニューから「並び替え」を外す。メニューは「時間帯全体をクリア」と「時間帯全体を連続 N 日コピー」の 2 つになる
- 品のメニューに「上へ移動」「下へ移動」を足す
  - セルに 2 品以上あるときだけ表示する
  - 先頭の品では「上へ移動」を、末尾の品では「下へ移動」を押せない状態にする
  - 作り置きに属する品でも使える (ドラッグできない品の並びを変える手段になる)
- 「セル内の品表示」の要件から、各メニューの項目の列挙を外す。項目はメニューを定める各要件に任せる

### Non-goals

- **作り置きの品のドラッグ**：作り置き全体を掴んでずらす操作は R7 で扱う。この proposal ではドラッグの挙動を変えない
- **「他 N 件」に畳まれた品の並び替え**：畳まれた品は画面に出ないので、メニューからも動かせない。セルの表示件数の仕組みは変えない
- **時間帯全体の `⋯` メニューの他の項目**：R6 (「連続 N 日コピー」を「作り置きを広げる」に寄せる) はこの proposal に含めない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：
  - 「セル内の品表示」から時間帯全体の `⋯` メニューの「並び替え」を外し、メニュー項目の列挙をやめる
  - 「時間帯内の並び替え」を、品のメニューの「上へ移動」「下へ移動」で並びを変える形に改める

## Impact

### 新設ファイル

- なし

### 更新ファイル

- `frontend/src/components/mealplan/meal-menu.tsx`：「並び替え」の項目と `onReorder` を削除する
- `frontend/src/components/mealplan/cell-item-menu.tsx`：「上へ移動」「下へ移動」の項目を追加する
- `frontend/src/components/mealplan/meal-plan-cell.tsx`：`MealAction` から `"reorder"` を外し、`EntryAction` に `"move-up"` と `"move-down"` を足す
- `frontend/src/components/mealplan/week-calendar.tsx`：`handleMealAction` の `reorder` 分岐を削除し、`handleEntryAction` から既存の `reorderWithinCell` を呼ぶ
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.meal_plan.meal_menu.reorder` と、どこからも参照されていない
  `pages.meal_plan.reorder` (上へ / 下へ / 完了) を削除し、`pages.meal_plan.item_menu.move_up` / `move_down` を追加する
- `frontend/e2e/mealplan-assign.spec.ts`：describe「時間帯内の並び替え」を追加する
- `docs/product.md`：§4 S01 の「セル内の操作」と「時間帯全体の操作」を更新する
- `openspec/specs/mealplan/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- なし。Go 側とバインディングは変えない (既存の並び替え API を使う)

### migration

- なし

### 前提と順序

- 進行中の `revise-recipe-detail-page` が `week-calendar.tsx` の `view-detail` 分岐を書き換えている。
  そちらを先に archive してから apply する
- `add-mealplan-cooked-mark` も品のメニューに項目を足す。どちらを先に入れてもよいが、後から入れる側で `cell-item-menu.tsx` の項目順を合わせる

### 後続 proposal への引き継ぎ

- R6：時間帯全体の「連続 N 日コピー」を「作り置きを広げる」のダイアログに寄せる
- R7：作り置きの品を掴んだら作り置き全体をずらす
