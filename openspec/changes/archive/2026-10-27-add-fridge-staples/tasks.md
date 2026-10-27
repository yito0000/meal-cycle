## 1. 食材名の表記をそろえる関数の移動

- [x] 1.1 `internal/domain/ingredient/name.go` を新設し、`internal/domain/shopping/normalize.go` の `normalizeKey` の本体を `NormalizeName(s string) string` として移す (design.md D3)。完了判定: `go build ./internal/...` が通る
- [x] 1.2 `internal/domain/shopping/normalize.go` を更新し、`normalizeKey` を `ingredient.NormalizeName` を呼ぶだけの関数にする。漢字とかなをそろえない理由のコメントは `name.go` 側に移す。完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/domain/shopping/normalize_test.go` のケースを `internal/domain/ingredient/name_test.go` (新設) に table-driven のまま移し、`normalize_test.go` を削除する。完了判定: `go test ./internal/domain/...` が pass する

## 2. 常備品のデータとリポジトリ

- [x] 2.1 `internal/storage/sqlite/migrations/00011_staples.sql` を新設し、design.md D1 の `staples` テーブルを Up で作り、Down で `DROP TABLE staples` する。
  `CREATE INDEX idx_staples_name ON staples (name)` も足す。完了判定: `go test ./internal/storage/sqlite/ -run "TestOpenAndMigrate|TestMigrateIsIdempotent"` が pass する
- [x] 2.2 `internal/domain/fridge/staple.go` を新設する (design.md D2)
    - `StapleState` 型と `StapleEnough = "enough"` / `StapleLow = "low"` / `StapleOut = "out"` を定義する
    - `Staple` struct を `ID` / `Name` / `State` / `AddedAt` / `UpdatedAt` の JSON タグ付きで定義する
    - `StapleState.Valid() bool` を定義する
    - 完了判定: `go build ./internal/...` が通る
- [x] 2.3 `internal/domain/fridge/errors.go` に `ErrStapleNameEmpty` / `ErrStapleNameTooLong` / `ErrStapleDuplicate` / `ErrStapleInvalidState` / `ErrStapleNotFound` / `ErrStapleInvalidRestore` を
  `fridge: staple ...` の文言で追加する (design.md D4)。完了判定: `go build ./internal/...` が通る
- [x] 2.4 `internal/storage/sqlite/staple_repo.go` を新設し、`StapleRepo` に `List` / `GetByID` / `Create` / `UpdateState` / `Delete` / `Restore` を実装する。
  ID は UUID v7、時刻は RFC3339 (UTC) で `stock_repo.go` にそろえる。見つからない ID には `fridge.ErrStapleNotFound` を返す。完了判定: `go build ./internal/...` が通る
- [x] 2.5 `internal/storage/sqlite/staple_repo_test.go` を新設し、実 SQLite でテストする。完了判定: `go test ./internal/storage/sqlite/ -run TestStapleRepo` が pass する
    - Create した常備品は状態と added_at を保って List に出る
    - UpdateState で state が変わり updated_at が進み、added_at は変わらない
    - 存在しない ID の UpdateState と Delete は ErrStapleNotFound を返す
    - Restore は渡した ID と added_at で作り直す
    - state に列挙外の値を直接 INSERT すると CHECK 制約で失敗する

## 3. FridgeService の常備品の操作

- [x] 3.1 `internal/service/fridge.go` を更新し、`FridgeService` に `staples *sqlite.StapleRepo` を持たせ、`NewFridgeService` で作る (design.md D4)。完了判定: `go build .` が通る
- [x] 3.2 `internal/service/fridge.go` に `validateStapleName` を追加する。食材名を trim し、空なら `ErrStapleNameEmpty`、100 文字超なら `ErrStapleNameTooLong` を返す。
  上限は既存の `stockNameMaxRunes` を使う。完了判定: `go build .` が通る
- [x] 3.3 `internal/service/fridge.go` に `ListStaples` / `AddStaple` / `SetStapleState` / `DeleteStaple` / `RestoreStaple` を追加する (design.md D3 / D4)
    - `AddStaple` と `RestoreStaple` は、`ingredient.NormalizeName` でそろえた名前が既存の常備品と一致したら `ErrStapleDuplicate` を返す
    - `AddStaple` は状態を `StapleEnough` で作る
    - `SetStapleState` と `RestoreStaple` は `StapleState.Valid()` でない状態に `ErrStapleInvalidState` を返す
    - `RestoreStaple` は ID が空か `AddedAt` が RFC3339 でなければ `ErrStapleInvalidRestore` を返す
    - 完了判定: `go build .` が通る
- [x] 3.4 `internal/service/fridge_test.go` に BDD 風のテストを追加する。完了判定: `go test ./internal/service/ -run 'TestFridgeService_.*Staple'` が pass する
    - `TestFridgeService_AddStaple`
      - should create staple with enough state when name is valid
      - should trim name when name has surrounding spaces
      - should return ErrStapleNameEmpty when name is blank
      - should return ErrStapleNameTooLong when name exceeds 100 runes
      - should return ErrStapleDuplicate when normalized name matches existing staple (「ミルク」と「みるく」)
      - should allow name that differs only in kanji and kana (「玉ねぎ」と「たまねぎ」)
      - should not touch stock items when stock with same name exists
    - `TestFridgeService_SetStapleState`
      - should persist new state when state is valid
      - should return ErrStapleInvalidState when state is unknown
      - should return ErrStapleNotFound when id does not exist
    - `TestFridgeService_DeleteStaple`
      - should remove staple when id exists
    - `TestFridgeService_RestoreStaple`
      - should recreate staple with same id, state and addedAt when restored after delete
      - should return ErrStapleInvalidRestore when id is empty
      - should return ErrStapleDuplicate when same name was added after delete
- [x] 3.5 `wails3 generate bindings -clean=true -ts -i` を実行し、`frontend/bindings/` に `Staple` と常備品のメソッドが出ることを確かめる。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.6 `frontend/e2e/bindings-shim/fridgeservice.ts` に `ListStaples` / `AddStaple` / `SetStapleState` / `DeleteStaple` / `RestoreStaple` を追加する。
  戻り値型は `fridge$0.Staple[]` のように null を含めない形で書く。完了判定: `cd frontend && npm run type-check` が通る

## 4. S07 の算出

- [x] 4.1 `internal/domain/shopping/list.go` に `StapleRef` (`ID` / `Name` / `State`) と `StapleItem` (`ID` / `Name` / `State` / `Recipes`、JSON タグ付き) を追加し、
  `List` に `Staples []StapleItem` と `HasStapleIngredients bool` を足す (design.md D5)。`State` は `fridge.StapleState` を使う。完了判定: `go build ./internal/...` が通る
- [x] 4.2 `internal/domain/shopping/aggregate.go` を更新し、`Aggregate` に `staples []StapleRef` を受け取らせる (design.md D6)
    - 合算の前に、表記をそろえた食材名が常備品のいずれかと一致する材料を飛ばす
    - 返り値に、飛ばした材料があったかどうかの bool を足す
    - 完了判定: `go build ./internal/...` が通る
- [x] 4.3 `internal/domain/shopping/staple.go` を新設し、`StapleItems(batches []PlannedBatch, staples []StapleRef) []StapleItem` を実装する (design.md D6)
    - `StapleEnough` の常備品を除く
    - 作った印の無い組の材料を献立の順に走査し、名前が一致する常備品の `Recipes` にレシピ名を重複なく足す
    - 名前が一致する常備品が 2 件あれば、`staples` の順で最初の 1 件にだけ足す
    - `StapleOut` を先、`StapleLow` を後にし、それぞれ食材名の昇順に並べる
    - 完了判定: `go build ./internal/...` が通る
- [x] 4.4 `internal/domain/shopping/aggregate_test.go` に table-driven のケースを追加し、既存のケースは `staples` に nil を渡す形に直す。完了判定: `go test ./internal/domain/shopping/` が pass する
    - 常備品と同名の量のある材料は items に入らない
    - 常備品と同名の量を持たない材料は items に入らない
    - カタカナとひらがなの違う材料名も常備品に寄せられる
    - 漢字とかなの違う材料名は常備品に寄せられない
    - 常備品に寄せた材料があると bool が true になり、無ければ false になる
    - 作った印の付いた組の材料は bool を true にしない
- [x] 4.5 `internal/domain/shopping/staple_test.go` を新設し、`StapleItems` を table-driven でテストする。完了判定: `go test ./internal/domain/shopping/` が pass する
    - enough の常備品は返らない
    - out が low より先に並び、それぞれ食材名の昇順になる
    - 献立に無い常備品は Recipes が空で返る
    - Recipes は献立の順 (日付、時間帯、セル内の並び) で重複なく並ぶ
    - 作った印の付いた組のレシピは Recipes に入らない
    - 表記をそろえた名前が同じ常備品が 2 件あると、先の 1 件にだけ Recipes が入る
- [x] 4.6 `internal/service/shopping.go` を更新し、`ShoppingService` に `staples *sqlite.StapleRepo` を持たせ、`List` で常備品を読み、
  `Aggregate` と `StapleItems` に渡して `Staples` と `HasStapleIngredients` を埋める。`Staples` は空でも空スライスで返す。完了判定: `go build .` が通る
- [x] 4.7 `internal/service/shopping_test.go` に BDD 風のテストを追加する。完了判定: `go test ./internal/service/ -run TestShoppingService_List` が pass する
    - should list low and out staples when period has no meal plans
    - should exclude ingredient from items when staple with same name exists
    - should not list enough staples even when their ingredient is used in period
    - should attach recipe names to staple item when its ingredient is used in period
    - should report HasStapleIngredients when every ingredient is a staple
    - should keep stock status of other items unchanged when staples exist
- [x] 4.8 `wails3 generate bindings -clean=true -ts -i` を再実行し、`shopping.List` に `staples` と `hasStapleIngredients` が出ることを確かめる。完了判定: `cd frontend && npm run type-check` が通る

## 5. S05 の常備品の欄

- [x] 5.1 `frontend/src/components/fridge/use-staples.ts` を新設する (design.md D8)
    - `useStapleList()`：query key `["fridge", "staples"]` で `FridgeService.ListStaples()` を呼び、null を空配列にそろえる
    - `useAddStaple()` / `useSetStapleState()` / `useDeleteStaple()` / `useRestoreStaple()`：成功したら常備品の一覧と `["shopping"]` を invalidate する
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.2 `frontend/src/components/fridge/error-messages.ts` に `stapleErrorOf(err)` を追加し、`fridge: staple ...` の各コードを `pages.fridge.staples.errors.*` に対応づける。
  既存の `RULES` より先に `staple` を含むコードを判定する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.3 `frontend/src/components/fridge/staple-state-toggle.tsx` を新設する (design.md D8)
    - props は `value: StapleState`、`disabled: boolean`、`onChange(state)`、`name: string`
    - 「ある / 少ない / ない」の 3 つのボタンを `role="group"` で並べ、group に `aria-label` (`pages.fridge.staples.state_group`、食材名入り) を付ける
    - 選択中のボタンに `aria-pressed="true"` を付ける。選択中のボタンを押しても `onChange` を呼ばない
    - 「ない」は赤系、「少ない」は黄系、「ある」は無彩色で選択中を示す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.4 `frontend/src/components/fridge/staple-row.tsx` を新設する。食材名、`StapleStateToggle`、「削除」ボタンを並べ、`data-testid="staple-row"` と `data-state` を付ける。
  状態の保存中は切替を無効にする。完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.5 `frontend/src/components/fridge/staple-panel.tsx` を新設する (design.md D7)
    - 見出し (`pages.fridge.staples.title`) と説明 (`pages.fridge.staples.description`) を出す
    - 追加行は `IngredientNameInput` と「追加」ボタン。`nameSuggestions` を props で受け取る
    - 食材名の欄の Enter で追加する。`isImeComposing` の Enter では追加しない
    - 送信前に空 (trim 後) なら `pages.fridge.staples.errors.name_empty` を出し、Go のエラーは `stapleErrorOf` で入力欄の下に出す
    - 追加に成功したら入力を空にし、入力欄にフォーカスを残す
    - 一覧は `name.localeCompare(other, "ja")` で並べて `StapleRow` を出し、0 件なら `pages.fridge.staples.empty` を出す
    - 削除と「元に戻す」のコールバックは props (`onDelete`) でページ側に任せる
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 5.6 `frontend/src/routes/fridge.tsx` を更新する (design.md D7)
    - 検索、並び替え、追加行、在庫のテーブルを左の列に、`StaplePanel` を右の列に置き、`grid grid-cols-1 items-start gap-6 lg:grid-cols-[1fr_20rem]` で組む
    - 常備品の削除を在庫と同じくページ側で扱い、トースト (`pages.fridge.staples.toast.deleted` と `pages.fridge.toast.undo`) の「元に戻す」で `RestoreStaple` を呼ぶ
    - 完了判定: `cd frontend && npm run type-check` が通り、`task dev` で常備品の欄が在庫の右に出ることを目視で確認する
- [x] 5.7 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` に `pages.fridge.staples.*` を追加する
    - `title`：「常備品」
    - `description`：「牛乳やパンのように家族が日常的に食べる品を、量ではなく状態で持ちます」
    - `add` / `name_placeholder` / `empty` (「常備品はまだありません」) / `delete`
    - `state.enough`：「ある」、`state.low`：「少ない」、`state.out`：「ない」、`state_group`：「{{name}} の状態」
    - `toast.deleted`：「{{name}} を削除しました」
    - `errors.name_empty` / `errors.name_too_long` / `errors.duplicate` (「すでに常備品にあります」) / `errors.invalid_state` / `errors.not_found` / `errors.unknown`
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 6. S07 の常備品の欄

