## 1. config とドメイン層

- [x] 1.1 `internal/config/defaults.go` を更新し、`Config` に `ShoppingPeriodDayPresets []int` と `ShoppingDefaultPeriodDays int` を追加する。
      `Defaults()` では `[]int{3, 4, 7, 14, 28}` と `7` を返す (design.md D9)。完了判定: `go build ./internal/...` が通る
- [x] 1.2 `internal/config/defaults_test.go` に既定値のテストを追加する
    - `ShoppingPeriodDayPresets` が `[3, 4, 7, 14, 28]` である
    - `ShoppingDefaultPeriodDays` が 7 で、`ShoppingPeriodDayPresets` に含まれる
    - `ShoppingPeriodDayPresets` が 7 / 14 / 28 (S01 の 1 / 2 / 4 週) をすべて含む
    - 完了判定: `go test ./internal/config/...` が pass する
- [x] 1.3 `internal/domain/shopping/list.go` を新設し、design.md D10 の型を定義する (JSON タグは camelCase)。
      完了判定: `go build ./internal/...` が通る
    - `PlannedBatch`：`FirstDate` / `FirstMeal` / `FirstPosition` / `BatchID` / `RecipeID` / `RecipeName` / `Dates []string` / `Meals []string` / `BatchMemberCount` /
      `StartedBeforePeriod` / `Ingredients []recipe.RecipeIngredient` (design.md D14)
    - `Dish`：`BatchID` / `RecipeID` / `RecipeName` / `Dates []string` / `Meals []string` / `BatchMemberCount` / `StartedBeforePeriod`
    - `StockAmount`：`Name` / `Amount` / `Unit` (期限切れを除いた在庫 1 行)
    - `StockStatus`：`none` / `partial` / `covered` の 3 値
    - `Item`：`Name` / `Unit` / `Amount` / `StockStatus` / `Recipes []string`
    - `List`：`PeriodStart` / `PeriodEnd` / `PeriodDays` / `Dishes []Dish` / `Items []Item`
- [x] 1.4 `internal/domain/shopping/errors.go` を新設し、`ErrInvalidPeriodDays` を定義する。完了判定: `go build ./internal/...` が通る
- [x] 1.5 `internal/domain/shopping/aggregate.go` を新設し、丸め桁数の定数 (小数第 2 位) と
      `Aggregate(batches []PlannedBatch, stocks []StockAmount) []Item` を実装する (design.md D4 / D5 / D7、spec の並び順)。
      完了判定: 1.7 のテストが pass する
- [x] 1.6 `aggregate.go` に `Dishes(batches []PlannedBatch) []Dish` を実装する。並び順は `FirstDate`、`FirstMeal` (朝 / 昼 / 夕)、
      `FirstPosition` の昇順 (design.md D14)。完了判定: 1.7 の Dishes ケースが pass する
- [x] 1.7 `internal/domain/shopping/aggregate_test.go` に table-driven test を追加する
    - 作り置き 0 件なら空スライス (nil でない) を返す
    - 1 レシピ 1 材料で在庫なしなら、必要量 = レシピの量、状態 `none` の 1 行を返す
    - 2 レシピの同名同単位の材料は 1 行に合算され、`Recipes` に両方のレシピ名が `FirstDate` 順で入る
    - 同じレシピが 2 回計上されたとき、`Recipes` にレシピ名は 1 回だけ入り、量は 2 回分になる
    - 同名で単位が違う材料は 2 行になる
    - 食材名と単位の前後の空白は無視して突き合わせる
    - 在庫が必要量より少なければ状態は `partial` になる
    - 在庫が必要量と等しければ状態は `covered` になる
    - 同名同単位の在庫 2 行は合計して判定する
    - 同名で単位違いの在庫は判定に含まれず、状態は `none` になる
    - 0.1 + 0.2 の必要量と 0.3 の在庫は `covered` になる (丸め誤差で `partial` にならない)
    - 量 0 の材料は、同名同単位の在庫が無ければ `none`、あれば `covered` になる
    - `covered` の行は、`FirstDate` が早くても `none` / `partial` の行より後ろに並ぶ
    - 同じまとまりの中では `FirstDate` の昇順、同日なら食材名の昇順、さらに単位の昇順になる
    - Dishes: 作り置き 0 件なら空スライス (nil でない) を返す
    - Dishes: `PlannedBatch` の日付 / 時間帯 / 作り置き行数 / 期間前フラグがそのまま `Dish` に移る
    - Dishes: `FirstDate` の昇順に並ぶ
    - Dishes: 同じ `FirstDate` なら朝 / 昼 / 夕の順、同じ時間帯なら `FirstPosition` の昇順に並ぶ
    - 同じ日の同じ時間帯の 2 品は、`Recipes` に `FirstPosition` の順で入る
    - 完了判定: `go test ./internal/domain/shopping/...` が pass する

