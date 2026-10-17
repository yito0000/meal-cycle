## Why

S04 で材料の量に負の値を入れると、そのまま保存される (docs/reviews/2026-10-15-design-ux-review.md R41)。
量の欄は `type="number" min={0}` だが、`min` は入力を止めないので `-1` は打てる。
Go 側の保存前の検証も、材料については名前しか見ていない。
冷蔵庫の在庫の追加と編集は量を「0 より大きく 99999 以下」で検証し、レシピのインポートも負の量を弾いており、手入力の経路だけが抜けている。
負の量は S07 の合算で必要量を減らす方向に効くので、買い物リストの数字が黙って狂う。

量の欄に `1e400` のような大きすぎる値を入れた場合も、画面では無限大になり、保存の要求では量 0 (量を持たない材料) に化ける。

## What Changes

- 材料の量を「0 以上 99999 以下の有限の数」に制限する。量 0 は従来どおり「量を持たない材料」(「塩 少々」など) を表す
  - S04 は保存時に検証し、違反した材料行の量の欄の近くにエラーを出す
  - 画面を経由しない保存の要求 (登録、編集、インポートの取り込み) も、サービスで拒否する
  - インポートのプレビューでは、上限を超える量を既存の「量の書き方が正しくない」問題として示す

### Non-goals

- **すでに保存されている負の量の修正**：migration で書き換えない。該当するレシピを編集して保存するときに、検証で気づける
- **量の小数の桁数の制限**：扱わない。インポートは小数第 2 位で丸めているが、S04 の手入力は丸めない (今の挙動のまま)
- **単位の長さの制限**：扱わない

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `recipe`：
  - 「入力バリデーション」の制約に、材料の量の範囲を足す

## Impact

### 新設ファイル

- なし

### 更新ファイル

- `internal/service/recipe.go`
  - 量の上限の定数 `recipeAmountMax` と、エラー `ErrRecipeIngredientAmountOutOfRange` を足す
  - `validateRecipeInput` で量を検証する
  - `recipeImportIssueByErr` に、このエラーを `recipe.ImportIssueAmountInvalid` に引く対応を足し、`importIssueFromErr` で該当する材料名を添える
- `internal/service/recipe_test.go`：`Create`、`Update`、`Import`、`PreviewImport` のテストを足す
- `frontend/src/components/recipe/recipe-edit-form.tsx`
  - `validate` で量を検証し、材料行ごとの量のエラーを持つ
  - `mapErrorToI18nKey` に `ingredient amount out of range` の対応を足す
- `frontend/src/components/recipe/recipe-ingredient-row.tsx`：`amountError` を受け取り、量の欄の近くに出す
- `frontend/src/locales/ja.json` / `frontend/src/locales/en.json`：`pages.recipes.edit.save_error.amount_out_of_range` を足す
- `frontend/e2e/recipe-edit.spec.ts`：describe「入力バリデーション」にシナリオを足す
- `docs/product.md`：§4 S04 の材料行の説明に、量の範囲を書き足す
- `docs/reviews/2026-10-15-design-ux-review.md`：R41 にチェックを入れる
- `openspec/specs/recipe/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- なし (サービスのメソッドと型は変わらない)

### migration

- なし

### 前提と順序

- ほかの 2 つの change (`fix-mealplan-batch-boundaries`、`revise-mealplan-spread-safety`) とは触るファイルが重ならない。順序は問わない

### 後続 proposal への引き継ぎ

- 量の上限 (99999) は、冷蔵庫の在庫の上限 (`stockAmountMax`) と同じ値を別の定数で持つ。S07 の一括登録は必要量を在庫の量の初期値にするので、
  2 つの上限を変えるときは一緒に変える
