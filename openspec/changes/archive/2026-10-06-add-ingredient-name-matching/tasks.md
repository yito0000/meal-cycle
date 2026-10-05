## 1. 表記をそろえる関数 (Go)

- [x] 1.1 `go.mod` の `golang.org/x/text` を直接の依存に上げる (`go get golang.org/x/text` のあと `go mod tidy`)。完了判定: `go build .` が通り、go.mod で `// indirect` が外れている
- [x] 1.2 `internal/domain/shopping/normalize.go` を新設し、`normalizeKey(s string) string` を定義する (design.md D2)
    - `strings.TrimSpace` → `norm.NFKC.String` → カタカナ (U+30A1〜U+30F6) を 0x60 引いてひらがなに → `strings.ToLower` の順に変換する
    - 漢字とかなをそろえない理由 (辞書が要るため R19 に回す) を WHY のコメントで 1 行書く
    - 完了判定: `go build ./internal/...` が通る
- [x] 1.3 `internal/domain/shopping/normalize_test.go` を新設し、`normalizeKey` の table-driven test を書く
    - 前後の空白を除く
    - 全角英字「ｇ」を半角「g」にする
    - 全角数字と全角空白を半角にする
    - 半角カタカナ「ﾀﾏﾈｷﾞ」を「たまねぎ」にする (濁点を合成する)
    - カタカナ「タマネギ」を「たまねぎ」にする
    - 長音符を残す (「ピーマン」→「ぴーまん」)
    - 英字の大文字を小文字にする (「KG」→「kg」)
    - 漢字は変えない (「玉ねぎ」→「玉ねぎ」)
    - 完了判定: `go test ./internal/domain/shopping/...` が pass する

## 2. 突き合わせ (Go)

- [x] 2.1 `internal/domain/shopping/aggregate.go` の `Aggregate` を更新する (design.md D3)
    - `ingredientKey` を `normalizeKey` を通した食材名と単位で作る
    - 行を初めて作るとき、`Item.Name` と `Item.Unit` には `TrimSpace` した材料の表記を入れる
    - `stockByKey` と `stockByName` のキーも `normalizeKey` で作る
    - 量る単位の判定は、`measuringUnits` の各要素と材料の単位を `normalizeKey` に通してから比べる
    - 関数のコメントの「trim 後の食材名 + 単位」を「表記をそろえた食材名 + 単位」に改める
    - 完了判定: `go build ./internal/...` が通る
- [x] 2.2 `internal/domain/shopping/aggregate_test.go` にケースを足す
    - ひらがなとカタカナの違う食材名が 1 行に合算され、表示は最初の材料の表記になる
    - 全角「ｇ」と半角「g」の単位が 1 行に合算される
    - カタカナの材料とひらがなの在庫が突き合わされ、在庫の状態が covered になる
    - 漢字とかなの違う食材名は別の行のまま
    - カタカナとひらがなの違う量なし材料が 1 行にまとまる
    - 量る単位をカタカナで書いても (「大サジ」) 量なし材料として扱う
    - 完了判定: `go test ./internal/domain/shopping/...` が pass する
- [x] 2.3 `internal/service/shopping_test.go` の `TestShoppingService_List` に BDD 風のテストを足す
    - should merge ingredients when names differ only in hiragana and katakana
    - should count stock when stock name differs only in width or kana
    - should keep separate rows when names differ in kanji and kana
    - 完了判定: `go test ./internal/service/...` が pass する

## 3. 候補補完の部品の移動 (Frontend)

- [x] 3.1 `frontend/src/lib/ime.ts` を新設し、`fridge/stock-form.ts` の `isImeComposing` を移す (design.md D4)。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.2 `frontend/src/components/fridge/stock-form.ts` から `isImeComposing` を削除する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.3 `frontend/src/components/ingredient/ingredient-name-input.tsx` を新設し、`fridge/stock-name-input.tsx` の中身を移す (design.md D4)
    - コンポーネント名を `IngredientNameInput` に改める
    - `isImeComposing` を `@/lib/ime` から import する
    - 候補一覧の aria-label を `ingredient.suggestions_label` に改める
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.4 `frontend/src/components/fridge/stock-name-input.tsx` を削除する。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.5 `frontend/src/components/ingredient/use-ingredient-suggestions.ts` を新設する (design.md D4)
    - `fridge/use-fridge.ts` の `useStockSuggestions` を `useIngredientSuggestions` と改名して移す
    - query key を `["ingredient", "suggestions"]` にし、`INGREDIENT_SUGGESTIONS_KEY` として export する
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.6 `frontend/src/components/fridge/use-fridge.ts` を更新する (design.md D5)
    - `useStockSuggestions` と `SUGGESTIONS_KEY` を削除する
    - `useInvalidateFridge` で `INGREDIENT_SUGGESTIONS_KEY` を捨てる
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.7 `frontend/src/components/fridge/fridge-add-row.tsx` と `fridge-edit-row.tsx` を、`IngredientNameInput` と `@/lib/ime` の `isImeComposing` を import する形に改める。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.8 `frontend/src/routes/fridge.tsx` の候補の取得を `useIngredientSuggestions` に替える。完了判定: `cd frontend && npm run type-check` が通る
- [x] 3.9 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新する
    - `ingredient.suggestions_label` を追加する (値は今の `pages.fridge.add.suggestions_label` と同じ)
    - `pages.fridge.add.suggestions_label` を削除する
    - 完了判定: `cd frontend && npm run build:dev` が通り、両言語で同じキーがそろっていることを目視で確認する

