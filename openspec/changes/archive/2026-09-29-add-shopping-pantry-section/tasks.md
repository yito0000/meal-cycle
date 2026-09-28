## 1. 量を持たない材料の判定

- [x] 1.1 `frontend/src/components/shopping/pantry.ts` を新設する (design.md D1)
    - `isPantryItem(item)` を定義し、`item.amount === 0` のとき true を返す
    - 必要量 0 の行が量を持たない材料の行だけであるという、Go 側の性質に依存していることを WHY のコメントで 1〜2 行書く
    - 完了判定: `cd frontend && npm run type-check` が通る

## 2. 画面

- [x] 2.1 `frontend/src/components/shopping/shopping-pantry-list.tsx` を新設する (design.md D2 / D3)
    - props は `items: (Item & { recipes: string[] })[]`
    - 見出し (`pages.shopping.pantry.title`) と説明 (`pages.shopping.pantry.description`) を出す
    - 各項目に食材名と用途 (`item.recipes.join(", ")`) を出し、`data-testid="shopping-pantry-item"` と `data-stock-status` を付ける
    - `StockStatus.StockCovered` の項目は薄い文字にし、`StockStatusBadge` を添える。それ以外の項目には何も添えない
    - 受け取った順のまま並べ、チェック欄は置かない
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/routes/shopping.tsx` を更新する (design.md D4)
    - `data.items` を `isPantryItem` で `tableItems` と `pantryItems` に分ける
    - 材料テーブルの行 (絞り込みの前)、`countByStockFilter`、`checkedItems` を `tableItems` から作る
    - `tableItems.length > 0` のときだけ、絞り込み、材料テーブル (または `filter.no_match`)、登録ボタンを出す
    - `data.items.length > 0` かつ `tableItems.length === 0` のときは、材料テーブルの位置に `pages.shopping.empty.no_quantified` を出す
    - `pantryItems.length > 0` のとき、材料の列の末尾 (登録ボタンまたは `empty.no_quantified` の下) に `ShoppingPantryList` を置く
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.3 `frontend/src/components/shopping/shopping-table.tsx` から、必要量 0 の行に `pages.shopping.amount_unspecified` を出す分岐を削除する (design.md D6)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.4 `frontend/src/components/shopping/purchase-form.ts` から、必要量 0 の行の量を空にする初期値の分岐とそのコメントを削除する (design.md D6)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.5 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する
    - `pages.shopping.pantry.title`：「家にあるか確かめる材料」
    - `pages.shopping.pantry.description`：「量を買う単位で書いていない材料です。切らしていないかだけ確かめてください」
    - `pages.shopping.empty.no_quantified`：「買う量の決まった材料はありません」
    - `pages.shopping.amount_unspecified` を削除する
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 3. E2E

- [x] 3.1 `frontend/e2e/shopping.spec.ts` の describe「量の指定が無い材料」を「量を持たない材料の欄」に改め、テストをそろえる
    - 「量る単位の材料は必要量を「—」で表示する」を削除する
    - 量を持たない材料は材料テーブルとは別の欄に並ぶ (新規)
    - 量を持たない材料は単位が違う在庫でも在庫ありになる (別の欄の項目の `data-stock-status` と「在庫あり」の表示を確かめる形に書き換える)
    - 量を持たない材料にはチェックを付けられない (新規)
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.2 `frontend/e2e/shopping.spec.ts` の describe「買い物リストの空状態」に Scenario を追加する
    - 量を持たない材料しか無い
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.3 `frontend/e2e/shopping-purchase.spec.ts` を更新する (design.md D5)
    - describe「冷蔵庫への一括登録」を「チェックした材料の冷蔵庫への一括登録」に改める
    - 「量の指定が無い材料は量が空になる」と「量る単位の材料は量と単位が空になる」を削除する
    - describe「一括登録の入力検証」の「量が空の行があると登録されない」を、「玉ねぎ 5 個」と「豚ロース 200 g」の 2 行で豚ロースの量を消す形に書き換える
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 3.4 `frontend/e2e/shopping-filter.spec.ts` の describe「在庫の状態での絞り込み」に Scenario を追加する
    - 量を持たない材料は件数に含めない
    - 完了判定: `task test:e2e` で上記が pass する

## 4. ドキュメントと検証

- [x] 4.1 `docs/product.md` §4 S07 を更新する
    - 「材料テーブル」の項の、量を持たない材料の説明を「材料テーブルには出さず、下の別の欄に食材名と用途だけを並べる」形に書き換える
    - 別の欄はチェックを持たず、在庫がある項目だけ薄く「在庫あり」を添えることを書く
    - 「購入済チェックと一括登録」の項から、量の指定が無い材料の量を空にする記述を削除する
    - 「空状態」の項に、量を持たない材料しか無い期間の表示を足す
    - 完了判定: 差分を目視で確認する
- [x] 4.2 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 4.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 4.4 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 4.5 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 4.6 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 4.7 `openspec validate add-shopping-pantry-section --strict` を実行する。完了判定: エラー 0 件
