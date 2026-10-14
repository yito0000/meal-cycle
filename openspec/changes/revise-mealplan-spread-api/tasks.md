## 1. ドメイン層

- [ ] 1.1 `internal/domain/mealplan/validate.go` (更新) に `ErrInvalidSpreadTarget = errors.New("mealplan: invalid spread target")` を足す。完了判定: `go build .` が通る
- [ ] 1.2 `internal/domain/mealplan/spread.go` (新設) に `SpreadTarget` を定義する (design.md D1)。完了判定: `go build .` が通る
- [ ] 1.3 `internal/domain/mealplan/spread.go` (更新) に `ValidateSpreadTargets` を書く (design.md D2)。完了判定: `go build .` が通る
- [ ] 1.4 `internal/domain/mealplan/spread_test.go` (新設) に table-driven test を書く
    - 起点日の翌日から 3 日後までの、同じ時間帯と別の時間帯のセルは通る
    - 起点日の別の時間帯のセルは通る
    - 空の一覧は `ErrInvalidSpreadTarget`
    - 起点セル自身を含むと `ErrInvalidSpreadTarget`
    - 起点日より前の日付は `ErrInvalidSpreadTarget`
    - 起点日 + maxOffsetDays より後の日付は `ErrInvalidSpreadTarget`
    - 同じセルが 2 回あると `ErrInvalidSpreadTarget`
    - 不正な日付は `ErrInvalidDate`
    - 不正な時間帯は `ErrInvalidMeal`
    - 完了判定: `go test ./internal/domain/mealplan/ -run TestValidateSpreadTargets` が pass する

## 2. リポジトリ層

- [ ] 2.1 `internal/storage/sqlite/mealplan_repo.go` (更新) に `Spread(ctx, date, meal string, position int, targets []mealplan.SpreadTarget, mode mealplan.CopyMode,
  shareBatch mealplan.ShareBatch) error` を書く (design.md D3)。完了判定: `go build .` が通る
- [ ] 2.2 `internal/storage/sqlite/mealplan_repo_test.go` (更新) に `TestMealPlanRepo_Spread` を足す
    - should insert rows only into the given cells when targets skip a day
    - should share the source batch across same-meal and cross-meal targets when shareBatch is true
    - should give every target its own batch and leave the source batch single when shareBatch is false
    - should append after existing rows when mode is append
    - should skip a cell that already has the same recipe when mode is append
    - should replace existing rows when mode is overwrite
    - should carry the cooked mark to shared rows and not to distinct rows
    - should return ErrEntryNotFound and change nothing when the source row does not exist
    - 完了判定: `go test ./internal/storage/sqlite/ -run TestMealPlanRepo_Spread` が pass する
- [ ] 2.3 `internal/storage/sqlite/mealplan_repo.go` (更新) の `ListMergeCandidates` を、`ROW_NUMBER()` で作り置きごとに 1 行を取る形に書き直す (design.md D6)。
  完了判定: `go build .` が通る
- [ ] 2.4 `internal/storage/sqlite/mealplan_repo_test.go` (更新) の `TestMealPlanRepo_Batch` にケースを足す
    - should return one candidate when a batch spans lunch and dinner on the same date
    - should pick the earliest row by date, meal order and position as the representative
    - 完了判定: `go test ./internal/storage/sqlite/ -run TestMealPlanRepo_Batch` が pass する
- [ ] 2.5 `internal/storage/sqlite/mealplan_repo.go` (更新) から `CopyToOtherMeal` を消す (design.md D4)。完了判定: `go build .` が、サービス層の参照が残っているために失敗する
- [ ] 2.6 `internal/storage/sqlite/mealplan_repo_test.go` (更新) から `CopyToOtherMeal` を使うテストを消し、前提づくりに使っている箇所は `Spread` に置き換える。
  完了判定: `go vet ./internal/storage/sqlite/` が通る

## 3. サービス層

- [ ] 3.1 `internal/service/mealplan.go` (更新) の `mealPlanRepo` に `Spread` を足し、`CopyToOtherMeal` を消す。完了判定: 3.2 と合わせて `go build .` が通る
- [ ] 3.2 `internal/service/mealplan.go` (更新) に `SpreadEntry` を書き、`CopyToOtherMeal` を消す
    - 起点の (日付, 時間帯, 並び順) と `mode` を検証する
    - `ValidateSpreadTargets` に `max(cfg.LeftoverCopyPresets) - 1` を渡す
    - 完了判定: `go build .` が通る
- [ ] 3.3 `internal/service/mealplan_test.go` (更新) のフェイクのリポジトリに `Spread` を足し、`CopyToOtherMeal` を消す。完了判定: `go vet ./internal/service/` が通る
- [ ] 3.4 `internal/service/mealplan_test.go` (更新) に `TestMealPlanService_SpreadEntry` を足し、`TestMealPlanService_CopyToOtherMeal` を消す
    - should pass the given targets to the repository unchanged when all targets are valid
    - should return ErrInvalidSpreadTarget and not reach the repository when a target is out of range
    - should return ErrInvalidSpreadTarget and not reach the repository when targets is empty
    - should return ErrInvalidCopyMode when mode is neither append nor overwrite
    - should return ErrInvalidPosition when the source position is negative
    - should derive the range from LeftoverCopyPresets when presets change
    - 完了判定: `go test ./internal/service/ -run TestMealPlanService_SpreadEntry` が pass する
