## 1. サービス層

- [x] 1.1 `internal/service/recipe_test.go` の `TestRecipeService_Create` と `TestRecipeService_Update` にテストを足す (更新)。
  足した時点で落ちることを `go test ./internal/service/ -run 'TestRecipeService_(Create|Update)'` で確かめる
  - should reject an ingredient with a negative amount
  - should reject an ingredient whose amount exceeds 99999
  - should accept an ingredient whose amount is 0
  - should accept an ingredient whose amount is exactly 99999
  - should leave the recipe unchanged when an update has an out-of-range amount
- [x] 1.2 `internal/service/recipe_test.go` の `TestRecipeService_Import` と `TestRecipeService_PreviewImport` にテストを足す (更新)
  - should reject the whole import when one recipe has a negative amount
  - should report amount_invalid with the ingredient name when a file has an amount over 99999
- [x] 1.3 `internal/service/recipe.go` を更新し、定数 `recipeAmountMax = 99999` とエラー `ErrRecipeIngredientAmountOutOfRange` を足す。
  `validateRecipeInput` で、各材料の量が `0 <= amount <= recipeAmountMax` で有限 (`math.IsNaN` / `math.IsInf` でない) かを確かめる。
  1.1 のテストが通ることで確かめる
- [x] 1.4 `internal/service/recipe.go` の `recipeImportIssueByErr` に `{ErrRecipeIngredientAmountOutOfRange, recipe.ImportIssueAmountInvalid}` を足し、
  `importIssueFromErr` で範囲外の量を持つ最初の材料名を `IssueArg` に入れる。1.2 のテストが通ることで確かめる

## 2. フロントエンド

- [x] 2.1 `frontend/src/locales/ja.json` と `frontend/src/locales/en.json` を更新し、`pages.recipes.edit.save_error.amount_out_of_range`
  (「量は 0 以上 99999 以下の数で入力してください」) を足す。`npm run type-check` で確かめる
- [x] 2.2 `frontend/src/components/recipe/recipe-ingredient-row.tsx` を更新し、`amountError?: string | null` を受け取って、
  量の欄に `aria-invalid` を付け、行の下にエラー文を出す (名前のエラーと同じ出し方)。`npm run type-check` で確かめる
- [x] 2.3 `frontend/src/components/recipe/recipe-edit-form.tsx` を更新し、`FieldErrors` に `ingredientAmounts?: (string | undefined)[]` を足す。
  `validate` で、空でない量の文字列を `Number()` に通した値が有限で 0 以上 99999 以下かを確かめる。
  `handleSave` は数に直す前の文字列で判定できるよう、`validate` に量の文字列を渡す形にする。`npm run type-check` で確かめる
- [x] 2.4 `frontend/src/components/recipe/recipe-edit-form.tsx` を更新し、`RecipeIngredientRow` に `amountError={fieldErrors.ingredientAmounts?.[i] ?? null}` を渡し、
  `mapErrorToI18nKey` に `ingredient amount out of range` → `pages.recipes.edit.save_error.amount_out_of_range` を足す。`npm run type-check` で確かめる

## 3. E2E

- [x] 3.1 `frontend/e2e/recipe-edit.spec.ts` に describe「入力バリデーション」を足す (更新)。`task test:e2e` で通ることを確かめる
  - 材料の量が負の値だと保存できない
  - 材料の量が上限を超えると保存できない
  - 量 0 と空の量は保存できる

サービス層で確かめるシナリオ (E2E には含めない):

- 「画面を経由しない保存の要求でも範囲外の量は拒否される」(1.1、1.2)

## 4. ドキュメント

- [x] 4.1 `docs/product.md` の §4 S04 の材料行の箇条に、「量は 0 以上 99999 以下。空と 0 は量を持たない材料」を書き足す。`git diff docs/product.md` で確かめる
- [x] 4.2 `docs/reviews/2026-10-15-design-ux-review.md` の R41 にチェックを入れ、proposal 名 `fix-recipe-amount-validation` を添える。目視で確かめる

## 5. 確認

- [x] 5.1 `go test ./internal/... ./cmd/...` を実行し、すべて通ることを確かめる
- [x] 5.2 `cd frontend && npm run type-check` を実行し、エラーが無いことを確かめる
- [x] 5.3 `cd frontend && npm run build:dev` を実行し、ビルドが通ることを確かめる
- [x] 5.4 `task test:e2e` を実行し、すべて通ることを確かめる (`check:scenarios` を含む)
- [x] 5.5 `task lint` を実行し、エラーが無いことを確かめる
