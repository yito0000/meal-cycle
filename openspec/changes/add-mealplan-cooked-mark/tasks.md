## 1. スキーマとドメイン

- [ ] 1.1 `internal/storage/sqlite/migrations/00008_meal_plan_cooked.sql` を新設する (design.md D2)
    - up で `meal_plan_cooked (batch_id, recipe_id, cooked_at)` を作り、主キーを `(batch_id, recipe_id)` にする
    - `batch_id` に `CHECK (length(batch_id) > 0)` を付ける
    - `meal_plans` に外部キーを張らない理由 (`(batch_id, recipe_id)` は一意でない) を WHY のコメントで 1 行書く
    - down でテーブルを drop する
    - 完了判定: `go test ./internal/storage/sqlite/ -run "TestOpenAndMigrate|TestMigrateIsIdempotent"` が pass する
- [ ] 1.2 `internal/domain/mealplan/entry.go` の `Entry` に `Cooked bool \`json:"cooked"\`` を足し、コメントに「同じ (BatchID, RecipeID) の組に作った印が付いているか」を書く。
  完了判定: `go build ./internal/...` が通る
- [ ] 1.3 `internal/domain/shopping/list.go` の `PlannedBatch` に `Cooked bool` を、`Dish` に `Cooked bool \`json:"cooked"\`` を足す。完了判定: `go build ./internal/...` が通る

## 2. リポジトリ

- [ ] 2.1 `internal/storage/sqlite/mealplan_repo.go` の `ListPeriod` の SELECT に、`meal_plan_cooked` の `EXISTS` を `cooked` として足す (design.md D3)。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.2 `internal/storage/sqlite/mealplan_repo.go` に `SetCooked(ctx, date, meal, position, cooked)` を足す (design.md D3)
    - 1 トランザクションで行の `batch_id` と `recipe_id` を引き、無ければ `mealplan.ErrEntryNotFound` を返す
    - `cooked` が true なら `INSERT ... ON CONFLICT DO NOTHING`、false なら `DELETE` する
    - 完了判定: `go build ./internal/...` が通る
- [ ] 2.3 `internal/storage/sqlite/mealplan_repo.go` に `pruneCookedMarks(ctx, tx)` を足す (design.md D4)。完了判定: `go build ./internal/...` が通る
- [ ] 2.4 `internal/storage/sqlite/mealplan_repo.go` の次のメソッドで、トランザクションの commit 前に `pruneCookedMarks` を呼ぶ (design.md D4)
    - `Replace`
    - `Remove`
    - `ClearMeal`
    - `CopyLeftover`
    - `CopyToOtherMeal`
    - `SplitBatch`
    - `MergeBatch`
    - `Replace` では掃除が正しさに要ることを WHY のコメントで 1 行書く
    - 完了判定: `go build ./internal/...` が通る
- [ ] 2.5 `internal/storage/sqlite/mealplan_repo.go` の `SplitBatch` で、新しい `batch_id` を振るとき元の組に印があれば新しい組にも印の行を足す (design.md D4)。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.6 `internal/storage/sqlite/mealplan_repo.go` の `MergeBatch` で、付け替える前の組に印があればまとめ先の組にも印の行を足す (design.md D4)。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.7 `internal/storage/sqlite/shopping_repo.go` の `loadPlannedGroups` で、`meal_plan_cooked` の `EXISTS` を読んで `PlannedBatch.Cooked` に入れる (design.md D5)。
  完了判定: `go build ./internal/...` が通る
- [ ] 2.8 `internal/storage/sqlite/mealplan_repo_test.go` にテストを足す
    - SetCooked で印を付けると、同じ batch_id と recipe_id の全行の Cooked が true になる
    - SetCooked で印を外すと、同じ組の全行の Cooked が false になる
    - 同じレシピでも別の batch_id の行には印が付かない
    - 存在しない行への SetCooked は ErrEntryNotFound を返す
    - 印を二重に付けても外してもエラーにならない
    - CopyLeftover の作り置きモードで足した行は印を引き継ぐ
    - CopyLeftover の別調理モードで足した行には印が付かない
    - MoveEntry で動かした行は印を保つ
    - Replace で入れ替えた行は印が無く、元のレシピに戻しても印は復活しない
    - SplitBatch で残った起点の行は印を保つ
    - MergeBatch で印のある組にまとめると、まとめた行にも印が付く
    - MergeBatch で印のある行を印の無い組にまとめると、まとめ先の組に印が付く
    - ClearMeal で組の行が無くなると、印の行も消える
    - 完了判定: `go test ./internal/storage/sqlite/...` が pass する

