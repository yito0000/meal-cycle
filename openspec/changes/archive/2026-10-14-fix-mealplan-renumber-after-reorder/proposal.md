## Why

S01 で 3 品以上あるセルの並び順を変えたあと、上のほうの品を削除すると失敗する
(docs/reviews/2026-10-14-design-ux-review.md R21)。
画面には汎用のエラー (「処理に失敗しました」) が出るだけで、利用者には原因がわからない。

削除のあとに後ろの品の並び順を 1 つずつ詰める処理が、行を並び順ではなく登録した順に動かすためである。
並べ替えで登録順と並び順が食い違ったセルでは、後ろの品が先に動き、まだ詰めていない品と並び順が重なって拒否される。
同じ詰め直しは、ドラッグ&ドロップでの移動と「作り置きから外す」にもある。

「上へ移動」「下へ移動」が入り、主菜、副菜、汁物の 3 品を並べ替える場面が増えた。早めに止める。

## What Changes

- 並べ替えたセルでも、品の削除、別セルへの移動、作り置きの解体による行の削除が成功するようにする
- 削除や移動のあと、残った品は相対的な順を保ったまま、並び順を 0 から隙間なく振り直す
- 並び順の重複による失敗を「存在しないレシピ」と取り違えないようにする (R27 の 1 点目)
- 使われていない宣言を 1 つ消す (R27 の 3 点目)

### Non-goals

- **献立の行に ID を持たせる**：並び順を主キーから外す根本対応は `add-mealplan-entry-id` で扱う (R25)
- **エラーコードの対応表の一本化**：R27 の 2 点目 (フロントが文言の部分一致でエラーを引く) は扱わない
- **エラーの見せ方**：赤い帯をトーストに変えるのは `revise-mealplan-destructive-actions` で扱う (R30)

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `mealplan`：並べ替えたあとのセルでも、品の削除と移動が成功し、残りの品が隙間なく並ぶことを要件として足す

## Impact

### 新設ファイル

- なし

### 更新ファイル

- `internal/storage/sqlite/mealplan_repo.go`
  - 並び順を詰める処理を `closePositionGap` に切り出し、`removeRowTx` と `SplitBatch` から呼ぶ
  - `Reorder` の退避幅 `1000` を定数 `positionShiftOffset` にして共有する
  - `Remove` のコメントの「小さい position から順に上書きするので衝突しない」を消す
  - `isForeignKeyErr` を外部キー違反だけに絞る
- `internal/storage/sqlite/mealplan_repo_test.go`：並べ替えたあとの削除、移動、作り置きの解体のテストを足す
- `internal/service/mealplan.go`：末尾の `var _ = errors.Is` と `errors` の import を消す
- `frontend/e2e/mealplan-assign.spec.ts`：describe「並べ替えたあとの品の削除と移動」を足す
- `docs/reviews/2026-10-14-design-ux-review.md`：R21 にチェックを入れ、R27 に対応した範囲を追記する
- `openspec/specs/mealplan/spec.md`：archive 時に delta を合成する (手で直接は触らない)

### 生成物 (再生成のみ)

- なし。サービスのメソッドと引数は変えない

### migration

- なし

### 前提と順序

- レビュー由来の 7 つの change のうち、最初に apply する。ほかの change に依存しない

### 後続 proposal への引き継ぎ

- `revise-mealplan-destructive-actions`：削除した品を元の位置へ戻すとき、後ろの品を 1 つずつ下げる。`closePositionGap` と同じ退避の方式を使う
- `add-mealplan-entry-id`：行に ID を足したあとも並び順の一意性は保つので、`closePositionGap` はそのまま残る