## 2. repository 層

- [x] 2.1 `internal/storage/sqlite/shopping_repo.go` を新設し、`ShoppingRepo` と `NewShoppingRepo(db)` を定義する。
      完了判定: `go build ./internal/...` が通る
- [x] 2.2 `shopping_repo.go` に `ListPlannedBatches(ctx, periodStart string, periodDays int) ([]shopping.PlannedBatch, error)` を実装する。
      `meal_plans` を期間 (`date >= start AND date < start + days`) で絞り、`(batch_id, recipe_id)` で `GROUP BY` して
      期間内の `MIN(date)` を `FirstDate` にする (design.md D3)。期間内で最初の行の時間帯と position を `FirstMeal` / `FirstPosition` に、期間内の日付と時間帯を重複なく詰め、
      `batch_id` の全行数を `BatchMemberCount`、開始日より前に同じ組の行があるかを `StartedBeforePeriod` (`EXISTS`) にする (design.md D14)。
      材料は `recipe_ingredients.position` 順に詰め、
      材料が 0 件のレシピも `Ingredients` を空スライスにして返す。返り値は `FirstDate` の昇順。完了判定: 2.3 のテストが pass する
- [x] 2.3 `internal/storage/sqlite/shopping_repo_test.go` を新設し、`t.TempDir()` の実 SQLite で以下を検証する
    - 献立が 0 件なら空スライス (nil でない) を返す
    - 単独割当 1 件は、その日付を `FirstDate` とする 1 件になり、材料が position 順に入る
    - 同じ `batch_id` の 3 行 (3 日分) は 1 件になり、`FirstDate` は最も早い日付、`Dates` は 3 日分、`BatchMemberCount` は 3 になる
    - 単独割当は `BatchMemberCount` が 1、`StartedBeforePeriod` が false になる
    - 同じ作り置きで朝食と夕食に置かれた行は、`Meals` に朝 / 夕の順で重複なく入る
    - 同じ日の同じ時間帯の 2 品は、`FirstMeal` が同じで `FirstPosition` がセル内の順になる
    - 同じレシピでも `batch_id` が違う 2 行は 2 件になる
    - 同じ `batch_id` に異なるレシピが混ざると、レシピごとに 1 件ずつになる
    - 期間の前日から期間初日にまたがる作り置きは 1 件になり、`FirstDate` は期間初日、`Dates` は期間初日だけ、
      `StartedBeforePeriod` は true、`BatchMemberCount` は 2 になる
    - 期間の最終日から翌日にまたがる作り置きは `StartedBeforePeriod` が false で、`BatchMemberCount` は 2 になる
    - 期間外の行だけの作り置きは含まれない
    - 3 日の期間では、開始日から 3 日目までの行だけが対象になる
    - 材料が 0 件のレシピも、空の `Ingredients` で 1 件として返る
    - 朝食と昼食と夕食の献立はいずれも対象になる
    - 完了判定: `go test ./internal/storage/sqlite/...` が pass する

## 3. service 層

- [x] 3.1 `internal/service/shopping.go` を新設し、`ShoppingService` (フィールド `repo *sqlite.ShoppingRepo` / `stock *sqlite.StockRepo` /
      `cfg config.Config` / `now func() time.Time`) と `NewShoppingService(db *sql.DB)` を定義する。完了判定: `go build ./internal/...` が通る
