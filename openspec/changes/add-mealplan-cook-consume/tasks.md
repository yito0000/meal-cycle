## 1. 候補の算出 (ドメイン)

- [x] 1.1 `internal/domain/fridge/consume.go` を新設し、型を定義する (design.md D3)。完了判定: `go build ./internal/...` が通る
    - `ConsumptionCandidate`：`StockID` / `ExpiresAt` / `AddedAt` / `CurrentAmount` / `Amount` (減らす量の初期値)
    - `ConsumptionLine`：`Name` / `Unit` / `RequiredAmount` / `Candidates []ConsumptionCandidate`
    - `ConsumptionPlan`：`Lines []ConsumptionLine` / `Missing []string`
    - `Consumption`：`StockID` / `Amount`
- [x] 1.2 `internal/domain/fridge/consume.go` に `PlanConsumption(ingredients []recipe.RecipeIngredient, stocks []StockItem, stapleNames []string, measuringUnits []string) ConsumptionPlan` を実装する。
  `stocks` は期限切れを除いた在庫を受け取る前提にし、比較は `ingredient.NormalizeName` で行う。完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/domain/fridge/consume_test.go` を新設し、table-driven でテストする。完了判定: `go test ./internal/domain/fridge/` が pass する
    - 賞味期限の近い在庫から割り当て、期限なしの在庫は最後になる
    - 同じ期限の在庫は追加日時の古い順に割り当てる
    - 在庫より多い材料は在庫の量までしか割り当てない
    - 割り当てが 0 の在庫も Amount 0 で返る
    - 単位が違う在庫は候補にしない
    - カタカナとひらがなの違う食材名は同じ食材として候補にする
    - 量 0 の材料と量る単位の材料は Lines にも Missing にも入らない
    - 常備品と同名の材料は Lines にも Missing にも入らない
    - 同じ食材名と単位の材料が 2 行あると量を合計して 1 行になる
    - 在庫の無い材料は Missing に入る

## 2. 在庫の反映 (リポジトリとサービス)

- [x] 2.1 `internal/domain/fridge/errors.go` に `ErrConsumeAmountOutOfRange` と `ErrConsumeDuplicateStock` を `fridge: consume ...` の文言で追加する。完了判定: `go build ./internal/...` が通る
- [x] 2.2 `internal/storage/sqlite/stock_repo.go` に `Consume(ctx, []fridge.Consumption) error` を追加する (design.md D4)
    - 1 トランザクションで各行を読み、量が等しければ DELETE、少なければ amount を減らして updated_at を進める。0 の行は飛ばす
    - 存在しない ID は `ErrStockNotFound`、今の量を超える量は `ErrConsumeAmountOutOfRange` を返し、ロールバックする
    - 完了判定: `go build ./internal/...` が通る
- [x] 2.3 `internal/storage/sqlite/stock_repo_test.go` に `TestStockRepo_Consume` を追加する。完了判定: `go test ./internal/storage/sqlite/ -run TestStockRepo_Consume` が pass する
    - 一部を減らすと amount が減り、added_at と expires_at は変わらない
    - 全量を減らすと行が消える
    - 0 の行は変わらない
    - 1 行でも今の量を超えるとどの行も変わらない
    - 存在しない ID を含むとどの行も変わらない
- [x] 2.4 `internal/service/fridge.go` を更新し、`FridgeService` に献立とレシピのリポジトリ (`sqlite.NewMealPlanRepo(db)` / `sqlite.NewRecipeRepo(db)`) と常備品のリポジトリを持たせる (design.md D2)。完了判定: `go build .` が通る
- [x] 2.5 `internal/service/fridge.go` に `PlanCookConsumption(ctx, entryID string) (fridge.ConsumptionPlan, error)` を追加する
    - 品からレシピを引き、期限切れを除いた在庫、常備品の名前、S09 の量る単位で `PlanConsumption` を呼ぶ
    - 品が無ければ `mealplan.ErrEntryNotFound` を返す
    - 完了判定: `go build .` が通る
- [x] 2.6 `internal/service/fridge.go` に `Consume(ctx, items []fridge.Consumption) error` を追加する。量が負なら `ErrConsumeAmountOutOfRange`、同じ ID が 2 回出たら `ErrConsumeDuplicateStock` を返してから repo を呼ぶ。
  完了判定: `go build .` が通る
- [x] 2.7 `internal/service/fridge_test.go` に BDD 風のテストを追加する。完了判定: `go test ./internal/service/ -run 'TestFridgeService_(PlanCookConsumption|Consume)'` が pass する
    - `TestFridgeService_PlanCookConsumption`
      - should return candidates ordered by expiry when matching stocks exist
      - should exclude expired stocks when building candidates
      - should return empty lines when recipe has no matching stock
      - should return ErrEntryNotFound when entry does not exist
    - `TestFridgeService_Consume`
      - should reduce amounts and delete emptied rows when all rows are valid
      - should return ErrConsumeAmountOutOfRange when amount is negative
      - should return ErrConsumeDuplicateStock when same stock appears twice
      - should leave all stocks unchanged when any row is invalid
- [x] 2.8 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/e2e/bindings-shim/fridgeservice.ts` に `PlanCookConsumption` と `Consume` を追加する。完了判定: `cd frontend && npm run type-check` が通る

