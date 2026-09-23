## 1. migration とドメイン層

- [x] 1.1 `internal/storage/sqlite/migrations/00006_stock_items.sql` を新設し、design.md D3 の `stock_items` テーブルと
      `idx_stock_items_name` を作成する (Down は `DROP TABLE stock_items`)。完了判定: `go test ./internal/storage/sqlite/...` が pass する
- [x] 1.2 `internal/domain/fridge/stock.go` を新設し、`StockItem` / `ExpiryStatus` (`none` / `fresh` / `soon` / `urgent`) /
      `NameSuggestion` / `Suggestions` を定義する (JSON タグは camelCase)。完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/domain/fridge/errors.go` を新設し、design.md D11 の表にある 8 つの sentinel を定義する。
      完了判定: `go build ./internal/...` が通る
- [x] 1.4 `internal/domain/fridge/expiry.go` を新設し、`Evaluate(expiresAt string, today time.Time, red, yellow int)
      (ExpiryStatus, *int)` を実装する (ローカル日付同士の日数差)。完了判定: 1.5 のテストが pass する
- [x] 1.5 `internal/domain/fridge/expiry_test.go` に table-driven test を追加する
    - 期限なし (空文字) は `none` と `nil`
    - 残り 8 日は `fresh` と 8
    - 残り 7 日 (yellow 境界) は `soon`
    - 残り 4 日は `soon`
    - 残り 3 日 (red 境界) は `urgent`
    - 残り 0 日 (当日) は `urgent` と 0
    - 残り -2 日 (期限切れ) は `urgent` と -2
    - `today` の時刻が 23:59 でも日数差は日付だけで決まる
    - 月末 / 年末をまたぐ日付でも日数差が正しい
    - (完了判定: `go test ./internal/domain/fridge/...` が pass する)

## 2. repository 層

- [x] 2.1 `internal/storage/sqlite/stock_repo.go` を新設し、`StockRepo` と `NewStockRepo(db)` を定義する。
      `expires_at` の `NULL ⇔ ""` 変換は scan / insert のヘルパ 1 箇所に置く。完了判定: `go build ./internal/...` が通る
- [x] 2.2 `stock_repo.go` に `List(ctx) ([]fridge.StockItem, error)` を実装する (並び順は `added_at DESC`)。
      完了判定: 2.8 の List ケースが pass する
- [x] 2.3 `stock_repo.go` に `Create(ctx, item)` を実装する。ID は `uuid.NewV7()`、`AddedAt` / `UpdatedAt` は現在時刻 (RFC3339 UTC)。
      完了判定: 2.8 の Create ケースが pass する
- [x] 2.4 `stock_repo.go` に `Update(ctx, id, item)` を実装する。`added_at` は変えず、`updated_at` を進める。
      該当行が無ければ `fridge.ErrStockNotFound`。完了判定: 2.8 の Update ケースが pass する
- [x] 2.5 `stock_repo.go` に `Delete(ctx, id)` を実装する。該当行が無ければ `fridge.ErrStockNotFound`。
      完了判定: 2.8 の Delete ケースが pass する
- [x] 2.6 `stock_repo.go` に `Restore(ctx, item)` を実装する。渡された ID / `AddedAt` で insert し、PK 重複は
      `fridge.ErrStockAlreadyExists` に変換する。完了判定: 2.8 の Restore ケースが pass する
- [x] 2.7 `stock_repo.go` に `ListSuggestions(ctx) (fridge.Suggestions, error)` を実装する。
      design.md D5 のウィンドウ関数で名前ごとの直近単位を決める。完了判定: 2.8 の ListSuggestions ケースが pass する
- [x] 2.8 `internal/storage/sqlite/stock_repo_test.go` を新設し、`t.TempDir()` の実 SQLite で以下を検証する
    - List: 0 件のとき空スライス (nil でない) を返す
    - Create: 期限つきで作成すると ID / AddedAt / UpdatedAt が埋まり、List で読み戻せる
    - Create: 期限なし (空文字) で作成すると DB は NULL、読み戻すと空文字
    - Create: 同じ名前を 2 回作成すると 2 行になる
    - Update: 量 / 単位 / 期限 / 名前が更新され、AddedAt は変わらない
    - Update: 期限を空文字にすると期限なしになる
    - Update: 存在しない ID は ErrStockNotFound
    - Delete: 削除後は List に出ない
    - Delete: 存在しない ID は ErrStockNotFound
    - Restore: 削除した行を同じ ID / AddedAt で復元できる
    - Restore: 既存 ID への Restore は ErrStockAlreadyExists
    - ListSuggestions: 在庫名とレシピ材料名が重複なく 1 件ずつ返る
    - ListSuggestions: 同名の在庫が複数あると added_at が最新の行の単位になる
    - ListSuggestions: 在庫に無い名前は updated_at が最新のレシピの材料行の単位になる
    - ListSuggestions: 在庫とレシピの両方にある名前は在庫側の単位が優先される
    - ListSuggestions: Units は空文字を除いた distinct になる
    - (完了判定: `go test ./internal/storage/sqlite/...` が pass する)

## 3. service 層

- [x] 3.1 `internal/service/fridge.go` を新設し、`FridgeService` (フィールド `repo *sqlite.StockRepo` / `now func() time.Time` /
      `cfg config.Config`) と `NewFridgeService(db *sql.DB)` を定義する。完了判定: `go build ./internal/...` が通る
- [x] 3.2 `fridge.go` に検証上限の定数 (食材名 100 字 / 量 99999 / 単位 20 字) と `validateStockInput` を実装する。
      trim 後の値を返し、違反は design.md D11 の sentinel で返す。完了判定: 3.6 のバリデーションケースが pass する
- [x] 3.3 `fridge.go` に `List(ctx)` を実装する。repo の結果の各行に `fridge.Evaluate` で `ExpiryStatus` / `DaysLeft` を載せる。
      閾値は `s.cfg.ExpiryAlertRedDays` / `s.cfg.ExpiryAlertYellowDays` から読む。完了判定: 3.6 の List ケースが pass する
- [x] 3.4 `fridge.go` に `Add` / `Update` / `Delete` を実装する。Add / Update は検証後に repo へ委譲し、戻り値にも期限状態を載せる。
      完了判定: 3.6 の該当ケースが pass する
- [x] 3.5 `fridge.go` に `Restore(ctx, item)` と `ListSuggestions(ctx)` を実装する。Restore は ID 空 / AddedAt が RFC3339 でない場合に
      `ErrStockInvalidRestore` を返す。完了判定: 3.6 の該当ケースが pass する
- [x] 3.6 `internal/service/fridge_test.go` を新設し、BDD 風 (`t.Run("should ... when ...")`) で以下を検証する (now は固定日付)
    - List: should return items with urgent status when expiry is within red days
    - List: should return items with none status and nil daysLeft when expiry is empty
    - List: should use config thresholds when evaluating status
    - Add: should trim name and unit when input has surrounding spaces
    - Add: should return ErrStockNameEmpty when name is blank
    - Add: should return ErrStockNameTooLong when name exceeds 100 runes
    - Add: should return ErrStockAmountOutOfRange when amount is zero, negative, or above 99999
    - Add: should return ErrStockUnitTooLong when unit exceeds 20 runes
    - Add: should return ErrStockInvalidExpiry when expiry is not a valid date (例: 2026-02-30)
    - Add: should accept past expiry date
    - Update: should keep addedAt when item is updated
    - Update: should return ErrStockNotFound when id does not exist
    - Delete: should remove item when id exists
    - Restore: should recreate item with same id and addedAt when restoring deleted item
    - Restore: should return ErrStockInvalidRestore when id is empty
    - ListSuggestions: should include recipe ingredient names when stock is empty
    - (完了判定: `go test ./internal/service/...` が pass する)
- [x] 3.7 `main.go` を更新し、`application.Options.Services` に `application.NewService(service.NewFridgeService(db))` を追加する。
      完了判定: `go build .` が通る

## 4. bindings と E2E サーバ

- [x] 4.1 `wails3 generate bindings` を実行する。完了判定: `frontend/bindings/.../internal/service/` に `fridgeservice.js` と
      `internal/domain/fridge/models.js` が生成される
- [x] 4.2 `cmd/e2e-server/main.go` の `newRegistry` に `"FridgeService": service.NewFridgeService(db)` を追加し、
      「3 サービス」のコメントを 4 に直す。完了判定: `go test ./cmd/e2e-server/...` が pass する
- [x] 4.3 `cmd/e2e-server/dispatch.go` の `serviceRegistry` コメントのサービス数を 4 に直す。完了判定: 目視
- [x] 4.4 `frontend/e2e/bindings-shim/fridgeservice.ts` を新設し、`List` / `Add` / `Update` / `Delete` / `Restore` /
      `ListSuggestions` を `call("FridgeService", ...)` で定義する。完了判定: 4.5 後に `npm run type-check` が通る
- [x] 4.5 `frontend/e2e/bindings-shim/index.ts` に `FridgeService` の import / export / `_typeCheck` 追加を行う。
      完了判定: `cd frontend && npm run type-check` が通る

## 5. フロント基盤 (トースト / i18n)

- [x] 5.1 `frontend/` で `npm install sonner` を実行する。完了判定: `frontend/package.json` の dependencies に `sonner` が入る
- [x] 5.2 `frontend/src/components/ui/sonner.tsx` を新設し、shadcn の Sonner をコピーして `next-themes` の import と
      `useTheme` を削る (削った理由を 1 行コメント)。完了判定: `npm run type-check` が通る
- [x] 5.3 `frontend/src/components/layout/root-layout.tsx` を更新し、`<Toaster />` を 1 つ配置する。完了判定: `npm run build:dev` が通る
- [x] 5.4 `frontend/src/locales/ja.json` と `en.json` の `pages.fridge` を更新し、`placeholder` を削除して以下の key を追加する。
      完了判定: 両ファイルで key 集合が一致する (目視 + type-check)
    - `search.placeholder` / `sort.label` / `sort.expiry_asc` / `sort.name_asc` / `sort.added_desc`
    - `column.name` / `column.amount` / `column.expires_at` / `column.days_left` / `column.actions`
    - `add.label` / `add.name_placeholder` / `add.amount_placeholder` / `add.unit_placeholder` / `add.expires_placeholder` / `add.submit`
    - `row.edit` / `row.delete` / `row.save` / `row.cancel`
    - `days_left.remaining` (「あと {{count}} 日」) / `days_left.today` / `days_left.overdue` (「{{count}} 日超過」) / `days_left.none`
    - `expiry_status.urgent` / `expiry_status.soon` / `expiry_status.fresh` (色ドットの aria-label)
    - `empty.no_items` / `empty.no_match`
    - `toast.deleted` / `toast.undo`
    - `errors.name_empty` / `errors.name_too_long` / `errors.amount_out_of_range` / `errors.unit_too_long` /
      `errors.invalid_expiry` / `errors.not_found` / `errors.unknown`

## 6. フロント: データ取得と純関数

- [x] 6.1 `frontend/src/components/fridge/use-fridge.ts` を新設し、`useStockList` / `useStockSuggestions` と
      `useAddStock` / `useUpdateStock` / `useDeleteStock` / `useRestoreStock` を定義する (design.md D12 の query key と invalidate)。
      完了判定: `npm run type-check` が通る
- [x] 6.2 `frontend/src/components/fridge/fridge-sort.ts` を新設し、`FridgeSortKey` / `DEFAULT_FRIDGE_SORT` / `compareStock` を実装する
      (期限なしは末尾、同順位は食材名)。完了判定: `npm run type-check` が通る
- [x] 6.3 `frontend/src/components/fridge/stock-form.ts` を新設し、`StockFormValues` / `emptyStockForm` / `fromStockItem` /
      `validateStockForm` / `toStockInput` / `isImeComposing` を実装する。完了判定: `npm run type-check` が通る
- [x] 6.4 `frontend/src/components/fridge/error-messages.ts` を新設し、`fridge: ` で始まる Go エラーを `pages.fridge.errors.*` に対応付ける
      `stockErrorOf` (i18n key と対象入力欄を返す) を実装する。完了判定: `npm run type-check` が通る

## 7. フロント: コンポーネント

- [x] 7.1 `frontend/src/components/fridge/expiry-dot.tsx` を新設し、`ExpiryStatus` から 🔴 / 🟡 / 🟢 の色ドット (aria-label つき) を
      描画する。`none` のときは何も描画しない。完了判定: `npm run type-check` が通る
- [x] 7.2 `frontend/src/components/fridge/stock-name-input.tsx` を新設し、WAI-ARIA combobox パターンの候補補完つき入力を実装する。
      完了判定: `npm run type-check` が通り、E2E の候補選択シナリオが pass する
    - 部分一致 (英字は大文字小文字を無視) で最大 8 件を表示する
    - 上下キーで移動、Enter で確定、Esc で閉じる。`isImeComposing` のときはこれらを無視する
    - 確定時に `onSelect(suggestion)` を呼ぶ (単位の自動入力は呼び出し側で行う)
- [x] 7.3 `frontend/src/components/fridge/fridge-add-row.tsx` を新設し、食材名 / 量 / 単位 / 賞味期限 / 追加ボタンの追加行を実装する。
      完了判定: E2E の追加系シナリオが pass する
    - 候補確定時、単位欄が空なら候補の単位を入れる
    - 食材名以外の欄の Enter で送信する (`isImeComposing` のときは送信しない)
    - 成功時に入力をクリアし、食材名欄にフォーカスを戻す
    - 検証エラーと Go エラーを該当欄の近くに表示する
- [x] 7.4 `frontend/src/components/fridge/fridge-item-row.tsx` を新設し、食材名 + 色ドット / 量 + 単位 / 賞味期限 / 残り日 /
      編集・削除ボタンの表示行を実装する (`disabled` prop で操作を無効化)。完了判定: `npm run type-check` が通る
- [x] 7.5 `frontend/src/components/fridge/fridge-edit-row.tsx` を新設し、行内編集行 (入力 4 つ + 保存 / キャンセル) を実装する。
      Enter で保存、Esc でキャンセル。完了判定: E2E の行内編集シナリオが pass する
- [x] 7.6 `frontend/src/components/fridge/fridge-table.tsx` を新設し、`editingId` を持って表示行と編集行を切り替える。
      編集中は他行に `disabled` を渡す。完了判定: E2E の「編集中は他の行を操作できない」が pass する
- [x] 7.7 `frontend/src/routes/fridge.tsx` の削除操作で `useDeleteStock` を呼び (最後の 1 行の削除でテーブルが外れても通知を出すためページ側に置く)、成功時に sonner の `toast` に「元に戻す」action を付けて 5 秒表示する。
      action で `useRestoreStock` に削除前の行を渡す。完了判定: E2E の削除シナリオが pass する
- [x] 7.8 `frontend/src/components/fridge/fridge-sort-select.tsx` を新設し、並び替え 3 択のセレクトを実装する
      (`recipe-sort-select.tsx` と同じ構成)。完了判定: `npm run type-check` が通る
- [x] 7.9 `frontend/src/components/fridge/fridge-search-input.tsx` を新設し、検索欄を実装する
      (`recipe-search-input.tsx` と同じ構成)。完了判定: `npm run type-check` が通る
- [x] 7.10 `frontend/src/routes/fridge.tsx` を更新し、見出し / 検索 + 並び替え / 追加行 / テーブル / 空状態 (0 件と検索 0 件) を組む。
      完了判定: `task dev` で S05 がワイヤー `S05_fridge.drawio.svg` と同じ配置で表示される (目視)

## 8. E2E

- [x] 8.1 `frontend/e2e/helpers.ts` に `openFridge(page)` (サイドバーから遷移) と `addStockItem(page, {name, amount, unit, expiresInDays?})`
      (HTTP 経由で直接作成) を追加する。完了判定: `npm run type-check` が通る
- [x] 8.2 `frontend/e2e/fridge.spec.ts` を新設し、9.1 で更新した `docs/scenarios/fridge.feature` の `シナリオ:` 名で以下を実装する。
      完了判定: `task test:e2e` が pass する
    - デフォルトで賞味期限の近い順に並ぶ
    - 食材名でソートする
    - 追加日新しい順でソートする
    - 食材名で検索する
    - 当日期限と期限切れは赤で表示される
    - 賞味期限つきで在庫を追加する
    - 賞味期限を省略して追加する
    - 候補から食材名を選ぶと単位が埋まる
    - 候補にない食材名で追加する
    - 食材名が空だと追加できない
    - 行内編集で在庫量を修正する
    - 行内編集をキャンセルする
    - 編集中は他の行を操作できない
    - 削除は即実行され、トーストの「元に戻す」で復元できる

## 9. ドキュメント

- [x] 9.1 `docs/scenarios/fridge.feature` を更新する。完了判定: 8.2 のテスト名と `シナリオ:` 行が一字一句一致する
    - ヘッダの対応 proposal を `add-fridge-inventory` に置き換える
    - 「既存 Ingredient を選んで在庫を追加する」を「候補から食材名を選ぶと単位が埋まる」に書き換える
    - 「新規 Ingredient を作りながら追加する」を「候補にない食材名で追加する」に書き換える
    - 「食材名で検索する」「当日期限と期限切れは赤で表示される」「食材名が空だと追加できない」「編集中は他の行を操作できない」を追加する
    - 「賞味期限つきで在庫を追加する」の名前に合わせて既存の追加シナリオ名を揃える
- [x] 9.2 `docs/product.md` §4 S05 に検索欄と期限切れ / 当日の扱い、S06 の食材欄を「自由入力 + 既存在庫名 / レシピ材料名の候補補完」
      「単位は候補の直近単位で埋まる」に改める。完了判定: `grep -n "defaultUnit\|新規作成" docs/product.md` が S06 節で 0 件
- [x] 9.3 `docs/design.md` §4 StockItem の `ingredientId` を `name` に改め、§8 の順序案の fridge 3 行を `add-fridge-inventory` 1 行に置き換える。
      完了判定: 目視

## 10. 検証

- [x] 10.1 `go test ./internal/...` を実行する。完了判定: 全 pass
- [x] 10.2 `go test ./cmd/e2e-server/...` を実行する。完了判定: 全 pass
- [x] 10.3 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0
- [x] 10.4 `cd frontend && npm run build:dev` を実行する。完了判定: ビルド成功
- [x] 10.5 `task test:e2e` を実行する。完了判定: fridge を含む全 spec が pass (skip は既存分のみ)
- [x] 10.6 `task lint` を実行する。完了判定: Go / Frontend ともに指摘 0
- [x] 10.7 `openspec validate add-fridge-inventory --strict` を実行する。完了判定: valid
- [x] 10.8 `task build:win` で Windows ビルドを作り、Windows 側で日本語入力を確認する。完了判定: 以下を目視で確認
    - 食材名に日本語を IME で入力し、変換確定の Enter で追加されない
    - 食材名の候補を IME 入力中に上下キーで動かしても候補選択が暴れない
    - 単位に「パック」を入力して追加でき、再起動後も残る
    - `<input type="date">` のピッカーで賞味期限を入力できる

## 11. 期限切れ表示の分離 (design.md D14)

- [x] 11.1 `internal/domain/fridge/stock.go` を更新し、`ExpiryExpired ExpiryStatus = "expired"` を追加する。
      `internal/domain/fridge/expiry.go` の `Evaluate` を更新し、`daysLeft < 0` を `ExpiryExpired` にする。完了判定: 11.2 が pass する
- [x] 11.2 `internal/domain/fridge/expiry_test.go` を更新する。完了判定: `go test ./internal/domain/fridge/...` が pass する
    - 残り -2 日 (期限切れ) は `expired` と -2
    - 残り -1 日は `expired`
    - 残り 0 日 (当日) は `urgent` のまま
- [x] 11.3 `internal/service/fridge_test.go` に以下を追加する。完了判定: `go test ./internal/service/...` が pass する
    - List: should return expired status when expiry is in the past
    - (既存の「should accept past expiry date」の期待値を `expired` に更新する)
- [x] 11.4 `wails3 generate bindings -clean=true -ts -i` を実行する。完了判定: 生成された `ExpiryStatus` に `ExpiryExpired` が含まれる
- [x] 11.5 `frontend/src/locales/ja.json` / `en.json` の `pages.fridge.expiry_status` に `expired` (「期限切れ」/ "Expired") を追加する。
      完了判定: 両ファイルで key 集合が一致する
- [x] 11.6 `frontend/src/components/fridge/expiry-dot.tsx` を更新し、`expired` のときは色ドットの代わりに「期限切れ」の文字バッジを描画する
      (`data-expiry-status="expired"` を付ける)。完了判定: `npm run type-check` が通る
- [x] 11.7 `frontend/src/components/fridge/fridge-item-row.tsx` を更新し、`expired` の行に薄い赤の背景 (`bg-red-50`) を付け、
      残り日の文字色も赤にする。完了判定: `npm run type-check` が通る
- [x] 11.8 `docs/scenarios/fridge.feature` のシナリオ「当日期限と期限切れは赤で表示される」を次の 2 つに分け、`frontend/e2e/fridge.spec.ts` の
      テスト名と期待値を揃える。完了判定: `task test:e2e` が pass し、シナリオ名とテスト名が一字一句一致する
    - 当日期限は赤で表示される
    - 期限切れは色ドットではなく期限切れバッジで表示される (バッジの文言と行の目印を検証する)
- [x] 11.9 `docs/product.md` §4 S05 の期限表示の記述を「🔴 は当日を含む 3 日以内、期限切れはバッジ + 行背景」に改める。完了判定: 目視
- [x] 11.10 検証一式を再実行する。完了判定: 以下がすべて通る
    - `go test ./internal/... ./cmd/e2e-server/...`
    - `cd frontend && npm run type-check` と `npm run build:dev`
    - `task test:e2e`
    - `task lint`
    - `openspec validate add-fridge-inventory --strict`
