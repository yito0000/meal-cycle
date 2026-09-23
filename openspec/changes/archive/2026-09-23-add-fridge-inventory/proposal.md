## Why

S05 冷蔵庫在庫は、MVP 4 機能のうち唯一まだ手が付いていない画面で、現状はプレースホルダしか表示されない。
買い物のあとに在庫を足し、料理のあとに量を減らすという「在庫更新」の起点 (product.md §1) が成立しないため、
後続の `add-shopping-list` (在庫との突き合わせ) や MVP 後の「使い切り」提案も前提を欠いたままになっている。

本 proposal では S05 一覧と S06 インライン追加 / 行内編集を 1 回で通し、在庫を「見る」「足す」「直す」「消す」が
end-to-end で動く状態にする。design.md §8 の順序案では `add-fridge-listing` / `add-fridge-add-item` /
`add-fridge-expiry-sort` の 3 本に分けていたが、いずれも同じテーブルと同じ画面に触り、単独では在庫更新の体験が
完結しないため 1 本にまとめる。

食材名は **食材マスタを持たない自由テキスト** で扱う。レシピの材料も自由テキスト (add-recipe-edit の決定) であり、
在庫側だけマスタ化しても名寄せは片側にしか効かない。入力の摩擦は、既存の在庫名とレシピ材料名からの候補補完で下げる。

## What Changes

- `fridge` capability を新設し、冷蔵庫在庫 (StockItem) を SQLite に永続化する。
  1 行 = 食材名 / 量 / 単位 / 賞味期限 (任意) / 追加日時。
- Wails サービス `FridgeService` を追加する。公開メソッドは次のとおり。
  - `List`：全在庫と、各行の期限状態 (🔴 / 🟡 / 🟢 / 期限切れ / なし) と残り日数を返す
  - `Add` / `Update` / `Delete`：在庫 1 行の追加 / 更新 / 削除
  - `Restore`：削除直後の「元に戻す」用。削除前と同じ ID / 追加日時で行を復元する
  - `ListSuggestions`：既存在庫名とレシピ材料名を合わせた食材名候補 (直近の単位つき) と単位候補を返す
- 期限状態の判定は Go 側ドメインで行い、閾値は `config.Defaults()` の `ExpiryAlertRedDays` (3) /
  `ExpiryAlertYellowDays` (7) を読む。期限切れ (残り日数が負) は 🔴 と分けた独立の状態とし、
  色ドットの代わりに「期限切れ」バッジ + 行背景の薄い赤で示す (当日期限は 🔴 のまま)。
- S05 画面 (`/fridge`) を実装する。ワイヤー `S05_fridge.drawio.svg` に従い、上から
  「検索欄 + 並び替え」「インライン追加行」「在庫テーブル (食材 / 量 / 賞味期限 / 残り日 / 操作)」を置く。
  - 並び替え：賞味期限が近い順 (既定。期限なしは末尾) / 食材名 / 追加日 新しい順
  - 検索：食材名の部分一致で表示を絞る (再問い合わせなし)
  - 0 件時は空状態メッセージを表示する
- S06 インライン追加行：食材名 (候補補完つき自由入力) / 量 / 単位 / 賞味期限 (任意) / `追加`。
  候補を選ぶと単位が直近の単位で埋まる。追加後は入力欄をクリアし、食材名欄にフォーカスを戻して連続追加できる。
- S06 行内編集：行の `編集` で、その行のセルが入力欄に切り替わる。`保存` で確定、`キャンセル` で元に戻す。
  編集中は他の行の `編集` / `削除` を無効化する。
- 削除：確認ダイアログを挟まず即削除し、トーストの `元に戻す` で復元できる。
- トースト基盤として `sonner` を導入し、shadcn の `sonner.tsx` を `src/components/ui/` に手動コピーする。
- 入力バリデーション (食材名 1〜100 字、量 0 より大きく 99999 以下、単位 20 字以内、賞味期限は日付形式) を
  フロントと Go 側の両方で行い、Go 側は型付きエラーで返す。
- `docs/scenarios/fridge.feature` と `docs/product.md` S06 節を、食材マスタ前提の記述 (「新規作成: "◯◯"」候補、
  Ingredient の defaultUnit) から自由テキスト + 候補補完の記述に改める。design.md §8 の順序案も 1 本化に合わせて直す。

### Non-goals

- **食材マスタ (Ingredient) の導入**：在庫もレシピも自由テキストのまま。名寄せ辞書 / 表記ゆれ吸収 / 単位変換は
  将来の `add-ingredient-master` で、レシピ材料と同時に移行する。
- **同名在庫の自動合算**：同じ食材名を追加しても既存行に足し込まず、別行として追加する (賞味期限が違う実物を区別するため)。
- **在庫の部分消費 UI** (「1 個使った」ボタン等)：量の修正は行内編集で行う。
- **献立との自動連動** (作ったら在庫を減らす)：MVP 後の `add-mealplan-cook-consume` で扱う。
- **買い物リストからの一括登録**：`add-shopping-purchase` で扱う。本 proposal の `Add` を再利用できる形にはしておく。
- **並び替え / 検索条件の永続化**：画面を離れると既定 (賞味期限が近い順、検索なし) に戻る。
- **期限閾値のユーザー変更**：S09 設定画面 (MVP 後)。本 proposal は `config.Defaults()` を読むだけ。
- **Frontend テスト runner の導入**：Vitest は未導入のまま。フロントは type-check / build:dev / E2E で検証する。

