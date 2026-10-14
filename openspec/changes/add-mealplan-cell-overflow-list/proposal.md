## Why

S01 のセルは、表示件数の上限を超えた品を「他 N 件」という文字にまとめる。
この文字は押せず、畳まれた品を開く手段が無い (docs/reviews/2026-10-14-design-ux-review.md R29)。

4 週表示の上限は 2 品である。
主菜、副菜、汁物の 3 品を組むと、3 品目には「作った」印も、削除も、レシピ詳細も使えない。
期間を 1 週に切り替えれば触れるが、それを示す手がかりは画面に無い。
献立を俯瞰する 4 週表示で、俯瞰した先の品に手が届かないことになる。

## What Changes

- 「他 N 件」を押せるボタンにする。押すと、そのセルの全品を並び順どおりに並べた一覧を、セルの近くに開く
- 一覧の品行は、セルの中の品行と同じ形にする (作った印のチェックボックス + レシピ名)。レシピ名を押すと、同じ品のメニューが開く
- 一覧の中の品はドラッグできない
- 一覧は、Esc、一覧の外のクリック、セルの品数が上限以下になったとき、のいずれかで閉じる

### Non-goals

- **表示件数の上限の変更**：`MealPlanCellSizePresets` の値は変えない
- **一覧の中でのドラッグ&ドロップ**：一覧は浮いた面なので、別のセルへ運ぶ操作と相性が悪い。並び替えは品のメニューの「上へ移動」「下へ移動」で行える
- **一覧からの品の追加**：品の追加はセル右上の `＋` で行う。一覧には `＋` を置かない
- **時間帯全体の `⋯` メニュー**：一覧には置かない。セル右下の `⋯` を使う

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：「セル内の品表示」に、「他 N 件」から開く一覧を足す

## Impact

### 新設ファイル

- `frontend/src/components/mealplan/meal-item-row.tsx`：品行 (チェックボックス + レシピ名 + 品のメニュー) を `meal-plan-cell.tsx` から切り出す
- `frontend/src/components/mealplan/cell-overflow-list.tsx`：「他 N 件」のボタンと、全品の一覧

### 更新ファイル

- `frontend/src/components/mealplan/meal-plan-cell.tsx`：品行の描画を `MealItemRow` に置き換え、「他 N 件」の `<div>` を `CellOverflowList` に置き換える
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.meal_plan.cell.overflow_list_label` (一覧の読み上げ用の名前) を足す
- `frontend/e2e/helpers.ts`：`openOverflowList(page, date, meal)` を足す
- `frontend/e2e/mealplan-item-row.spec.ts`：describe「セル内の品表示」に一覧のシナリオを足す
- `docs/product.md`：§4 S01 の「超過は『他 N 件』で省略」に、押すと一覧が開くことを足す
- `docs/reviews/2026-10-14-design-ux-review.md`：R29 にチェックを入れる
- `openspec/specs/mealplan/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- なし。Go 側とバインディングは変えない

### migration

- なし

### 前提と順序

- `revise-mealplan-destructive-actions` のあとに apply する。一覧からの削除が「元に戻す」の通知に乗り、一覧の中の「作り置きをやめる」が確認のダイアログを開く
- `add-mealplan-entry-id` より先に apply する。そちらが品行の `key` と操作の引数を ID に変えるので、品行を 1 つの部品にまとめてから入れる

### 後続 proposal への引き継ぎ

- `add-mealplan-entry-id`：`MealItemRow` の `key` と `DragSource` を行の ID に変える
- R8 (作り置きの枠色の重なり)：枠色の決め方を変えるときは `MealItemRow` の 1 か所で済む
