## 0. 前提

- [x] 0.1 `add-shopping-purchase` が `openspec/changes/archive/` に移っていることを確認する。
      完了判定: `openspec/specs/shopping/spec.md` に `### Requirement: 購入済チェック` がある

## 1. 絞り込みの判定

- [x] 1.1 `frontend/src/components/shopping/stock-filter.ts` を新設する (design.md D1)
    - `StockFilter` 型 (`"all" | "needed" | "covered"`) と、並び順の配列 `STOCK_FILTERS`、既定値 `DEFAULT_STOCK_FILTER = "all"` を定義する
    - `matchesStockFilter(item, filter)`：`all` は常に true、`needed` は `StockNone` / `StockPartial`、`covered` は `StockCovered` で true を返す
    - `countByStockFilter(items)`：3 種類それぞれに当てはまる行の件数を `Record<StockFilter, number>` で返す
    - 完了判定: `cd frontend && npm run type-check` が通る

## 2. 画面

- [x] 2.1 `frontend/src/components/shopping/shopping-stock-filter.tsx` を新設する (design.md D4)
    - props は `value: StockFilter` / `counts: Record<StockFilter, number>` / `onChange: (next: StockFilter) => void`
    - `shopping-period-picker.tsx` の日数プリセットと同じ見た目で、`aria-pressed` を持つボタンを `fieldset` に並べる
    - `legend` (sr-only) は `pages.shopping.filter.label`、各ボタンの文言は `pages.shopping.filter.<種類>` に件数を埋め込む
    - 各ボタンに `data-testid={`shopping-filter-${filter}`}` を付ける
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/components/shopping/shopping-table.tsx` の `toggleAll` を、表示中の行だけを付け外しする形に書き換える (design.md D3)
    - 表示中の行がすべてチェック済みなら、表示中の行のキーだけを集合から除く
    - それ以外なら、表示中の行のキーを集合に足す
    - 集合にある他のキー (隠れた行) は変えない
    - 完了判定: `cd frontend && npm run type-check` が通り、`shopping-purchase.spec.ts` の既存 E2E が 4.2 で pass する
- [x] 2.3 `frontend/src/routes/shopping.tsx` を更新する (design.md D2 / D5)
    - `useState<StockFilter>(DEFAULT_STOCK_FILTER)` で絞り込みを持つ。期間の切替ではリセットしない
    - `data.items` を `matchesStockFilter` で絞った配列を `ShoppingTable` に渡す
    - 件数は `countByStockFilter(data.items)` で数えて `ShoppingStockFilter` に渡す
    - 材料テーブルがあるとき (`data.items.length > 0`) だけ、テーブルの直上に `ShoppingStockFilter` を置く
    - 絞った結果が 0 行なら、テーブルの代わりに `pages.shopping.filter.no_match` を表示する。切替と登録ボタンは残す
    - 登録ボタンの件数とダイアログの行は、これまでどおり絞る前の `data.items` とチェックの集合から数える
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.4 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.shopping.filter.*` を追加する
    - `label`：「在庫の状態で絞り込む」
    - `all`：「すべて ({{count}})」
    - `needed`：「在庫なし・一部 ({{count}})」
    - `covered`：「在庫あり ({{count}})」
    - `no_match`：「該当する食材がありません」
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 3. E2E

- [x] 3.1 `frontend/e2e/shopping-filter.spec.ts` を新設し、spec の Requirement 名を describe、Scenario 名をテスト名にして実装する
    - describe「在庫の状態での絞り込み」
      - 在庫なし・一部で絞り込む
      - 在庫ありで絞り込む
      - 選択肢に件数が表示される
      - 開き直すと絞り込みはすべてに戻る
      - 該当する行が無いとその旨を表示する
      - 絞り込みを切り替えてもチェックは残る
      - 全選択は表示中の行だけに効く
      - 隠れた行のチェックも登録の対象になる
      - 登録して在庫が足りた行は在庫なし・一部の表示から消える
    - 絞り込みは `data-testid="shopping-filter-<種類>"` で押し、チェックは `helpers.ts` の `checkShoppingItem` を使う
    - 完了判定: `task test:e2e` で上記がすべて pass する

## 4. ドキュメントと検証

- [x] 4.1 `docs/product.md` §4 S07 の「材料テーブル」の項に、絞り込み (すべて / 在庫なし・一部 / 在庫あり、件数つき、保存しない、
      チェックは隠れても残り、全選択は表示中の行だけに効く) を追記する。完了判定: 差分を目視で確認する
- [x] 4.2 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 4.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 4.4 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 4.5 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 4.6 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 4.7 `openspec validate add-shopping-stock-filter --strict` を実行する。完了判定: エラー 0 件