- [x] 6.1 `frontend/src/components/shopping/use-shopping-list.ts` を更新し、`staples` を `?? []` に、各項目の `recipes` も `?? []` にそろえる。完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.2 `frontend/src/components/shopping/shopping-staple-list.tsx` を新設する (design.md D9 / D10)
    - 見出し (`pages.shopping.staples.title`) と説明 (`pages.shopping.staples.description`) を出す
    - 各項目に食材名、状態のバッジ (`pages.fridge.staples.state.*`)、用途 (`recipes.join(", ")`、空なら出さない)、「買った」ボタンを出す
    - 各項目に `data-testid="shopping-staple-item"` と `data-state` を付ける。チェック欄は置かない
    - 「買った」のコールバックは props (`onBought(item)`) でページ側に任せる
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.3 `frontend/src/routes/shopping.tsx` を更新する (design.md D9 / D10)
    - `useSetStapleState` で「買った」を処理し、トースト (`pages.shopping.staples.bought`) の「元に戻す」で押す前の状態に戻す
    - 献立が無い期間は、`empty.no_plans` の下に `ShoppingStapleList` を出す (`staples.length > 0` のとき)
    - 2 列の組みの左の列では、登録ボタン (または空状態のメッセージ) の下、`ShoppingPantryList` の上に `ShoppingStapleList` を置く
    - `items.length === 0` かつ `hasStapleIngredients` のときは `empty.no_ingredients` の代わりに `empty.only_staples` を出す
    - `CopyShoppingTextButton` を、`dishes.length > 0 && items.length > 0` または `staples.length > 0` のときに出す
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.4 `frontend/src/components/shopping/shopping-text.ts` を更新し、`buildShoppingText` に `staples` を受け取らせて design.md D11 の順で組み立てる。件数に常備品の行を含める。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.5 `frontend/src/components/shopping/copy-shopping-text-button.tsx` を更新し、`staples` を props で受け取って `buildShoppingText` に渡す。完了判定: `cd frontend && npm run type-check` が通る
- [x] 6.6 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する
    - `pages.shopping.staples.title`：「常備品」
    - `pages.shopping.staples.description`：「少ない、ない、にした常備品です。献立に関係なく並びます」
    - `pages.shopping.staples.bought_button`：「買った」、`pages.shopping.staples.bought`：「{{name}} をあるにしました」、`pages.shopping.staples.undo`：「元に戻す」
    - `pages.shopping.staples.recipes`：「用途: {{recipes}}」
    - `pages.shopping.empty.only_staples`：「献立の材料はすべて常備品です」
    - `pages.shopping.copy.staple_heading`：「常備品」、`pages.shopping.copy.staple_item`：「- {{name}}」、`pages.shopping.copy.staple_item_low`：「- {{name}} (少ない)」
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する
- [x] 6.7 `frontend/src/components/shopping/shopping-text.test.ts` を新設し、`buildShoppingText` を `it.each` でテストする。`t` はキーと引数を返すスタブにする。
  完了判定: `cd frontend && npm run test` が pass する
    - 常備品が無いときは、これまでどおり材料と家にあるか確かめる材料だけを組み立てる
    - 常備品は材料のあとに空行と見出しを挟んで並ぶ
    - 少ないの常備品には (少ない) が添えられ、ないの常備品には添えられない
    - 常備品は家にあるか確かめる材料より前に並ぶ
    - 材料が無く常備品だけのときは、見出しのあとに空行と常備品の見出しが続く
    - 件数は材料、常備品、家にあるか確かめる材料の行の合計になる

