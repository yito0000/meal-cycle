## 1. ドメイン層

- [x] 1.1 `internal/domain/mealplan/entry.go` を更新し、`MealRank(m Meal) int` を足す (朝食 0、昼食 1、夕食 2、それ以外 3)。
  `go build ./internal/...` が通ることで確かめる
- [x] 1.2 `internal/domain/mealplan/spread.go` を更新し、`SpreadResult` struct を足す。
  フィールドは `AddedEntryIDs []string` (`json:"addedEntryIds"`) と `Removed []RemovedEntry` (`json:"removed"`)。`go build ./internal/...` で確かめる
- [x] 1.3 `internal/domain/mealplan/validate.go` を更新し、`ErrArchivedRecipe = errors.New("mealplan: archived recipe")` を足す。`go build ./internal/...` で確かめる
- [x] 1.4 `internal/domain/mealplan/spread_test.go` に `ValidateSpreadTargets` のケースを足す (更新)。足した時点で 1 件目と 2 件目が落ちることを
  `go test ./internal/domain/mealplan/` で確かめる
  - 夕食が起点で、同じ日の昼食を指定すると `ErrInvalidSpreadTarget` を返す
  - 夕食が起点で、同じ日の朝食を指定すると `ErrInvalidSpreadTarget` を返す
  - 昼食が起点で、同じ日の夕食を指定すると通る
  - 夕食が起点で、翌日の朝食を指定すると通る
- [x] 1.5 `internal/domain/mealplan/spread.go` の `ValidateSpreadTargets` を更新し、`tg.Date == sourceDate && MealRank(tg.Meal) <= MealRank(sourceMeal)` を
  `ErrInvalidSpreadTarget` で拒否する (起点セル自身の判定もこの条件に含める)。関数のコメントを「起点より後のセル」に直す。1.4 のテストが通ることで確かめる

## 2. リポジトリ層

- [x] 2.1 `internal/storage/sqlite/mealplan_repo.go` の `mealRank` を消し、呼び出し元を `mealplan.MealRank` に置き換える。
  `go test ./internal/storage/sqlite/` が通ることで確かめる
- [x] 2.2 `internal/storage/sqlite/mealplan_repo_test.go` の `TestMealPlanRepo_Spread` にテストを足す (更新)。足した時点で落ちることを
  `go test ./internal/storage/sqlite/ -run TestMealPlanRepo_Spread` で確かめる
  - 追加で広げると、`AddedEntryIDs` に追加した行の ID が対象セルの日付と時間帯の順で入り、`Removed` は空である
  - 追加で、同じレシピがすでにあるセルを飛ばしたときは、そのセルの ID は `AddedEntryIDs` に入らない
  - 上書きで広げると、`Removed` に消した行が作った印の日時つきで入る
  - 起点のレシピが削除済みなら `ErrArchivedRecipe` を返し、どのセルも変わらない
- [x] 2.3 `internal/storage/sqlite/mealplan_repo.go` の `Spread` を更新し、戻り値を `(mealplan.SpreadResult, error)` にする。
  起点の行を読んだ直後に `recipes.archived_at` を確かめて `ErrArchivedRecipe` を返し、上書きの `DELETE` の前に `loadRemovedEntries` で消す行を読む。
  2.2 のテストが通ることで確かめる
- [x] 2.4 `internal/storage/sqlite/mealplan_repo_test.go` に `TestMealPlanRepo_UndoSpread` を足す (更新)。`go test ./internal/storage/sqlite/ -run TestMealPlanRepo_UndoSpread` で落ちることを確かめる
  - 追加で広げた結果を渡すと、追加した行が消え、起点の行だけが残る
  - 上書きで広げた結果を渡すと、追加した行が消え、消した 2 品が元の並び順で戻る
  - 上書きで消えた品の作った印が戻る
  - 追加した行の 1 つを先に削除しておいても、残りの行が消え、消した品が戻る
  - 消した品と同じ ID の行がすでにあると `ErrInvalidRestore` を返し、追加した行も消えない
- [x] 2.5 `internal/storage/sqlite/mealplan_repo.go` に `UndoSpread(ctx, result mealplan.SpreadResult) error` を足す。
  1 トランザクションで、残っている追加行を `DELETE` してセルごとに `renumberMeal` で詰め、`Removed` を `RestoreEntries` と同じ並びで `restoreEntryTx` に通し、
  最後に `pruneCookedMarks` を呼ぶ。2.4 のテストが通ることで確かめる