## 3. サービス

- [ ] 3.1 `internal/service/mealplan.go` を更新する
    - repo の interface に `SetCooked(ctx, date, meal string, position int, cooked bool) error` を足す
    - `MealPlanService.SetCooked(ctx, date, meal, position, cooked)` を足し、`validateMealAddress` と position の検証を通してから repo を呼ぶ
    - 完了判定: `go build .` が通る
- [ ] 3.2 `internal/service/mealplan_test.go` に `TestMealPlanService_SetCooked` を足す
    - should mark all rows of the same batch and recipe when marking one row
    - should not spread the mark to a separate batch of the same recipe
    - should unmark all rows of the same batch and recipe when unmarking one row
    - should return ErrEntryNotFound when the row does not exist
    - should return ErrInvalidDate when the date is malformed
    - should keep the mark after reopening the database
    - 完了判定: `go test ./internal/service/...` が pass する
- [ ] 3.3 `internal/domain/shopping/aggregate.go` の `Aggregate` で、`Cooked` の `PlannedBatch` を走査から飛ばす (design.md D5)。完了判定: `go build ./internal/...` が通る
- [ ] 3.4 `internal/domain/shopping/aggregate.go` の `Dishes` で、`Dish.Cooked` に `PlannedBatch.Cooked` を写す。完了判定: `go build ./internal/...` が通る
- [ ] 3.5 `internal/domain/shopping/aggregate_test.go` にケースを足す
    - Cooked の作り置きの材料は items に出ない
    - Cooked の作り置きと同じ食材を使う Cooked でない作り置きは、Cooked でない側の量だけが計上される
    - Cooked の作り置きは Dishes に Cooked=true で残る
    - 完了判定: `go test ./internal/domain/shopping/...` が pass する
- [ ] 3.6 `internal/service/shopping_test.go` の `TestShoppingService_List` にテストを足す
    - should exclude ingredients of a cooked batch
    - should keep a cooked batch in dishes with cooked flag
    - 完了判定: `go test ./internal/service/...` が pass する
- [ ] 3.7 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: `frontend/bindings/` の `MealPlanService` に `SetCooked` があり、`Entry` と `Dish` に `cooked` がある
- [ ] 3.8 `frontend/e2e/bindings-shim/mealplanservice.ts` に `SetCooked(date, meal, position, cooked)` を足す。完了判定: `cd frontend && npm run type-check` が通る

## 4. S01 (Frontend)

- [ ] 4.1 `frontend/src/components/mealplan/cell-item-menu.tsx` を更新する (design.md D6)
    - props に `cooked: boolean` と `onToggleCooked: () => void` を足す
    - メニューの先頭に、`cooked` が false なら「作った」、true なら「作った印を外す」を置き、区切り線で他の項目と分ける
    - 冒頭のコメントのメニュー項目の列挙に「作った / 作った印を外す」を足す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.2 `frontend/src/components/mealplan/meal-plan-cell.tsx` を更新する (design.md D6)
    - `EntryAction` に `"toggle-cooked"` を足し、`CellItemMenu` に `cooked={entry.cooked}` と `onToggleCooked` を渡す
    - `entry.cooked` のとき、🔗 の後ろ、レシピ名の前に lucide の `Check` アイコンを `aria-label={t("pages.meal_plan.cell.cooked")}` つきで出す
    - 品行に `data-cooked` 属性を付ける
    - 完了判定: `cd frontend && npm run type-check` が通る
- [ ] 4.3 `frontend/src/components/mealplan/week-calendar.tsx` を更新する
    - `MealPlanService.SetCooked` を呼ぶ `cookedMut` を足し、成功時に既存の `invalidate` を呼ぶ
    - `handleEntryAction` に `case "toggle-cooked"` を足し、`cookedMut.mutate({ date, meal, position: entry.position, cooked: !entry.cooked })` を呼ぶ
    - 失敗時は他の mutation と同じくエラーの表示に流す
    - 完了判定: `cd frontend && npm run type-check` が通る

## 5. S07 (Frontend)

- [ ] 5.1 `frontend/src/components/shopping/shopping-dish-list.tsx` で、`dish.cooked` の行に「作った」の印を文字で添え、行に `data-cooked` 属性を付ける。
  完了判定: `cd frontend && npm run type-check` が通る
