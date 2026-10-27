## 1. 候補の算出 (ドメイン)

- [x] 1.1 `internal/config/defaults.go` に `SuggestionLimit int` (既定 5) を追加し、`defaults_test.go` に既定値の検査を足す。完了判定: `go test ./internal/config/` が pass する
- [x] 1.2 `internal/domain/suggestion/suggestion.go` を新設し、`StockReason` (`Name` / `DaysLeft`)、`Candidate` (`RecipeID` / `RecipeName` / `Stocks`)、
  `Result` (`ByStock []Candidate`) を JSON タグ付きで定義する (design.md D2)。完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/domain/suggestion/by_stock.go` を新設し、`ByStock(recipes []recipe.Recipe, stocks []fridge.StockItem, today time.Time, red, yellow, limit int) []Candidate` を実装する (design.md D3)。
  比較は `ingredient.NormalizeName` で行い、削除済み (`ArchivedAt` が空でない) レシピは除く。完了判定: `go build ./internal/...` が通る
- [x] 1.4 `internal/domain/suggestion/by_stock_test.go` を新設し、table-driven でテストする。完了判定: `go test ./internal/domain/suggestion/` が pass する
    - 🔴 と 🟡 の在庫を持つレシピが候補になる
    - 🟢、期限切れ、期限なしの在庫だけのレシピは候補にならない
    - 単位が違っても食材名が同じなら候補になる
    - カタカナとひらがなの違う食材名も一致する
    - 一致した在庫の数の降順、最も近い期限の昇順、レシピ名の昇順に並ぶ
    - 同じ食材名の在庫が 2 行あっても 1 件として数え、近いほうの期限を使う
    - 削除済みのレシピは候補にならない
    - limit を超える候補は切られる

## 2. サービス

- [x] 2.1 `internal/service/suggestion.go` を新設し、`SuggestionService` と `NewSuggestionService(db *sql.DB, cfg configSource)` を定義する。
  `List(ctx) (suggestion.Result, error)` で全レシピ (材料つき) と在庫を読み、`ByStock` を呼ぶ。`ByStock` は空でも空スライスで返す。完了判定: `go build .` が通る
- [x] 2.2 `internal/service/suggestion_test.go` を新設し、BDD 風にテストする。完了判定: `go test ./internal/service/ -run TestSuggestionService_List` が pass する
    - should return empty by-stock candidates when no stock is near expiry
    - should return recipes using near-expiry stock when such stock exists
    - should follow saved expiry thresholds when settings change
- [x] 2.3 `main.go` を更新し、`application.Options.Services` に `service.NewSuggestionService(db, settingsSvc)` を追加する。完了判定: `go build .` が通る
- [x] 2.4 `cmd/e2e-server/main.go` の registry に `SuggestionService` を足し、`cmd/e2e-server/dispatch_test.go` の `TestNewRegistry` の名前と件数を 7 にそろえる。
  完了判定: `go test ./cmd/...` が pass する
- [x] 2.5 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: `frontend/bindings/` に `SuggestionService` が出る
- [x] 2.6 `frontend/e2e/bindings-shim/suggestionservice.ts` を新設して `List` を書き、`frontend/e2e/bindings-shim/index.ts` で re-export と `_typeCheck` に足す。
  完了判定: `cd frontend && npm run type-check` が通る

## 3. ダイアログ

- [x] 3.1 `frontend/src/components/mealplan/use-suggestions.ts` を新設し、`useSuggestions(enabled)` を `queryKey: ["suggestions"]`、`staleTime: 0` で定義する。
  null の配列を空配列にそろえる。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/components/mealplan/suggestion-section.tsx` を新設する (design.md D4)
    - props は `title`、`countLabel`、`items: { recipeId; recipeName; reason }[]`、`onSelect`
    - 候補が 0 件なら `pages.meal_plan.picker.suggestion.none` を出す
    - 各行に `data-testid="suggestion-item"` を付け、押すと `onSelect` を呼ぶ
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.3 `frontend/src/components/mealplan/recipe-picker-dialog.tsx` を更新する (design.md D4)
    - 幅を `max-w-lg` にする
    - 検索欄が空のとき、検索欄と一覧のあいだに「使い切り」の `SuggestionSection` を置く
    - 根拠は `pages.meal_plan.picker.suggestion.by_stock_reason` に食材名 (「、」区切り) と最も近い期限の日数を渡して組む
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.4 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.meal_plan.picker.suggestion.*` を追加する
    - `by_stock_title`：「使い切り (期限が近い在庫を使う)」、`count`：「{{count}} 件」、`none`：「該当なし」
    - `by_stock_reason`：「{{names}} が期限 {{days}} 日以内」、`by_stock_reason_today`：「{{names}} が今日まで」
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 4. E2E

- [x] 4.1 `frontend/e2e/suggestion.spec.ts` を新設する。describe 名は Requirement 名、test 名は Scenario 名と一字一句そろえる。完了判定: `task test:e2e` で以下が pass する
    - describe「使い切りの候補」
      - 期限の近い在庫を使うレシピが候補に出る
      - 使える在庫の多いレシピが先に並ぶ
      - 期限に余裕のある在庫と期限切れの在庫は候補の根拠にならない
      - 候補を押すとそのレシピが割り当たる
      - 検索中は候補の欄を隠す
      - 削除済みのレシピは候補に出ない

## 5. ドキュメントと検証

- [x] 5.1 `docs/product.md` §4 S01 の「MVP 後拡張」と §5 の「使い切り」の行に、`add-mealplan-suggest-by-stock` で入れたことと、検索中は欄を隠すことを足す。完了判定: 差分を目視で確認する
- [x] 5.2 `docs/design.md` §8 の `add-mealplan-suggest-by-stock` の行に、capability `suggestion` を新設したことを添える。完了判定: 差分を目視で確認する
- [x] 5.3 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 5.4 `cd frontend && npm run test` と `npm run type-check` と `npm run build:dev` を実行する。完了判定: 3 つとも成功する
- [x] 5.5 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 5.6 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 5.7 `openspec validate add-mealplan-suggest-by-stock --strict` を実行する。完了判定: エラー 0 件