## 4. S04 の材料行 (Frontend)

- [x] 4.1 `frontend/src/components/recipe/recipe-ingredient-row.tsx` を更新する (design.md D4 / D6)
    - props に `nameSuggestions: NameSuggestion[]` と `unitSuggestions: string[]` を足す
    - 名前の `Input` を `IngredientNameInput` に替え、`onSelect` で名前を入れ、単位が空 (trim 後) のときだけ候補の単位を入れる
    - 単位の `Input` に `list` 属性を付け、`useId` で作った id の `<datalist>` に `unitSuggestions` を並べる
    - 冒頭のコメントの「食材コンボボックスは持たない (design.md D9)」を、候補補完を持つ旨と本 change の design.md D4 への参照に改める
    - 完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.2 `frontend/src/components/recipe/recipe-edit-form.tsx` で `useIngredientSuggestions` を呼び、各 `RecipeIngredientRow` に `names` と `units` を渡す。
  完了判定: `cd frontend && npm run type-check` が通る
- [x] 4.3 `frontend/src/routes/recipes/new.tsx` と `frontend/src/routes/recipes/$recipeId.edit.tsx` の保存の成功時に、
  `INGREDIENT_SUGGESTIONS_KEY` の query も捨てる (design.md D5)。完了判定: `cd frontend && npm run type-check` が通る

## 5. E2E

- [x] 5.1 `frontend/e2e/recipe-edit.spec.ts` に describe「材料名と単位の候補補完」を追加する
    - 在庫の食材名が材料名の候補に出る
    - 他のレシピの材料名が材料名の候補に出る
    - 材料名の候補を選ぶと単位が埋まる
    - 材料行の単位を先に入れていれば上書きしない
    - 保存したレシピの材料名が次から候補に出る
    - 単位欄に既知の単位が候補として示される (単位欄の `list` が指す `datalist` の `option` を確かめる)
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 5.2 `frontend/e2e/shopping.spec.ts` の describe「同一食材の合算」に Scenario を足す
    - ひらがなとカタカナの違う食材名は 1 行に合算される
    - 表記の違う在庫も同じ食材として数える
    - 漢字とかなの違う食材名は別の行になる
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 5.3 `frontend/e2e/shopping.spec.ts` の describe「量を持たない材料の欄」に Scenario を足す
    - 表記の違う量なし材料は 1 項目にまとまる
    - 完了判定: `task test:e2e` で上記が pass する
- [x] 5.4 `frontend/e2e/fridge.spec.ts` の describe「食材名の候補補完」が、部品の移動のあとも変更なしで pass することを確かめる。完了判定: `task test:e2e` で pass する

## 6. ドキュメントと検証

- [x] 6.1 `docs/product.md` を更新する
    - §4 S04 の材料行の説明を、「食材 (S06 と同じ候補補完つきの自由入力)」「量」「単位 (既知の単位を候補に出す)」「×」に改める
    - §4 S07 の材料テーブルの合算の説明に、ひらがなとカタカナ、全角と半角、大文字と小文字をそろえてから比べること、漢字とかなはそろえないことを足す
    - 完了判定: 差分を目視で確認する
- [x] 6.2 `docs/reviews/2026-09-25-design-ux-review.md` の R3 のチェックを付け、`add-ingredient-name-matching` で対応した旨を追記する。完了判定: 差分を目視で確認する
- [x] 6.3 `go test ./internal/... ./cmd/...` を実行する。完了判定: 全テストが pass する
- [x] 6.4 `cd frontend && npm run type-check` を実行する。完了判定: エラー 0 件
- [x] 6.5 `cd frontend && npm run build:dev` を実行する。完了判定: ビルドが成功する
- [x] 6.6 `task test:e2e` を実行する。完了判定: 既存の E2E を含めて全テストが pass する
- [x] 6.7 `task lint` を実行する。完了判定: Go と Frontend の両方で指摘 0 件
- [x] 6.8 `openspec validate add-ingredient-name-matching --strict` を実行する。完了判定: エラー 0 件
- [x] 6.9 `task build:win` で Windows ビルドを作り、S04 の材料名に IME で「たま」と入力して候補を選べることと、変換確定の Enter で候補が確定しないことを確かめる。
  完了判定: 目視で確認する