- [ ] 3.5 `cmd/e2e-server/dispatch_test.go` (更新) に、構造体のスライスと bool を受けるメソッドへ JSON の引数を詰められるケースを足す。完了判定: `go test ./cmd/...` が pass する
- [ ] 3.6 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: `frontend/bindings/` に `SpreadEntry` があり、`CopyToOtherMeal` が無い

## 4. フロントエンド

- [ ] 4.1 `frontend/e2e/bindings-shim/mealplanservice.ts` (更新) に `SpreadEntry` を足し、`CopyToOtherMeal` を消す。完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.2 `frontend/src/lib/spread-batch-selection.ts` (更新) から `planSpread`、`SpreadPlan`、`dateGap` を消す。完了判定: 4.3 と合わせて `npm run type-check` が通る
- [ ] 4.3 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `submitSpread` を、`useMutation` で包んだ 1 回の `SpreadEntry` の呼び出しに置き換える
    - `computeCopyDays` を消す
    - 失敗件数の集計と `spread.partial_failure` の分岐を消す
    - `errorState` の `params` を消す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.4 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `SpreadState` に起点の `batchId` と `recipeId` を足し、`SpreadBatchDialog` に渡す。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.5 `frontend/src/components/mealplan/spread-batch-dialog.tsx` (更新) で、起点と同じ `batchId` と `recipeId` の行があるセルを選択不可にする (design.md D5)
    - チェックボックスの代わりに、起点セルと同じ印を出す
    - 既定のチェックがこのセルに当たるときは、チェックを付けない
    - 完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.6 `frontend/src/components/mealplan/error-messages.ts` (更新) に `["mealplan: invalid spread target", "pages.meal_plan.errors.invalid_spread_target"]` を足す。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.7 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` (更新) を直す
    - `pages.meal_plan.spread.partial_failure` を消す
    - `pages.meal_plan.errors.invalid_spread_target`：「広げる先に選べないセルが含まれています」/ "Some selected cells cannot be used"
    - `pages.meal_plan.spread.same_batch_cell_label`：「すでに同じ作り置きが入っています」/ "Already in this batch"
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっている

## 5. E2E

- [ ] 5.1 `frontend/e2e/helpers.ts` (更新) に `openSpreadDialog(page, recipeName)` と `spreadCell(page, date, meal)` (格子のセルの locator を返す) を足す。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 5.2 `frontend/e2e/mealplan-batch.spec.ts` (更新) の describe「作り置きを広げる (統合ダイアログ)」を「作り置きを広げる操作」に改名し、
  `test.skip` を外してシナリオを実装する
    - 起点日の翌日の同じ時間帯が既定でチェックされる (旧名「起点日 + 2 日ぶん同時間帯が既定でチェックされる」を改める)
    - 対象日 × 時間帯を自由に組み合わせて広げる
    - チェックしていない日には追加されない
    - 別調理を選ぶと各セルが独立した作り置きになる
    - 別調理は別の時間帯のセルにも効く
    - 衝突セルは追加 / 上書きの指定に従って反映される
    - 上書きを選ぶと対象セルの既存の品が置き換わる
    - 同じ作り置きがすでに入っているセルは選べない
    - 起点日 + 未来 3 日ぶんの 4 行だけが表示される
    - キャンセルすると割当は変わらない
    - 完了判定: `task test:e2e` で上記が pass する
- [ ] 5.3 `frontend/e2e/mealplan-batch.spec.ts` (更新) に describe「まとめ候補の一意性」を足す
    - 同じ日の昼食と夕食にまたがる作り置きは候補に 1 行だけ出る
    - 完了判定: `task test:e2e` で上記が pass する

## 6. ドキュメントと検証

- [ ] 6.1 `docs/product.md` (更新) §4 S01 の「作り置きを広げる」を直す
    - 対象日を「起点日 ±4 日」から「起点日から未来 3 日ぶんの 4 行」に改める
    - 既定のチェックは起点日の翌日の同じ時間帯の 1 セルであることを書く
    - チェックしたセルだけに入ることと、別調理が別の時間帯にも効くことを書く
    - 完了判定: 差分を目視で確認する
- [ ] 6.2 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R22 と R23 にチェックを入れ、R34 に対応した範囲を追記する。完了判定: 差分を目視で確認する
- [ ] 6.3 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [ ] 6.4 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [ ] 6.5 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [ ] 6.6 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [ ] 6.7 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [ ] 6.8 `openspec validate revise-mealplan-spread-api --strict` を実行する。完了判定: エラー 0 件