- [x] 3.2 `shopping.go` に `List(ctx, periodStart string, periodDays int) (shopping.List, error)` を実装する。
      完了判定: 3.3 のテストが pass する
    - 開始日は `mealplan.ValidateDate` で検証する (design.md D13)
    - 日数が `s.cfg.ShoppingPeriodDayPresets` に無ければ `shopping.ErrInvalidPeriodDays` を返す
    - 在庫は `s.stock.List` で読み、`fridge.Evaluate` が `expired` を返す行を除いて `shopping.StockAmount` に変換する。
      閾値は `s.cfg.ExpiryAlertRedDays` / `s.cfg.ExpiryAlertYellowDays` から読む (design.md D6)
    - `PeriodEnd` は開始日 + 日数 − 1 日、`Dishes` は `shopping.Dishes`、`Items` は `shopping.Aggregate` の結果にする
- [x] 3.3 `internal/service/shopping_test.go` を新設し、BDD 風 (`t.Run("should ... when ...")`) で以下を検証する (now は固定日付、実 SQLite)
    - should return empty dishes and empty items when no meal plans exist in period
    - should return period end as the last day included when period days is 3
    - should count ingredients once when a batch spans three days
    - should count ingredients per cooking when the same recipe is assigned separately twice
    - should count a batch once when it starts before the period and continues into it
    - should mark covered when non-expired stock with same name and unit is enough
    - should mark none when the only stock is expired
    - should count stock that expires today
    - should count stock that has no expiry
    - should return dishes and empty items when planned recipes have no ingredients
    - should mark started before period when a batch has a row before period start
    - should return ErrInvalidDate when period start is not a valid date (例: 2026-02-30)
    - should return ErrInvalidPeriodDays when period days is 5
    - 完了判定: `go test ./internal/service/...` が pass する
- [x] 3.4 `main.go` を更新し、`application.Options.Services` に `application.NewService(service.NewShoppingService(db))` を追加する。
      完了判定: `go build .` が通る
- [x] 3.5 `cmd/e2e-server/main.go` を更新し、`newRegistry` に `"ShoppingService": service.NewShoppingService(db)` を追加し、
      コメントの「4 サービス」を「5 サービス」に直す。完了判定: `go build ./cmd/...` が通る
- [x] 3.6 `cmd/e2e-server/dispatch_test.go` の `TestNewRegistry` の期待一覧に `ShoppingService` を追加する。
      完了判定: `go test ./cmd/...` が pass する
- [x] 3.7 `wails3 generate bindings -clean=true -ts -i` で bindings を再生成する。
      完了判定: `frontend/bindings/github.com/yito0000/meal-cycle/internal/service/` に `shoppingservice.ts` が生成される

## 4. フロントエンド共通部品

