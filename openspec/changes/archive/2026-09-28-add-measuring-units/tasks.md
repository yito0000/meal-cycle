## 1. 量る単位の既定値

- [x] 1.1 `internal/config/defaults.go` を更新し、`Config` に `MeasuringUnits []string` を追加する (design.md D5)
    - 「買い物リスト」の区切りの下に置き、WHY のコメントを 1〜2 行で書く
    - `Defaults()` で `[]string{"大さじ", "小さじ", "カップ"}` を返す
    - 完了判定: `go build ./internal/...` が通る
- [x] 1.2 `internal/config/defaults_test.go` の `TestDefaults` に `MeasuringUnits` の検証を追加する
    - `MeasuringUnits` が `[大さじ 小さじ カップ]` と一致する
    - 完了判定: `go test ./internal/config/...` が pass する
- [x] 1.3 `docs/product.md` §1 の可変値の表に「量る単位 / 大さじ、小さじ、カップ / S09 設定 (`MeasuringUnits`)」の行を追加する。完了判定: 差分を目視で確認する

## 2. 買い物リストの算出

- [x] 2.1 `internal/domain/shopping/aggregate.go` の `Aggregate` に引数 `measuringUnits []string` を追加する (design.md D1)。完了判定: `go build ./internal/...` が通る (呼び出し側は 2.4 で直す)
- [x] 2.2 `internal/domain/shopping/aggregate.go` の合算を、量を持たない材料に対応させる (design.md D3)
    - 量 0 の材料と、trim 後の単位が `measuringUnits` に含まれる材料を、量を持たない材料と判定する
    - 量を持たない材料は、キーを「trim 後の食材名 + 空の単位」にし、足し込む量を 0 にする
    - 量のある材料は、これまでどおり「trim 後の食材名 + trim 後の単位」をキーにして量を足す
    - 完了判定: 2.5 のテストが pass する
- [x] 2.3 `internal/domain/shopping/aggregate.go` の在庫の状態を、量を持たない行に対応させる (design.md D4)
    - 在庫を「食材名 + 単位」で合計した表に加えて、食材名だけで合計した表を作る
    - 必要量の合計が 0 の行は、食材名だけの合計を `stockStatus(0, stock)` に渡す
    - 必要量の合計が 0 より大きい行は、これまでどおり「食材名 + 単位」の合計を渡す
    - 完了判定: 2.5 のテストが pass する
- [x] 2.4 `internal/service/shopping.go` の `List` を更新し、`shopping.Aggregate` に `s.cfg.MeasuringUnits` を渡す。完了判定: `go build ./internal/...` が通る
- [x] 2.5 `internal/domain/shopping/aggregate_test.go` の `TestAggregate` に table-driven のケースを追加する。既存の呼び出しには既定の量る単位を渡す
    - 「醤油 大さじ 2」は単位が空、必要量 0、在庫なしの 1 行になる
    - 「醤油 大さじ 2」と「醤油 小さじ 1」は「醤油」の 1 行にまとまり、用途に両方のレシピが並ぶ
    - 「塩 (量 0、単位 少々)」と「塩 小さじ 1」は「塩」の 1 行にまとまる
    - 「醤油 大さじ 2」は在庫「醤油 1 本」で在庫ありになる
    - 「醤油 大さじ 2」は同名の在庫が無ければ在庫なしになる
    - 「醤油 大さじ 2」と「醤油 1 本」は、単位が空の行と「本」の行の 2 行になる
    - 「塩 2 (単位なし)」と「塩 小さじ 1」は必要量 2 の 1 行になり、在庫「塩 1 袋」は数えず在庫なしになる
    - 前後に空白のある単位「 大さじ 」も量る単位として扱う
    - 量る単位の一覧が空なら「醤油 大さじ 2」はこれまでどおり単位「大さじ」、必要量 2 の行になる
    - 完了判定: `go test ./internal/domain/shopping/...` が pass する
- [x] 2.6 `internal/service/shopping_test.go` の `TestShoppingService_List` にケースを追加する
    - should merge measuring-unit ingredients into one unquantified row when the same name uses tablespoons and teaspoons
    - should mark covered when a measuring-unit ingredient has non-expired stock with a different unit
    - 完了判定: `go test ./internal/service/...` が pass する

## 3. 量と単位の表示

- [x] 3.1 `frontend/src/lib/format-quantity.ts` を新設する (design.md D5 / D6)
    - `MEASURING_UNITS` を定義し、`internal/config/defaults.go` の mirror で source of truth が `defaults_test.go` であることをコメントに書く
    - `formatQuantity(amount, unit)` を定義する。量 0 なら単位だけ、量る単位なら「単位 量」、それ以外は「量 単位」(単位が空なら量だけ) を返す
    - 単位は trim してから量る単位と比べる
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/components/recipe/recipe-detail-drawer.tsx` を更新し、`formatIngredientAmount` を削除して `formatQuantity` を呼ぶ。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.3 `frontend/src/components/fridge/fridge-item-row.tsx` を更新し、量の列を `formatQuantity(item.amount, item.unit)` で表示する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.4 `frontend/src/components/shopping/shopping-table.tsx` を更新し、量が 0 でない行の必要量を `formatQuantity` で表示する。量 0 の行の「—」の分岐は残す。完了判定: `cd frontend && npm run type-check` が通る

## 4. E2E

- [x] 4.1 `frontend/e2e/shopping.spec.ts` に describe「量の指定が無い材料」を追加し、次の Scenario を実装する
    - 量る単位の材料は必要量を「—」で表示する
    - 量を持たない材料は単位が違う在庫でも在庫ありになる
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 4.2 `frontend/e2e/shopping-purchase.spec.ts` の describe「冷蔵庫への一括登録」に Scenario を追加する
    - 量る単位の材料は量と単位が空になる
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 4.3 `frontend/e2e/recipe-detail.spec.ts` の describe「S03 レシピ詳細ドロワーの表示」に Scenario を追加する
    - 量る単位は量の前に表示される
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 4.4 `frontend/e2e/fridge.spec.ts` に describe「在庫一覧の表示」を追加し、Scenario を実装する
    - 量る単位の在庫は単位を量の前に表示する
    - 完了判定: `task test:e2e` で上記が pass する

## 5. ドキュメントと検証

- [x] 5.1 `docs/product.md` §4 S07 の「材料テーブル」の項を更新する
    - 量 0 の材料の説明を、量を持たない材料 (量 0 と量る単位) の説明に置き換える
    - 量を持たない材料は食材名だけで 1 行にまとめ、在庫は単位を問わず数えることを書く
    - 完了判定: 差分を目視で確認する
- [x] 5.2 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 5.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 5.4 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 5.5 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 5.6 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 5.7 `openspec validate add-measuring-units --strict` を実行する。完了判定: エラー 0 件