- [x] 2.6 `internal/storage/sqlite/mealplan_repo_test.go` の `TestMealPlanRepo_CopyLeftover` にテストを足す (更新)
  - 起点の時間帯に削除済みレシピの品があると `ErrArchivedRecipe` を返し、どの日にも行が増えない
  - 削除済みレシピの品が無ければ従来どおりコピーされる
- [x] 2.7 `internal/storage/sqlite/mealplan_repo.go` の `CopyLeftover` を更新し、トランザクションの最初に `sourceEntries` のレシピがすべて未削除かを確かめる。
  2.6 のテストが通ることで確かめる

## 3. サービス層

- [x] 3.1 `internal/service/mealplan.go` を更新し、`RestoreEntries` の入力検証を `validateRemovedEntries(entries []mealplan.RemovedEntry) error` に切り出す。
  `go test ./internal/service/ -run TestMealPlanService_RestoreEntries` が通ることで確かめる
- [x] 3.2 `internal/service/mealplan.go` の `mealPlanRepo` を更新し、`Spread` の戻り値を `(mealplan.SpreadResult, error)` にして `UndoSpread` を足す。
  `internal/service/mealplan_test.go` の `fakeMealPlanRepo` も合わせる (更新)。`go build ./internal/...` で確かめる
- [x] 3.3 `internal/service/mealplan_test.go` の `TestMealPlanService_SpreadEntry` にテストを足し、`TestMealPlanService_UndoSpread` を新設する (更新)
  - should return the added entry ids when spreading succeeds
  - should reject a target in an earlier meal of the same day
  - should reject an undo whose removed entry has an invalid date
  - should reject an undo whose removed entry has an empty id
  - should pass the result to the repository when the undo input is valid
  - should do nothing when the undo result is empty
- [x] 3.4 `internal/service/mealplan.go` を更新し、`SpreadEntry` の戻り値を `(mealplan.SpreadResult, error)` にし、
  `UndoSpread(ctx, result mealplan.SpreadResult) error` を足す (`Removed` を `validateRemovedEntries` で確かめ、`AddedEntryIDs` の空文字を拒否する)。
  3.3 のテストが通ることで確かめる