- [ ] 5.2 `frontend/src/routes/shopping.tsx` を更新する (design.md D5)
    - `data.dishes.length > 0` かつ `data.dishes.every((d) => d.cooked)` のとき、材料の列に `pages.shopping.empty.all_cooked` だけを出す
    - この判定を `data.items.length === 0` の判定より先に置き、順序の理由を WHY のコメントで 1 行書く
    - 完了判定: `cd frontend && npm run type-check` が通る
- [ ] 5.3 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する
    - `pages.meal_plan.item_menu.mark_cooked`：「作った」/ "Mark as cooked"
    - `pages.meal_plan.item_menu.unmark_cooked`：「作った印を外す」/ "Unmark as cooked"
    - `pages.meal_plan.cell.cooked`：「作った」/ "Cooked"
    - `pages.shopping.dishes.cooked`：「作った」/ "Cooked"
    - `pages.shopping.empty.all_cooked`：「この期間の献立はすべて作りました。買う材料はありません」/ "All meals in this period are cooked. Nothing to buy"
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 6. E2E

- [ ] 6.1 `frontend/e2e/mealplan-cooked.spec.ts` を新設し、describe「作った印」を置く
    - 作ったを選ぶと品に印が付く
    - 作り置きの 1 行に印を付けると同じ作り置きの全行に印が付く
    - 別調理の行には印が広がらない
    - 作った印を外す
    - 作った印は再起動後も保持される (ページを再読み込みして確かめる)
    - 完了判定: `task test:e2e` で上記が pass する
- [ ] 6.2 `frontend/e2e/mealplan-cooked.spec.ts` に describe「作り置きの編集と作った印」を置く
    - 作り置きとして広げた行は印を引き継ぐ
    - 入れ替えた品は印の無い状態になる
    - それ以外の Scenario はリポジトリとサービスのテストで確かめる (タスク 2.8 / 3.2)
    - 完了判定: `task test:e2e` で上記が pass する
- [ ] 6.3 `frontend/e2e/shopping.spec.ts` に Scenario を足す
    - describe「作り置き単位の計上」：作った印の付いた作り置きの材料は計上しない
    - describe「作り置き単位の計上」：作った印の無い作り置きの材料は計上し続ける
    - describe「期間の献立一覧」：作った印の付いた献立は印を添えて一覧に残る
    - describe「買い物リストの空状態」：期間の献立がすべて作った印付き
    - 完了判定: `task test:e2e` で上記が pass する

## 7. ドキュメントと検証

- [ ] 7.1 `docs/product.md` を更新する
    - §4 S01 の品の `⋯` の項目列挙に「作った / 作った印を外す」を足し、印の単位 (同じ作り置きの同じレシピ) とチェックの表示を書く
    - §4 S07 の「作り置きの数え方」に、作った印の付いた作り置きは計上しないことを足す
    - §4 S07 の「この期間の献立」と「空状態」に、作った印の表示とすべて印付きのときのメッセージを足す
    - §5 の「久しぶりに」の最終使用日を、作った印の付いた作り置きの行の日付から数えると書く
    - 完了判定: 差分を目視で確認する
- [ ] 7.2 `docs/design.md` を更新する
    - §4 MealPlan に `meal_plan_cooked (batch_id, recipe_id, cooked_at)` を書き足す
    - §8 の順序案で、`add-mealplan-cook-consume` の前に `add-mealplan-cooked-mark` を置き、cook-consume の説明を「作った印に合わせて在庫を減らす」に改める
    - 完了判定: 差分を目視で確認する
- [ ] 7.3 `docs/wireframes/S01_meal_calendar.v2.drawio.svg` の品行の 1 つに、レシピ名の前のチェックの印を描き足す
    - 可視 SVG と `content=` 属性内の編集 XML の両方を更新する (`docs/wireframes/README.md` の手順に従う)
    - 完了判定: ブラウザで SVG を開いて印が見えることと、drawio で開いて同じ位置に印があることを目視で確認する
- [ ] 7.4 `docs/reviews/2026-09-25-design-ux-review.md` の R4 のチェックを付け、`add-mealplan-cooked-mark` で印を入れ、在庫の消費は `add-mealplan-cook-consume` に回した旨を追記する。
  完了判定: 差分を目視で確認する
- [ ] 7.5 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [ ] 7.6 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [ ] 7.7 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [ ] 7.8 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [ ] 7.9 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [ ] 7.10 `openspec validate add-mealplan-cooked-mark --strict` を実行する。完了判定: エラー 0 件
- [ ] 7.11 `task build:win` で Windows ビルドを作り、既存の DB に 00008 が当たって起動することと、S01 で印を付け外しできることを確かめる。完了判定: 目視で確認する
