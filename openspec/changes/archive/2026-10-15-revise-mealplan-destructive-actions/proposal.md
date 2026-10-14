## Why

S01 の「消える」操作は、ほかの画面と作法が違う (docs/reviews/2026-10-14-design-ux-review.md R30、R31)。

- S05 の削除は確認なしで実行し、通知の「元に戻す」で救える。S01 の品の削除と時間帯のクリアは即時に実行され、取り消せない
- S05 と S07 は結果を通知で知らせる。S01 は失敗をカレンダー上部の赤い帯に出し、次の操作が成功するまで残る
- 「作り置きから外す」は、作り置きの最初の日の行だけを残し、ほかの行を削除する。
  名前から受ける印象は「この行を切り離す」だが、実際には選んだ行のほうが消えることがある

3 つめは、挙動を変えずに名前を中身へ合わせる (利用者の判断)。
行が消える操作であることを名前だけで伝えきるのは難しいので、確認のダイアログで削除される行の数を見せる。

product.md と spec は、作り置きの行を動かす手順を「外して単独行にしてから D&D する 2 段操作」と説明している。
最初の日以外の行では、外した時点でその行が無くなるので、この説明は成り立たない。あわせて直す。

## What Changes

- 「この品を削除」と「時間帯全体をクリア」のあと、通知に「元に戻す」を出す。押すと、削除した品が元のセルの元の並び順に戻る
  - 作り置きへの属し方と「作った」印も、削除の前の状態に戻る
- S01 の操作の失敗を、赤い帯ではなく通知で知らせる
- 「作り置きから外す」を「作り置きをやめる」に改名する。挙動 (最初の日の行だけを残す) は変えない
- 「作り置きをやめる」を選ぶと、確認のダイアログを開く。最初の日の行だけが残ることと、削除される行の数を示す
- 作り置きの日をずらす手順の説明を、「作り置きをやめる → 残った行を D&D → 作り置きを広げる」に直す
- spec の誤りを直す (R34 のうち、この要件にかかる分)
  - シナリオ名「まとめ候補は同じ 時間帯・同じレシピ・±4 日以内だけを提示する」から「時間帯」を外す
  - 「1 行 移動」の要件から、サービスのメソッド名とエラーの定数名を外す
- **BREAKING** (内部 API)：品の削除と時間帯のクリアが、削除した行の内容を返すようになる

### Non-goals

- **「作り置きをやめる」の取り消し**：確認のダイアログで代える。やめたあとに戻す操作は作らない (design.md D3)
- **選んだ行だけを切り離す操作**：挙動は変えない。作り置きごと日をずらす操作 (R7) も扱わない
- **入れ替え、移動、広げる操作の取り消し**：消える行が無い、または同じ操作で戻せるので対象にしない
- **「他 N 件」に畳まれた品の操作**：`add-mealplan-cell-overflow-list` で扱う (R29)
- **エラーコードの対応表の一本化** (R27 の 2 点目)

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：
  - 「事後編集による作り置きの外し / まとめ」を「作り置きをやめる操作とまとめる操作」に差し替え、新しい名前と確認のダイアログに合わせる
  - 「1 行 移動 (ドラッグ&ドロップ)」の、作り置きの行を動かす手順の説明を直す
  - 「作り置きの編集と作った印」を「編集操作と作った印」に差し替え、元に戻した品の印の扱いを足す
  - 上の 2 つは、シナリオ名に古い操作名が入っているので要件ごと差し替える (MODIFIED ではシナリオ名を変えられない)
  - 「品の削除と時間帯クリアの取り消し」と「S01 の操作結果の通知」を足す

## Impact

### 新設ファイル

- `internal/domain/mealplan/removed.go`：`RemovedEntry` (削除した行の内容)
- `frontend/src/components/mealplan/dissolve-batch-dialog.tsx`：「作り置きをやめる」の確認のダイアログ
- `frontend/src/lib/undo-toast.ts`：「元に戻す」付きの通知の表示時間 `UNDO_TOAST_MS` (S05 と共有する)