## 7. E2E

- [x] 7.1 `frontend/e2e/helpers.ts` に次を追加する。完了判定: `cd frontend && npm run type-check` が通る
    - `addStaple(page, name, state?)`：`/api/FridgeService/AddStaple` で作り、state が渡されたら `/api/FridgeService/SetStapleState` で変える
    - `stapleRow(page, name)`：S05 の `staple-row` から食材名で 1 行を取る
    - `shoppingStapleItem(page, name)`：S07 の `shopping-staple-item` から食材名で 1 項目を取る
- [x] 7.2 `frontend/e2e/fridge-staples.spec.ts` を新設する。describe 名は Requirement 名、test 名は Scenario 名と一字一句そろえる。完了判定: `task test:e2e` で以下が pass する
    - describe「常備品の永続化」
      - 常備品は在庫の一覧に出ない
    - describe「常備品の欄の表示」
      - 常備品が食材名の順に並ぶ
      - 常備品が無い場合はその旨を示す
      - 在庫の検索は常備品の一覧を絞り込まない
    - describe「常備品の追加」
      - 常備品を追加すると状態はあるになる
      - Enter で続けて追加できる
    - describe「常備品の入力検証」
      - 食材名が空だと常備品を追加できない
      - 同じ常備品は 2 つ登録できない
      - 同名の在庫があっても常備品を追加できる
    - describe「常備品の状態の切替」
      - 状態を押すだけで切り替わる
      - 状態を切り替えても並び順は変わらない
    - describe「常備品の削除と元に戻す」
      - 常備品を削除してトーストの元に戻すで復元する