## Capabilities

### New Capabilities

- `fridge`：冷蔵庫在庫の一覧 (期限状態の色分け / 並び替え / 検索)、インライン追加 (食材名の候補補完つき)、
  行内編集、削除と元に戻す。

### Modified Capabilities

(なし。レシピ材料名は候補の供給元として読むだけで、`recipe` の要件は変わらない。)

## Impact

### 新設ファイル

- `internal/storage/sqlite/migrations/00006_stock_items.sql`：`stock_items` テーブル追加
- `internal/domain/fridge/stock.go`：`StockItem` / `ExpiryStatus` / `Suggestions` 型
- `internal/domain/fridge/errors.go`：repo と service が共有するエラー sentinel
- `internal/domain/fridge/expiry.go`：期限状態と残り日数の判定 (`Evaluate`)
- `internal/domain/fridge/expiry_test.go`：期限判定の table-driven test
- `internal/storage/sqlite/stock_repo.go`：`StockRepo` (List / GetByID / Create / Update / Delete / Restore / ListSuggestions)
- `internal/storage/sqlite/stock_repo_test.go`：実 SQLite での CRUD / Restore / 候補集計のテスト
- `internal/service/fridge.go`：`FridgeService` と入力バリデーション
- `internal/service/fridge_test.go`：サービス層の BDD 風テスト
- `frontend/src/components/fridge/fridge-table.tsx`：在庫テーブル本体 (表示行と編集行の切替)
- `frontend/src/components/fridge/fridge-item-row.tsx`：表示モードの 1 行
- `frontend/src/components/fridge/fridge-edit-row.tsx`：行内編集モードの 1 行
- `frontend/src/components/fridge/fridge-add-row.tsx`：インライン追加行
- `frontend/src/components/fridge/stock-name-input.tsx`：食材名の候補補完つき入力 (ARIA combobox)
- `frontend/src/components/fridge/expiry-dot.tsx`：期限状態の色ドットと期限切れバッジ
- `frontend/src/components/fridge/fridge-sort.ts`：並び替えキーと比較関数
- `frontend/src/components/fridge/fridge-sort-select.tsx`：並び替えセレクト
- `frontend/src/components/fridge/fridge-search-input.tsx`：検索欄
- `frontend/src/components/fridge/stock-form.ts`：追加行 / 編集行が共有する入力値の型とフロント側バリデーション
- `frontend/src/components/fridge/error-messages.ts`：Go エラー → i18n key の対応表
- `frontend/src/components/fridge/use-fridge.ts`：TanStack Query の query / mutation フック
- `frontend/src/components/ui/sonner.tsx`：shadcn の Toaster ラッパ (手動コピー)
- `frontend/e2e/bindings-shim/fridgeservice.ts`：E2E 用 shim
- `frontend/e2e/fridge.spec.ts`：`fridge.feature` のシナリオ名に揃えた E2E

### 更新ファイル

- `main.go`：`application.Options.Services` に `service.NewFridgeService(db)` を追加
- `cmd/e2e-server/main.go`：`newRegistry` に `FridgeService` を追加 (コメントの「3 サービス」も更新)
- `cmd/e2e-server/dispatch.go`：`serviceRegistry` コメントのサービス数を更新
- `frontend/e2e/bindings-shim/index.ts`：`FridgeService` を re-export と型突き合わせに追加
- `frontend/e2e/helpers.ts`：`openFridge` / `addStockItem` ヘルパを追加
- `frontend/src/routes/fridge.tsx`：プレースホルダを S05 本体に差し替え
- `frontend/src/components/layout/root-layout.tsx`：`<Toaster />` を配置
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.fridge.*` を追加し、`placeholder` を削除
- `frontend/package.json` / `frontend/package-lock.json`：`sonner` を追加
- `docs/scenarios/fridge.feature`：食材名の候補補完シナリオへの書き換え、期限切れ / 検索 / 空状態 / バリデーションのシナリオ追加、
  ヘッダの対応 proposal 名を `add-fridge-inventory` に更新
- `docs/product.md`：§4 S05 に検索欄と期限切れの扱いを追記、S06 の Ingredient 前提の記述を自由テキスト + 候補補完に改める
- `docs/design.md`：§4 StockItem を `name` 直持ちに改め、§8 の順序案を `add-fridge-inventory` 1 本に置き換える

### 生成物 (再生成のみ)

- `frontend/bindings/`：`wails3 generate bindings` で `FridgeService` と `fridge` モデルを生成

### migration

- `00006_stock_items.sql`：`stock_items (id, name, amount, unit, expires_at, added_at, updated_at)` を作成

### 後続 proposal への引き継ぎ

- `add-shopping-list`：在庫との突き合わせは `StockRepo.List` を再利用する。名前一致は trim 後の完全一致になる
  (表記ゆれは吸収しない) ことを前提に設計する
- `add-shopping-purchase`：一括登録は `FridgeService.Add` を行ごとに呼ぶか、`StockRepo` に一括 insert を足す
- `add-ingredient-master`：`stock_items.name` と `recipe_ingredients.name` を同時に FK 列へ移行する。
  `ListSuggestions` はマスタ検索に差し替わる
- `add-mealplan-suggest-by-stock`：期限状態は `fridge.Evaluate` を再利用する (Go 側に置いた理由の一つ)。
  「使い切り」軸で `expired` の在庫を候補に含めるかは、その proposal で決める
