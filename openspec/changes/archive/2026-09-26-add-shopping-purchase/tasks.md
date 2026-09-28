## 1. Repository

- [x] 1.1 `internal/storage/sqlite/stock_repo.go` に `CreateMany(ctx, inputs []fridge.StockItem) ([]fridge.StockItem, error)` を追加する (design.md D1)
    - 1 つのトランザクションの中で、行ごとに `uuid.NewV7()` で ID を発行し、既存の `insertStockItem(ctx, tx, item)` を呼ぶ
    - `AddedAt` と `UpdatedAt` は全行で同じ `time.Now().UTC().Format(time.RFC3339)` にする
    - 途中の INSERT が失敗したら rollback して、エラーを返す
    - `inputs` が空なら DB に触れず空スライスを返す
    - 完了判定: 1.2 のテストが通る
- [x] 1.2 `internal/storage/sqlite/stock_repo_test.go` に `TestStockRepo_CreateMany` を追加する
    - 3 行を渡すと 3 行が入り、`List` で同じ内容 (名前 / 量 / 単位 / 期限) が返る
    - 返り値の各行に異なる ID が振られ、`AddedAt` が全行で等しい
    - 空の賞味期限は NULL として保存され、読み出すと空文字になる
    - 空スライスを渡すと空スライスを返し、行は増えない
    - 2 行目の INSERT が失敗するとエラーを返し、1 行目も残らない
      (テスト内で `name = 'fail'` の行に `RAISE(ABORT)` する BEFORE INSERT トリガを作って失敗させる)
    - 完了判定: `go test ./internal/storage/sqlite/ -run TestStockRepo_CreateMany` が通る

## 2. Service と bindings

- [x] 2.1 `internal/service/fridge.go` に `AddMany(ctx, inputs []fridge.StockItem) ([]fridge.StockItem, error)` を追加する (design.md D1 / D2)
    - 全行を `validateStockInput` で検証し、最初に不正だった行のエラーを `fmt.Errorf("row %d: %w", i, err)` で包んで返す
    - 1 行でも不正なら `CreateMany` を呼ばない
    - 挿入後の各行に `withExpiry` で期限の状態を載せて返す
    - メソッドのコメントに、S07 の一括登録から呼ばれ、全行を 1 トランザクションで入れる理由を 1 行で残す
    - 完了判定: `go build .` が通る
- [x] 2.2 `internal/service/fridge_test.go` に `TestFridgeService_AddMany` を追加する
    - should add all items with trimmed name and unit when every input is valid
    - should return added items with expiry status evaluated when expiry is set
    - should not add any item when one of the inputs has an empty name
    - should wrap ErrStockAmountOutOfRange with the row index when the second input has zero amount
    - should not merge into existing stock when the same name and unit already exist
    - should return empty slice without error when inputs are empty
    - 完了判定: `go test ./internal/service/ -run TestFridgeService_AddMany` が通る
- [x] 2.3 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/bindings/` に `FridgeService.AddMany` を生成する。
      完了判定: `frontend/bindings/github.com/yito0000/meal-cycle/internal/service/fridgeservice.ts` に `AddMany` がある
- [x] 2.4 `frontend/e2e/bindings-shim/fridgeservice.ts` に `AddMany(inputs: fridge$0.StockItem[])` を追加し、
      戻り値の型を `fridge$0.StockItem[]` にする。完了判定: `cd frontend && npm run type-check` が通る

## 3. フロントの状態とフック

- [x] 3.1 `frontend/src/components/fridge/use-fridge.ts` を更新し、`useInvalidateFridge` を export する。
      完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/components/shopping/purchase-form.ts` を新設する (design.md D5)
    - `itemKey(item)`：`name + "\u0000" + unit` を返す。`shopping-table.tsx` の行の key もこれに置き換える
    - `toPurchaseRows(items)`：`Item[]` から `StockFormValues[]` を作る。量は `formatAmount`、必要量 0 は空文字、賞味期限は空文字
    - `validatePurchaseRows(rows)`：各行に `validateStockForm` をかけ、行ごとの `StockFormErrors[]` と、エラーが 1 件でもあるかを返す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.3 `frontend/src/components/shopping/use-purchase.ts` を新設する (design.md D6)
    - `useRegisterPurchase()`：`FridgeService.AddMany(rows.map(toStockInput))` を呼ぶ mutation を返す
    - 成功時に `useInvalidateFridge` と `queryClient.invalidateQueries({ queryKey: ["shopping"] })` を実行する
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.4 `frontend/src/components/shopping/use-shopping-list.ts` のコメントを、他画面の mutation からは invalidate しないこと、
      S07 自身の一括登録では invalidate することに合わせて書き換える (design.md D6)。完了判定: 差分を目視で確認する