- [x] 7.3 `frontend/e2e/shopping-staples.spec.ts` を新設する。完了判定: `task test:e2e` で以下が pass する
    - describe「常備品の欄」
      - 少ないとないの常備品が献立に関係なく並ぶ
      - 常備品を使うレシピが用途として添えられる
      - 常備品はチェックと絞り込みの件数の対象にならない
    - describe「常備品と同名の材料」
      - 常備品と同名の材料は材料テーブルに出ない
      - 量を持たない材料も常備品に寄せられる
    - describe「S07 の常備品の「買った」」
      - 買ったを押すと常備品の欄から消える
      - 買ったを元に戻すと元の状態で欄に戻る
      - 買ったを押しても購入済チェックは外れない
    - describe「買い物リストの空状態」
      - 献立の材料がすべて常備品と同名
      - 献立が無い期間でも買う常備品は表示される
- [x] 7.4 `frontend/e2e/shopping-copy.spec.ts` の describe「買い物リストのテキストコピー」にテストを追加する。完了判定: `task test:e2e` で以下が pass する
    - 常備品は材料のあとに見出しを付けて続く
    - 献立が無い期間でも常備品をコピーできる

サービス層で確かめるシナリオ (「再起動後も常備品の状態が残る」「変換確定の Enter では常備品を追加しない」「画面を経由しない不正な状態は拒否される」
「表記の違う材料も常備品に寄せられる」) は E2E に含めない。
変換確定の Enter は IME を通らない headless Chromium では再現できないので、Windows 実機の確認 (8.9) で見る。

