## Why

献立の行は自分の ID を持たず、画面からの操作は「日付、時間帯、並び順」の 3 つ組で行を指している
(docs/reviews/2026-10-14-design-ux-review.md R25)。
並び順は、同じセルの品を削除したり並べ替えたりするたびに振り直される。行の住所が、操作のたびに変わることになる。

ここから、次の不都合が出ている。

- 画面は操作のあとで一覧を取り直すが、取り直しが終わる前の次の操作は、古い並び順で行を指す。
  先頭の品を削除した直後なら、2 品目を指したつもりの操作が 3 品目に当たる
- セルの中の並び替えは行を指せないので、レシピ ID の並びを渡している。
  同じレシピの品が同じセルに 2 つあると、それぞれを区別して並べ替えられない
- 並び順が主キーの一部なので、詰め直しのたびに主キーを書き換える (`fix-mealplan-renumber-after-reorder` の不具合の土台)

docs/design.md §4 の MealPlan には最初から `id` がある。実装をそこへ合わせる。

## What Changes

- 献立の行に ID を持たせる。既存の行には、スキーマの更新で ID を振る
- 1 品に対する操作 (入れ替え、削除、作った印、別のセルへの移動、作り置きを広げる、作り置きをやめる、他の作り置きにまとめる、まとめ候補の取得) を、行の ID で指す
- セルの中の並び替えを、行の ID の並びで指す
- 削除した品を元に戻すと、戻した行は削除の前と同じ ID を持つ
- **BREAKING** (内部 API)：`MealPlanService` の上記のメソッドの引数が、(日付, 時間帯, 並び順) から行の ID に変わる
- **BREAKING** (内部 API)：`CopyLeftover` から「1 品だけをコピーする」ための並び順の引数を外す。
  画面からは時間帯全体のコピーにしか使っていない (`revise-mealplan-spread-api` のあと)

### Non-goals

- **並び順の一意性をやめる**：(日付, 時間帯, 並び順) の一意性は残す (design.md D2)
- **作り置きの ID や「作った」印の持ち方の変更**：`batch_id` と `meal_plan_cooked` はそのままにする
- **画面の見た目や操作の変更**：利用者から見える変化は、連続操作が別の品に当たらなくなることだけである
- **買い物リスト**：S07 は (作り置き, レシピ) の単位で集計しており、行の ID を使わない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：
  - 「品の指定は並び順に左右されない」を足す
  - 「作った印」の、存在しない品への要求の書き方を、並び順によらない形に改める

## Impact

### 新設ファイル

- `internal/storage/sqlite/migrations/00009_meal_plans_id.sql`

### 更新ファイル

- `internal/domain/mealplan/entry.go`：`Entry` に `ID` を足す
- `internal/domain/mealplan/batch.go`：`NewEntryID` を足す
- `internal/domain/mealplan/removed.go`：`RemovedEntry` に `ID` を足す
- `internal/domain/mealplan/copy.go` / `copy_test.go`：`PlanCopy` が返す `Entry` に ID を振る
- `internal/storage/sqlite/mealplan_repo.go`：全クエリを `id` で行を指す形に直す。`Reorder` はレシピ ID ではなく行の ID の並びを受け取る
- `internal/storage/sqlite/mealplan_repo_test.go`：引数の変更に合わせ、ID に関するケースを足す
- `internal/storage/sqlite/sqlite_test.go`：migration 00009 が既存の行に ID を振ることを確かめる
- `internal/service/mealplan.go`：メソッドの引数を変える。`mealPlanRepo` も合わせる
- `internal/service/mealplan_test.go`：フェイクのリポジトリとテストを引数の変更に合わせる
- `frontend/src/components/mealplan/week-calendar.tsx`：全 mutation の引数を `entry.id` にする。`reorderWithinCell` は行の ID の並びを組む
- `frontend/src/components/mealplan/meal-plan-cell.tsx` / `meal-item-row.tsx` / `cell-overflow-list.tsx`：`key` と `DragSource` を行の ID にする
- `frontend/src/components/mealplan/spread-batch-dialog.tsx`：起点を行の ID で受け取る
- `frontend/src/components/mealplan/batch-merge-dialog.tsx`：変更なし (候補は作り置きの ID で選ぶ)
- `frontend/e2e/bindings-shim/mealplanservice.ts`：引数の変更に合わせる
- `frontend/e2e/helpers.ts`：`copyLeftover` から並び順の引数を外し、`setCooked` は (日付, 時間帯, 並び順) から行の ID を引いてから呼ぶ
- `frontend/e2e/mealplan-assign.spec.ts`：describe「品の指定は並び順に左右されない」を足す
- `docs/design.md`：§4 の MealPlan を実装に合わせる (`id`、`batch_id`、`meal` の値、一意性の持ち方)
- `docs/reviews/2026-10-14-design-ux-review.md`：R25 にチェックを入れる
- `openspec/specs/mealplan/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- `frontend/bindings/` (`wails3 generate bindings -clean=true -ts -i`)：`Entry.id` が増え、メソッドの引数が変わる

### migration

- `00009_meal_plans_id.sql`：`meal_plans` を作り直し、`id TEXT PRIMARY KEY` を足す。既存の行には無作為な ID を振る。
  (日付, 時間帯, 並び順) は一意インデックスとして残す

### 前提と順序

レビュー由来の 7 つの change のうち、最後に apply する。

- `add-shell-startup-safety`：テーブルを作り直す migration を当てる前に、更新前の控えが取られるようにしておく
- `fix-mealplan-renumber-after-reorder`：並び順の一意性を残すので、詰め直しの退避の方式がそのまま要る
- `revise-mealplan-spread-api`：`SpreadEntry` ができ、`CopyToOtherMeal` が消えてから引数を変える
- `revise-mealplan-destructive-actions`：`RemovedEntry` と `RestoreEntries` ができてから ID を足す
- `add-mealplan-cell-overflow-list`：品行が `MealItemRow` にまとまってから `key` と操作の引数を変える

### 後続 proposal への引き継ぎ

- R7 (作り置きごと日をずらす)：作り置きの行をまとめて動かす操作を、行の ID の一覧で書ける
- `add-mealplan-cook-consume`：作った印と在庫の消費を結ぶとき、消費の記録から献立の行を ID で指せる