## 4. 画面

- [x] 4.1 `frontend/src/components/shopping/shopping-table.tsx` を更新し、先頭にチェックの列を追加する (design.md D4)
    - props に `checked: Set<string>` と `onCheckedChange: (next: Set<string>) => void` を足す
    - 各行にネイティブの `<input type="checkbox">` を置き、`aria-label` は `pages.shopping.purchase.check_row` (食材名を埋め込む)
    - 見出しのチェックボックスは、全行チェックで checked、一部だけで indeterminate (ref で設定)、0 件で unchecked にする
    - 見出しのチェックボックスを押すと、全行チェック済みなら全解除、それ以外は全選択にする。`aria-label` は `pages.shopping.purchase.check_all`
    - 列幅を、チェック列を足しても 4 列の比率が崩れないよう調整する
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/src/components/shopping/purchase-dialog.tsx` を新設する (design.md D5 / D7)
    - props は `open` / `items` (チェックした材料、材料テーブルの順) / `onOpenChange` / `onRegistered(count)`
    - 開いた時点で `toPurchaseRows(items)` から行の state を作り、開き直すたびに作り直す
    - 各行に食材名 / 量 / 単位 / 賞味期限 (`<Input type="date">`) の入力欄を並べ、エラーは fridge の `FieldError` で出す
    - 「登録」で `validatePurchaseRows` を通し、エラーがあれば該当欄に表示して送信しない
    - 検証を通れば `useRegisterPurchase` を呼び、成功で `onRegistered(rows.length)` を呼ぶ
    - 失敗時はダイアログを開いたまま、`stockErrorOf(err).key` の文言をダイアログ上部に出す
    - 本体に最大の高さを設けて縦スクロールさせ、フッタの「登録」「キャンセル」は常に見えるようにする
    - 送信中は「登録」を押せないようにする
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/routes/shopping.tsx` を更新する (design.md D3 / D6)
    - `useState<Set<string>>` でチェック状態を持つ
    - `periodStart` と `periodDays` の組が前回と変わったら、描画中にチェック状態を空にする
    - チェック状態を `data.items` と突き合わせ、表示中の行に対応するものだけを件数とダイアログの行に使う
    - 材料テーブルの下に登録ボタンを置き、文言は `pages.shopping.purchase.register` (件数を埋め込む)、0 件なら disabled にする
    - 材料テーブルを出さない空状態 (献立なし / 材料なし) ではボタンを出さない
    - ボタンで `PurchaseDialog` を開く
    - `onRegistered` でダイアログを閉じ、チェック状態を空にし、`toast(t("pages.shopping.purchase.registered", { count }))` を出す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.4 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.shopping.purchase.*` を追加する
    - `check_all`：「すべての食材を選択」
    - `check_row`：「{{name}} を購入済みにする」
    - `register`：「チェックした食材を冷蔵庫に登録 ({{count}} 件)」
    - `dialog_title`：「冷蔵庫に登録する食材」
    - `dialog_description`：「量と賞味期限を確かめてから登録してください」
    - `submit`：「登録」
    - `cancel`：「キャンセル」
    - `registered`：「{{count}} 件を冷蔵庫に登録しました」
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 5. E2E

- [x] 5.1 `frontend/e2e/shopping.spec.ts` の `ingredientNames` と `itemRow` を、先頭にチェック列が増えても食材名の列を指すよう直す
      (`td:first-child` を `td:nth-child(2)` にする)。完了判定: 5.4 で既存の shopping の E2E が pass する
- [x] 5.2 `frontend/e2e/helpers.ts` に `checkShoppingItem(page, name)` を追加する。
      `pages.shopping.purchase.check_row` の `aria-label` でチェックボックスを取って check する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.3 `frontend/e2e/shopping-purchase.spec.ts` を新設し、spec の Requirement 名を describe、Scenario 名をテスト名にして実装する
    - describe「購入済チェック」
      - チェックを付けても在庫は増えない
      - 在庫ありの行にもチェックを付けられる
      - 見出しの操作で全行のチェックを付け外しする
      - 期間を切り替えるとチェックが外れる
    - describe「冷蔵庫への一括登録」
      - チェックが無いと登録ボタンを押せない
      - ダイアログにチェックした材料が初期値つきで並ぶ
      - 量の指定が無い材料は量が空になる
      - 量と期限を直して登録すると在庫に加わる
      - 登録すると在庫の状態が変わりチェックが外れる
      - キャンセルすると在庫もチェックも変わらない
    - describe「一括登録の入力検証」
      - 量が空の行があると登録されない
      - 食材名を空にした行があると登録されない
    - 賞味期限の日付は `dateAfter` で今日からの相対に組み立て、S05 の表示 (yyyy/mm/dd) と突き合わせる
    - 「既存の在庫には合算しない」は 2.2 のサービス層のテストで確かめ、E2E に含めない
    - 完了判定: `task test:e2e` で上記がすべて pass する

## 6. ドキュメント

- [x] 6.1 `docs/product.md` §4 S07 の「購入済チェックと一括登録」を書き換える。完了判定: 差分を目視で確認する
    - 見出しから「(add-shopping-purchase で追加予定)」を外す
    - 全行にチェックを付けられること、見出しで全選択できること、チェックは保存せず期間の切替と開き直しで外れることを書く
    - 登録ボタンで確認のダイアログが開き、食材名 / 量 / 単位 / 賞味期限を直してから一括で登録することを書く
    - 量の初期値は必要量、賞味期限は空、既存在庫に合算しないことを書く
    - 「賞味期限は空、後で S05 で編集」の記述を削除する
- [x] 6.2 `docs/product.md` §7 決定事項の「買い物リストの購入済チェック」を、確認のダイアログで量と期限を直してから一括登録する流れに書き換え、
      帰宅後に登録する使い方を前提にした理由を 1 文添える。完了判定: 差分を目視で確認する
- [x] 6.3 `docs/design.md` §9 の「購入済チェックで在庫に加える」を「購入済チェックと確認のダイアログで在庫に加える」に直す。
      完了判定: 差分を目視で確認する
- [x] 6.4 `docs/wireframes/S07_shopping_list.v2.drawio.svg` を新設し、現在の S07 を描く。
      `docs/wireframes/README.md` の「初期案の凍結」に従い、画面構造が変わったため既存の `S07_shopping_list.drawio.svg` は残して v2 を足す
    - 期間の切替 (開始日と日数プリセット) と対象期間の表示
    - チェック列 + 「食材 / 必要量 / 在庫 / 用途」の列、在庫の状態の表示、在庫ありの行を薄く末尾に置く
    - 右側の「この期間の献立」欄
    - 材料テーブルの下の「チェックした食材を冷蔵庫に登録 (N 件)」ボタン
    - 完了判定: ブラウザで SVG を開いて目視で確認し、`content=` 内の編集 XML が XML として読めることを確認する
- [x] 6.5 `docs/wireframes/S07a_purchase_dialog.drawio.svg` を新設し、一括登録のダイアログ (行ごとの 4 つの入力欄、エラー表示、登録とキャンセル) を描く。
      `docs/wireframes/README.md` の一覧に追加する。完了判定: ブラウザで SVG を開いて目視で確認する
- [x] 6.6 `docs/product.md` §4 S07 のワイヤーへのリンクに、S07 v2 と S07a へのリンクを追加する。完了判定: 差分を目視で確認する
- [x] 6.7 `docs/reviews/2026-09-25-design-ux-review.md` の R1 に、購入済チェックの流れを `add-shopping-purchase` で
      確認のダイアログ方式に見直した旨を引用ブロックで追記する。持ち出しの手段は未対応のため、チェックは付けない。完了判定: 差分を目視で確認する

## 7. 検証

- [x] 7.1 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 7.2 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 7.3 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 7.4 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 7.5 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 7.6 `openspec validate add-shopping-purchase --strict` を実行する。完了判定: エラー 0 件
- [x] 7.7 `task build:win` で Windows 版を作り、日本語の食材名を含む行を一括登録して S05 に表示されることを確かめる。
      完了判定: Windows 実機で目視確認する
