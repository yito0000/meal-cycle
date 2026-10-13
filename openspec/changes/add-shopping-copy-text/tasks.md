## 1. テキストの組み立てとクリップボード

- [x] 1.1 `frontend/src/components/shopping/shopping-text.ts` を新設する (design.md D1 / D4)
    - `buildShoppingText(items, period, t)` を定義し、`{ text: string; count: number }` を返す
    - `items` を `isPantryItem` で材料テーブルの行と別の欄の項目に分ける
    - 材料テーブルの行は `StockStatus.StockCovered` 以外を、受け取った順のまま `pages.shopping.copy.item` (一部在庫ありは `pages.shopping.copy.item_partial`) で 1 行にする
    - 必要量は `formatQuantity(item.amount, item.unit)` で作る
    - 別の欄の項目は `StockStatus.StockNone` だけを `pages.shopping.copy.pantry_item` で 1 行にし、1 件以上あれば空行と `pages.shopping.copy.pantry_heading` を前に置く
    - 1 行目は `pages.shopping.copy.heading` に `formatDisplayDate` した開始日と終了日を入れる
    - `count` は材料テーブルの行と別の欄の項目の合計にする
    - 行は `\n` でつなぐ
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 1.2 `frontend/src/lib/clipboard.ts` を新設する (design.md D2)
    - `writeClipboardText(text: string): Promise<void>` を定義する
    - `navigator.clipboard?.writeText` を先に試し、無いか例外になったら `await import("@wailsio/runtime")` の `Clipboard.SetText(text)` を呼ぶ
    - runtime を動的 import にする理由 (E2E のビルドで runtime を読ませない) を WHY のコメントで 1 行書く
    - 完了判定: `cd frontend && npm run type-check` と `cd frontend && npm run build:dev` が通る

## 2. 画面

- [x] 2.1 `frontend/src/components/shopping/copy-shopping-text-button.tsx` を新設する (design.md D3)
    - props は `items: Item[]` と `period: ShoppingPeriod`
    - `buildShoppingText` の `count` が 0 のとき、または書き込み中は `disabled` にする
    - 押したら `writeClipboardText` を呼び、成功で `pages.shopping.copy.copied` (件数つき)、失敗で `pages.shopping.copy.failed` をトーストで出す
    - `data-testid="shopping-copy-text"` を付ける
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.2 `frontend/src/routes/shopping.tsx` を更新する (design.md D3)
    - 対象期間の表示 (`shopping-period`) とボタンを 1 行に並べ、ボタンを右端に置く
    - `data.dishes.length > 0` かつ `data.items.length > 0` のときだけボタンを出す
    - ボタンには `data.items` (絞り込み前の全行) と `period` を渡す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 2.3 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.shopping.copy.*` を追加する
    - `button`：「テキストでコピー」/ "Copy as text"
    - `heading`：「買い物リスト {{start}}〜{{end}}」/ "Shopping list {{start}}–{{end}}"
    - `item`：「- {{name}} {{quantity}}」/ "- {{name}} {{quantity}}"
    - `item_partial`：「- {{name}} {{quantity}} (一部在庫あり)」/ "- {{name}} {{quantity}} (partly in stock)"
    - `pantry_heading`：「家にあるか確かめる材料」/ "Check at home"
    - `pantry_item`：「- {{name}}」/ "- {{name}}"
    - `copied`：「買い物リストをコピーしました ({{count}} 件)」/ "Copied the shopping list ({{count}} items)"
    - `failed`：「コピーできませんでした」/ "Could not copy the shopping list"
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 3. E2E

- [x] 3.1 `frontend/e2e/shopping-copy.spec.ts` を新設し、describe「買い物リストのテキストコピー」を置く
    - テストの先頭で `page.context().grantPermissions(["clipboard-read", "clipboard-write"])` を呼び、コピー後に `navigator.clipboard.readText()` で中身を読む
    - 在庫なしと一部在庫ありの材料がテキストでコピーされる
    - 家にあるか確かめる材料は見出しの下に続く
    - 絞り込みにかかわらず同じ内容がコピーされる
    - 買う必要のある材料が無いとコピーできない
    - 献立が無い期間にはコピーのボタンを出さない
    - 「クリップボードに書き込めないとその旨を知らせる」は E2E に入れない (design.md D2)
    - 完了判定: `task test:e2e` で上記が pass する

## 4. ドキュメントと検証

- [x] 4.1 `docs/product.md` §4 S07 に「テキストでコピー」の項を足す
    - 対象期間の表示の横にボタンがあること
    - コピーするのは在庫なし・一部在庫ありの行と、在庫の無い「家にあるか確かめる材料」だけであること
    - 絞り込みとチェックはコピーの内容に影響しないこと
    - 完了判定: 差分を目視で確認する
- [x] 4.2 `docs/wireframes/S07_shopping_list.v2.drawio.svg` の対象期間の行の右端に「テキストでコピー」ボタンを描き足す
    - 可視 SVG と `content=` 属性内の編集 XML の両方を更新する (`docs/wireframes/README.md` の手順に従う)
    - 完了判定: ブラウザで SVG を開いてボタンが見えることと、drawio で開いて同じ位置にボタンがあることを目視で確認する
- [x] 4.3 `docs/reviews/2026-09-25-design-ux-review.md` の R1 に、`add-shopping-copy-text` でテキストのコピーに対応し、印刷は見送った旨を追記してチェックを付ける。完了判定: 差分を目視で確認する
- [x] 4.4 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 4.5 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 4.6 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 4.7 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 4.8 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 4.9 `openspec validate add-shopping-copy-text --strict` を実行する。完了判定: エラー 0 件
- [x] 4.10 `task build:win` で Windows ビルドを作り、Windows 実機で「テキストでコピー」を押してメモ帳とスマホ向けアプリ (LINE の PC 版など) に貼れることを確かめる。
  完了判定: 貼り付けたテキストが spec の形どおりであることを目視で確認する
