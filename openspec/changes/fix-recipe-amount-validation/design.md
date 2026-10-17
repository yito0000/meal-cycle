## Context

動機は proposal.md の Why を参照。

レシピの保存前の検証は、Go 側の `validateRecipeInput` (最終防衛線) と、S04 の `validate` (即時の表示用) の 2 か所に同じ規則を書いている。
どちらも材料については名前しか見ていない。
S04 は量を文字列で持ち、保存時に `Number()` で数に直す。空は 0 にする。
`Number("1e400")` は `Infinity` になり、JSON にすると `null` になるので、Go 側では 0 として受け取られる。

インポートは `ParseAmount` で負の値と非有限の値を「量の書き方が正しくない」(`ImportIssueAmountInvalid`) として弾くが、上限は無い。
冷蔵庫の在庫は、サービスの定数 `stockAmountMax = 99999` で上限を持つ。

## Goals / Non-Goals

**Goals:**

- 材料の量を 0 以上 99999 以下の有限の数に制限し、Go 側と S04 の両方で検証する
- インポートの上限超えを、既存の問題コードで示す

**Non-Goals:**

- 保存済みのデータの修正 (proposal の Non-goals を参照)

## Decisions

### D1. 上限の値と置き場所

**選択肢:**

- (a) `internal/service/recipe.go` に `recipeAmountMax = 99999` を置く ← 採用 — Pros: 既存の `recipeServingsMax` などと同じ「実装上限」の並びに入る。
  冷蔵庫の在庫の上限と同じ値なので、S07 の一括登録で必要量をそのまま在庫に入れても上限に当たりにくい / Cons: 在庫の上限と別の定数になる
- (b) `internal/config/defaults.go` に置く — Pros: 1 か所に集まる / Cons: 設定画面で利用者が変える値ではない。
  defaults.go には利用者が変える値だけを置く方針 (CLAUDE.md「可変値は 1 箇所に集約」) に反する
- (c) 在庫の `stockAmountMax` を共有する — Pros: 値が 1 つになる / Cons: 2 つの capability のサービスが互いの実装上限に依存する

**採用: (a)**。2 つの上限を一緒に変える必要があることは、proposal の「後続 proposal への引き継ぎ」に残す。

### D2. 量 0 と空の扱い

量 0 は「量を持たない材料」として既存の spec (S03 の表示、S07 の「家にあるか確かめる材料」) が使っている。
下限は 0 を含める (`0 <= amount`)。冷蔵庫の在庫 (`0 < amount`) とは下限が違う。
S04 の空欄は従来どおり 0 として送る。

### D3. S04 での検出

**選択肢:**

- (a) `validate` で、空でない量の文字列を `Number()` に通し、`Number.isFinite` で有限かと、範囲内かを確かめる ← 採用
  — Pros: `1e400` の `Infinity` も弾ける。既存の名前の検証と同じく、材料行ごとのエラーの配列に載せられる / Cons: なし
- (b) `<input>` の `min` / `max` と `checkValidity()` に任せる — Pros: コードが少ない /
  Cons: 量の欄は入力途中の文字列を保つ作り (fix-recipe-form-and-list D3) で、ブラウザの検証メッセージは i18n を通らない

**採用: (a)**。エラーは `ingredientAmounts` の配列として `FieldErrors` に足し、`RecipeIngredientRow` の `amountError` に渡す。
名前のエラーと同時に出ることがあるので、既存の `ingredients` (名前のエラー) とは別の配列にする。

### D4. Go 側のエラーと問題コード

`ErrRecipeIngredientAmountOutOfRange = errors.New("recipe ingredient amount out of range")` を足す。
`math.IsNaN` と `math.IsInf` も同じエラーにする (JSON からは来ないが、`Import` は画面の値を受け取るので念のため弾く)。
インポートでは、`recipeImportIssueByErr` でこのエラーを `ImportIssueAmountInvalid` に引き、`importIssueFromErr` で範囲外の量を持つ最初の材料名を `IssueArg` に入れる。
既存の文言「材料「{{arg}}」の量を読み取れません」は上限超えにも通じるので、新しい問題コードは足さない。

**他案:** 上限超え専用の問題コード (`amount_out_of_range`) を足す案も考えたが、インポートで 99999 を超える量を書く場面はまず無く、文言を分ける利点が小さい。

## Risks / Trade-offs

- [Risk] すでに負の量や 99999 を超える量で保存されたレシピは、編集して保存しようとすると弾かれる
  → 弾かれた材料行の量の欄にエラーが出るので、その場で直せる。S03 の表示と S07 の計算はこれまでどおり動く
- [Risk] Go 側と S04 に同じ規則が 2 か所で書かれる → 既存のほかの制約と同じ持ち方であり、範囲の値はテストで両方を確かめる

## Migration Plan

スキーマとデータは変わらない。既存の DB はそのまま使える。