### 更新ファイル

- `internal/storage/sqlite/mealplan_repo.go`
  - `Remove` と `ClearMeal` が `RemovedEntry` を返す
  - `RestoreEntries` を足す
- `internal/storage/sqlite/mealplan_repo_test.go`：`TestMealPlanRepo_RestoreEntries` を足し、`Remove` と `ClearMeal` の戻り値のケースを足す
- `internal/service/mealplan.go`：`RemoveEntry` と `ClearMeal` の戻り値を変え、`RestoreEntries` を足す。`mealPlanRepo` も合わせる
- `internal/service/mealplan_test.go`：フェイクのリポジトリを合わせ、`TestMealPlanService_RestoreEntries` を足す
- `internal/domain/mealplan/validate.go`：`ErrInvalidRestore` を足す
- `frontend/src/components/mealplan/week-calendar.tsx`
  - 赤い帯 (`errorState`) を消し、失敗を `toast.error` で出す
  - 削除とクリアの成功時に、「元に戻す」付きの通知を出す
  - `split-batch` を選んだら確認のダイアログを開く
- `frontend/src/components/mealplan/error-messages.ts`：`mealplan: invalid restore` の対応を足す
- `frontend/src/components/fridge/use-fridge.ts` / `frontend/src/routes/fridge.tsx`：`UNDO_TOAST_MS` を `frontend/src/lib/undo-toast.ts` から読む
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`
  - `pages.meal_plan.item_menu.split_batch` の文言を「作り置きをやめる」/ "Stop batching" に変える
  - `pages.meal_plan.dissolve.*`、`pages.meal_plan.toast.*`、`pages.meal_plan.errors.invalid_restore` を足す
- `frontend/e2e/bindings-shim/mealplanservice.ts`：`RemoveEntry` と `ClearMeal` の戻り値を合わせ、`RestoreEntries` を足す
- `frontend/e2e/mealplan-assign.spec.ts`：describe「品の削除と時間帯クリアの取り消し」と「S01 の操作結果の通知」を足す
- `frontend/e2e/mealplan-batch.spec.ts`：describe「事後編集による作り置きの外し / まとめ」を「作り置きをやめる操作とまとめる操作」に改名し、
  やめる操作の `test.skip` を外して実装する
- `frontend/e2e/mealplan-cooked.spec.ts`：describe「作り置きの編集と作った印」を「編集操作と作った印」に改名し、「元に戻した品は印を保つ」を足す
- `docs/product.md`：§4 S01 の品のメニューの項目名、D&D の 2 段操作の説明、削除とクリアの取り消し、エラーの通知を直す。§7 の決定事項に追記する
- `docs/reviews/2026-10-14-design-ux-review.md`：R30、R31 にチェックを入れる。`docs/reviews/2026-09-25-design-ux-review.md` の R9 にもチェックを入れる
- `openspec/specs/mealplan/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`RemovedEntry` と `RestoreEntries` が増える

### migration

- なし

### 前提と順序

- `fix-mealplan-renumber-after-reorder` が先に必要である。元に戻すときに後ろの品を 1 つずつ下げる処理は、そこで入れる退避の方式を使う
- `revise-mealplan-spread-api` のあとに apply する。そちらが `week-calendar.tsx` の `errorState` の `params` を消すので、赤い帯の撤去が単純になる

### 後続 proposal への引き継ぎ

- `add-mealplan-cell-overflow-list`：畳まれた品の削除も、同じ「元に戻す」の経路に乗る
- `add-mealplan-entry-id`：`RemovedEntry` に行の ID を足し、元に戻した行が同じ ID を持つようにする
- R7：作り置きごと日をずらす操作を入れるときに、「やめる → 動かす → 広げる」の 3 手を 1 手にできる