## 8. ドキュメントと検証

- [x] 8.1 `docs/product.md` を更新する。完了判定: 差分を目視で確認する
    - §1「家族で使う場合」の末尾に、常備品を add-fridge-staples で入れたことを添える
    - §4 S05 に、常備品の欄 (置き場所、追加、状態の切替、削除と元に戻す、検索と並び替えの対象外) の項を足す
    - §4 S07 に、常備品の欄、同名の材料を寄せる規則、「買った」、空状態とテキストコピーの変更を足す
    - §7 決定事項に、常備品と同名の材料を常備品の欄に寄せる判断と理由を足す
- [x] 8.2 `docs/design.md` §4 の StockItem の注記を、`staples` テーブルの列と状態の値に書き換え、§9 の「常備品」の候補に proposal 名を添える。完了判定: 差分を目視で確認する
- [x] 8.3 `docs/wireframes/S05_fridge.v2.drawio.svg` を新設し、在庫のテーブルの右に常備品の欄 (見出し、説明、追加行、3 つの状態ボタンと削除の行) を描く (design.md D12)。
  可視 SVG と `content=` 内の drawio XML の両方に同じ図形を入れる。完了判定: ブラウザで SVG を開いて目視で確認する
- [x] 8.4 `docs/wireframes/README.md` のファイル一覧に `S05_fridge.v2.drawio.svg` の行を追加する。完了判定: 差分を目視で確認する
- [x] 8.5 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 8.6 `cd frontend && npm run test` と `npm run type-check` と `npm run build:dev` を実行する。完了判定: 3 つとも成功する
- [x] 8.7 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 8.8 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 8.9 `task build:win` で Windows 版を作り、Windows 実機で常備品の日本語名の追加、変換確定の Enter、S07 の「買った」を確認する。完了判定: 目視で確認する
- [x] 8.10 `openspec validate add-fridge-staples --strict` を実行する。完了判定: エラー 0 件
