## Why

S01 の献立は「作る予定」しか持たず、実際に作ったかどうかを記録する手段が無い
(docs/reviews/2026-09-25-design-ux-review.md R4)。
予定と実績の区別が無いことで、次の 2 つが困る。

- S07 買い物リストは、期間より前から続く作り置きも計上する。
  たとえば日曜に作ったカレーを月曜と火曜にも食べる場合、月曜から始まる期間の S07 に、もう買って作り終えたカレーの材料が並ぶ
- MVP 後の献立候補の「久しぶりに」軸は、レシピの最終使用日から計算する (product.md §5)。
  予定と実績を区別できないと、まだ作っていない未来の予定まで「最近使った」と数えてしまう

design.md §8 の順序案には、「作った」印を付けて在庫を減らす `add-mealplan-cook-consume` がある。
在庫を減らすには、材料と在庫の突き合わせ (単位の違い、表記の揺れ) の扱いを決める必要があり、それだけで 1 つの proposal になる。
この proposal では、その土台になる「作った」印だけを先に入れ、S07 の計上に反映する。

## What Changes

- S01 の品の `⋯` メニューに「作った」を足す。印の付いた品では「作った印を外す」に変わる
- 「作った」印は、材料を 1 回分として数える単位 (同じ作り置きに属する同じレシピ) に付ける
  - 作り置きの 1 行で「作った」を選ぶと、同じ作り置きの同じレシピの全行に印が付く
  - 別調理として割り当てた行には、同じレシピでも印は広がらない
  - 日付の制限は設けない (前の日に作り置きを作る使い方があるため、未来の日付の品にも付けられる)
- 印の付いた品は、S01 でレシピ名の前にチェックの印を出す
- 作り置きを編集したときの印の扱いを決める
  - 作り置きを広げて足した行、ドラッグ&ドロップで動かした行は、印を引き継ぐ
  - 品を入れ替えると、入れ替えた品は印の無い状態になる
  - 作り置きから外したとき、残った起点の行は印を保つ
  - 他の作り置きにまとめるとき、まとめる前の 2 つのどちらかに印があれば、まとめたあとの作り置きに印が付く
- S07 の計上を変える
  - 「作った」印の付いた作り置きの材料は、材料テーブルと「家にあるか確かめる材料」に計上しない
  - 「この期間の献立」欄には印の付いた献立も残し、「作った」の印を添える
  - 期間の献立がすべて「作った」印付きのときは、材料テーブルの位置にその旨を出す

### Non-goals

- **在庫の消費**：「作った」印を付けても冷蔵庫の在庫は減らない。`add-mealplan-cook-consume` で扱う
- **食べたかどうかの記録**：作り置きを何日目に食べたか、食べ残したかは記録しない。印は「調理した」ことだけを表す
- **作った日付の入力**：印を付けた日時は保存するが、利用者が「何日に作った」を入力・表示する UI は持たない
- **「久しぶりに」軸の実装**：最終使用日の計算方法は product.md §5 に書くだけで、計算は `add-mealplan-suggest-by-recency` で実装する
- **S01 の見た目の大きな変更**：印の付いた品を薄くしたり、別の場所へまとめたりはしない (作り置きの残りの日にも食べるため)

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：
  - 新しい要件「作った印」を ADDED で足す (印の付け外し、印の単位、表示、永続化)
  - 新しい要件「作り置きの編集と作った印」を ADDED で足す (広げる、移動、入れ替え、外す、まとめるときの印の扱い)
- `shopping`：
  - 「作り置き単位の計上」に、作った印の付いた作り置きを計上しないことを足す (MODIFIED)
  - 「期間の献立一覧」に、作った印を添えることを足す (MODIFIED)
  - 「買い物リストの空状態」に、期間の献立がすべて作った印付きの場合を足す (MODIFIED)

## Impact

### 新設ファイル

- `internal/storage/sqlite/migrations/00008_meal_plan_cooked.sql`：作った印のテーブル
- `frontend/e2e/mealplan-cooked.spec.ts`：作った印の E2E

### 更新ファイル

