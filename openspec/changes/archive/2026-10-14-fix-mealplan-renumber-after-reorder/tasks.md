## 1. リポジトリ層

- [x] 1.1 `internal/storage/sqlite/mealplan_repo_test.go` (更新) に、失敗するテストを先に足す
    - `TestMealPlanRepo_Remove`：A、B、C を B、C、A に並べ替えてから position 0 を消すと成功し、C、A が position 0、1 になる
    - `TestMealPlanRepo_MoveEntry`：3 品を登録順と逆に並べ替えてから先頭を別セルへ動かすと成功し、元のセルの 2 品は順を保って position 0、1 になる
    - `TestMealPlanRepo_Batch`：火曜のセルを並べ替えた状態で `SplitBatch` すると成功し、火曜に残った 2 品は順を保って position 0、1 になる
    - `TestMealPlanRepo_Batch`：同じセルに同じ作り置きの行が 2 つある状態で `SplitBatch` すると、起点以外の行がすべて消える
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'TestMealPlanRepo_(Remove|MoveEntry|Batch)'` で、足したケースが `UNIQUE constraint failed` で落ちる
- [x] 1.2 `internal/storage/sqlite/mealplan_repo.go` (更新) に定数 `positionShiftOffset = 1000` を足し、`Reorder` のリテラル `1000` を置き換える。
  完了判定: `go build .` が通る
- [x] 1.3 `internal/storage/sqlite/mealplan_repo.go` (更新) に `closePositionGap(ctx, q querier, date, meal string, removed int) error` を足す (design.md D1)。
  完了判定: `go build .` が通る
- [x] 1.4 `internal/storage/sqlite/mealplan_repo.go` (更新) の `removeRowTx` の詰め直しを `closePositionGap` の呼び出しに置き換える。
  完了判定: 1.1 の `Remove` と `MoveEntry` のケースが pass する
- [x] 1.5 `internal/storage/sqlite/mealplan_repo.go` (更新) に `renumberMeal(ctx, q querier, date, meal string) error` を足す (design.md D2)。
  完了判定: `go build .` が通る
- [x] 1.6 `internal/storage/sqlite/mealplan_repo.go` (更新) の `SplitBatch` を、起点以外の行を先にすべて消してから、影響を受けたセルごとに `renumberMeal` を呼ぶ形に直す。
  完了判定: 1.1 の `Batch` のケースが pass する
- [x] 1.7 `internal/storage/sqlite/mealplan_repo.go` (更新) の `Remove` のコメントから「小さい position 側から順に上書きしていけば衝突しない」の説明を消す。
  完了判定: 差分を目視で確認する
- [x] 1.8 `internal/storage/sqlite/mealplan_repo.go` (更新) の `isForeignKeyErr` を `FOREIGN KEY constraint failed` を含む場合だけに絞る (design.md D3)。
  完了判定: `go test ./internal/storage/sqlite/` の既存の `ErrUnknownRecipe` のテストが pass する

## 2. サービス層

- [x] 2.1 `internal/service/mealplan.go` (更新) の末尾の `var ( _ = errors.Is )` と、使われなくなる `errors` の import を消す。完了判定: `go build .` が通る

## 3. E2E

- [x] 3.1 `frontend/e2e/mealplan-assign.spec.ts` (更新) に describe「並べ替えたあとの品の削除と移動」を足す
    - 並べ替えたセルで先頭の品を削除できる (3 品を API で割り当て、品のメニューの「下へ移動」を 2 回押してから先頭を削除する)
    - 完了判定: `task test:e2e` で上記が pass する

## 4. ドキュメントと検証

- [x] 4.1 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R21 にチェックを入れ、`fix-mealplan-renumber-after-reorder` で対応した旨を追記する。完了判定: 差分を目視で確認する
- [x] 4.2 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R27 に、1 点目と 3 点目を対応した旨を追記する (チェックは入れない)。完了判定: 差分を目視で確認する
- [x] 4.3 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 4.4 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 4.5 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 4.6 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 4.7 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 4.8 `openspec validate fix-mealplan-renumber-after-reorder --strict` を実行する。完了判定: エラー 0 件