- [x] 4.1 `frontend/src/lib/format-amount.ts` を新設し、`frontend/src/components/fridge/stock-form.ts` の `formatAmount` を
      挙動を変えずに移設する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/src/components/fridge/stock-form.ts` から `formatAmount` を削除する。
      完了判定: `grep -n formatAmount frontend/src/components/fridge/stock-form.ts` が何も返さない
- [x] 4.3 `frontend/src/components/fridge/fridge-item-row.tsx` の `formatAmount` の import 元を `@/lib/format-amount` に変える。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.4 `frontend/src/components/shopping/shopping-config.ts` を新設し、`SHOPPING_DEFAULTS` (`periodDayPresets: [3, 4, 7, 14, 28]` /
      `defaultPeriodDays: 7`) を `internal/config/defaults.go` の mirror として定義する (同期元をコメントで示す)。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.5 `frontend/src/components/shopping/shopping-period.ts` を新設し、search の zod スキーマ (`from` / `days` とも任意) と
      `resolveShoppingPeriod(search, today: Date): { periodStart: string; periodDays: number }` を実装する。
      `from` が実在する日付で `days` が `SHOPPING_DEFAULTS.periodDayPresets` に含まれるときだけ採用し、
      それ以外は今日 + `defaultPeriodDays` を返す (design.md D8)。日付の整形は `frontend/src/components/mealplan/week-utils.ts` の
      `formatDate` を使う。完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.6 `frontend/src/router.tsx` を更新し、`shoppingRoute` に 4.5 のスキーマで `validateSearch` を付ける。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.7 `frontend/src/components/shopping/use-shopping-list.ts` を新設し、queryKey `["shopping", periodStart, periodDays]` で
      `ShoppingService.List` を呼ぶ `useShoppingList` を実装する。完了判定: `cd frontend && npm run type-check` が通る

## 5. S07 画面と S01 の導線

- [x] 5.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する。
      完了判定: 両ファイルで同じ key 集合になっていることを目視で確認する
    - `pages.shopping.placeholder` を削除する
    - `pages.shopping.description` (「この期間の献立で使う材料の目安です」) を追加する
    - `pages.shopping.period` (「対象期間: {{start}} 〜 {{end}}」) を追加する
    - `pages.shopping.picker.start` (「開始日」) / `picker.days` (「日数」) / `picker.day_option` (「{{count}} 日」) を追加する
    - `pages.shopping.column.name` / `column.amount` / `column.stock` / `column.usage` を追加する
    - `pages.shopping.stock_status.none` / `stock_status.partial` / `stock_status.covered` を追加する
    - `pages.shopping.empty.no_plans` / `empty.no_ingredients` / `loading` / `load_error` を追加する
    - `pages.shopping.dishes.title` (「この期間の献立」) / `dishes.batch` (作り置きの印の aria-label) /
      `dishes.started_before` (「期間前から」) / `dishes.date_range` / `dishes.meal.breakfast` / `meal.lunch` / `meal.dinner` を追加する
    - `pages.meal_plan.shopping_cta` (「この期間の買い物リストを見る」) を追加する
- [x] 5.2 `frontend/src/components/shopping/shopping-period-picker.tsx` を新設する。
      `src/components/ui/input.tsx` の `type="date"` で開始日、`SHOPPING_DEFAULTS.periodDayPresets` のボタン群で日数を選び、
      選択中の日数は `aria-pressed` で示す。空の日付入力は無視する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.3 `frontend/src/components/shopping/stock-status-badge.tsx` を新設し、3 状態を文字 + 色のバッジで描画する
      (`data-stock-status` 属性に状態値を載せる)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.4 `frontend/src/components/shopping/shopping-table.tsx` を新設し、`src/components/ui/table.tsx` で
      「食材 / 必要量 / 在庫 / 用途」の 4 列テーブルを描画する。必要量は `formatAmount` + 単位、量 0 は「—」を出す。
      `covered` の行は薄い文字色にする。用途はレシピ名を「, 」で連結する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.5 `frontend/src/components/shopping/format-dish-dates.ts` を新設し、`formatDishDates(dates: string[], t)` を実装する。
      連続する日付は「9/24〜26」、月をまたぐ連続は「9/30〜10/2」、飛び飛びは「9/24, 9/26」を返す (design.md D14)。
      完了判定: `cd frontend && npm run type-check` が通り、6.5 の献立一覧シナリオが pass する
- [x] 5.6 `frontend/src/components/shopping/shopping-dish-list.tsx` を新設し、「この期間の献立」欄を描画する。
      各行はレシピ名、`formatDishDates` の日付、時間帯、`batchMemberCount > 1` なら 🔗 (`Link2` アイコン + aria-label)、
      `startedBeforePeriod` なら「期間前から」のバッジを出す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.7 `frontend/src/routes/shopping.tsx` を S07 本体に差し替える。search から `resolveShoppingPeriod` で期間を決め、
      見出し / 説明文 / 期間の切替 / 対象期間 / 読み込み中 / 読み込み失敗 / 空状態 2 種 (`dishes` と `items` で判定) / テーブル / 献立の欄を出し分ける。
      本体は `lg:grid-cols-[1fr_20rem]` の 2 列、狭い幅では 1 列にする。`dishes` が空のときは献立の欄を出さない。
      切替は `navigate({ to: "/shopping", search: { from, days }, replace: true })` で反映する。
      完了判定: `task dev` で S07 を開き、サイドバーから開くと今日から 7 日間、「3 日」を押すと 3 日間になることを目視で確認する
- [x] 5.8 `frontend/src/routes/meal-plan.tsx` を更新し、カレンダーの下に「この期間の買い物リストを見る」ボタンを置く。
      押すと `navigate({ to: "/shopping", search: { from: periodStartKey, days: periodWeeks * 7 } })` で遷移する。
      完了判定: `task dev` で S01 を 2 週表示にしてボタンを押し、S07 の対象期間が 14 日分になることを目視で確認する

## 6. E2E

- [x] 6.1 `frontend/e2e/bindings-shim/shoppingservice.ts` を新設し、`List` を `call("ShoppingService", "List", ...)` で転送する
      (戻り値型は生成物より狭く書く)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.2 `frontend/e2e/bindings-shim/index.ts` を更新し、`ShoppingService` を re-export と `_typeCheck` に追加する。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.3 `frontend/e2e/helpers.ts` を更新し、以下のヘルパを追加する。完了判定: `cd frontend && npm run type-check` が通る
    - `openShoppingList(page)`：サイドバーから S07 を開き、見出しを待つ
    - `createRecipeWithIngredients(page, name, ingredients)`：`/api/RecipeService/Create` で作成し、ID を返す
    - `assignRecipe(page, date, meal, recipeId)`：`/api/MealPlanService/AssignRecipe` で 1 品割り当てる
    - `copyLeftover(page, date, meal, days, shareBatch)`：`/api/MealPlanService/CopyLeftover` で作り置きを広げる
- [x] 6.4 `docs/scenarios/shopping.feature` を書き換える。完了判定: `openspec/changes/add-shopping-list/specs/shopping/spec.md` の
      各 Requirement に対応するシナリオが 1 つ以上あることを目視で確認する
    - 機能名「献立から買い物リストを集計する」は変えない
    - 「サイドバーから直接開いた場合の既定期間」の前提を「S01 で 4 週表示に切り替えたあと」に直し、結果を「今日から 7 日間」と書く
    - 「在庫にある分は自動除外される」を「在庫で一部を賄える食材は一部在庫ありと表示される」に置き換える
    - 「必要量が在庫量以下の食材は表示されない」を「在庫で足りる食材は在庫ありとして末尾に表示される」に置き換える
    - 「複数レシピで必要な同一食材は 1 行に合算される」と「同一時間帯内の複数品も合算対象になる」から「冷蔵庫に在庫はない」前提を外す
    - シナリオ「S07 で開始日と日数を切り替える」を追加する
    - シナリオ「作り置きは何日分でも材料を 1 回分だけ計上する」を追加する
    - シナリオ「期限切れの在庫は在庫として扱わない」を追加する
    - シナリオ「期間に献立が無いと空状態になる」を追加する
    - シナリオ「献立のレシピに材料が無いと空状態になる」を追加する
    - シナリオ「作り置きは献立の一覧で 1 行にまとまる」を追加する
    - シナリオ「期間より前から続く作り置きには印が付く」を追加する
    - 機能「購入済チェックと在庫への一括登録」はそのまま残す
- [x] 6.5 `frontend/e2e/shopping.spec.ts` を新設し、`shopping.feature` の `シナリオ:` 行と一字一句同じテスト名で実装する。
      日付は `dateAfter` と今週の月曜からの相対で組み立てる。完了判定: `task test:e2e` で下記がすべて pass する
    - describe「機能: 献立から買い物リストを集計する」
      - S01 の期間切替が S07 に引き継がれる
      - サイドバーから直接開いた場合の既定期間
      - S07 で開始日と日数を切り替える
      - 在庫で一部を賄える食材は一部在庫ありと表示される
      - 在庫で足りる食材は在庫ありとして末尾に表示される
      - 複数レシピで必要な同一食材は 1 行に合算される
      - 同一時間帯内の複数品も合算対象になる
      - 作り置きは何日分でも材料を 1 回分だけ計上する
      - 期限切れの在庫は在庫として扱わない
      - 期間に献立が無いと空状態になる
      - 献立のレシピに材料が無いと空状態になる
      - 作り置きは献立の一覧で 1 行にまとまる
      - 期間より前から続く作り置きには印が付く
    - describe「機能: 購入済チェックと在庫への一括登録」は `add-shopping-purchase` 用に `test.skip` で名前だけ置く
      - 購入済チェックしても在庫は増えない
      - チェック済みを一括で冷蔵庫に登録する
      - 中間ステップを取り消せる

## 7. ドキュメント

- [x] 7.1 `docs/product.md` §1 の可変値一覧に 2 行を追加する。完了判定: 差分を目視で確認する
    - 「買い物リストの日数プリセット」(3 / 4 / 7 / 14 / 28 日、S09 設定、MVP で変更不可)
    - 「買い物リストの既定日数」(7 日、S09 設定、MVP で変更不可)
- [x] 7.2 `docs/product.md` §4 S07 を書き換える。完了判定: 差分を目視で確認する
    - リストを「期間の献立で使う材料の目安」と位置づけ、在庫で足りる材料も表示することを書く
    - 対象期間は S01 から開けば S01 の期間、サイドバーからは今日から 7 日間で、S07 上で開始日と日数を切り替えられることを書く
    - 作り置きは期間内に何日分あっても 1 回分として数えることを書く
    - 在庫の状態 3 段階、期限切れと単位違いの在庫は数えないこと、「在庫あり」の行は末尾にまとめることを書く
    - 列 (食材 / 必要量 / 在庫 / 用途)、空状態 2 種を書く
    - 右側の「この期間の献立」欄 (1 行 = 調理 1 回分、日付と時間帯、🔗、「期間前から」の印、参照専用) を書く
    - 「冷蔵庫にある分は自動除外」の記述を削除する
    - チェックボックスと一括登録は `add-shopping-purchase` で入ることを書く

## 8. 検証

- [x] 8.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 8.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 8.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 8.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する (skip は除く)
- [x] 8.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 8.6 `openspec validate add-shopping-list --strict` を実行する。完了判定: エラー 0 件

## 9. 開始日のカレンダーと日付表記 (task dev での確認を受けた見直し、design.md D15)

- [x] 9.1 `frontend/package.json` に `react-day-picker` と `@radix-ui/react-popover` を追加する。完了判定: `cd frontend && npm install` が成功する
- [x] 9.2 `frontend/src/components/ui/popover.tsx` を新設し、shadcn の Popover を手動コピーする。完了判定: `cd frontend && npm run type-check` が通る
- [x] 9.3 `frontend/src/components/ui/calendar.tsx` を新設し、shadcn の Calendar を手動コピーしたうえで、日付セルを大きめ (40px 四方、text-base) にする。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 9.4 `frontend/src/lib/format-date.ts` を新設し、`YYYY-MM-DD` を `yyyy/mm/dd` に変える `formatDisplayDate` を実装する。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 9.5 `frontend/src/components/ui/date-picker.tsx` を新設する。`yyyy/mm/dd` を表示するボタンと、押すと開く Calendar を持ち、
      日付を選ぶと `YYYY-MM-DD` で `onChange` を呼んで閉じる。曜日と月の見出しは i18next の言語に合わせる。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 9.6 `frontend/src/components/shopping/shopping-period-picker.tsx` の `<input type="date">` を DatePicker に置き換える。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 9.7 `frontend/src/routes/shopping.tsx` の対象期間の表示を `formatDisplayDate` で `yyyy/mm/dd` にする。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 9.8 `frontend/e2e/shopping.spec.ts` を更新する。完了判定: `task test:e2e` で shopping のシナリオがすべて pass する
    - 対象期間の期待値を `yyyy/mm/dd` 表記にする
    - 「S07 で開始日と日数を切り替える」の開始日の変更を、カレンダーを開いて明日の日付を選ぶ操作にする (月をまたぐ場合は翌月へ送る)
- [x] 9.9 `go test ./internal/... ./cmd/...` / `npm run type-check` / `npm run build:dev` / `task test:e2e` / `task lint` /
      `openspec validate add-shopping-list --strict` を実行する。完了判定: すべて成功する