- `internal/domain/mealplan/entry.go`：`Entry` に `Cooked bool` を足す
- `internal/storage/sqlite/mealplan_repo.go`：`ListPeriod` で印を返す。`SetCooked` を足す。`SplitBatch` と `MergeBatch` で印を引き継ぐ。
  `Replace` / `Remove` / `ClearMeal` / `CopyLeftover` (上書き) / `SplitBatch` / `MergeBatch` で、対応する行の無くなった印を消す
- `internal/storage/sqlite/mealplan_repo_test.go`：印の付け外しと、作り置きの編集での印の扱いのテストを足す
- `internal/service/mealplan.go`：`SetCooked(ctx, date, meal, position, cooked)` を足す。repo の interface にも足す
- `internal/service/mealplan_test.go`：`TestMealPlanService_SetCooked` を足す
- `internal/domain/shopping/list.go`：`PlannedBatch` と `Dish` に `Cooked` を足す
- `internal/domain/shopping/aggregate.go`：印の付いた作り置きを計上から外す
- `internal/domain/shopping/aggregate_test.go`：印の付いた作り置きを外すケースを足す
- `internal/storage/sqlite/shopping_repo.go`：`loadPlannedGroups` で印を読む
- `internal/service/shopping_test.go`：印の付いた作り置きを計上しないシナリオを足す
- `frontend/e2e/bindings-shim/mealplanservice.ts`：`SetCooked` を足す
- `frontend/src/components/mealplan/cell-item-menu.tsx`：「作った」「作った印を外す」の項目を足す
- `frontend/src/components/mealplan/meal-plan-cell.tsx`：印の表示と、`EntryAction` の `"toggle-cooked"` を足す
- `frontend/src/components/mealplan/week-calendar.tsx`：`SetCooked` の mutation を足し、`handleEntryAction` から呼ぶ
- `frontend/src/components/shopping/shopping-dish-list.tsx`：「作った」の印を添える
- `frontend/src/routes/shopping.tsx`：期間の献立がすべて作った印付きのときのメッセージを出す
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.meal_plan.item_menu.mark_cooked` / `unmark_cooked`、
  `pages.meal_plan.cell.cooked`、`pages.shopping.dishes.cooked`、`pages.shopping.empty.all_cooked` を追加する
- `frontend/e2e/shopping.spec.ts`：「作り置き単位の計上」「期間の献立一覧」「買い物リストの空状態」に Scenario を足す
- `docs/product.md`：§4 S01 の品の `⋯` の項目、§4 S07 の作り置きの数え方と献立欄、§5 の「久しぶりに」の最終使用日の定義を更新する
- `docs/design.md`：§4 MealPlan に作った印のテーブルを書き足す。§8 の順序案で `add-mealplan-cook-consume` の前に本 change を置く
- `docs/wireframes/S01_meal_calendar.v2.drawio.svg`：品行のチェックの印を描き足す (可視 SVG と `content=` 内の編集 XML の両方)
- `openspec/specs/mealplan/spec.md` / `openspec/specs/shopping/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- `frontend/bindings/`：`MealPlanService.SetCooked` と、`Entry.cooked` / `Dish.cooked` を反映する (`wails3 generate bindings -clean=true -ts -i`)

### migration

- `00008_meal_plan_cooked.sql`：`meal_plan_cooked (batch_id, recipe_id, cooked_at)` を作る。主キーは `(batch_id, recipe_id)`。既存の行は変えない

### 前提と順序

- 進行中の `revise-recipe-detail-page` が `week-calendar.tsx` を書き換えている。そちらを先に archive してから apply する
- `revise-mealplan-reorder-menu` も品の `⋯` メニューに項目を足す。どちらが先でもよいが、後から入れる側で `cell-item-menu.tsx` の項目順を design.md D6 に合わせる
- `add-ingredient-name-matching` は `aggregate.go` の突き合わせのキーを変える。この change は同じファイルの走査の入口で作り置きを外すだけなので、どちらが先でも衝突は小さい

### 後続 proposal への引き継ぎ

- `add-mealplan-cook-consume`：「作った」を選んだときに、材料分の在庫を減らす (減らす量の確認ダイアログを含む)
- `add-mealplan-suggest-by-recency`：最終使用日を、作った印の付いた作り置きの行の日付から計算する