## 3. S01 のダイアログ

- [x] 3.1 `frontend/src/components/mealplan/consume-form.ts` を新設する
    - `toConsumeRows(plan)`：候補ごとに減らす量の文字列を初期値に持つ行を作る
    - `validateConsumeRows(rows)`：空、数値でない、負、今の量を超える行にエラーを付ける
    - `toConsumptions(rows)`：0 の行を除いて `Consumption[]` を返す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/components/mealplan/consume-form.test.ts` を新設し、`it.each` でテストする。完了判定: `cd frontend && npm run test` が pass する
    - 初期値は候補の Amount の表示になる
    - 今の量を超える値はエラーになる
    - 負の値と数値でない値はエラーになる
    - 0 の行は toConsumptions から除かれる
- [x] 3.3 `frontend/src/components/mealplan/consume-stock-dialog.tsx` を新設する (design.md D5)
    - props は `plan: ConsumptionPlan | null`、`onClose()`、`onConsumed(count)`
    - 材料ごとの見出しに食材名とレシピの量、その下に賞味期限 / 今の量 / 減らす量の行を並べる
    - `Missing` があれば「在庫が見つからない材料」として名前を並べる
    - 「減らす」で `FridgeService.Consume` を呼び、成功したら `["fridge"]` を invalidate する。「減らさない」と Esc で閉じる
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.4 `frontend/src/components/mealplan/week-calendar.tsx` を更新し、`toggle-cooked` で `cooked` を true にした `SetCooked` が成功したら `PlanCookConsumption` を呼ぶ。
  候補の在庫が 1 件以上ならダイアログを開き、消費の件数をトースト (`pages.meal_plan.consume.consumed`) で知らせる。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.5 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.meal_plan.consume.*` を追加する
    - `title`：「使った在庫を減らす」、`description`：「作った料理で使った分だけ冷蔵庫の在庫を減らします。量は直せます」
    - `column.expires_at` / `column.current` / `column.amount`、`missing_title`：「在庫が見つからない材料」
    - `submit`：「減らす」、`skip`：「減らさない」、`consumed`：「在庫を {{count}} 件減らしました」
    - `errors.amount_out_of_range`：「0 以上、今の量以下の数を入れてください」
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 4. E2E

- [x] 4.1 `frontend/e2e/mealplan-cook-consume.spec.ts` を新設する。describe 名は Requirement 名、test 名は Scenario 名と一字一句そろえる。完了判定: `task test:e2e` で以下が pass する
    - describe「作った印を付けたときの在庫の消費」
      - 印を付けると減らせる在庫がダイアログに並ぶ
      - 減らせる在庫が無いとダイアログを開かない
      - 減らさないを選んでも印は残る
      - 印を外しても在庫は変わらない
- [x] 4.2 `frontend/e2e/fridge-cook-consume.spec.ts` を新設する。シナリオ検査はファイル名の capability で照合するので、fridge の Requirement はこちらに置く。
  完了判定: `task test:e2e` で以下が pass する
    - describe「料理で使った在庫の候補」
      - 賞味期限の近い在庫から割り当てる
      - 在庫の無い材料は名前だけ示す
    - describe「料理で使った在庫の反映」
      - 減らすと在庫の量が減る
      - 使い切った在庫は消える
      - 今の量より多く減らそうとすると何も変わらない

サービス層とドメイン層で確かめるシナリオ (「在庫より多く使う材料は在庫の量までしか割り当てない」「期限切れと単位の違う在庫は候補にしない」
「量を持たない材料は候補にしない」「画面を経由しない不正な消費は拒否される」) は E2E に含めない。

## 5. ドキュメントと検証

- [x] 5.1 `docs/product.md` を更新する。完了判定: 差分を目視で確認する
    - §1 ナラティブの「在庫と献立の自動連動は MVP 後」を、作った印から確認のダイアログで在庫を減らせる記述に改める
    - §4 S01 の「作った」チェックボックスの項に、印を付けたときのダイアログと、印を外しても在庫が戻らないことを足す
    - §7 決定事項に、自動では減らさず確認のダイアログを挟む判断と理由を足す
- [x] 5.2 `docs/design.md` §9 の「料理実施と在庫消費の連動」に proposal 名を添える。完了判定: 差分を目視で確認する
- [x] 5.3 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 5.4 `cd frontend && npm run test` と `npm run type-check` と `npm run build:dev` を実行する。完了判定: 3 つとも成功する
- [x] 5.5 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 5.6 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 5.7 `openspec validate add-mealplan-cook-consume --strict` を実行する。完了判定: エラー 0 件
