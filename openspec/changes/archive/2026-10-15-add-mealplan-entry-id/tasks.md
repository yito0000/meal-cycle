## 1. migration とドメイン層

- [x] 1.1 `internal/storage/sqlite/migrations/00009_meal_plans_id.sql` (新設) を書く (design.md D6)
    - Up: `meal_plans_new` を作り、既存の行に `lower(hex(randomblob(16)))` の ID を付けてコピーし、改名して、2 つのインデックスを張り直す
    - Down: `id` を落とし、`(date, meal, position)` を主キーに戻す
    - 完了判定: `go build .` が通り、`go test ./internal/storage/sqlite/ -run TestOpenAndMigrate` が pass する
- [x] 1.2 `internal/storage/sqlite/sqlite_test.go` (更新) に、migration 00009 のテストを足す
    - 00008 までを当てた DB に献立を 3 行入れてから 00009 を当てると、3 行とも空でない別々の ID を持つ
    - 00009 のあとも、(日付, 時間帯, 並び順) の重複する行は挿入できない
    - 00009 のあとも、作った印 (`meal_plan_cooked`) の行は残る
    - 完了判定: `go test ./internal/storage/sqlite/ -run 'TestMigrate|TestOpenAndMigrate'` が pass する
- [x] 1.3 `internal/domain/mealplan/batch.go` (更新) に `NewEntryID()` を足す (design.md D1)。完了判定: `go build .` が通る
- [x] 1.4 `internal/domain/mealplan/entry.go` (更新) の `Entry` に `ID string \`json:"id"\`` を足す。完了判定: `go build .` が通る
- [x] 1.5 `internal/domain/mealplan/removed.go` (更新) の `RemovedEntry` に `ID string \`json:"id"\`` を足す。完了判定: `go build .` が通る
- [x] 1.6 `internal/domain/mealplan/copy.go` (更新) の `PlanCopy` が返す `Entry` に `NewEntryID()` を振る。完了判定: `go build .` が通る
- [x] 1.7 `internal/domain/mealplan/copy_test.go` (更新) にケースを足す
    - 返される Entry はすべて空でない別々の ID を持つ
    - 完了判定: `go test ./internal/domain/mealplan/` が pass する

## 2. リポジトリ層

- [x] 2.1 `internal/storage/sqlite/mealplan_repo_test.go` (更新) に、`Assign` の戻り値から ID を取るテスト用のヘルパー `assignID(t, repo, date, meal, recipeID) string` を足す。
  完了判定: `go vet ./internal/storage/sqlite/` が通る
- [x] 2.2 `internal/storage/sqlite/mealplan_repo.go` (更新) の `insertMealPlan` に `id` の引数を足し、`ListPeriod` と `loadMealEntries` の SELECT に `id` を足す。
  完了判定: `go build ./internal/storage/sqlite/` が通る
- [x] 2.3 `internal/storage/sqlite/mealplan_repo.go` (更新) に、ID から行の住所と内容を読む `loadEntryByID(ctx, q, id)` を足す。行が無ければ `ErrEntryNotFound` を返す。
  完了判定: `go build ./internal/storage/sqlite/` が通る
- [x] 2.4 `internal/storage/sqlite/mealplan_repo.go` (更新) の `Replace`、`Remove`、`SetCooked`、`SplitBatch`、`MergeBatch`、`ListMergeCandidates`、`MoveEntry`、`Spread` を、
  `entryID` を受け取り `loadEntryByID` で行を特定する形に直す (design.md D3)。完了判定: `go build ./internal/storage/sqlite/` が通る
- [x] 2.5 `internal/storage/sqlite/mealplan_repo.go` (更新) の `Reorder` を、行の ID の並びを受け取る形に直す
    - 渡された ID の集合が、そのセルの行の ID の集合と一致しなければ `ErrReorderMismatch`
    - 全行を退避してから、渡された順に position を振る
    - 完了判定: `go build ./internal/storage/sqlite/` が通る
- [x] 2.6 `internal/storage/sqlite/mealplan_repo.go` (更新) の `Remove` と `ClearMeal` が返す `RemovedEntry` に `ID` を入れ、`RestoreEntries` は同じ ID で行を入れる。
  同じ ID の行がすでにあれば `ErrInvalidRestore` を返す (design.md D5)。完了判定: `go build ./internal/storage/sqlite/` が通る
- [x] 2.7 `internal/storage/sqlite/mealplan_repo_test.go` (更新) の既存のテストを、引数の変更に合わせて書き換える (`assignID` と `ListPeriod` の戻り値の ID を使う)。
  完了判定: `go test ./internal/storage/sqlite/` が pass する
- [x] 2.8 `internal/storage/sqlite/mealplan_repo_test.go` (更新) に、ID に関するケースを足す
    - Assign should return an entry with a non-empty id
    - Remove should hit the given entry even after an earlier row in the same cell was removed
    - SetCooked should mark the given entry even after the cell was reordered
    - Replace should return ErrEntryNotFound when the entry was already removed
    - MoveEntry should keep the entry id
    - Reorder should swap two rows of the same recipe when their ids are given in the new order
    - Reorder should return ErrReorderMismatch when an id is duplicated
    - Reorder should return ErrReorderMismatch when an id belongs to another cell
    - RestoreEntries should bring the row back with the same id
    - RestoreEntries should return ErrInvalidRestore when a row with the same id already exists
    - Spread should give every inserted row a new id
    - 完了判定: `go test ./internal/storage/sqlite/ -run TestMealPlanRepo` が pass する

