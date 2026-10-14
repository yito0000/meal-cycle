## 1. ドメイン層とリポジトリ層

- [x] 1.1 `internal/domain/mealplan/removed.go` (新設) に `RemovedEntry` を定義する (design.md D1)。完了判定: `go build .` が通る
- [x] 1.2 `internal/domain/mealplan/validate.go` (更新) に `ErrInvalidRestore = errors.New("mealplan: invalid restore")` を足す。完了判定: `go build .` が通る
- [x] 1.3 `internal/storage/sqlite/mealplan_repo.go` (更新) の `Remove` を、消す前に行の内容と印の日時を読み、`RemovedEntry` を返す形に変える。
  完了判定: 2.1 と合わせて `go build .` が通る
- [x] 1.4 `internal/storage/sqlite/mealplan_repo.go` (更新) の `ClearMeal` を、消した行を並び順の昇順で `[]RemovedEntry` として返す形に変える。
  完了判定: 2.1 と合わせて `go build .` が通る
- [x] 1.5 `internal/storage/sqlite/mealplan_repo.go` (更新) に `RestoreEntries(ctx, entries []mealplan.RemovedEntry) error` を書く (design.md D2)。
  完了判定: `go build .` が通る
- [x] 1.6 `internal/storage/sqlite/mealplan_repo_test.go` (更新) にテストを足す
    - Remove should return the removed row with its batch id and createdAt
    - Remove should return cookedAt when the removed row carried a cooked mark
    - ClearMeal should return all removed rows ordered by position
    - ClearMeal should return an empty slice when the meal has no rows
    - RestoreEntries should put a row back at its original position and shift later rows down
    - RestoreEntries should append at the end when the original position exceeds the current count
    - RestoreEntries should restore all rows of a cleared meal in the original order
    - RestoreEntries should keep the original batch id so the row rejoins its batch
    - RestoreEntries should bring back the cooked mark with the original cookedAt when the mark was pruned
    - RestoreEntries should not duplicate the cooked mark when other rows of the batch still carry it
    - RestoreEntries should succeed in a cell whose rows were reordered before the removal
    - RestoreEntries should return ErrUnknownRecipe and change nothing when the recipe row does not exist
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'TestMealPlanRepo_(Remove|ClearMeal|RestoreEntries)'` が pass する

## 2. サービス層

- [x] 2.1 `internal/service/mealplan.go` (更新) の `mealPlanRepo`、`RemoveEntry`、`ClearMeal` の戻り値を `RemovedEntry` / `[]RemovedEntry` に合わせる。完了判定: `go build .` が通る
- [x] 2.2 `internal/service/mealplan.go` (更新) に `RestoreEntries(ctx, entries []mealplan.RemovedEntry) error` を書く (design.md D2 の検証を含む)。
  完了判定: `go build .` が通る
- [x] 2.3 `internal/service/mealplan_test.go` (更新) のフェイクのリポジトリを、戻り値の変更と `RestoreEntries` に合わせる。完了判定: `go vet ./internal/service/` が通る
- [x] 2.4 `internal/service/mealplan_test.go` (更新) に `TestMealPlanService_RestoreEntries` を足す
    - should pass valid entries to the repository
    - should do nothing when entries is empty
    - should return ErrInvalidRestore when recipeId or batchId is empty
    - should return ErrInvalidRestore when createdAt is not RFC3339
    - should return ErrInvalidRestore when cookedAt is neither empty nor RFC3339
    - should return ErrInvalidDate, ErrInvalidMeal or ErrInvalidPosition when the address is invalid
    - 完了判定: `go test ./internal/service/ -run TestMealPlanService_RestoreEntries` が pass する
- [x] 2.5 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: `frontend/bindings/` に `RemovedEntry` と `RestoreEntries` がある

## 3. フロントエンド

- [x] 3.1 `frontend/src/lib/undo-toast.ts` (新設) に `UNDO_TOAST_MS = 5000` を置く。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/components/fridge/use-fridge.ts` (更新) から `UNDO_TOAST_MS` を消し、`frontend/src/routes/fridge.tsx` (更新) の import を
  `@/lib/undo-toast` に変える。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.3 `frontend/e2e/bindings-shim/mealplanservice.ts` (更新) の `RemoveEntry` と `ClearMeal` の戻り値を合わせ、`RestoreEntries` を足す。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.4 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` (更新) を直す
    - `pages.meal_plan.item_menu.split_batch`：「作り置きをやめる」/ "Stop batching"
    - `pages.meal_plan.dissolve.title`：「作り置きをやめますか」
    - `pages.meal_plan.dissolve.description`：「最初の日の行だけを残し、ほかの {{count}} 行を削除します。選んだ行が残るとは限りません。」
    - `pages.meal_plan.dissolve.confirm` / `cancel`：「やめる」/「キャンセル」
    - `pages.meal_plan.toast.removed`：「「{{name}}」を削除しました」
    - `pages.meal_plan.toast.cleared`：「{{count}} 品を削除しました」
    - `pages.meal_plan.toast.undo`：「元に戻す」
    - `pages.meal_plan.toast.restore_failed`：「元に戻せませんでした」
    - `pages.meal_plan.errors.invalid_restore`：「元に戻す内容が正しくありません」
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっている
- [x] 3.5 `frontend/src/components/mealplan/error-messages.ts` (更新) に `["mealplan: invalid restore", "pages.meal_plan.errors.invalid_restore"]` を足す。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.6 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `errorState` と赤い帯の `<p>` を消し、`withHandlers` の失敗時を `toast.error(t(errorI18nKey(err)))` にする
  (design.md D5)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.7 `frontend/src/components/mealplan/week-calendar.tsx` (更新) に `restoreMut` (`MealPlanService.RestoreEntries`) を足す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.8 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `removeMut` の成功時に、`toast.removed` を `元に戻す` 付きで出す。
  `元に戻す` は、返ってきた `RemovedEntry` を `restoreMut` に渡す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.9 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `clearMut` の成功時に、消した品が 1 つ以上あれば `toast.cleared` を `元に戻す` 付きで出す。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.10 `frontend/src/components/mealplan/dissolve-batch-dialog.tsx` (新設) に確認のダイアログを書く (props: `open`、`removedCount`、`onConfirm`、`onCancel`)。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.11 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `case "split-batch"` を、ダイアログを開く形に変える。
  実行を選んだら `splitMut` を呼び、`removedCount` には `entry.batchMemberCount - 1` を渡す (design.md D3)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.12 `frontend/src/components/mealplan/cell-item-menu.tsx` と `meal-plan-cell.tsx` (更新) のコメントにある「作り置きから外す」を「作り置きをやめる」に直す。
  完了判定: `grep -rn "作り置きから外す" frontend/src` が 0 件

## 4. E2E

- [x] 4.1 `frontend/e2e/mealplan-assign.spec.ts` (更新) に describe「品の削除と時間帯クリアの取り消し」を足す
    - 削除した品を元に戻す
    - 削除すると通知に元に戻すが表示される
    - クリアした時間帯を元に戻す
    - 作り置きの 1 行を削除して元に戻すと同じ作り置きに戻る
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 4.2 `frontend/e2e/mealplan-assign.spec.ts` (更新) に describe「S01 の操作結果の通知」を足す
    - 失敗した操作は通知で知らされる (レシピ選択ダイアログを開いたあと、`deleteRecipe` で API からレシピを削除してから選ぶ)
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 4.3 `frontend/e2e/mealplan-batch.spec.ts` (更新) の describe「事後編集による作り置きの外し / まとめ」を「作り置きをやめる操作とまとめる操作」に改名し、中身を直す
    - 作り置きをやめると最初の日だけが残る (`test.skip` を外して実装する。旧名「作り置きから外すと起点日だけが残る」)
    - 作り置きをやめる前に削除される行の数を確認する (新規)
    - 確認でキャンセルすると作り置きは変わらない (新規)
    - 単独の行には「作り置きをやめる」項目が出ない (新規)
    - `test.skip("単独作り置きの行に対して外す操作は何もしない")` を消す (サービス層のテストで確かめているシナリオで、画面からは到達できない)
    - 完了判定: `task test:e2e` で上記が pass し、`cd frontend && npm run check:scenarios` が通る
- [x] 4.4 `frontend/e2e/mealplan-cooked.spec.ts` (更新) の describe「作り置きの編集と作った印」を「編集操作と作った印」に改名する。
  完了判定: `cd frontend && npm run check:scenarios` が通る
- [x] 4.5 `frontend/e2e/mealplan-cooked.spec.ts` (更新) の describe「編集操作と作った印」に「元に戻した品は印を保つ」を足す。完了判定: `task test:e2e` で上記が pass する

## 5. ドキュメント

- [x] 5.1 `docs/product.md` (更新) §4 S01 の品のメニューの項目名を「作り置きをやめる」に直し、確認のダイアログが開くことを書く。完了判定: 差分を目視で確認する
- [x] 5.2 `docs/product.md` (更新) §4 S01 の D&D の「作り置きから外すで単独行にしてから D&D する 2 段操作」を、
  「作り置きをやめる → 残った行を D&D → 作り置きを広げる」に直す。完了判定: 差分を目視で確認する
- [x] 5.3 `docs/product.md` (更新) §4 S01 に、削除とクリアの「元に戻す」と、失敗を通知で知らせることを足す。完了判定: 差分を目視で確認する
- [x] 5.4 `docs/product.md` (更新) §7 の決定事項に「S01 の削除とクリアは確認なし + 元に戻す、作り置きをやめるは事前の確認」を理由つきで足す (design.md D3)。
  完了判定: 差分を目視で確認する
- [x] 5.5 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R30 と R31 にチェックを入れ、R34 に対応した範囲を追記する。完了判定: 差分を目視で確認する
- [x] 5.6 `docs/reviews/2026-09-25-design-ux-review.md` (更新) の R9 にチェックを入れ、`revise-mealplan-destructive-actions` で対応した旨を追記する。完了判定: 差分を目視で確認する

## 6. 検証

- [x] 6.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 6.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 6.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 6.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 6.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 6.6 `openspec validate revise-mealplan-destructive-actions --strict` を実行する。完了判定: エラー 0 件