- [x] 3.5 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/bindings/` に `SpreadResult` と `UndoSpread` が出ることを確かめる
- [x] 3.6 `go test ./cmd/...` を実行し、`cmd/e2e-server/dispatch_test.go` の `SpreadEntry` のテストが通ることを確かめる。
  落ちた場合は応答の形 (`SpreadResult` の JSON) に合わせて同ファイルを直す

## 4. フロントエンド

- [x] 4.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新し、次のキーを足す / 改める。`npm run type-check` で確かめる
  - `pages.meal_plan.toast.spread` (「{{cells}} セルに広げました」)
  - `pages.meal_plan.toast.spread_overwritten` (「{{cells}} セルに広げ、{{replaced}} 品を置き換えました」)
  - `pages.meal_plan.errors.archived_recipe` (「削除済みのレシピは広げられません」)
  - `pages.meal_plan.spread.earlier_meal_cell_label` (「起点より前の時間帯」)
  - `pages.meal_plan.spread.load_failed` (「格子の献立を読み込めませんでした」)
  - `pages.meal_plan.cell.cooked` を「{{name}}を作った」に改める
- [x] 4.2 `frontend/src/components/mealplan/error-messages.ts` を更新し、`["mealplan: archived recipe", "pages.meal_plan.errors.archived_recipe"]` を足す。
  あわせて、`MergeBatch` が `ErrEntryNotFound` を返すとした古いコメントを消す。`npm run type-check` で確かめる
- [x] 4.3 `frontend/src/components/mealplan/spread-batch-dialog.tsx` を更新し、`existingEntries` の prop を消して、
  `useQuery({ queryKey: ["mealplan", "spread-grid", source.date], queryFn: () => MealPlanService.ListPeriod(source.date, 1) })` で取得した行を
  `occupancy` と `sameBatchCells` に使う。取得が終わるまでは格子のチェックボックスを `disabled` にし、失敗したら `onOpenChange(false)` してエラーを通知する。
  `npm run type-check` で確かめる
- [x] 4.4 `frontend/src/components/mealplan/spread-batch-dialog.tsx` を更新し、起点日の、`MEAL_RENDER_ORDER` の添字が起点より小さいセルを、
  起点セルと同じ選択不可の印 (`earlier_meal_cell_label`) で描く。`npm run type-check` で確かめる
- [x] 4.5 `frontend/src/components/mealplan/spread-batch-dialog.tsx` を更新し、`origin_hint` の `date` と `cell_checkbox_label` の `date` に `formatRow(date)` を渡す。
  `npm run type-check` で確かめる
- [x] 4.6 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、`spreadMut` の `onSuccess` で、置き換えた品があれば `toast.spread_overwritten`、
  無ければ `toast.spread` の通知を `UNDO_TOAST_MS` のあいだ出し、`元に戻す` で `MealPlanService.UndoSpread(result)` を呼ぶ mutation を足す
  (失敗時は `restore_failed` を見出しにする)。`SpreadBatchDialog` に `existingEntries` を渡すのをやめる。`npm run type-check` で確かめる
- [x] 4.7 `frontend/src/components/mealplan/meal-item-row.tsx` を更新し、チェックボックスの `aria-label` を `t("pages.meal_plan.cell.cooked", { name: entry.recipeName })` にする。
  `npm run type-check` で確かめる
- [x] 4.8 `frontend/e2e/bindings-shim/mealplanservice.ts` を更新し、`SpreadEntry` の戻り値を `call<mealplan$0.SpreadResult>` にして、`UndoSpread` を足す。
  `npm run type-check` で確かめる
- [x] 4.9 `frontend/e2e/mealplan-cooked.spec.ts` と `frontend/e2e/mealplan-item-row.spec.ts` を更新し、
  `getByRole("checkbox", { name: t.cell.cooked })` を `t.cell.cooked.replace("{{name}}", レシピ名)` で組み立てた名前に置き換える。`npm run type-check` で確かめる

## 5. E2E

- [x] 5.1 `frontend/e2e/mealplan-batch.spec.ts` の describe「作り置きを広げる操作」に次のテストを足す (更新)。`task test:e2e` で通ることを確かめる
  - 起点より前の時間帯のセルは選べない
  - 表示期間の外のセルでも既存割当が示される
  - 起点の日付は格子の行見出しと同じ表記で示される
- [x] 5.2 `frontend/e2e/mealplan-batch.spec.ts` に describe「作り置きを広げる操作の取り消し」を足す (更新)。`task test:e2e` で通ることを確かめる
  - 広げた品を元に戻す
  - 上書きで消えた品を元に戻す
  - 上書きで消えた品の作った印が戻る
  - 広げると通知にセルの数と置き換えた品の数が示される
- [x] 5.3 `frontend/e2e/mealplan-batch.spec.ts` に describe「削除済みレシピの品は広げられない」を足す (更新)。`task test:e2e` で通ることを確かめる
  - 削除済みレシピの品を広げようとすると通知で知らされる
- [x] 5.4 `frontend/e2e/mealplan-cooked.spec.ts` に describe「作った印のチェックボックスの読み上げ名」を足す (更新)。`task test:e2e` で通ることを確かめる
  - チェックボックスの読み上げ名でどの品かわかる

サービス層とリポジトリ層で確かめるシナリオ (E2E には含めない):

- 「起点より前の時間帯へ広げる要求は拒否される」(1.4、3.3)
- 「削除済みレシピを含む時間帯はコピーされない」(2.6)
- recipe の「削除済みレシピの品を別のセルへ増やす要求は拒否される」(2.2、2.6)

## 6. ドキュメント

- [x] 6.1 `docs/product.md` の §4 S01「作り置きを広げる」の箇条を更新し、次の 3 点を書き足す。`git diff docs/product.md` で確かめる
  - 起点と同じ日の起点より前の時間帯は選べない
  - 既存割当の背景色は表示期間の外のセルにも付く
  - 反映後の通知の `元に戻す` で、追加した行が消え、上書きで消えた品が戻る
- [x] 6.2 `docs/product.md` の §4 S03 の削除の箇条に、削除済みレシピの品は「作り置きを広げる」と「時間帯全体を連続 N 日コピー」で増やせないことを書き足す。
  `git diff docs/product.md` で確かめる
- [x] 6.3 `docs/reviews/2026-10-15-design-ux-review.md` の R37、R39、R40、R42、R43 にチェックを入れ、proposal 名 `revise-mealplan-spread-safety` を添える。目視で確かめる

## 7. 確認

- [x] 7.1 `go test ./internal/... ./cmd/...` を実行し、すべて通ることを確かめる
- [x] 7.2 `cd frontend && npm run type-check` を実行し、エラーが無いことを確かめる
- [x] 7.3 `cd frontend && npm run build:dev` を実行し、ビルドが通ることを確かめる
- [x] 7.4 `task test:e2e` を実行し、すべて通ることを確かめる (`check:scenarios` を含む)
- [x] 7.5 `task lint` を実行し、エラーが無いことを確かめる