## 3. サービス層

- [x] 3.1 `internal/service/mealplan.go` (更新) の `mealPlanRepo` と各メソッドの引数を design.md D3 の表のとおりに変える。
  空の `entryID` は `ErrEntryNotFound` にする。完了判定: `go build .` が通る
- [x] 3.2 `internal/service/mealplan.go` (更新) の `CopyLeftover` から `position` の引数と、1 品だけを選ぶ分岐を消す (design.md D4)。完了判定: `go build .` が通る
- [x] 3.3 `internal/service/mealplan_test.go` (更新) のフェイクのリポジトリを、ID で行を持つ形に直す。完了判定: `go vet ./internal/service/` が通る
- [x] 3.4 `internal/service/mealplan_test.go` (更新) の既存のテストを、引数の変更に合わせて書き換える。完了判定: `go test ./internal/service/` が pass する
- [x] 3.5 `internal/service/mealplan_test.go` (更新) にケースを足す
    - RemoveEntry should return ErrEntryNotFound when entryID is empty
    - ReorderMeal should pass the ordered entry ids to the repository
    - ListMergeCandidates should take the recipe from the entry itself
    - CopyLeftover should copy all dishes of the meal
    - 続けて 2 品を削除しても別の品は消えない (spec のシナリオ。実 SQLite のリポジトリを使う)
    - 並べ替えのあとの作った印は指定した品に付く (同上)
    - すでに無い品への操作は拒否される (同上)
    - 完了判定: `go test ./internal/service/ -run TestMealPlanService` が pass する
- [x] 3.6 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: `frontend/bindings/` の `Entry` に `id` があり、`RemoveEntry` の引数が 1 つになっている

## 4. フロントエンド

- [x] 4.1 `frontend/e2e/bindings-shim/mealplanservice.ts` (更新) を引数の変更に合わせる。完了判定: 4.2 以降と合わせて `npm run type-check` が通る
- [x] 4.2 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の mutation (`replaceMut`、`removeMut`、`cookedMut`、`splitMut`、`mergeMut`、`moveMut`、`spreadMut`) の引数を
  `entry.id` にする。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `reorderWithinCell` と `reorderMut` を、行の ID の並びを組んで渡す形に直す。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.4 `frontend/src/components/mealplan/week-calendar.tsx` (更新) の `PickerState`、`MergeDialogState`、`SpreadState`、やめる確認の state を、`position` ではなく行の ID を持つ形に直す。
  `copyMut` から `position` を外す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.5 `frontend/src/components/mealplan/meal-plan-cell.tsx` (更新) の `DragSource` に `id` を足し、`handleDropEntry` に渡す (design.md D7)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.6 `frontend/src/components/mealplan/meal-plan-cell.tsx`、`meal-item-row.tsx`、`cell-overflow-list.tsx` (更新) の品行の `key` を `entry.id` にする。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.7 `frontend/src/components/mealplan/spread-batch-dialog.tsx` (更新) の `source` を `{ id, date, meal }` にする。完了判定: `cd frontend && npm run type-check` が通る

## 5. E2E

- [x] 5.1 `frontend/e2e/helpers.ts` (更新) に `entryId(page, date, meal, position)` (期間を取得して ID を返す) を足し、`setCooked` をこれ経由にする。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.2 `frontend/e2e/helpers.ts` (更新) の `copyLeftover` から `position` の引数を外す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.3 `frontend/e2e/mealplan-assign.spec.ts` (更新) に describe「品の指定は並び順に左右されない」を足す
    - 同じレシピの 2 品を区別して並べ替えられる
    - 完了判定: `task test:e2e` で上記が pass する

## 6. ドキュメント

- [x] 6.1 `docs/design.md` (更新) §4 の MealPlan を実装に合わせる
    - `id` (UUID v7)、`batchId`、`createdAt` を載せる
    - `meal` の値から `"other"` を外す
    - 「`(date, meal, position)` にユニークインデックス」を、主キーは `id`、(日付, 時間帯, 並び順) は一意インデックス、と書く
    - 完了判定: 差分を目視で確認する
- [x] 6.2 `CLAUDE.md` (更新) の「データモデルの癖」に、献立の行は `id` で指し、`position` は並び順だけを表すことを 1 行で足す。完了判定: 差分を目視で確認する
- [x] 6.3 `docs/reviews/2026-10-14-design-ux-review.md` (更新) の R25 にチェックを入れ、`add-mealplan-entry-id` で対応した旨を追記する。完了判定: 差分を目視で確認する

## 7. 検証

- [x] 7.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 7.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 7.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 7.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 7.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 7.6 WSL 側で、献立の入った既存の DB (`~/.local/share/meal-cycle/meal-cycle.db`) に対して `task dev` で起動する。
  完了判定: `backups/` に控えが 1 つ増え、S01 に更新前と同じ献立が表示されることを目視で確認する
- [x] 7.7 `task build:win` で作った `.exe` を、Windows 側の既存の DB に対して起動する。
  完了判定: `%APPDATA%\meal-cycle\backups\` に控えが 1 つ増え、S01 に更新前と同じ献立が表示されることを目視で確認する
- [x] 7.8 `openspec validate add-mealplan-entry-id --strict` を実行する。完了判定: エラー 0 件
